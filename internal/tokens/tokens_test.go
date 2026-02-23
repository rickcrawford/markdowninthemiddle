package tokens

import (
	"strings"
	"testing"
)

func newTestCounter(t testing.TB) *Counter {
	t.Helper()
	c, err := NewCounter("cl100k_base")
	if err != nil {
		t.Skipf("skipping: tiktoken encoding unavailable (network required): %v", err)
	}
	return c
}

func TestNewCounter(t *testing.T) {
	tests := []struct {
		encoding string
		wantErr  bool
	}{
		{"cl100k_base", false},
		{"p50k_base", false},
		{"invalid_encoding_xyz", true},
	}

	for _, tt := range tests {
		t.Run(tt.encoding, func(t *testing.T) {
			c, err := NewCounter(tt.encoding)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for encoding %q", tt.encoding)
				}
				return
			}
			if err != nil {
				t.Skipf("skipping: tiktoken encoding unavailable (network required): %v", err)
			}
			if c == nil {
				t.Fatal("expected non-nil counter")
			}
		})
	}
}

func TestCounter_Count(t *testing.T) {
	c := newTestCounter(t)

	tests := []struct {
		name    string
		text    string
		wantMin int
		wantMax int
	}{
		{
			name:    "empty string",
			text:    "",
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "single word",
			text:    "hello",
			wantMin: 1,
			wantMax: 1,
		},
		{
			name:    "simple sentence",
			text:    "The quick brown fox jumps over the lazy dog.",
			wantMin: 5,
			wantMax: 15,
		},
		{
			name:    "markdown content",
			text:    "# Title\n\nThis is a paragraph with **bold** and *italic*.",
			wantMin: 5,
			wantMax: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := c.Count(tt.text)
			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("Count(%q) = %d, want between %d and %d", tt.text, count, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCounter_CountDeterministic(t *testing.T) {
	c := newTestCounter(t)

	text := "This is a deterministic test string."
	first := c.Count(text)
	for i := 0; i < 10; i++ {
		got := c.Count(text)
		if got != first {
			t.Errorf("non-deterministic: iteration %d got %d, first was %d", i, got, first)
		}
	}
}

func BenchmarkCounter_Count_Short(b *testing.B) {
	c := newTestCounter(b)
	text := "Hello, world!"
	for b.Loop() {
		c.Count(text)
	}
}

func BenchmarkCounter_Count_Long(b *testing.B) {
	c := newTestCounter(b)
	text := strings.Repeat("The quick brown fox jumps over the lazy dog. ", 100)
	for b.Loop() {
		c.Count(text)
	}
}
