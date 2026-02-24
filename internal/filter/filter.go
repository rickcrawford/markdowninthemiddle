package filter

import (
	"fmt"
	"net/http"
	"regexp"
)

// Filter holds compiled regexes for allowed request URLs.
// If empty, all requests are allowed.
type Filter struct {
	patterns []*regexp.Regexp
}

// New compiles a slice of regex strings into a Filter.
// Returns an error if any pattern is invalid.
func New(patterns []string) (*Filter, error) {
	if len(patterns) == 0 {
		return &Filter{patterns: []*regexp.Regexp{}}, nil
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", p, err)
		}
		compiled[i] = re
	}

	return &Filter{patterns: compiled}, nil
}

// Allowed reports whether the given URL matches any allowed pattern.
// If no patterns are configured, all requests are allowed.
func (f *Filter) Allowed(rawURL string) bool {
	if len(f.patterns) == 0 {
		return true
	}

	for _, p := range f.patterns {
		if p.MatchString(rawURL) {
			return true
		}
	}
	return false
}

// Middleware returns an http.Handler wrapper that returns 403
// for requests not matched by the filter.
func (f *Filter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reconstruct the full URL
		rawURL := r.URL.String()
		if !r.URL.IsAbs() {
			// Try to reconstruct with scheme and host if needed
			scheme := "http"
			if r.TLS != nil {
				scheme = "https"
			}
			rawURL = scheme + "://" + r.Host + r.URL.String()
		}

		if !f.Allowed(rawURL) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
