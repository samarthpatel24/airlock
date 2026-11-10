package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Each case here deletes a guard's reason to exist: if the guard is removed,
// the matching case fails.

func TestVerifyGuards(t *testing.T) {
	bundle, pub, mb := sealedBundle(t)

	os.Remove(filepath.Join(bundle, "tok/config.json"))
	if err := doVerify(bundle, pub, &report{}); err == nil || !strings.Contains(err.Error(), "missing file") {
		t.Errorf("deleted file: %v", err)
	}
	write(t, filepath.Join(bundle, "tok/config.json"), []byte(`{}`))

	rep := filepath.Join(t.TempDir(), "r.md")
	os.Symlink(filepath.Join(t.TempDir(), "target"), rep)
	if err := verify([]string{bundle, "--key", pub, "--report", rep}); err == nil {
		t.Error("report written through a symlink")
	}

	sig := filepath.Join(bundle, signatureName)
	good, _ := os.ReadFile(sig)
	for label, edit := range map[string]func(*sigBundle){
		"media type": func(b *sigBundle) { b.MediaType = "x" },
		"algorithm":  func(b *sigBundle) { b.MessageSignature.MessageDigest.Algorithm = "SHA2_512" },
	} {
		var b sigBundle
		json.Unmarshal(good, &b)
		edit(&b)
		out, _ := json.Marshal(b)
		write(t, sig, out)
		if err := doVerify(bundle, pub, &report{}); err == nil {
			t.Errorf("%s: accepted", label)
		}
	}
	write(t, sig, good)
	_ = mb

	rk, _ := rsa.GenerateKey(rand.Reader, 2048)
	der, _ := x509.MarshalPKIXPublicKey(&rk.PublicKey)
	rsaPub := filepath.Join(t.TempDir(), "rsa.pub")
	write(t, rsaPub, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	priv := filepath.Join(t.TempDir(), "priv.pem")
	write(t, priv, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	for _, k := range []string{rsaPub, priv} {
		if err := doVerify(bundle, k, &report{}); err == nil {
			t.Errorf("%s accepted as the trusted key", filepath.Base(k))
		}
	}
}

func TestNoWeights(t *testing.T) {
	keys, bundle := t.TempDir(), t.TempDir()
	k := newKey()
	pub := pubPEM(t, keys, k)
	write(t, filepath.Join(bundle, "c.json"), []byte(`{}`))
	files, _ := listFiles(bundle, nil)
	writeJSON(filepath.Join(bundle, manifestName), &Manifest{Version: 1, Files: files})
	mb, _ := os.ReadFile(filepath.Join(bundle, manifestName))
	write(t, filepath.Join(bundle, signatureName), sign(t, k, mb))
	if err := doVerify(bundle, pub, &report{}); err == nil || !strings.Contains(err.Error(), "no model weights") {
		t.Errorf("signed bundle without weights: %v", err)
	}
}

func TestNamesAndLimits(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "m.gguf"), gguf(""))
	write(t, filepath.Join(dir, "a:b.json"), []byte(`{}`))
	if _, err := listFiles(dir, nil); err == nil || !strings.Contains(err.Error(), "unsafe file name") {
		t.Errorf("':' in name: %v", err)
	}

	limit := filepath.Join(t.TempDir(), "big.md")
	write(t, limit, []byte(strings.Repeat("a", maxMetaSize)))
	if _, err := inspect(limit); err != nil {
		t.Errorf("metadata at the limit refused: %v", err)
	}
	write(t, limit, []byte(strings.Repeat("a", maxMetaSize+1)))
	if _, err := inspect(limit); err == nil {
		t.Error("metadata over the limit accepted")
	}
}

func TestHFNegatives(t *testing.T) {
	dir := t.TempDir()
	weights, cfg := gguf("w"), []byte(`{"x":1}`)
	write(t, filepath.Join(dir, "m.gguf"), weights)
	write(t, filepath.Join(dir, "config.json"), cfg)
	files, _ := listFiles(dir, nil)
	rev := strings.Repeat("b", 40)
	var lfs, blob string
	for _, f := range files {
		if f.Path == "m.gguf" {
			lfs = f.SHA256
		} else {
			blob = f.GitSHA1
		}
	}
	mode := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sha, b, name := rev, blob, "config.json"
		switch mode {
		case "404":
			http.NotFound(w, r)
			return
		case "redirect":
			http.Redirect(w, r, "/elsewhere", http.StatusMovedPermanently)
			return
		case "sha":
			sha = strings.Repeat("c", 40)
		case "blob":
			b = strings.Repeat("0", 40)
		case "missing":
			name = "other.json"
		}
		fmt.Fprintf(w, `{"sha":%q,"siblings":[{"rfilename":%q,"blobId":%q},{"rfilename":"m.gguf","lfs":{"sha256":%q}}]}`, sha, name, b, lfs)
	}))
	defer srv.Close()
	hfBase = srv.URL

	if err := checkHF(dir, "org/m", rev, files); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	for _, m := range []string{"404", "redirect", "sha", "blob", "missing"} {
		mode = m
		if err := checkHF(dir, "org/m", rev, files); err == nil {
			t.Errorf("%s: accepted", m)
		}
	}
	mode = ""
	if err := checkHF(dir, "org/m", strings.Repeat("g", 40), files); err == nil {
		t.Error("40-char non-hex revision accepted")
	}
}

func TestZarfFlags(t *testing.T) {
	bundle, pub, _ := sealedBundle(t)
	out := filepath.Join(t.TempDir(), "elsewhere", "zarf.yaml")
	os.MkdirAll(filepath.Dir(out), 0o755)
	bin := filepath.Join(t.TempDir(), "airlock")
	write(t, bin, []byte("binary"))

	if zarfCmd([]string{bundle, "--key", pub, "--out", out, "--name", "Bad_Name"}) == nil {
		t.Error("invalid package name accepted")
	}
	if zarfCmd([]string{bundle, "--out", out}) == nil {
		t.Error("missing --key accepted")
	}
	if err := zarfCmd([]string{bundle, "--key", pub, "--out", out, "--bin", bin}); err != nil {
		t.Fatal(err)
	}
	y, _ := os.ReadFile(out)
	ob, _ := filepath.EvalSymlinks(filepath.Dir(out))
	bb, _ := filepath.EvalSymlinks(bundle)
	rel, _ := filepath.Rel(ob, filepath.Join(bb, "model.gguf"))
	for _, want := range []string{"executable: true", fmt.Sprintf("source: %q", filepath.ToSlash(rel))} {
		if !strings.Contains(string(y), want) {
			t.Errorf("zarf.yaml missing %q", want)
		}
	}
}

// TestSealWithCosign runs the real seal command, so cosign's flags are tested too.
func TestSealWithCosign(t *testing.T) {
	if _, err := exec.LookPath("cosign"); err != nil {
		t.Skip("cosign not installed")
	}
	dir := t.TempDir()
	bundle := filepath.Join(dir, "b")
	write(t, filepath.Join(bundle, "m.gguf"), gguf("weights"))
	t.Setenv("COSIGN_PASSWORD", "")
	gen := exec.Command("cosign", "generate-key-pair")
	gen.Dir = dir
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("cosign generate-key-pair: %v\n%s", err, out)
	}
	for i := 0; i < 2; i++ { // sealing twice replaces the old manifest
		if err := seal([]string{bundle, "--key", filepath.Join(dir, "cosign.key")}); err != nil {
			t.Fatal(err)
		}
	}
	if err := verify([]string{bundle, "--key", filepath.Join(dir, "cosign.pub"), "--report", filepath.Join(dir, "r.md")}); err != nil {
		t.Fatal(err)
	}
	if seal([]string{bundle}) == nil {
		t.Error("seal without --key accepted")
	}
}
