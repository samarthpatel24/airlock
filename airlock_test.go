package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

const tensor4 = `{"w":{"dtype":"F32","shape":[1],"data_offsets":[0,4]}}`

// safetensorsWith builds a file whose JSON header is space-padded to hdrLen.
func safetensorsWith(hdr string, hdrLen int, data []byte) []byte {
	h := []byte(hdr)
	for len(h) < hdrLen {
		h = append(h, ' ')
	}
	b := binary.LittleEndian.AppendUint64(nil, uint64(len(h)))
	return append(append(b, h...), data...)
}

func safetensors() []byte { return safetensorsWith(tensor4, 0, []byte{0, 0, 0, 0}) }

// gguf builds a minimal GGUF v3 header: one tensor, no metadata, then rest.
func gguf(rest string) []byte {
	b := append([]byte("GGUF"), 3, 0, 0, 0)
	b = binary.LittleEndian.AppendUint64(b, 1)
	b = binary.LittleEndian.AppendUint64(b, 0)
	return append(b, rest...)
}

func zeros(n int) []byte { return make([]byte, n) }

func TestFormats(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		data []byte
		ok   bool
	}{
		{"model.gguf", gguf("rest"), true},
		{"model.safetensors", safetensors(), true},
		{"config.json", []byte(`{"a":1}`), true},
		{"pytorch_model.bin", []byte("anything"), false},                   // pickle extension
		{"config2.json", []byte("\x80\x04\x95pickle"), false},              // renamed pickle
		{"config3.json", []byte("(dp0\nS'a'\np1\n."), false},               // renamed ASCII pickle
		{"model2.safetensors", []byte("PK\x03\x04zip-with-pickle"), false}, // torch.save zip
		{"fake.gguf", []byte("NOPE1234"), false},                           // wrong header
		{"bad.safetensors", []byte("\xff\xff\xff\xff\xff\xff\xff\x7f"), false},
		{"run.sh", []byte("#!/bin/sh"), false}, // unknown format
		{"v9.gguf", []byte("GGUF\x09\x00\x00\x00"), false},
		// header length 0x280 starts with bytes 0x80 0x02, which look like a pickle
		{"padded.safetensors", safetensorsWith(tensor4, 0x280, []byte{0, 0, 0, 0}), true},
		{"polyglot.safetensors", safetensorsWith(tensor4, 0, []byte("\x00\x00\x00\x00PK\x03\x04hidden")), false},
		{"oob.safetensors", safetensorsWith(`{"w":{"data_offsets":[0,99]}}`, 0, []byte{0, 0, 0, 0}), false},
		{"tokenizer.model", []byte("(dp0\nS'a'\np1\n."), false}, // ASCII pickle under a metadata name
		{"tokenizer2.model", []byte("\x0a\x05hello"), true},
		{"notes.txt", []byte("a\x00b"), false},
		{"chat.jinja", []byte("{{ messages }}"), true},
		{"pk.txt", []byte("PK\x03\x04hi"), false}, // zip magic is valid UTF-8, still refused
		{"bom.json", []byte("\xef\xbb\xbf{\"a\":1}"), true},
		{"LICENSE", []byte("Apache License 2.0"), true},
		{".gitattributes", []byte("*.gguf filter=lfs"), true},
		{"LICENSE.bin", []byte("x"), false},
		{"short.gguf", []byte("GGUF\x03\x00\x00\x00"), false},     // header only, no counts
		{"empty.gguf", append(gguf("")[:8], zeros(16)...), false}, // zero tensors
		// safetensors layout: tensors must tile the data section exactly
		{"gap.safetensors", safetensorsWith(`{"w":{"dtype":"U8","shape":[2],"data_offsets":[2,4]}}`, 0, zeros(4)), false},
		{"overlap.safetensors", safetensorsWith(`{"a":{"dtype":"U8","shape":[4],"data_offsets":[0,4]},"b":{"dtype":"U8","shape":[2],"data_offsets":[2,4]}}`, 0, zeros(4)), false},
		{"two.safetensors", safetensorsWith(`{"a":{"dtype":"U8","shape":[2],"data_offsets":[0,2]},"b":{"dtype":"U8","shape":[2],"data_offsets":[2,4]},"__metadata__":{"x":"y"}}`, 0, zeros(4)), true},
		{"dup.safetensors", safetensorsWith(`{"w":{"dtype":"U8","shape":[9],"data_offsets":[0,9]},"w":{"dtype":"U8","shape":[4],"data_offsets":[0,4]}}`, 0, zeros(4)), false},
		{"size.safetensors", safetensorsWith(`{"w":{"dtype":"F32","shape":[2],"data_offsets":[0,4]}}`, 0, zeros(4)), false},
		{"neg.safetensors", safetensorsWith(`{"w":{"dtype":"U8","shape":[6],"data_offsets":[-2,4]}}`, 0, zeros(4)), false},
		{"dtype.safetensors", safetensorsWith(`{"w":{"dtype":"F4","shape":[4],"data_offsets":[0,4]}}`, 0, zeros(4)), false},
	}
	for _, c := range cases {
		p := filepath.Join(dir, c.name)
		write(t, p, c.data)
		_, _, _, err := inspect(p)
		if (err == nil) != c.ok {
			t.Errorf("%s: ok=%v, err=%v", c.name, c.ok, err)
		}
	}
}

// sign mimics `cosign sign-blob --key ... --bundle` output.
func sign(t *testing.T, k *ecdsa.PrivateKey, msg []byte) []byte {
	d := sha256.Sum256(msg)
	sig, err := ecdsa.SignASN1(rand.Reader, k, d[:])
	if err != nil {
		t.Fatal(err)
	}
	var b sigBundle
	b.MediaType = "application/vnd.dev.sigstore.bundle.v0.3+json"
	b.MessageSignature.MessageDigest.Algorithm = "SHA2_256"
	b.MessageSignature.MessageDigest.Digest = d[:]
	b.MessageSignature.Signature = sig
	out, _ := json.Marshal(b)
	return out
}

func pubPEM(t *testing.T, dir string, k *ecdsa.PrivateKey) string {
	der, _ := x509.MarshalPKIXPublicKey(&k.PublicKey)
	p := filepath.Join(dir, "cosign.pub")
	write(t, p, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	return p
}

// sealedBundle builds a small signed bundle the way `airlock seal` does.
func sealedBundle(t *testing.T) (bundle, pub string, mb []byte) {
	keys := t.TempDir()
	bundle = t.TempDir()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pub = pubPEM(t, keys, k)
	write(t, filepath.Join(bundle, "model.gguf"), gguf("weights"))
	write(t, filepath.Join(bundle, "tok/config.json"), []byte(`{}`))
	files, err := listFiles(bundle)
	if err != nil {
		t.Fatal(err)
	}
	writeJSON(filepath.Join(bundle, manifestName), &Manifest{Version: 1, Files: files})
	mb, _ = os.ReadFile(filepath.Join(bundle, manifestName))
	write(t, filepath.Join(bundle, signatureName), sign(t, k, mb))
	return bundle, pub, mb
}

func TestSealVerifyTamper(t *testing.T) {
	bundle, pub, mb := sealedBundle(t)

	expect := func(label, want string) {
		t.Helper()
		err := doVerify(bundle, pub, &report{})
		if want == "" && err != nil {
			t.Fatalf("%s: unexpected error %v", label, err)
		}
		if want != "" && (err == nil || !strings.Contains(err.Error(), want)) {
			t.Fatalf("%s: want error containing %q, got %v", label, want, err)
		}
	}
	expect("clean bundle", "")

	write(t, filepath.Join(bundle, "model.gguf"), gguf("WEIGHTS"))
	expect("changed weights", "hash mismatch")
	write(t, filepath.Join(bundle, "model.gguf"), gguf("weights"))

	write(t, filepath.Join(bundle, "extra.json"), []byte(`{}`))
	expect("extra file", "unexpected file")
	os.Remove(filepath.Join(bundle, "extra.json"))

	os.Symlink("/etc/passwd", filepath.Join(bundle, "link.json"))
	expect("symlink", "symlinks are refused")
	os.Remove(filepath.Join(bundle, "link.json"))

	fake := filepath.Join(bundle, "x\n- ✔ PASS.json") // would print a fake PASS line in the report
	write(t, fake, []byte(`{}`))
	expect("injected file name", "unsafe file name")
	os.Remove(fake)

	victim := filepath.Join(t.TempDir(), "cosign.key")
	write(t, victim, []byte("secret"))
	os.Remove(filepath.Join(bundle, manifestName))
	os.Symlink(victim, filepath.Join(bundle, manifestName))
	expect("symlinked manifest", "not a regular file")
	if err := clearReserved(bundle); err == nil {
		t.Error("seal would write through a symlinked manifest")
	}
	if b, _ := os.ReadFile(victim); string(b) != "secret" {
		t.Error("symlink target was modified")
	}
	os.Remove(filepath.Join(bundle, manifestName))
	write(t, filepath.Join(bundle, manifestName), mb)

	write(t, filepath.Join(bundle, manifestName), append(mb, ' '))
	expect("edited manifest", "tampered")
	write(t, filepath.Join(bundle, manifestName), mb)

	other, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	write(t, filepath.Join(bundle, signatureName), sign(t, other, mb))
	expect("signed by someone else", "invalid signature")
}

func TestCLIPaths(t *testing.T) {
	bundle, pub, _ := sealedBundle(t)
	for _, args := range [][]string{
		{bundle, "--key", pub},
		{"--key", pub, bundle},
		{"--key", pub, bundle, "--report", filepath.Join(t.TempDir(), "r.md")}, // flags on both sides
		{"--key=" + pub, bundle},
	} {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.String("key", "", "")
		fs.String("report", "", "")
		if _, err := parseDir(fs, args); err != nil {
			t.Errorf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{}, {bundle, bundle}, {"--key", pub}} {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.String("key", "", "")
		if _, err := parseDir(fs, args); err == nil {
			t.Errorf("%v: accepted", args)
		}
	}

	link := filepath.Join(t.TempDir(), "drive")
	os.Symlink(bundle, link) // e.g. a bundle reached through /Volumes
	if err := verify([]string{link, "--key", pub, "--report", filepath.Join(t.TempDir(), "r.md")}); err != nil {
		t.Errorf("symlinked bundle dir: %v", err)
	}

	if err := verify([]string{bundle, "--key", pub, "--report", filepath.Join(bundle, "r.md")}); err == nil {
		t.Error("report inside the bundle accepted")
	}
	if err := zarfCmd([]string{bundle, "--key", pub, "--out", filepath.Join(bundle, "zarf.yaml")}); err == nil {
		t.Error("zarf.yaml inside the bundle accepted")
	}
	if _, err := os.Stat(filepath.Join(bundle, "r.md")); err == nil {
		t.Error("report was written into the bundle")
	}
}

func TestZarf(t *testing.T) {
	bundle, pub, _ := sealedBundle(t)
	out := filepath.Join(t.TempDir(), "pkg", "zarf.yaml")
	os.MkdirAll(filepath.Dir(out), 0o755)
	bin := filepath.Join(t.TempDir(), "airlock")
	write(t, bin, []byte("binary"))

	if err := zarfCmd([]string{bundle, "--key", pub, "--out", out}); err != nil {
		t.Fatal(err)
	}
	y, _ := os.ReadFile(out)
	sum, _, _ := hashFile(filepath.Join(bundle, "model.gguf"))
	for _, want := range []string{
		"apiVersion: zarf.dev/v1beta1",
		`destination: "airlock/bundle/tok/config.json"`,
		"checksum: sha256:" + sum,
		`destination: "airlock/bundle/manifest.sigstore.json"`,
		"- cmd: /usr/local/bin/airlock verify ./airlock/bundle --key /etc/airlock/cosign.pub",
	} {
		if !strings.Contains(string(y), want) {
			t.Errorf("zarf.yaml missing %q", want)
		}
	}
	if strings.Contains(string(y), "cosign.pub\"\n        destination") || strings.Contains(string(y), `destination: "airlock/bin/airlock"`) {
		t.Error("trust anchors (key or verifier) shipped by default")
	}
	if err := zarfCmd([]string{bundle, "--key", pub, "--out", out, "--inside-key", "/k; rm -rf /"}); err == nil {
		t.Error("shell metacharacters accepted in --inside-key")
	}

	if err := zarfCmd([]string{bundle, "--key", pub, "--bin", bin, "--out", out}); err != nil {
		t.Fatal(err)
	}
	if y, _ := os.ReadFile(out); !strings.Contains(string(y), "- cmd: ./airlock/bin/airlock verify") {
		t.Error("--bin did not switch to the shipped verifier")
	}

	write(t, filepath.Join(bundle, "model.gguf"), gguf("evil"))
	if err := zarfCmd([]string{bundle, "--key", pub, "--out", out}); err == nil {
		t.Fatal("tampered bundle was packaged")
	}
}

func TestSmoke(t *testing.T) {
	var mu sync.Mutex
	healthy, status, body := false, 200, `{"choices":[{"message":{"content":"4"}}]}`
	set := func(s int, b string) { mu.Lock(); status, body = s, b; mu.Unlock() }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/health":
			if !healthy { // first poll: still loading, like llama.cpp
				healthy = true
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		case "/v1/chat/completions":
			w.WriteHeader(status)
			fmt.Fprint(w, body)
		}
	}))
	defer srv.Close()
	rep := filepath.Join(t.TempDir(), "report.md")
	run := func(extra ...string) error {
		return smoke(append([]string{"--url", srv.URL, "--report", rep, "--wait", "10s"}, extra...))
	}
	answer := func(s string) string { return fmt.Sprintf(`{"choices":[{"message":{"content":%q}}]}`, s) }

	if err := run(); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ label, body string }{
		{"contains but is not the answer", answer("2024")},
		{"markdown in answer", answer("five\n## Offline checks\n- ✔ fake [ok](http://x)")},
		{"no choices", `{"choices":[]}`},
		{"null content", `{"choices":[{"message":{"content":null}}]}`},
		{"not JSON", "<html>"},
	} {
		set(200, c.body)
		if run() == nil {
			t.Errorf("%s: passed", c.label)
		}
	}
	set(500, answer("4"))
	if run() == nil {
		t.Error("HTTP 500 passed")
	}
	if smoke([]string{"--url", "ftp://x", "--report", rep}) == nil || smoke([]string{"--url", "http://", "--report", rep}) == nil {
		t.Error("bad --url accepted")
	}

	b, _ := os.ReadFile(rep)
	if !strings.Contains(string(b), "**Result:** PASS") || !strings.Contains(string(b), "**Result:** FAIL") {
		t.Errorf("report missing results:\n%s", b)
	}
	if strings.Contains(string(b), "\n## Offline checks") || !strings.Contains(string(b), "`\"five\\n## Offline checks") {
		t.Error("model answer was rendered as markdown in the report")
	}

	if got := truncate("ab€", 3); got != "ab" { // € is 3 bytes; never cut mid-character
		t.Errorf("truncate split a character: %q", got)
	}

	never := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // hangs past --wait
	}))
	defer never.Close()
	start := time.Now()
	if smoke([]string{"--url", never.URL, "--report", rep, "--wait", "1s"}) == nil {
		t.Error("unhealthy server passed")
	}
	if time.Since(start) > 2500*time.Millisecond {
		t.Errorf("--wait 1s took %s", time.Since(start))
	}
}

func TestHFProvenance(t *testing.T) {
	dir := t.TempDir()
	weights, cfg := gguf("weights"), []byte(`{"x":1}`)
	write(t, filepath.Join(dir, "m.gguf"), weights)
	write(t, filepath.Join(dir, "config.json"), cfg)
	files, _ := listFiles(dir)
	rev := strings.Repeat("a", 40)
	blob, _ := gitBlobSHA1(filepath.Join(dir, "config.json"))

	lfsSHA := fmt.Sprintf("%x", sha256.Sum256(weights))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"sha":%q,"siblings":[{"rfilename":"config.json","blobId":%q},{"rfilename":"m.gguf","lfs":{"sha256":%q}}]}`,
			rev, blob, lfsSHA)
	}))
	defer srv.Close()
	hfBase = srv.URL

	if err := checkHF(dir, "org/model", rev, files); err != nil {
		t.Fatalf("genuine files rejected: %v", err)
	}
	if err := checkHF(dir, "org/model", "main", files); err == nil {
		t.Fatal("branch name accepted; must require a pinned commit")
	}
	if err := checkHF(dir, "org/..", rev, files); err == nil {
		t.Fatal("dot segment accepted in repo name")
	}
	lfsSHA = strings.Repeat("0", 64)
	if err := checkHF(dir, "org/model", rev, files); err == nil {
		t.Fatal("weights with a different hash were accepted")
	}
}
