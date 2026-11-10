package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// verify runs on the offline side with nothing but this binary and the
// public key. Order matters: the signature is checked before the manifest is
// even parsed, and files are trusted only through the signed manifest.
func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	key := fs.String("key", "", "cosign public key (required)")
	out := fs.String("report", "airlock-report.md", "where to write the report (outside the bundle)")
	dir, err := parseDir(fs, args)
	if err != nil {
		return err
	}
	if *key == "" {
		return errors.New("--key is required")
	}
	r := report{Dir: dir}
	err = doVerify(dir, *key, &r)
	if werr := r.write(*out, err); werr != nil {
		return werr
	}
	if err != nil {
		return fmt.Errorf("FAIL — %w (report: %s)", err, *out)
	}
	fmt.Printf("✔ PASS — report: %s\n", *out)
	return nil
}

func doVerify(dir, keyPath string, r *report) error {
	pub, fp, err := loadPublicKey(keyPath)
	if err != nil {
		return err
	}
	r.KeyFP = fp
	mb, err := os.ReadFile(filepath.Join(dir, manifestName))
	if err != nil {
		return err
	}
	sb, err := os.ReadFile(filepath.Join(dir, signatureName))
	if err != nil {
		return err
	}
	if err := verifySignature(pub, mb, sb); err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	r.pass("signature valid, manifest signed by key " + fp)

	var m Manifest
	if err := json.Unmarshal(mb, &m); err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	r.M = &m
	if err := checkFiles(dir, &m); err != nil {
		return err
	}
	r.pass(fmt.Sprintf("all %d files match the manifest (no missing, extra, or changed files)", len(m.Files)))
	if err := checkFormats(dir, m.Files); err != nil {
		return err
	}
	r.pass("formats re-checked offline: no pickle, no unknown files")
	return nil
}

func loadPublicKey(path string) (*ecdsa.PublicKey, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	blk, _ := pem.Decode(b)
	if blk == nil || blk.Type != "PUBLIC KEY" {
		return nil, "", errors.New("public key: not a PEM PUBLIC KEY")
	}
	k, err := x509.ParsePKIXPublicKey(blk.Bytes)
	if err != nil {
		return nil, "", err
	}
	pub, ok := k.(*ecdsa.PublicKey)
	if !ok {
		return nil, "", errors.New("public key: only ECDSA (cosign default) is supported")
	}
	sum := sha256.Sum256(blk.Bytes)
	return pub, "sha256:" + hex.EncodeToString(sum[:]), nil
}

// sigBundle is the part of a Sigstore bundle (as written by cosign
// sign-blob --key) that a key-based offline check needs.
type sigBundle struct {
	MediaType        string `json:"mediaType"`
	MessageSignature struct {
		MessageDigest struct {
			Algorithm string `json:"algorithm"`
			Digest    []byte `json:"digest"`
		} `json:"messageDigest"`
		Signature []byte `json:"signature"`
	} `json:"messageSignature"`
}

func verifySignature(pub *ecdsa.PublicKey, msg, bundle []byte) error {
	var b sigBundle
	if err := json.Unmarshal(bundle, &b); err != nil {
		return err
	}
	if !strings.HasPrefix(b.MediaType, "application/vnd.dev.sigstore.bundle") {
		return errors.New("not a sigstore bundle")
	}
	if b.MessageSignature.MessageDigest.Algorithm != "SHA2_256" {
		return errors.New("unsupported digest algorithm")
	}
	d := sha256.Sum256(msg)
	if subtle.ConstantTimeCompare(d[:], b.MessageSignature.MessageDigest.Digest) != 1 {
		return errors.New("manifest does not match the signed digest (tampered)")
	}
	if !ecdsa.VerifyASN1(pub, d[:], b.MessageSignature.Signature) {
		return errors.New("invalid signature (wrong key or tampered)")
	}
	return nil
}

// --- report for the security team ---

type report struct {
	Dir    string
	KeyFP  string
	M      *Manifest // set only after the signature verified
	Passed []string
}

func (r *report) pass(s string) { r.Passed = append(r.Passed, s) }

func (r *report) write(path string, failure error) error {
	var b strings.Builder
	result := "PASS"
	if failure != nil {
		result = "FAIL"
	}
	abs, _ := filepath.Abs(r.Dir)
	fmt.Fprintf(&b, "# airlock verification report\n\n")
	fmt.Fprintf(&b, "- **Result:** %s\n- **Bundle:** `%s`\n- **Verified at:** %s\n- **Tool:** airlock %s\n",
		result, abs, time.Now().UTC().Format(time.RFC3339), version)
	if r.KeyFP != "" {
		fmt.Fprintf(&b, "- **Trusted key:** `%s`\n", r.KeyFP)
	}
	if r.M != nil && r.M.Model != nil {
		fmt.Fprintf(&b, "- **Model:** %s @ `%s`\n", r.M.Model.Repo, r.M.Model.Revision)
	}
	fmt.Fprintf(&b, "\n## Offline checks\n\n")
	for _, p := range r.Passed {
		fmt.Fprintf(&b, "- ✔ %s\n", p)
	}
	if failure != nil {
		fmt.Fprintf(&b, "- ✘ %s\n", failure)
	}
	if r.M != nil {
		fmt.Fprintf(&b, "\n## Checks recorded when sealed (covered by the signature)\n\n")
		for _, c := range r.M.Checks {
			fmt.Fprintf(&b, "- %s\n", c)
		}
		fmt.Fprintf(&b, "\nSealed %s by %s.\n\n## Files\n\n| File | Size | SHA-256 |\n|---|---|---|\n", r.M.Created, r.M.Tool)
		for _, f := range r.M.Files {
			fmt.Fprintf(&b, "| `%s` | %d | `%s` |\n", f.Path, f.Size, f.SHA256)
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
