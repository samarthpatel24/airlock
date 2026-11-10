package main

import (
	"bytes"
	"crypto/sha1"
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
)

// Formats are allowed by content (magic bytes), not by file name alone.
var (
	weightExts = map[string]bool{".gguf": true, ".safetensors": true}
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
	if err := checkFormats(fs.Arg(0), files); err != nil {
		return err
	}
	fmt.Printf("✔ %d files: safe formats only\n", len(files))
	return nil
}

func checkFormats(dir string, files []FileEntry) error {
	weights := 0
	for _, f := range files {
		kind, err := checkModelFile(filepath.Join(dir, filepath.FromSlash(f.Path)), f.Size)
		if err != nil {
			return fmt.Errorf("%s: %w", f.Path, err)
		}
		if kind == "weights" {
			weights++
		}
	}
	if weights == 0 {
		return errors.New("no model weights found (.gguf or .safetensors)")
	}
	return nil
}

// checkModelFile returns "weights" or "meta", or an error if the file is a
// pickle-based format (can run code when loaded) or anything unrecognised.
func checkModelFile(path string, size int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	head := make([]byte, 8)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	ext := strings.ToLower(filepath.Ext(path))

	if pickleExts[ext] || looksLikePickle(head) {
		return "", errors.New("pickle-based format refused: it can execute code when loaded; convert to safetensors or GGUF")
	}
	switch ext {
	case ".gguf":
		if !bytes.HasPrefix(head, []byte("GGUF")) {
			return "", errors.New("named .gguf but missing GGUF header")
		}
		return "weights", nil
	case ".safetensors":
		if len(head) < 8 {
			return "", errors.New("safetensors file too short")
		}
		hdr := binary.LittleEndian.Uint64(head)
		if hdr == 0 || hdr > maxSafetensorsHdr || int64(hdr)+8 > size {
			return "", errors.New("invalid safetensors header length")
		}
		buf := make([]byte, hdr)
		if _, err := io.ReadFull(f, buf); err != nil {
			return "", err
		}
		if buf[0] != '{' || !json.Valid(bytes.TrimRight(buf, " ")) {
			return "", errors.New("invalid safetensors JSON header")
		}
		return "weights", nil
	}
	if metaExts[ext] {
		if size > maxMetaSize {
			return "", fmt.Errorf("metadata file unexpectedly large (%d bytes)", size)
		}
		if ext == ".json" { // also catches old ASCII pickles (protocol 0/1) renamed to .json
			b, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			if !json.Valid(b) {
				return "", errors.New("named .json but not valid JSON")
			}
		}
		return "meta", nil
	}
	return "", fmt.Errorf("unknown format %q not allowed", ext)
}

// looksLikePickle catches renamed pickles: protocol 2+ starts with 0x80, and
// torch.save writes a zip archive that contains a pickle.
func looksLikePickle(head []byte) bool {
	if len(head) >= 2 && head[0] == 0x80 && head[1] >= 2 && head[1] <= 5 {
		return true
	}
	return bytes.HasPrefix(head, []byte("PK\x03\x04"))
}

// --- provenance: compare local files against a pinned Hugging Face commit ---

var (
	hfBase   = "https://huggingface.co" // overridden in tests
	commitRe = regexp.MustCompile(`^[0-9a-f]{40}$`)
	repoRe   = regexp.MustCompile(`^[A-Za-z0-9][\w.-]*/[\w.-]+$`)
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
	client := http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("hugging face: %s", resp.Status)
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
			return fmt.Errorf("%s: not in %s@%s", f.Path, repo, rev[:12])
		}
		if s.LFS != nil {
			if s.LFS.SHA256 != f.SHA256 {
				return fmt.Errorf("%s: sha256 differs from upstream", f.Path)
			}
			continue
		}
		sum, err := gitBlobSHA1(filepath.Join(dir, filepath.FromSlash(f.Path)))
		if err != nil {
			return err
		}
		if sum != s.BlobID {
			return fmt.Errorf("%s: git blob hash differs from upstream", f.Path)
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
