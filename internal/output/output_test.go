package output

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_EmptyDir(t *testing.T) {
	w, err := New("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != nil {
		t.Error("expected nil writer for empty dir")
	}
}

func TestNew_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub", "output")

	w, err := New(subdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w == nil {
		t.Fatal("expected non-nil writer")
	}

	info, err := os.Stat(subdir)
	if err != nil {
		t.Fatalf("output dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestSafeFilename(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{
			url:  "http://example.com/blog/my-post",
			want: "example.com__blog__my-post.md",
		},
		{
			url:  "http://example.com/",
			want: "example.com.md",
		},
		{
			url:  "http://example.com",
			want: "example.com.md",
		},
		{
			url:  "http://example.com/search?q=test&page=1",
			want: "example.com__search__q_test_page_1.md",
		},
		{
			url:  "http://example.com/path/with spaces/file",
			want: "example.com__path__with_spaces__file.md",
		},
		{
			url:  "https://sub.domain.com/a/b/c",
			want: "sub.domain.com__a__b__c.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := SafeFilename(tt.url)
			if got != tt.want {
				t.Errorf("SafeFilename(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestSafeFilename_Truncation(t *testing.T) {
	// Build a URL with a very long path.
	longPath := "http://example.com/"
	for i := 0; i < 50; i++ {
		longPath += "verylongsegment/"
	}
	name := SafeFilename(longPath)
	// Should be at most 200 chars + ".md" = 203
	if len(name) > 203 {
		t.Errorf("filename too long: %d chars", len(name))
	}
	if !hasSuffix(name, ".md") {
		t.Errorf("expected .md extension, got %q", name)
	}
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func TestWriter_Write(t *testing.T) {
	dir := t.TempDir()
	w, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	url := "http://example.com/test-page"
	content := []byte("# Test Page\n\nHello world")

	err = w.Write(url, content)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	expectedFile := filepath.Join(dir, "example.com__test-page.md")
	got, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("file content = %q, want %q", got, content)
	}
}

func TestWriter_NilSafe(t *testing.T) {
	var w *Writer
	err := w.Write("http://example.com", []byte("data"))
	if err != nil {
		t.Errorf("expected nil error from nil writer, got %v", err)
	}
}

func BenchmarkSafeFilename(b *testing.B) {
	url := "http://example.com/blog/2024/my-great-post?ref=twitter&utm_source=test"
	for b.Loop() {
		SafeFilename(url)
	}
}

func BenchmarkWriter_Write(b *testing.B) {
	dir := b.TempDir()
	w, _ := New(dir)
	content := []byte("# Benchmark\n\nSome markdown content here.")

	for b.Loop() {
		w.Write("http://example.com/bench", content)
	}
}
