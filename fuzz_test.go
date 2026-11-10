package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"testing"
)

var exts = []string{".gguf", ".safetensors", ".json", ".model", ".txt", ".md", ".jinja", ".bin", ".sh"}

// Accepted content must satisfy the invariants the README promises, and the
// checker must never panic or read past what it was given.
func FuzzCheckFormat(f *testing.F) {
	f.Add(uint8(1), safetensors())
	f.Add(uint8(1), safetensorsWith(tensor4, 0x280, []byte{0, 0, 0, 0}))
	f.Add(uint8(0), gguf("rest"))
	f.Add(uint8(0), ggufFile([][]byte{kvStr("general.name", "q"), kvStrs("t", "a", "b"), kvU32("general.alignment", 32)}, []gt{{name: "a", dims: []uint64{8}, data: zeros(32)}, {name: "b", typ: 12, dims: []uint64{256}, data: zeros(144)}}))
	f.Add(uint8(2), []byte(`{"a":1}`))
	f.Add(uint8(3), []byte("\x0a\x05hello"))
	f.Add(uint8(4), []byte("(dp0\nS'a'\np1\n."))
	f.Fuzz(func(t *testing.T, e uint8, b []byte) {
		ext := exts[int(e)%len(exts)]
		kind, err := checkFormat(bytes.NewReader(b), "f"+ext, int64(len(b)))
		if err != nil {
			return
		}
		switch ext {
		case ".bin", ".sh":
			t.Fatalf("%s accepted", ext)
		case ".gguf":
			if !bytes.HasPrefix(b, []byte("GGUF")) || kind != "weights" {
				t.Fatal("gguf accepted without magic")
			}
		case ".safetensors":
			n := binary.LittleEndian.Uint64(b[:8])
			var hdr map[string]json.RawMessage
			if json.Unmarshal(bytes.TrimRight(b[8:8+n], " "), &hdr) != nil {
				t.Fatal("safetensors accepted with bad header")
			}
		default:
			if looksLikePickle(b) || kind != "meta" {
				t.Fatal("pickle/zip accepted under a metadata name")
			}
		}
	})
}

// No byte string other than a real signature over this exact message may verify.
func FuzzVerifySignature(f *testing.F) {
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	msg := []byte(`{"version":1}`)
	good := sign(f, k, msg)
	f.Add(good, msg)
	f.Add(good, []byte(`{"version":2}`))
	f.Add([]byte(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`), msg)
	f.Fuzz(func(t *testing.T, bundle, m []byte) {
		if verifySignature(&k.PublicKey, m, bundle) != nil {
			return
		}
		if !bytes.Equal(m, msg) {
			t.Fatalf("signature for one manifest verified a different one: %q", m)
		}
	})
}
