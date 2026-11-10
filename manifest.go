package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	manifestName  = "manifest.json"
	signatureName = "manifest.sigstore.json"
)

// Manifest lists every file in a bundle with its hash. It is the only thing
// that gets signed; files are trusted only through it.
type Manifest struct {
	Version int         `json:"version"`
	Tool    string      `json:"tool"`
	Created string      `json:"created"`
	Model   *ModelInfo  `json:"model,omitempty"`
	Checks  []string    `json:"checks"`
	Files   []FileEntry `json:"files"`
}

type ModelInfo struct {
	Repo     string `json:"repo"`
	Revision string `json:"revision"` // pinned commit, never a branch
}

type FileEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"-"` // "weights" or "meta", set when read from disk
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), n, err
}

// listFiles format-checks and hashes every file under dir (one read each),
// except airlock's own manifest and signature. Symlinks and other
// non-regular files are rejected outright.
func listFiles(dir string) ([]FileEntry, error) {
	var out []FileEntry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		if rel == manifestName || rel == signatureName {
			return nil
		}
		if !safeName(rel) {
			return fmt.Errorf("%q: unsafe file name (control characters, backticks, '|' or ':')", rel)
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%q: not a regular file (symlinks are refused)", rel)
		}
		sum, size, kind, err := inspect(p)
		if err != nil {
			return fmt.Errorf("%q: %w", rel, err)
		}
		out = append(out, FileEntry{Path: filepath.ToSlash(rel), Size: size, SHA256: sum, Kind: kind})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, err
}

// checkFiles requires the bundle on disk to match the manifest exactly:
// no missing files, no extra files, same sizes and hashes. It returns what it
// read from disk.
func checkFiles(dir string, m *Manifest) ([]FileEntry, error) {
	onDisk, err := listFiles(dir)
	if err != nil {
		return nil, err
	}
	want := map[string]FileEntry{}
	for _, f := range m.Files {
		want[f.Path] = f
	}
	for _, f := range onDisk {
		w, ok := want[f.Path]
		if !ok {
			return nil, fmt.Errorf("unexpected file not in manifest: %q", f.Path)
		}
		if w.Size != f.Size || w.SHA256 != f.SHA256 {
			return nil, fmt.Errorf("hash mismatch: %q (tampered or corrupted)", f.Path)
		}
		delete(want, f.Path)
	}
	for p := range want {
		return nil, fmt.Errorf("missing file: %q", p)
	}
	return onDisk, nil
}

// safeName refuses names that could fake lines in the report or send
// escape sequences to a terminal.
func safeName(s string) bool {
	if !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if !unicode.IsPrint(r) || strings.ContainsRune("`|:", r) {
			return false
		}
	}
	return true
}

const maxReserved = 16 << 20 // manifest and signature are small

// readReserved reads airlock's own manifest or signature, refusing symlinks,
// devices and oversized files.
func readReserved(dir, name string) ([]byte, error) {
	p := filepath.Join(dir, name)
	st, err := os.Lstat(p)
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > maxReserved {
		return nil, fmt.Errorf("%s: not a regular file of sane size", name)
	}
	return os.ReadFile(p)
}

// clearReserved removes an old manifest and signature before sealing, and
// refuses if either is a symlink, so a planted link can't redirect the write.
func clearReserved(dir string) error {
	for _, n := range []string{manifestName, signatureName} {
		p := filepath.Join(dir, n)
		st, err := os.Lstat(p)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() {
			return fmt.Errorf("%s exists and is not a regular file; refusing to write through it", n)
		}
		if err := os.Remove(p); err != nil {
			return err
		}
	}
	return nil
}

// writeJSON creates path exclusively: it never follows or overwrites an existing file.
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
