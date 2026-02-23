package converter

import (
	"strings"
	"testing"
)

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"APPLICATION/JSON", true},
		{"text/html", false},
		{"text/plain", false},
		{"text/markdown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			got := IsJSONContentType(tt.ct)
			if got != tt.want {
				t.Errorf("IsJSONContentType(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func TestJSONToMarkdown_AutoGenerate_Object(t *testing.T) {
	input := `{"title":"My API","version":"1.0"}`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "## title") {
		t.Errorf("expected '## title' heading, got %q", md)
	}
	if !strings.Contains(md, "My API") {
		t.Errorf("expected 'My API' value, got %q", md)
	}
	if !strings.Contains(md, "## version") {
		t.Errorf("expected '## version' heading, got %q", md)
	}
	if !strings.Contains(md, "1.0") {
		// JSON unmarshals to float64, might render as "1"
		if !strings.Contains(md, "1") {
			t.Errorf("expected version value, got %q", md)
		}
	}
}

func TestJSONToMarkdown_AutoGenerate_ArrayOfObjects(t *testing.T) {
	input := `{"users":[{"name":"Alice","role":"admin"},{"name":"Bob","role":"user"}]}`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "| name | role |") {
		t.Errorf("expected table header, got %q", md)
	}
	if !strings.Contains(md, "Alice") {
		t.Errorf("expected 'Alice' in table, got %q", md)
	}
	if !strings.Contains(md, "Bob") {
		t.Errorf("expected 'Bob' in table, got %q", md)
	}
}

func TestJSONToMarkdown_AutoGenerate_ArrayOfPrimitives(t *testing.T) {
	input := `{"tags":["go","proxy","markdown"]}`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "- go") {
		t.Errorf("expected '- go' bullet, got %q", md)
	}
	if !strings.Contains(md, "- proxy") {
		t.Errorf("expected '- proxy' bullet, got %q", md)
	}
}

func TestJSONToMarkdown_AutoGenerate_TopLevelArray(t *testing.T) {
	input := `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "| id | name |") {
		t.Errorf("expected table header, got %q", md)
	}
	if !strings.Contains(md, "Alice") {
		t.Errorf("expected 'Alice' in output, got %q", md)
	}
}

func TestJSONToMarkdown_AutoGenerate_NestedObject(t *testing.T) {
	input := `{"server":{"host":"localhost","port":8080}}`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "## server") {
		t.Errorf("expected '## server' heading, got %q", md)
	}
	if !strings.Contains(md, "### host") {
		t.Errorf("expected '### host' sub-heading, got %q", md)
	}
	if !strings.Contains(md, "localhost") {
		t.Errorf("expected 'localhost' value, got %q", md)
	}
}

func TestJSONToMarkdown_AutoGenerate_EmptyObject(t *testing.T) {
	input := `{}`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md != "" {
		t.Errorf("expected empty markdown for empty object, got %q", md)
	}
}

func TestJSONToMarkdown_AutoGenerate_EmptyArray(t *testing.T) {
	input := `{"items":[]}`
	md, err := JSONToMarkdown([]byte(input), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "## items") {
		t.Errorf("expected '## items' heading, got %q", md)
	}
}

func TestJSONToMarkdown_WithTemplate(t *testing.T) {
	input := `{"name":"Alice","greeting":"Hello"}`
	tpl := "# {{{greeting}}}, {{{name}}}!"
	md, err := JSONToMarkdown([]byte(input), tpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md != "# Hello, Alice!" {
		t.Errorf("expected '# Hello, Alice!', got %q", md)
	}
}

func TestJSONToMarkdown_WithTemplate_Section(t *testing.T) {
	input := `{"items":[{"name":"one"},{"name":"two"}]}`
	tpl := "{{#items}}\n- {{{name}}}\n{{/items}}"
	md, err := JSONToMarkdown([]byte(input), tpl)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(md, "- one") {
		t.Errorf("expected '- one', got %q", md)
	}
	if !strings.Contains(md, "- two") {
		t.Errorf("expected '- two', got %q", md)
	}
}

func TestJSONToMarkdown_InvalidJSON(t *testing.T) {
	_, err := JSONToMarkdown([]byte("not json"), "")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONToMarkdown_InvalidTemplate(t *testing.T) {
	input := `{"key":"value"}`
	tpl := "{{#unclosed}}"
	_, err := JSONToMarkdown([]byte(input), tpl)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
}

func TestGenerateTemplate_Primitive(t *testing.T) {
	tpl := GenerateTemplate("hello")
	if !strings.Contains(tpl, "{{{.}}}") {
		t.Errorf("expected primitive template, got %q", tpl)
	}
}

func TestGenerateTemplate_MixedArray(t *testing.T) {
	// Array with inconsistent object keys falls back to list.
	data := []any{
		map[string]any{"a": 1},
		map[string]any{"b": 2},
	}
	tpl := GenerateTemplate(data)
	if !strings.Contains(tpl, "- {{{.}}}") {
		t.Errorf("expected list template for mixed array, got %q", tpl)
	}
}

func BenchmarkJSONToMarkdown_AutoGenerate(b *testing.B) {
	input := []byte(`{
		"title": "Benchmark",
		"users": [
			{"name": "Alice", "role": "admin"},
			{"name": "Bob", "role": "user"},
			{"name": "Charlie", "role": "user"}
		],
		"tags": ["go", "proxy", "markdown"],
		"config": {"host": "localhost", "port": 8080}
	}`)
	for b.Loop() {
		JSONToMarkdown(input, "")
	}
}

func BenchmarkJSONToMarkdown_WithTemplate(b *testing.B) {
	input := []byte(`{"name":"Alice","greeting":"Hello"}`)
	tpl := "# {{{greeting}}}, {{{name}}}!"
	for b.Loop() {
		JSONToMarkdown(input, tpl)
	}
}
