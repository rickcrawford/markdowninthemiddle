# Plan: JSON-to-Markdown via Mustache Templates

## Problem

Currently the proxy only converts `text/html` responses to Markdown. JSON API responses (`application/json`) pass through untouched. We want to optionally convert JSON responses to Markdown using Mustache templates — with support for both **user-defined templates** (per URL pattern) and **auto-generated templates** that dynamically produce reasonable Markdown from an arbitrary JSON shape.

## Research Findings

### Mustache Library

**[cbroglie/mustache](https://github.com/cbroglie/mustache)** — actively maintained fork of hoisie/mustache with idiomatic Go error handling (`(string, error)` returns), Mustache spec v1.2.1 compliance, custom partial support, and missing variable error options.

### Auto-Generation Strategy

The key insight: Mustache operates on the same data structures as `encoding/json` unmarshals to (`map[string]any`, `[]any`). We can walk the JSON structure at runtime and produce a Mustache template that generates valid Markdown:

| JSON Type | Markdown Output |
|---|---|
| Object (top-level) | Iterate keys as `## Key` headings with values as content |
| Array of objects | Markdown table with keys as column headers |
| Array of primitives | Bulleted list |
| Nested object | Recursive sub-section with incremented heading level |
| Primitive (string, number, bool) | Inline value |

## Architecture

### New Package: `internal/converter/json.go`

Add to the existing `converter` package (not a new package — it's the natural home alongside `HTMLToMarkdown`):

```go
// IsJSONContentType checks Content-Type for application/json.
func IsJSONContentType(ct string) bool

// JSONToMarkdown converts a JSON byte slice to Markdown.
// If a Mustache template is provided, it renders JSON data through the template.
// If template is empty, it auto-generates a template from the JSON shape.
func JSONToMarkdown(jsonBytes []byte, mustacheTemplate string) (string, error)

// GenerateTemplate inspects the JSON structure and produces a Mustache
// template that will render as well-formatted Markdown.
func GenerateTemplate(data any) string
```

### New Package: `internal/templates/templates.go`

Manages user-defined Mustache templates matched by URL pattern:

```go
// Store holds Mustache templates keyed by URL glob patterns.
type Store struct { ... }

// New loads templates from a directory. Each .mustache file's name
// (without extension) is treated as a URL glob pattern (with "__" for "/").
func New(dir string) (*Store, error)

// Match returns the template string for the best-matching URL pattern,
// or empty string if no match (triggering auto-generation).
func (s *Store) Match(rawURL string) string
```

### Config Changes (`internal/config/config.go`)

Add to `ConversionConfig`:

```go
type ConversionConfig struct {
    Enabled          bool   `mapstructure:"enabled"`
    TiktokenEncoding string `mapstructure:"tiktoken_encoding"`
    NegotiateOnly    bool   `mapstructure:"negotiate_only"`
    ConvertJSON      bool   `mapstructure:"convert_json"`
    TemplateDir      string `mapstructure:"template_dir"`
}
```

### config.yml additions

```yaml
conversion:
  convert_json: false
  template_dir: ""  # directory of .mustache files for URL-matched templates
```

### Middleware Changes (`internal/middleware/middleware.go`)

Extend `ResponseProcessor`:

```go
type ResponseProcessor struct {
    // ... existing fields ...
    ConvertJSON   bool
    TemplateStore *templates.Store  // nil = auto-generate only
}
```

In `RoundTrip`, after the existing HTML check, add a parallel JSON path:

```go
if converter.IsJSONContentType(ct) && shouldConvertJSON {
    // 1. Decompress + read body (reuse existing logic)
    // 2. Look up user template: tpl := rp.TemplateStore.Match(req.URL.String())
    // 3. Convert: md, err := converter.JSONToMarkdown(jsonBytes, tpl)
    // 4. Token count, output write, set headers (same as HTML path)
}
```

### CLI Flag

```
--convert-json    Enable JSON-to-Markdown conversion (overrides config)
--template-dir    Directory containing .mustache template files
```

## Auto-Generation Algorithm (Detail)

```
func GenerateTemplate(data any) string:
    switch v := data.(type) {
    case map[string]any:
        For each key/value pair:
            emit "## {key}\n\n"
            if value is []any of maps → emit table
            if value is []any of primitives → emit list
            if value is map → recurse with ### heading level
            if value is primitive → emit "{{{key}}}\n\n"

    case []any:
        If all elements are maps with consistent keys → emit table
        Otherwise → emit bulleted list

    default (primitive at top level):
        emit "{{{.}}}\n"
    }
```

Example — given this JSON:

```json
{
  "title": "My API",
  "users": [
    {"name": "Alice", "role": "admin"},
    {"name": "Bob", "role": "user"}
  ],
  "tags": ["go", "proxy", "markdown"]
}
```

Auto-generated template:

```mustache
## title

{{{title}}}

## users

| name | role |
|---|---|
{{#users}}
| {{{name}}} | {{{role}}} |
{{/users}}

## tags

{{#tags}}
- {{{.}}}
{{/tags}}
```

Rendered Markdown:

```markdown
## title

My API

## users

| name | role |
|---|---|
| Alice | admin |
| Bob | user |

## tags

- go
- proxy
- markdown
```

## User-Defined Templates

Users place `.mustache` files in the template directory. Filename conventions map to URL patterns:

```
templates/
  api.example.com__users.mustache     → matches http://api.example.com/users*
  api.example.com__products.mustache  → matches http://api.example.com/products*
  _default.mustache                   → fallback for all JSON responses
```

If a user template matches, it's used instead of auto-generation. This allows fine-grained control over Markdown output for known API endpoints.

## Implementation Steps

1. `go get github.com/cbroglie/mustache`
2. Add `IsJSONContentType()` and `JSONToMarkdown()` to `internal/converter/json.go`
3. Add `GenerateTemplate()` with the auto-generation algorithm
4. Create `internal/templates/templates.go` for URL-matched template store
5. Add `ConvertJSON` + `TemplateDir` to config, CLI flags, and proxy Options
6. Extend `ResponseProcessor.RoundTrip` with the JSON conversion branch
7. Unit tests for: JSON detection, auto-template generation (various shapes), template rendering, URL matching, full middleware pipeline
8. Benchmark tests for: auto-generation, template rendering, full JSON→Markdown pipeline
9. Update README with JSON conversion docs
10. Commit and push

## Trade-offs & Considerations

- **Auto-generation is best-effort**: Deeply nested or irregular JSON may produce verbose Markdown. User templates are the escape hatch.
- **Triple braces `{{{...}}}`**: We use unescaped output since Mustache defaults to HTML-escaping, but we're producing Markdown, not HTML.
- **Array-of-objects detection**: If all items in an array are objects with the same keys, render as a table. If keys vary, fall back to per-item sections.
- **Large JSON payloads**: Subject to the existing `max_body_size` limit.
- **Content negotiation**: The `--negotiate-only` flag would also apply to JSON conversion (only convert when `Accept: text/markdown` is sent).
