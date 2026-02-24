package mcp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew_CreatesServer(t *testing.T) {
	deps := Deps{
		HTTPClient: &http.Client{},
	}

	s := New(deps)
	if s == nil {
		t.Fatal("New should return a non-nil server")
	}
}

func TestHandler_IsHTML(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"application/xhtml+xml", true},
		{"application/json", false},
		{"text/plain", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isHTML(tt.contentType)
		if result != tt.expected {
			t.Errorf("isHTML(%q) = %v, want %v", tt.contentType, result, tt.expected)
		}
	}
}

func TestHandler_IsJSON(t *testing.T) {
	tests := []struct {
		contentType string
		expected    bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/ld+json", true},
		{"text/html", false},
		{"text/plain", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isJSON(tt.contentType)
		if result != tt.expected {
			t.Errorf("isJSON(%q) = %v, want %v", tt.contentType, result, tt.expected)
		}
	}
}

func TestHandler_MockHTTPServer(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test</h1></body></html>"))
	}))
	defer mockServer.Close()

	// Verify the mock server works
	resp, err := mockServer.Client().Get(mockServer.URL)
	if err != nil {
		t.Fatalf("Failed to fetch from mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
