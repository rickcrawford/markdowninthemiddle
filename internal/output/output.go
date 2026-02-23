package output

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Writer writes converted Markdown files to a directory.
type Writer struct {
	dir string
}

// New creates a Writer that saves .md files to dir.
// Returns nil if dir is empty.
func New(dir string) (*Writer, error) {
	if dir == "" {
		return nil, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}
	return &Writer{dir: dir}, nil
}

// unsafeChars matches characters that are not safe for filenames.
var unsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

// SafeFilename converts a URL into a file-safe name with .md extension.
// The naming structure is: {host}__{path_segments}.md
// For example: example.com__blog__my-post.md
func SafeFilename(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// Fallback: sanitize the entire string.
		return sanitize(rawURL) + ".md"
	}

	host := u.Hostname()
	path := strings.Trim(u.Path, "/")

	var parts []string
	if host != "" {
		parts = append(parts, sanitize(host))
	}
	if path != "" {
		for _, seg := range strings.Split(path, "/") {
			s := sanitize(seg)
			if s != "" {
				parts = append(parts, s)
			}
		}
	}

	// Include query string in the name if present, to distinguish URLs.
	if u.RawQuery != "" {
		parts = append(parts, sanitize(u.RawQuery))
	}

	name := strings.Join(parts, "__")
	if name == "" {
		name = "index"
	}

	// Truncate to a reasonable filename length (200 chars before extension).
	if len(name) > 200 {
		name = name[:200]
	}

	return name + ".md"
}

func sanitize(s string) string {
	return unsafeChars.ReplaceAllString(s, "_")
}

// Write saves the markdown content to the output directory using a
// file-safe name derived from the request URL.
func (w *Writer) Write(rawURL string, markdown []byte) error {
	if w == nil {
		return nil
	}
	filename := SafeFilename(rawURL)
	path := filepath.Join(w.dir, filename)
	return os.WriteFile(path, markdown, 0o644)
}
