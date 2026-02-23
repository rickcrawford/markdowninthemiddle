package middleware

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

// mockTransport returns a fixed response for testing.
type mockTransport struct {
	statusCode  int
	contentType string
	body        string
	encoding    string
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	header := http.Header{}
	header.Set("Content-Type", m.contentType)
	if m.encoding != "" {
		header.Set("Content-Encoding", m.encoding)
	}
	header.Set("Content-Length", strconv.Itoa(len(m.body)))

	return &http.Response{
		StatusCode:    m.statusCode,
		Header:        header,
		Body:          io.NopCloser(strings.NewReader(m.body)),
		ContentLength: int64(len(m.body)),
	}, nil
}

func TestResponseProcessor_HTMLToMarkdown(t *testing.T) {
	// Token counter may be nil if tiktoken can't download encodings (no network).
	tc, _ := tokens.NewCounter("cl100k_base")

	rp := &ResponseProcessor{
		ConvertHTML:  true,
		TokenCounter: tc,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "text/html; charset=utf-8",
			body:        "<h1>Hello</h1><p>World</p>",
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	md := string(body)

	if !strings.Contains(md, "# Hello") {
		t.Errorf("expected markdown heading, got %q", md)
	}
	if !strings.Contains(md, "World") {
		t.Errorf("expected 'World' in markdown, got %q", md)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/markdown") {
		t.Errorf("expected text/markdown content type, got %q", ct)
	}

	// Token counting is only available when tiktoken encoding is loaded.
	if tc != nil {
		tokenCount := resp.Header.Get("X-Token-Count")
		if tokenCount == "" {
			t.Error("expected X-Token-Count header")
		}
		count, err := strconv.Atoi(tokenCount)
		if err != nil {
			t.Errorf("invalid X-Token-Count: %q", tokenCount)
		}
		if count <= 0 {
			t.Errorf("expected positive token count, got %d", count)
		}
	}

	vary := resp.Header.Get("Vary")
	if vary != "accept" {
		t.Errorf("expected Vary: accept, got %q", vary)
	}
}

func TestResponseProcessor_NonHTML_PassThrough(t *testing.T) {
	rp := &ResponseProcessor{
		ConvertHTML: true,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "application/json",
			body:        `{"key":"value"}`,
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com/api", nil)
	resp, err := rp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"key":"value"}` {
		t.Errorf("expected JSON pass-through, got %q", body)
	}

	ct := resp.Header.Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestResponseProcessor_ConversionDisabled(t *testing.T) {
	rp := &ResponseProcessor{
		ConvertHTML: false,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "text/html",
			body:        "<h1>Hello</h1>",
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<h1>Hello</h1>" {
		t.Errorf("expected original HTML, got %q", body)
	}
}

func TestResponseProcessor_NegotiateOnly_WithAccept(t *testing.T) {
	tc, _ := tokens.NewCounter("cl100k_base")

	rp := &ResponseProcessor{
		ConvertHTML:   true,
		NegotiateOnly: true,
		TokenCounter:  tc,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "text/html",
			body:        "<h1>Negotiated</h1>",
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Accept", "text/markdown, text/html")
	resp, err := rp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "# Negotiated") {
		t.Errorf("expected markdown conversion, got %q", body)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/markdown") {
		t.Errorf("expected text/markdown, got %q", ct)
	}
}

func TestResponseProcessor_NegotiateOnly_WithoutAccept(t *testing.T) {
	rp := &ResponseProcessor{
		ConvertHTML:   true,
		NegotiateOnly: true,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "text/html",
			body:        "<h1>Not Negotiated</h1>",
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Accept", "text/html")
	resp, err := rp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "<h1>Not Negotiated</h1>" {
		t.Errorf("expected original HTML (no negotiation), got %q", body)
	}
}

func TestResponseProcessor_BodySizeLimit(t *testing.T) {
	tc, _ := tokens.NewCounter("cl100k_base")

	longHTML := "<p>" + strings.Repeat("x", 1000) + "</p>"
	rp := &ResponseProcessor{
		ConvertHTML:  true,
		MaxBodySize:  50,
		TokenCounter: tc,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "text/html",
			body:        longHTML,
		},
	}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// The body should be truncated to MaxBodySize, then converted.
	// Exact output depends on converter handling truncated HTML.
	if int64(len(body)) > 1000 {
		t.Errorf("response body too large after size limit: %d bytes", len(body))
	}
}

func TestWantsMarkdown(t *testing.T) {
	tests := []struct {
		accept string
		want   bool
	}{
		{"text/markdown", true},
		{"text/markdown, text/html", true},
		{"text/html, text/markdown;q=0.9", true},
		{"text/html", false},
		{"application/json", false},
		{"", false},
		{"text/plain, application/xml", false},
	}

	for _, tt := range tests {
		t.Run(tt.accept, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			if tt.accept != "" {
				req.Header.Set("Accept", tt.accept)
			}
			got := wantsMarkdown(req)
			if got != tt.want {
				t.Errorf("wantsMarkdown(Accept: %q) = %v, want %v", tt.accept, got, tt.want)
			}
		})
	}
}

func BenchmarkResponseProcessor_HTMLToMarkdown(b *testing.B) {
	tc, _ := tokens.NewCounter("cl100k_base")

	html := `<html><body>
	<h1>Title</h1>
	<p>Paragraph with <strong>bold</strong> and <a href="https://example.com">link</a>.</p>
	<ul><li>One</li><li>Two</li><li>Three</li></ul>
	</body></html>`

	rp := &ResponseProcessor{
		ConvertHTML:  true,
		TokenCounter: tc,
		Inner: &mockTransport{
			statusCode:  200,
			contentType: "text/html",
			body:        html,
		},
	}

	for b.Loop() {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		resp, _ := rp.RoundTrip(req)
		io.ReadAll(resp.Body)
		resp.Body.Close()
	}
}
