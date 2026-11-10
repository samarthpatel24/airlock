package main

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// Formats are allowed by extension AND content. Anything else is refused.
var (
	metaExts   = map[string]bool{".json": true, ".txt": true, ".md": true, ".jinja": true, ".model": true}
	pickleExts = map[string]bool{".bin": true, ".pt": true, ".pth": true, ".pkl": true, ".pickle": true,
		".ckpt": true, ".joblib": true, ".npy": true, ".npz": true}
)

const (
	maxMetaSize       = 50 << 20  // tokenizer/config files are small; big ones are suspicious
	maxSafetensorsHdr = 100 << 20 // spec limit for the JSON header
)

func checkCmd(args []string) error {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() != 1 {
		return errors.New("usage: airlock check <dir>")
	}
	files, err := listFiles(fs.Arg(0))
	if err != nil {
		return err
	}
	if err := needWeights(files); err != nil {
		return err
	}
	fmt.Printf("✔ %d files: safe formats only\n", len(files))
	return nil
}

func needWeights(files []FileEntry) error {
	for _, f := range files {
		if f.Kind == "weights" {
			return nil
		}
	}
	return errors.New("no model weights found (.gguf or .safetensors)")
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// inspect reads a file exactly once: the bytes that pass the format check
// are the same bytes that get hashed.
func inspect(path string) (sum string, size int64, kind string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", 0, "", err
	}
	h := sha256.New()
	cr := &countingReader{r: f}
	kind, err = checkFormat(io.TeeReader(cr, h), strings.ToLower(filepath.Ext(path)), st.Size())
	if err != nil {
		return "", 0, "", err
	}
	if _, err := io.Copy(h, cr); err != nil {
		return "", 0, "", err
	}
	if cr.n != st.Size() {
		return "", 0, "", errors.New("file changed while being read")
	}
	return hex.EncodeToString(h.Sum(nil)), cr.n, kind, nil
}

// checkFormat returns "weights" or "meta", or an error for pickle-based or
// unrecognised content. It reads from r, which may stop partway.
func checkFormat(r io.Reader, ext string, size int64) (string, error) {
	if pickleExts[ext] {
		return "", errors.New("pickle-based format refused: it can execute code when loaded; convert to safetensors or GGUF")
	}
	switch ext {
	case ".gguf":
		var hdr [8]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil || string(hdr[:4]) != "GGUF" {
			return "", errors.New("named .gguf but missing GGUF header")
		}
		if v := binary.LittleEndian.Uint32(hdr[4:]); v < 1 || v > 3 {
			return "", fmt.Errorf("unsupported GGUF version %d", v)
		}
		return "weights", nil
	case ".safetensors":
		return "weights", checkSafetensors(r, size)
	}
	if !metaExts[ext] {
		return "", fmt.Errorf("unknown format %q not allowed", ext)
	}
	if size > maxMetaSize {
		return "", fmt.Errorf("metadata file unexpectedly large (%d bytes)", size)
	}
	b, err := io.ReadAll(io.LimitReader(r, maxMetaSize))
	if err != nil {
		return "", err
	}
	if looksLikePickle(b) {
		return "", errors.New("pickle or zip content under a metadata name refused")
	}
	switch ext {
	case ".json":
		if !json.Valid(b) {
			return "", errors.New("named .json but not valid JSON")
		}
	case ".model": // sentencepiece ModelProto: starts with field 1 (pieces), wire type 2
		if len(b) == 0 || b[0] != 0x0a {
			return "", errors.New("named .model but not a sentencepiece model")
		}
	default: // .txt .md .jinja: plain text only (also rules out ASCII pickles' binary siblings)
		if !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
			return "", errors.New("text file contains binary data")
		}
	}
	return "meta", nil
}

// checkSafetensors validates the header and that the tensors exactly cover
// the data section, so nothing (a zip, a pickle) can hide after them.
func checkSafetensors(r io.Reader, size int64) error {
	var lb [8]byte
	if _, err := io.ReadFull(r, lb[:]); err != nil {
		return errors.New("safetensors file too short")
	}
	n := binary.LittleEndian.Uint64(lb[:])
	if n == 0 || n > maxSafetensorsHdr || int64(n)+8 > size {
		return errors.New("invalid safetensors header length")
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	var hdr map[string]json.RawMessage
	if buf[0] != '{' || json.Unmarshal(bytes.TrimRight(buf, " "), &hdr) != nil {
		return errors.New("invalid safetensors JSON header")
	}
	dataLen := size - 8 - int64(n)
	var end int64
	for k, raw := range hdr {
		if k == "__metadata__" {
			continue
		}
		var t struct {
			Offsets []int64 `json:"data_offsets"`
		}
		if json.Unmarshal(raw, &t) != nil || len(t.Offsets) != 2 {
			return fmt.Errorf("tensor %q: bad data_offsets", k)
		}
		b, e := t.Offsets[0], t.Offsets[1]
		if b < 0 || e < b || e > dataLen {
			return fmt.Errorf("tensor %q: offsets outside the file", k)
		}
		end = max(end, e)
	}
	if end != dataLen {
		return errors.New("safetensors has data not covered by any tensor (possible hidden payload)")
	}
	return nil
}

// looksLikePickle catches binary pickles (protocol 2+ starts with 0x80) and
// torch.save zip archives. Only used on non-weight files: a safetensors
// header length can legitimately start with these bytes.
func looksLikePickle(b []byte) bool {
	if len(b) >= 2 && b[0] == 0x80 && b[1] >= 2 && b[1] <= 5 {
		return true
	}
	return bytes.HasPrefix(b, []byte("PK\x03\x04"))
}

// --- provenance: compare local files against a pinned Hugging Face commit ---

var (
	hfBase   = "https://huggingface.co" // overridden in tests
	commitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
	repoRe   = regexp.MustCompile(`^[A-Za-z0-9][\w.-]*/[A-Za-z0-9][\w.-]*$`) // no "." or ".." segments
)

type hfSibling struct {
	Name   string `json:"rfilename"`
	BlobID string `json:"blobId"`
	LFS    *struct {
		SHA256 string `json:"sha256"`
	} `json:"lfs"`
}

// checkHF fails unless every local file exists in the upstream repo at that
// exact commit with the same hash. Branch names are refused: they can move.
func checkHF(dir, repo, rev string, files []FileEntry) error {
	if !repoRe.MatchString(repo) {
		return fmt.Errorf("invalid repo %q", repo)
	}
	if !commitRe.MatchString(rev) {
		return fmt.Errorf("revision must be a full 40-char commit hash, got %q", rev)
	}
	u := fmt.Sprintf("%s/api/models/%s/revision/%s?blobs=true", hfBase, repo, url.PathEscape(rev))
	client := http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("redirect refused (repo renamed? use its current name)")
	}}
	resp, err := client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("hugging face: HTTP %d", resp.StatusCode) // never the server's status text
	}
	var info struct {
		SHA      string      `json:"sha"`
		Siblings []hfSibling `json:"siblings"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 10<<20)).Decode(&info); err != nil {
		return err
	}
	if info.SHA != rev {
		return fmt.Errorf("hugging face returned commit %s, want %s", info.SHA, rev)
	}
	upstream := map[string]hfSibling{}
	for _, s := range info.Siblings {
		upstream[s.Name] = s
	}
	for _, f := range files {
		s, ok := upstream[f.Path]
		if !ok {
			return fmt.Errorf("%q: not in %s@%s", f.Path, repo, rev[:12])
		}
		if s.LFS != nil {
			if s.LFS.SHA256 != f.SHA256 {
				return fmt.Errorf("%q: sha256 differs from upstream", f.Path)
			}
			continue
		}
		// ponytail: small non-LFS files are matched by git's SHA-1 blob id, the
		// only hash HF publishes for them; fetch and sha256 them if that matters.
		sum, err := gitBlobSHA1(filepath.Join(dir, filepath.FromSlash(f.Path)))
		if err != nil {
			return err
		}
		if sum != s.BlobID {
			return fmt.Errorf("%q: git blob hash differs from upstream", f.Path)
		}
	}
	return nil
}

// gitBlobSHA1 is how git names small (non-LFS) files: sha1("blob <len>\0" + data).
func gitBlobSHA1(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "", err
	}
	h := sha1.New()
	fmt.Fprintf(h, "blob %d\x00", st.Size())
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
