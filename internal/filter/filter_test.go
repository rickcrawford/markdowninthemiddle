package filter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_InvalidRegex(t *testing.T) {
	_, err := New([]string{"[invalid(regex"})
	if err == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
}

func TestFilter_Allowed(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		url      string
		want     bool
	}{
		{
			name:     "empty patterns allow all",
			patterns: []string{},
			url:      "https://example.com/any/path",
			want:     true,
		},
		{
			name:     "exact domain match",
			patterns: []string{"^https://example\\.com"},
			url:      "https://example.com/path",
			want:     true,
		},
		{
			name:     "domain not matching",
			patterns: []string{"^https://example\\.com"},
			url:      "https://other.com/path",
			want:     false,
		},
		{
			name:     "multiple patterns, first matches",
			patterns: []string{"^https://api\\.example\\.com", "^https://www\\.example\\.com"},
			url:      "https://api.example.com/v1/data",
			want:     true,
		},
		{
			name:     "multiple patterns, second matches",
			patterns: []string{"^https://api\\.example\\.com", "^https://www\\.example\\.com"},
			url:      "https://www.example.com/index.html",
			want:     true,
		},
		{
			name:     "multiple patterns, none match",
			patterns: []string{"^https://api\\.example\\.com", "^https://www\\.example\\.com"},
			url:      "https://other.com/path",
			want:     false,
		},
		{
			name:     "path-based filtering",
			patterns: []string{"^https://(www\\.)?example\\.com/docs/"},
			url:      "https://example.com/docs/api.html",
			want:     true,
		},
		{
			name:     "path-based filtering, non-matching path",
			patterns: []string{"^https://(www\\.)?example\\.com/docs/"},
			url:      "https://example.com/other/api.html",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.patterns)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}
			got := f.Allowed(tt.url)
			if got != tt.want {
				t.Errorf("Allowed(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestFilter_Middleware(t *testing.T) {
	tests := []struct {
		name           string
		patterns       []string
		requestURL     string
		expectedStatus int
	}{
		{
			name:           "allowed request passes through",
			patterns:       []string{"^https://example\\.com"},
			requestURL:     "https://example.com/path",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blocked request returns 403",
			patterns:       []string{"^https://example\\.com"},
			requestURL:     "https://other.com/path",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "empty patterns allow all",
			patterns:       []string{},
			requestURL:     "https://any.com/path",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.patterns)
			if err != nil {
				t.Fatalf("New failed: %v", err)
			}

			// Create a handler that responds with 200 OK if reached
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := f.Middleware(nextHandler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.requestURL, nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.expectedStatus)
			}
		})
	}
}
