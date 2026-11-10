package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
)

// GGUF v2/v3 (little-endian): magic, version, tensor count, metadata count,
// metadata key/values, tensor infos, padding to the alignment, tensor data.
// checkGGUF reads the header as a stream (metadata values are read and
// discarded, so the caller still hashes them) and requires the tensors to
// tile the data section, like the safetensors check.

const (
	ggufMaxKey     = 1 << 16
	ggufMaxName    = 1 << 10  // llama.cpp's limit is 64
	ggufMaxKept    = 64 << 20 // keys and names held in memory
	ggufTypeString = 8
	ggufTypeArray  = 9
)

// metadata value types with a fixed size, in bytes
var ggufScalar = map[uint32]uint64{0: 1, 1: 1, 2: 2, 3: 2, 4: 4, 5: 4, 6: 4, 7: 1, 10: 8, 11: 8, 12: 8}

// ggml tensor types: elements per block, bytes per block. Unknown types are
// refused rather than guessed.
var ggmlTypes = map[uint32][2]uint64{
	0: {1, 4}, 1: {1, 2}, 2: {32, 18}, 3: {32, 20}, 6: {32, 22}, 7: {32, 24}, 8: {32, 34}, 9: {32, 36},
	10: {256, 84}, 11: {256, 110}, 12: {256, 144}, 13: {256, 176}, 14: {256, 210}, 15: {256, 292},
	16: {256, 66}, 17: {256, 74}, 18: {256, 98}, 19: {256, 50}, 20: {32, 18}, 21: {256, 110},
	22: {256, 82}, 23: {256, 136}, 24: {1, 1}, 25: {1, 2}, 26: {1, 4}, 27: {1, 8}, 28: {1, 8},
	29: {256, 56}, 30: {1, 2}, 34: {256, 54}, 35: {256, 66}, 39: {32, 17},
}

type ggufReader struct {
	r    io.Reader
	n    int64 // bytes consumed
	size int64
	kept int64
	buf  [8]byte
}

var errTruncated = errors.New("GGUF is truncated or a field runs past the end of the file")

func (g *ggufReader) left() uint64 { return uint64(g.size - g.n) }

func (g *ggufReader) read(p []byte) error {
	k, err := io.ReadFull(g.r, p)
	g.n += int64(k)
	if err != nil {
		return errTruncated
	}
	return nil
}

func (g *ggufReader) u32() (uint32, error) {
	err := g.read(g.buf[:4])
	return binary.LittleEndian.Uint32(g.buf[:4]), err
}

func (g *ggufReader) u64() (uint64, error) {
	err := g.read(g.buf[:8])
	return binary.LittleEndian.Uint64(g.buf[:8]), err
}

func (g *ggufReader) skip(n uint64) error {
	if n > g.left() {
		return errTruncated
	}
	k, err := io.CopyN(io.Discard, g.r, int64(n))
	g.n += k
	if err != nil {
		return errTruncated
	}
	return nil
}

// str reads a length-prefixed string it needs to keep (a key or tensor name).
func (g *ggufReader) str(max uint64) (string, error) {
	l, err := g.u64()
	if err != nil {
		return "", err
	}
	if l > max {
		return "", fmt.Errorf("GGUF name of %d bytes is too long", l)
	}
	if l > g.left() {
		return "", errTruncated
	}
	if g.kept += int64(l); g.kept > ggufMaxKept {
		return "", errors.New("GGUF header is implausibly large")
	}
	b := make([]byte, l)
	return string(b), g.read(b)
}

// value reads and discards one metadata value of type t.
func (g *ggufReader) value(t uint32, depth int) error {
	if sz, ok := ggufScalar[t]; ok {
		return g.skip(sz)
	}
	switch t {
	case ggufTypeString:
		l, err := g.u64()
		if err != nil {
			return err
		}
		return g.skip(l)
	case ggufTypeArray:
		if depth > 2 {
			return errors.New("GGUF arrays nested too deep")
		}
		et, err := g.u32()
		if err != nil {
			return err
		}
		count, err := g.u64()
		if err != nil {
			return err
		}
		if sz, ok := ggufScalar[et]; ok {
			if count > g.left()/sz {
				return errTruncated
			}
			return g.skip(count * sz)
		}
		if et != ggufTypeString && et != ggufTypeArray {
			return fmt.Errorf("unknown GGUF metadata type %d", et)
		}
		if count > g.left()/8 { // every element needs at least an 8-byte length
			return errTruncated
		}
		for i := uint64(0); i < count; i++ {
			if err := g.value(et, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return fmt.Errorf("unknown GGUF metadata type %d", t)
}

func pad(x, align uint64) uint64 { return (x + align - 1) / align * align }

func checkGGUF(r io.Reader, size int64) error {
	g := &ggufReader{r: r, size: size}
	var magic [4]byte
	if err := g.read(magic[:]); err != nil || string(magic[:]) != "GGUF" {
		return errors.New("named .gguf but missing GGUF header")
	}
	v, err := g.u32()
	if err != nil {
		return err
	}
	if v < 2 || v > 3 {
		return fmt.Errorf("unsupported GGUF version %d", v)
	}
	tensors, err := g.u64()
	if err != nil {
		return err
	}
	kvs, err := g.u64()
	if err != nil {
		return err
	}
	// smallest possible entries: a tensor info is 32 bytes, a key/value 13
	if tensors == 0 || tensors > g.left()/32 || kvs > g.left()/13 {
		return fmt.Errorf("implausible GGUF header (%d tensors, %d metadata keys)", tensors, kvs)
	}

	align := uint64(32)
	keys := map[string]bool{}
	for i := uint64(0); i < kvs; i++ {
		k, err := g.str(ggufMaxKey)
		if err != nil {
			return err
		}
		if keys[k] {
			return fmt.Errorf("duplicate GGUF metadata key %q", k)
		}
		keys[k] = true
		t, err := g.u32()
		if err != nil {
			return err
		}
		if k == "general.alignment" {
			a, err := g.u32()
			if err != nil {
				return err
			}
			if t != 4 || a == 0 || a&(a-1) != 0 || a > 1<<20 {
				return errors.New("invalid general.alignment")
			}
			align = uint64(a)
			continue
		}
		if err := g.value(t, 0); err != nil {
			return err
		}
	}

	type span struct {
		name     string
		off, len uint64
	}
	spans := make([]span, 0, min(tensors, 1<<16))
	names := map[string]bool{}
	for i := uint64(0); i < tensors; i++ {
		name, err := g.str(ggufMaxName)
		if err != nil {
			return err
		}
		if names[name] {
			return fmt.Errorf("duplicate GGUF tensor %q", name)
		}
		names[name] = true
		nd, err := g.u32()
		if err != nil {
			return err
		}
		if nd < 1 || nd > 4 {
			return fmt.Errorf("tensor %q: %d dimensions", name, nd)
		}
		elems := uint64(1)
		for d := uint32(0); d < nd; d++ {
			dim, err := g.u64()
			if err != nil {
				return err
			}
			if dim == 0 || elems > math.MaxInt64/dim {
				return fmt.Errorf("tensor %q: bad shape", name)
			}
			elems *= dim
		}
		typ, err := g.u32()
		if err != nil {
			return err
		}
		bt, ok := ggmlTypes[typ]
		if !ok {
			return fmt.Errorf("tensor %q: unknown tensor type %d", name, typ)
		}
		if elems%bt[0] != 0 || elems/bt[0] > math.MaxInt64/bt[1] {
			return fmt.Errorf("tensor %q: shape does not fit its type", name)
		}
		off, err := g.u64()
		if err != nil {
			return err
		}
		spans = append(spans, span{name, off, elems / bt[0] * bt[1]})
	}

	// Tensors must tile the data section, with only alignment padding
	// between them, so no bytes can hide between or after them.
	start := pad(uint64(g.n), align)
	sort.Slice(spans, func(i, j int) bool { return spans[i].off < spans[j].off })
	var at uint64
	for _, s := range spans {
		if s.off != pad(at, align) {
			return fmt.Errorf("tensor %q: gap, overlap or misalignment in data section", s.name)
		}
		if s.len > math.MaxInt64-s.off {
			return fmt.Errorf("tensor %q: too large", s.name)
		}
		at = s.off + s.len
	}
	if end := start + at; uint64(size) != end && uint64(size) != start+pad(at, align) {
		if uint64(size) < end {
			return errors.New("GGUF tensors run past the end of the file")
		}
		return errors.New("GGUF has data not covered by any tensor (possible hidden payload)")
	}
	return nil
}
