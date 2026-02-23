package converter

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cbroglie/mustache"
)

// IsJSONContentType returns true if the content type header indicates JSON.
func IsJSONContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.Contains(ct, "application/json")
}

// JSONToMarkdown converts a JSON byte slice to Markdown.
// If mustacheTemplate is non-empty, the JSON data is rendered through it.
// If mustacheTemplate is empty, a template is auto-generated from the JSON shape.
func JSONToMarkdown(jsonBytes []byte, mustacheTemplate string) (string, error) {
	var data any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return "", fmt.Errorf("parsing JSON: %w", err)
	}

	tpl := mustacheTemplate
	if tpl == "" {
		tpl = GenerateTemplate(data)
	}

	result, err := mustache.Render(tpl, data)
	if err != nil {
		return "", fmt.Errorf("rendering mustache template: %w", err)
	}

	return strings.TrimSpace(result), nil
}

// GenerateTemplate inspects the JSON structure and produces a Mustache
// template that will render as well-formatted Markdown.
func GenerateTemplate(data any) string {
	var b strings.Builder
	generateTemplateRecursive(&b, data, "", 2)
	return b.String()
}

// generateTemplateRecursive builds the Mustache template recursively.
// prefix is the Mustache context path (empty at top level).
// headingLevel controls the Markdown heading depth.
func generateTemplateRecursive(b *strings.Builder, data any, prefix string, headingLevel int) {
	switch v := data.(type) {
	case map[string]any:
		keys := sortedKeys(v)
		for _, key := range keys {
			val := v[key]
			ref := key
			if prefix != "" {
				ref = prefix + "." + key
			}
			heading := strings.Repeat("#", headingLevel)
			b.WriteString(fmt.Sprintf("%s %s\n\n", heading, key))

			switch child := val.(type) {
			case map[string]any:
				// Nested object: recurse with deeper heading.
				generateTemplateRecursive(b, child, ref, headingLevel+1)
			case []any:
				generateArrayTemplate(b, child, key, headingLevel+1)
			default:
				// Primitive value: emit unescaped.
				b.WriteString(fmt.Sprintf("{{{%s}}}\n\n", ref))
			}
		}

	case []any:
		generateArrayTemplate(b, v, ".", 2)

	default:
		// Top-level primitive.
		b.WriteString("{{{.}}}\n")
	}
}

// generateArrayTemplate writes a Mustache template for a JSON array.
// It detects arrays of objects (renders as table) vs arrays of primitives (renders as list).
func generateArrayTemplate(b *strings.Builder, arr []any, sectionKey string, headingLevel int) {
	if len(arr) == 0 {
		b.WriteString(fmt.Sprintf("{{#%s}}\n{{/%s}}\n\n", sectionKey, sectionKey))
		return
	}

	// Check if all elements are objects with the same keys (→ table).
	if cols := consistentObjectKeys(arr); cols != nil {
		// Table header.
		b.WriteString("| " + strings.Join(cols, " | ") + " |\n")
		b.WriteString("|" + strings.Repeat("---|", len(cols)) + "\n")
		// Table rows via Mustache section.
		b.WriteString(fmt.Sprintf("{{#%s}}\n", sectionKey))
		cells := make([]string, len(cols))
		for i, col := range cols {
			cells[i] = fmt.Sprintf("{{{%s}}}", col)
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
		b.WriteString(fmt.Sprintf("{{/%s}}\n\n", sectionKey))
		return
	}

	// Check if all elements are primitives (→ bulleted list).
	if allPrimitives(arr) {
		b.WriteString(fmt.Sprintf("{{#%s}}\n- {{{.}}}\n{{/%s}}\n\n", sectionKey, sectionKey))
		return
	}

	// Mixed array: render each element as a sub-section.
	b.WriteString(fmt.Sprintf("{{#%s}}\n", sectionKey))
	b.WriteString("- {{{.}}}\n")
	b.WriteString(fmt.Sprintf("{{/%s}}\n\n", sectionKey))
}

// consistentObjectKeys returns the sorted list of keys if every element in arr
// is a map[string]any with the exact same keys. Returns nil otherwise.
func consistentObjectKeys(arr []any) []string {
	if len(arr) == 0 {
		return nil
	}
	first, ok := arr[0].(map[string]any)
	if !ok {
		return nil
	}
	keys := sortedKeys(first)
	keySet := strings.Join(keys, "\x00")

	for _, elem := range arr[1:] {
		m, ok := elem.(map[string]any)
		if !ok {
			return nil
		}
		if strings.Join(sortedKeys(m), "\x00") != keySet {
			return nil
		}
	}
	return keys
}

// allPrimitives returns true if every element in arr is a JSON primitive
// (string, float64, bool, or nil).
func allPrimitives(arr []any) bool {
	for _, elem := range arr {
		switch elem.(type) {
		case string, float64, bool, nil:
			// ok
		default:
			return false
		}
	}
	return true
}

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
