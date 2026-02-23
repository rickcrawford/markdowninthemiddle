package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"testing"
)

func TestDecompress_Identity(t *testing.T) {
	input := "hello world"
	r, err := Decompress(bytes.NewReader([]byte(input)), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := io.ReadAll(r)
	if string(got) != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestDecompress_IdentityExplicit(t *testing.T) {
	input := "hello world"
	r, err := Decompress(bytes.NewReader([]byte(input)), "identity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := io.ReadAll(r)
	if string(got) != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestDecompress_Gzip(t *testing.T) {
	input := "hello gzip world"
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write([]byte(input))
	w.Close()

	r, err := Decompress(&buf, "gzip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := io.ReadAll(r)
	if string(got) != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestDecompress_Deflate(t *testing.T) {
	input := "hello deflate world"
	var buf bytes.Buffer
	w, _ := flate.NewWriter(&buf, flate.DefaultCompression)
	w.Write([]byte(input))
	w.Close()

	r, err := Decompress(&buf, "deflate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := io.ReadAll(r)
	if string(got) != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestDecompress_Unsupported(t *testing.T) {
	_, err := Decompress(bytes.NewReader([]byte("data")), "br")
	if err == nil {
		t.Error("expected error for unsupported encoding")
	}
}

func BenchmarkDecompress_Gzip(b *testing.B) {
	input := bytes.Repeat([]byte("benchmark data for gzip decompression "), 100)
	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	w.Write(input)
	w.Close()
	data := compressed.Bytes()

	for b.Loop() {
		r, _ := Decompress(bytes.NewReader(data), "gzip")
		io.ReadAll(r)
	}
}
