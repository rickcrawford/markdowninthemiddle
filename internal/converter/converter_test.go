package converter

import (
	"strings"
	"testing"
)

func TestHTMLToMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		contains string
	}{
		{
			name:     "simple paragraph",
			html:     "<p>Hello world</p>",
			contains: "Hello world",
		},
		{
			name:     "heading",
			html:     "<h1>Title</h1>",
			contains: "# Title",
		},
		{
			name:     "link",
			html:     `<a href="https://example.com">click</a>`,
			contains: "[click](https://example.com)",
		},
		{
			name:     "bold text",
			html:     "<strong>bold</strong>",
			contains: "**bold**",
		},
		{
			name:     "unordered list",
			html:     "<ul><li>one</li><li>two</li></ul>",
			contains: "- one",
		},
		{
			name:     "complex html",
			html:     "<div><h2>Section</h2><p>Text with <em>emphasis</em> and <code>code</code>.</p></div>",
			contains: "## Section",
		},
		{
			name:     "empty html",
			html:     "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md, err := HTMLToMarkdown(tt.html)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.contains != "" && !strings.Contains(md, tt.contains) {
				t.Errorf("expected markdown to contain %q, got %q", tt.contains, md)
			}
		})
	}
}

func TestIsHTMLContentType(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"TEXT/HTML", true},
		{"text/plain", false},
		{"application/json", false},
		{"text/markdown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			got := IsHTMLContentType(tt.ct)
			if got != tt.want {
				t.Errorf("IsHTMLContentType(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func BenchmarkHTMLToMarkdown_Simple(b *testing.B) {
	html := "<p>Hello world</p>"
	for b.Loop() {
		HTMLToMarkdown(html)
	}
}

func BenchmarkHTMLToMarkdown_Complex(b *testing.B) {
	html := `
	<html><body>
	<h1>Page Title</h1>
	<p>First paragraph with <strong>bold</strong> and <em>italic</em> text.</p>
	<h2>Section Two</h2>
	<ul>
		<li>Item one with <a href="https://example.com">a link</a></li>
		<li>Item two with <code>inline code</code></li>
		<li>Item three</li>
	</ul>
	<p>Another paragraph with more content.</p>
	<table>
		<tr><th>Header 1</th><th>Header 2</th></tr>
		<tr><td>Cell 1</td><td>Cell 2</td></tr>
		<tr><td>Cell 3</td><td>Cell 4</td></tr>
	</table>
	</body></html>`
	for b.Loop() {
		HTMLToMarkdown(html)
	}
}
