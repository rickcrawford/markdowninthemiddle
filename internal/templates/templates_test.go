package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_LoadsTemplates(t *testing.T) {
	dir := t.TempDir()

	// Create a pattern-matched template.
	os.WriteFile(filepath.Join(dir, "api.example.com__users.mustache"), []byte("# Users\n{{#users}}\n- {{{name}}}\n{{/users}}"), 0644)

	// Create a default template.
	os.WriteFile(filepath.Join(dir, "_default.mustache"), []byte("# Default\n{{{.}}}"), 0644)

	// Create a non-mustache file (should be ignored).
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a template"), 0644)

	store, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.templates) != 1 {
		t.Errorf("expected 1 pattern template, got %d", len(store.templates))
	}

	if store.defaultTemplate == "" {
		t.Error("expected default template to be loaded")
	}
}

func TestStore_Match_ExactPrefix(t *testing.T) {
	store := &Store{
		templates: map[string]string{
			"http://api.example.com/users": "users-template",
			"http://api.example.com/products": "products-template",
		},
	}

	got := store.Match("http://api.example.com/users?page=1")
	if got != "users-template" {
		t.Errorf("expected users-template, got %q", got)
	}

	got = store.Match("http://api.example.com/products/123")
	if got != "products-template" {
		t.Errorf("expected products-template, got %q", got)
	}
}

func TestStore_Match_LongestPrefix(t *testing.T) {
	store := &Store{
		templates: map[string]string{
			"http://api.example.com/":         "broad-template",
			"http://api.example.com/users":    "users-template",
		},
	}

	got := store.Match("http://api.example.com/users/123")
	if got != "users-template" {
		t.Errorf("expected users-template (longest match), got %q", got)
	}
}

func TestStore_Match_FallbackToDefault(t *testing.T) {
	store := &Store{
		templates:       map[string]string{},
		defaultTemplate: "default-tpl",
	}

	got := store.Match("http://unknown.com/api")
	if got != "default-tpl" {
		t.Errorf("expected default template, got %q", got)
	}
}

func TestStore_Match_NoMatch(t *testing.T) {
	store := &Store{
		templates: map[string]string{
			"http://api.example.com/users": "users-template",
		},
	}

	got := store.Match("http://other.com/api")
	if got != "" {
		t.Errorf("expected empty string for no match, got %q", got)
	}
}

func TestStore_Match_NilStore(t *testing.T) {
	var store *Store
	got := store.Match("http://example.com")
	if got != "" {
		t.Errorf("expected empty string for nil store, got %q", got)
	}
}

func TestNew_FilenameToPattern(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "api.example.com__v1__users.mustache"), []byte("template"), 0644)

	store, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "__" should be converted to "/"
	if _, ok := store.templates["api.example.com/v1/users"]; !ok {
		t.Errorf("expected pattern 'api.example.com/v1/users', got templates: %v", store.templates)
	}
}

func TestNew_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	store, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.templates) != 0 {
		t.Errorf("expected 0 templates, got %d", len(store.templates))
	}
	if store.defaultTemplate != "" {
		t.Error("expected empty default template")
	}
}

func TestNew_InvalidDir(t *testing.T) {
	_, err := New("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestStore_Match_SchemeStripping(t *testing.T) {
	dir := t.TempDir()

	// Create a template file with no scheme (as would come from filename)
	os.WriteFile(filepath.Join(dir, "api.example.com__users.mustache"), []byte("users-template"), 0644)

	store, err := New(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Match against a full URL with scheme
	got := store.Match("http://api.example.com/users?page=1")
	if got != "users-template" {
		t.Errorf("expected users-template, got %q", got)
	}

	// Also test with https
	got = store.Match("https://api.example.com/users/123")
	if got != "users-template" {
		t.Errorf("expected users-template with https, got %q", got)
	}
}
