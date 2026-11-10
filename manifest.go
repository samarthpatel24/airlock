package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
