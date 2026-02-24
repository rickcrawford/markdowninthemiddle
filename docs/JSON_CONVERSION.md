# JSON-to-Markdown Conversion

Convert JSON API responses to clean, structured Markdown using **Mustache templates**. Perfect for feeding API data to Claude and other LLMs.

---

## Quick Start

```bash
# Copy example templates
cp -r examples/mustache-templates ~/my-templates

# Enable JSON conversion in proxy
./markdowninthemiddle --convert-json --template-dir ~/my-templates

# Test with an API
curl -x http://localhost:8080 https://api.github.com/users/octocat
```

The API response (JSON) is automatically converted to Markdown based on your templates.

---

## How It Works

### 1. Request Flow

```
JSON API Response
        ↓
Content-Type: application/json detected
        ↓
Template matching by URL pattern
        ↓
Mustache template + JSON data
        ↓
Markdown output (or auto-generated if no template)
```

### 2. Template Selection

Templates are matched against request URLs using **longest prefix match**:

| Filename | Matches URLs |
|----------|---|
| `_default.mustache` | Fallback for all JSON (no specific template found) |
| `api.example.com.mustache` | `http://api.example.com/*` (host-only) |
| `api.example.com__users.mustache` | `http://api.example.com/users*` |
| `api.example.com__v1__posts.mustache` | `http://api.example.com/v1/posts*` |

**Note:** URL schemes (`http://`, `https://`) are stripped before matching.

---

## Configuration

### Enable JSON Conversion

**CLI flag:**
```bash
./markdowninthemiddle --convert-json --template-dir ./my-templates
```

**Environment variable:**
```bash
MITM_CONVERSION_CONVERT_JSON=true \
MITM_CONVERSION_TEMPLATE_DIR="./my-templates" \
./markdowninthemiddle
```

**Config file (config.yml):**
```yaml
conversion:
  convert_json: true                    # Enable JSON conversion
  template_dir: "./my-templates"        # Directory with .mustache files
```

### Auto-Generation (No Templates)

If `convert_json: true` but no templates found, templates are **auto-generated** from the JSON structure:

```bash
# No templates - auto-generate from JSON
./markdowninthemiddle --convert-json
# Outputs generic Markdown: **field**: value
```

---

## Creating Templates

### Template Syntax (Mustache)

Mustache is a simple templating language:

```mustache
# {{title}}

**ID**: {{id}}

{{#description}}
{{{description}}}
{{/description}}

{{#items}}
- {{name}} ({{type}})
{{/items}}
```

**Key syntax:**
- `{{variable}}` - Escaped HTML (safe output)
- `{{{variable}}}` - Unescaped HTML (raw output for rich content)
- `{{#section}}...{{/section}}` - Conditional/loop (renders if truthy)
- `{{^section}}...{{/section}}` - Inverse (renders if falsy)

### Step-by-Step Example

#### 1. Examine the JSON

```bash
curl https://api.github.com/users/octocat | jq .
```

Output:
```json
{
  "login": "octocat",
  "id": 1,
  "type": "User",
  "name": "The Octocat",
  "company": "GitHub",
  "blog": "https://github.blog",
  "location": "San Francisco",
  "bio": "There once was...",
  "public_repos": 2,
  "followers": 3938,
  "following": 9,
  "html_url": "https://github.com/octocat"
}
```

#### 2. Create the Template

File: `api.github.com__users.mustache`

```mustache
# {{login}}

**ID**: {{id}} | **Type**: {{type}}

{{#name}}## {{name}}
{{/name}}

{{#company}}**Company**: {{company}}
{{/company}}

{{#blog}}**Blog**: [{{blog}}]({{blog}})
{{/blog}}

{{#location}}📍 {{location}}
{{/location}}

{{#bio}}
{{bio}}
{{/bio}}

---

**Stats:**
- Public Repos: {{public_repos}}
- Followers: {{followers}}
- Following: {{following}}

**Profile**: [View on GitHub]({{html_url}})
```

#### 3. Place Template

```bash
mkdir -p my-templates
cp api.github.com__users.mustache my-templates/
```

#### 4. Use It

```bash
./markdowninthemiddle --convert-json --template-dir ./my-templates

curl -x http://localhost:8080 https://api.github.com/users/octocat
```

**Output** (converted to Markdown):
```markdown
# octocat

**ID**: 1 | **Type**: User

## The Octocat

**Company**: GitHub

**Blog**: [https://github.blog](https://github.blog)

📍 San Francisco

There once was...

---

**Stats:**
- Public Repos: 2
- Followers: 3938
- Following: 9

**Profile**: [View on GitHub](https://github.com/octocat)
```

---

## Template Naming Convention

Use **double underscore `__`** as path separators in filenames:

```
api.example.com__users.mustache
                └── converted to /
                    api.example.com/users
```

### Common Patterns

| Filename | Matches |
|----------|---------|
| `api.github.com__users__{{username}}.mustache` | `api.github.com/users/:username` |
| `api.example.com__v1__posts.mustache` | `api.example.com/v1/posts` |
| `api.example.com__v2__posts.mustache` | `api.example.com/v2/posts` (separate v2 template) |
| `_default.mustache` | Any unmatched JSON (fallback) |

---

## Use Cases

### 1. LLM API Integration

```bash
# Proxy internal APIs for Claude
./markdowninthemiddle --convert-json --template-dir ./api-templates

# Claude via MCP can now call APIs and get clean Markdown
./markdowninthemiddle mcp --convert-json --template-dir ./api-templates
```

### 2. API Documentation

```bash
# Convert live API responses to Markdown docs
curl -x http://localhost:8080 https://api.example.com/endpoints | tee api-docs.md
```

### 3. Data Processing Pipeline

```bash
# Save converted JSON as Markdown files
./markdowninthemiddle --convert-json --output-dir ./markdown

# Process with tools that consume Markdown
```

### 4. Conditional Formatting

```mustache
{{#is_premium}}
⭐ Premium User
{{/is_premium}}

{{^is_premium}}
Free Plan
{{/is_premium}}
```

---

## Advanced Features

### Nested Objects

For nested JSON, use dot notation:

```json
{
  "user": {
    "name": "John",
    "address": {
      "city": "NYC"
    }
  }
}
```

```mustache
**Name**: {{user.name}}
**City**: {{user.address.city}}
```

### Arrays & Loops

```json
{
  "repositories": [
    { "name": "repo1", "stars": 100 },
    { "name": "repo2", "stars": 200 }
  ]
}
```

```mustache
## Repositories

{{#repositories}}
- **{{name}}** - {{stars}} stars
{{/repositories}}
```

### Custom Formatting

Use unescaped HTML for custom formatting:

```mustache
{{#description}}
> {{{description}}}
{{/description}}
```

---

## MCP Server Integration

Use JSON conversion in Claude Desktop via MCP:

```bash
# Start MCP server with templates
./markdowninthemiddle mcp --convert-json --template-dir ./my-templates
```

Claude can now fetch APIs and get beautifully formatted Markdown:

```
User: Can you fetch and summarize https://api.github.com/users/octocat?

Claude: [Uses fetch_markdown tool with JSON template]
# octocat
**ID**: 1 | **Type**: User
...
```

---

## Debugging

### Check Template Match

Enable debug logging:

```bash
./markdowninthemiddle --convert-json --template-dir ./my-templates --log-level debug
```

Log output shows which template matched (or if fallback was used).

### Test Template Locally

You can test Mustache templates offline:

```bash
# Using a Mustache CLI tool
echo '{"name":"John"}' | mstch template.mustache

# Or use an online renderer: https://mustache.github.io/mustache.5.html
```

### Common Issues

**Template not matching:**
- Check filename follows `host__path.mustache` pattern
- Verify `://` (scheme) is NOT in filename
- Test with `-x http://localhost:8080 https://api.example.com/path`

**JSON not converting:**
- Ensure `Content-Type: application/json` header in response
- Check `convert_json: true` is enabled
- Verify template directory path is correct

**Weird formatting:**
- Use `{{{raw}}}` for unescaped HTML
- Use `{{escaped}}` for safe text (default)
- Test with simpler template first

---

## Examples

See `examples/mustache-templates/` for working templates:

- **`_default.mustache`** - Generic JSON template (all fields)
- **`api.github.com__users.mustache`** - GitHub Users API

Copy and customize for your APIs!

---

## See Also

- [Mustache Documentation](https://mustache.github.io/) - Full syntax reference
- [MCP_SERVER.md](./MCP_SERVER.md) - Integrate with Claude Desktop
- [examples/README.md](../examples/README.md) - More template examples
