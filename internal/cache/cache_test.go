package cache

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew_EmptyDir(t *testing.T) {
	c, err := New("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c != nil {
		t.Error("expected nil cache for empty dir")
	}
}

func TestNew_CreatesDir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub", "cache")

	c, err := New(subdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil cache")
	}

	info, err := os.Stat(subdir)
	if err != nil {
		t.Fatalf("cache dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestDiskCache_PutGet(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	url := "http://example.com/page"
	body := []byte("<html>test</html>")
	ttl := 1 * time.Hour

	err = c.Put(url, body, ttl)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}

	got, ok := c.Get(url)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != string(body) {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestDiskCache_Expiry(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	url := "http://example.com/expired"
	body := []byte("<html>expired</html>")

	// Write with a TTL that's already expired.
	key := keyFor(url)
	bodyPath := filepath.Join(dir, key+".html")
	metaPath := filepath.Join(dir, key+".meta")

	os.WriteFile(bodyPath, body, 0o644)
	expiry := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	os.WriteFile(metaPath, []byte(expiry), 0o644)

	_, ok := c.Get(url)
	if ok {
		t.Error("expected cache miss for expired entry")
	}
}

func TestDiskCache_Miss(t *testing.T) {
	dir := t.TempDir()
	c, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := c.Get("http://example.com/nonexistent")
	if ok {
		t.Error("expected cache miss")
	}
}

func TestDiskCache_NilSafe(t *testing.T) {
	var c *DiskCache
	_, ok := c.Get("http://example.com")
	if ok {
		t.Error("expected false from nil cache")
	}
	err := c.Put("http://example.com", []byte("data"), time.Hour)
	if err != nil {
		t.Errorf("expected nil error from nil cache Put, got %v", err)
	}
}

func TestIsCacheable(t *testing.T) {
	tests := []struct {
		name   string
		status int
		header http.Header
		want   bool
	}{
		{
			name:   "no-store",
			status: 200,
			header: http.Header{"Cache-Control": []string{"no-store"}},
			want:   false,
		},
		{
			name:   "private",
			status: 200,
			header: http.Header{"Cache-Control": []string{"private"}},
			want:   false,
		},
		{
			name:   "max-age",
			status: 200,
			header: http.Header{"Cache-Control": []string{"max-age=3600"}},
			want:   true,
		},
		{
			name:   "s-maxage",
			status: 200,
			header: http.Header{"Cache-Control": []string{"s-maxage=600"}},
			want:   true,
		},
		{
			name:   "etag",
			status: 200,
			header: func() http.Header { h := http.Header{}; h.Set("ETag", `"abc123"`); return h }(),
			want:   true,
		},
		{
			name:   "error status",
			status: 500,
			header: http.Header{"Cache-Control": []string{"max-age=3600"}},
			want:   false,
		},
		{
			name:   "no cache headers",
			status: 200,
			header: http.Header{},
			want:   false,
		},
		{
			name:   "future expires",
			status: 200,
			header: http.Header{"Expires": []string{time.Now().Add(1 * time.Hour).Format(http.TimeFormat)}},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.status,
				Header:     tt.header,
			}
			got := IsCacheable(resp)
			if got != tt.want {
				t.Errorf("IsCacheable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTTL(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		min    time.Duration
		max    time.Duration
	}{
		{
			name:   "max-age 600",
			header: http.Header{"Cache-Control": []string{"max-age=600"}},
			min:    600 * time.Second,
			max:    600 * time.Second,
		},
		{
			name:   "s-maxage takes priority",
			header: http.Header{"Cache-Control": []string{"max-age=600, s-maxage=300"}},
			min:    300 * time.Second,
			max:    300 * time.Second,
		},
		{
			name:   "default ttl with etag",
			header: func() http.Header { h := http.Header{}; h.Set("ETag", `"abc"`); return h }(),
			min:    5 * time.Minute,
			max:    5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{Header: tt.header}
			got := TTL(resp)
			if got < tt.min || got > tt.max {
				t.Errorf("TTL() = %v, want between %v and %v", got, tt.min, tt.max)
			}
		})
	}
}

func BenchmarkDiskCache_PutGet(b *testing.B) {
	dir := b.TempDir()
	c, _ := New(dir)
	body := []byte("<html><body>benchmark content</body></html>")

	for b.Loop() {
		c.Put("http://example.com/bench", body, time.Hour)
		c.Get("http://example.com/bench")
	}
}
