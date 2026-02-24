# Examples

This directory contains example configurations and templates for Markdown in the Middle.

## Files

### Configuration

- **config.example.yml** - Example configuration file with all available options and defaults

To use:
```bash
# Copy to root directory and customize
cp config.example.yml ../config.yml
```

Or for Docker:
```bash
# Copy to docker directory
cp config.example.yml ../docker/config.yml
```

### Mustache Templates

The `mustache-templates/` directory contains example templates for JSON-to-Markdown conversion:

- **_default.mustache** - Generic fallback template for any JSON response
- **api.github.com__users.mustache** - Example template for GitHub Users API

#### Using Templates

1. Create a templates directory:
   ```bash
   mkdir -p /path/to/templates
   ```

2. Copy example templates:
   ```bash
   cp -r mustache-templates/* /path/to/templates/
   ```

3. Run the proxy with templates:
   ```bash
   ./markdowninthemiddle --convert-json --template-dir /path/to/templates
   ```

Or configure in `config.yml`:
```yaml
conversion:
  convert_json: true
  template_dir: "/path/to/templates"
```

#### Template Naming Convention

Template filenames use `__` (double underscore) as path separators:

| Filename | Matches URLs |
|----------|---|
| `_default.mustache` | Fallback for all JSON |
| `api.example.com.mustache` | `http://api.example.com/*` (host-only) |
| `api.example.com__users.mustache` | `http://api.example.com/users*` |
| `api.example.com__v1__posts.mustache` | `http://api.example.com/v1/posts*` |

**Longest prefix match wins.** Scheme (`http://`/`https://`) is stripped before matching.

#### Template Syntax

Templates use [Mustache syntax](https://mustache.github.io/):

```mustache
# {{title}}

{{#description}}
{{{description}}}
{{/description}}

{{#items}}
- {{name}}
{{/items}}
```

**Key points:**
- `{{variable}}` - Escaped HTML
- `{{{variable}}}` - Unescaped (raw) HTML
- `{{#section}}...{{/section}}` - Conditional/loop
- `{{#if}}...{{else}}...{{/if}}` - If/else (with extensions)

#### Creating Custom Templates

1. Examine JSON structure:
   ```bash
   curl https://api.example.com/endpoint | jq .
   ```

2. Create matching template:
   ```bash
   cat > api.example.com__endpoint.mustache << 'EOF'
   # {{name}}
   **ID**: {{id}}
   {{#description}}
   {{{description}}}
   {{/description}}
   EOF
   ```

3. Test with proxy:
   ```bash
   curl -x http://localhost:8080 https://api.example.com/endpoint
   ```

## More Information

See documentation in the `docs/` folder:
- [CODE_DETAILS.md](../docs/CODE_DETAILS.md) - Full configuration reference
- [MCP_SERVER.md](../docs/MCP_SERVER.md) - JSON-to-Markdown via MCP
