package cache

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Entry represents a cached response.
type Entry struct {
	Body      []byte
	ExpiresAt time.Time
}

// DiskCache stores HTML response bodies on disk, keyed by request URL.
// It respects RFC 7234 Cache-Control and Expires headers.
type DiskCache struct {
	dir string
}

// New creates a new DiskCache writing to the given directory.
// Returns nil if dir is empty.
func New(dir string) (*DiskCache, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating cache dir: %w", err)
	}
	return &DiskCache{dir: dir}, nil
}

// IsCacheable checks RFC 7234 headers to determine if a response may be cached.
func IsCacheable(resp *http.Response) bool {
	// Don't cache non-success responses.
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return false
	}

	cc := resp.Header.Get("Cache-Control")
	ccLower := strings.ToLower(cc)

	// Explicit no-store directive: must not cache.
	if strings.Contains(ccLower, "no-store") {
		return false
	}
	// Private responses should not be stored in shared caches.
	if strings.Contains(ccLower, "private") {
		return false
	}

	// If max-age or s-maxage is present, it's cacheable.
	if strings.Contains(ccLower, "max-age") || strings.Contains(ccLower, "s-maxage") {
		return true
	}

	// If Expires header is present and in the future, it's cacheable.
	if exp := resp.Header.Get("Expires"); exp != "" {
		t, err := http.ParseTime(exp)
		if err == nil && t.After(time.Now()) {
			return true
		}
	}

	// If there's an ETag or Last-Modified, consider it cacheable for validation.
	if resp.Header.Get("ETag") != "" || resp.Header.Get("Last-Modified") != "" {
		return true
	}

	return false
}

// TTL computes how long a response should be cached based on RFC cache headers.
func TTL(resp *http.Response) time.Duration {
	cc := resp.Header.Get("Cache-Control")
	ccLower := strings.ToLower(cc)

	// Check s-maxage first (takes priority for shared caches).
	if idx := strings.Index(ccLower, "s-maxage="); idx >= 0 {
		if d := parseMaxAge(cc[idx+9:]); d > 0 {
			return d
		}
	}

	// Then max-age.
	if idx := strings.Index(ccLower, "max-age="); idx >= 0 {
		if d := parseMaxAge(cc[idx+8:]); d > 0 {
			return d
		}
	}

	// Fall back to Expires header.
	if exp := resp.Header.Get("Expires"); exp != "" {
		t, err := http.ParseTime(exp)
		if err == nil {
			ttl := time.Until(t)
			if ttl > 0 {
				return ttl
			}
		}
	}

	// Default TTL for responses with ETag/Last-Modified but no explicit expiry.
	return 5 * time.Minute
}

func parseMaxAge(s string) time.Duration {
	// Extract the numeric portion, stopping at comma or space.
	end := strings.IndexAny(s, ", ")
	if end == -1 {
		end = len(s)
	}
	secs, err := strconv.Atoi(strings.TrimSpace(s[:end]))
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// keyFor produces a filesystem-safe cache key from a URL.
func keyFor(rawURL string) string {
	h := sha256.Sum256([]byte(rawURL))
	return fmt.Sprintf("%x", h)
}

// Get returns cached body bytes if a valid cache entry exists and hasn't expired.
func (c *DiskCache) Get(rawURL string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	key := keyFor(rawURL)
	metaPath := filepath.Join(c.dir, key+".meta")
	bodyPath := filepath.Join(c.dir, key+".html")

	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, false
	}
	expiry, err := time.Parse(time.RFC3339, strings.TrimSpace(string(metaBytes)))
	if err != nil || time.Now().After(expiry) {
		// Expired — clean up.
		os.Remove(metaPath)
		os.Remove(bodyPath)
		return nil, false
	}

	body, err := os.ReadFile(bodyPath)
	if err != nil {
		return nil, false
	}
	return body, true
}

// Put stores response body bytes with an expiration.
func (c *DiskCache) Put(rawURL string, body []byte, ttl time.Duration) error {
	if c == nil {
		return nil
	}
	key := keyFor(rawURL)
	bodyPath := filepath.Join(c.dir, key+".html")
	metaPath := filepath.Join(c.dir, key+".meta")

	if err := os.WriteFile(bodyPath, body, 0o644); err != nil {
		return fmt.Errorf("writing cache body: %w", err)
	}
	expiry := time.Now().Add(ttl).Format(time.RFC3339)
	if err := os.WriteFile(metaPath, []byte(expiry), 0o644); err != nil {
		return fmt.Errorf("writing cache meta: %w", err)
	}
	return nil
}
