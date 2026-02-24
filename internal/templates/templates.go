package templates

import (
	"os"
	"path/filepath"
	"strings"
)

// Store holds Mustache templates keyed by URL glob patterns.
type Store struct {
	// templates maps URL patterns to template content.
	templates map[string]string
	// defaultTemplate is used when no pattern matches (from _default.mustache).
	defaultTemplate string
}

// New loads Mustache templates from a directory. Each .mustache file's name
// (without extension) is treated as a URL pattern where "__" is replaced by "/".
// A file named _default.mustache serves as the fallback for unmatched URLs.
func New(dir string) (*Store, error) {
	s := &Store{
		templates: make(map[string]string),
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".mustache") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		base := strings.TrimSuffix(name, ".mustache")
		if base == "_default" {
			s.defaultTemplate = string(content)
			continue
		}

		// Convert filename to URL pattern: "__" → "/"
		pattern := strings.ReplaceAll(base, "__", "/")
		s.templates[pattern] = string(content)
	}

	return s, nil
}

// Match returns the template string for the best-matching URL pattern,
// or empty string if no match (triggering auto-generation).
func (s *Store) Match(rawURL string) string {
	if s == nil {
		return ""
	}

	// Exact prefix match: find the longest matching pattern.
	var bestPattern string
	var bestTemplate string
	for pattern, tpl := range s.templates {
		if strings.HasPrefix(rawURL, pattern) && len(pattern) > len(bestPattern) {
			bestPattern = pattern
			bestTemplate = tpl
		}
	}
	if bestTemplate != "" {
		return bestTemplate
	}

	// Check host-only matches (pattern without path matches any path on that host).
	for pattern, tpl := range s.templates {
		// If pattern has no "/" after the scheme-less form, treat as host prefix.
		if !strings.Contains(pattern, "/") && strings.Contains(rawURL, pattern) {
			return tpl
		}
	}

	return s.defaultTemplate
}
