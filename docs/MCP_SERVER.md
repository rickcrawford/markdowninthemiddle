# MCP Server Guide

This document covers setup and usage of the **Model Context Protocol (MCP) server** for Markdown in the Middle.

## What is MCP?

The **Model Context Protocol** allows LLMs (like Claude) to call tools defined by your application. The MCP server in Markdown in the Middle exposes two tools:

- **`fetch_markdown`** - Fetch a URL and convert to Markdown
- **`fetch_raw`** - Fetch a URL and return raw HTML/JSON body

This allows Claude and other MCP-compatible clients to fetch and convert web content on demand.

## Quick Start - Claude Desktop

### 1. Start the MCP Server

```bash
# In stdio mode (recommended for Claude Desktop)
./markdowninthemiddle mcp --transport chromedp
```

This starts the MCP server in **stdio mode**, which reads JSON-RPC requests from stdin and writes responses to stdout.

### 2. Configure Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "markdowninthemiddle": {
      "command": "/full/path/to/markdowninthemiddle",
      "args": ["mcp", "--transport", "chromedp"]
    }
  }
}
```

Restart Claude Desktop.

### 3. Use in Claude

Simply ask Claude to fetch and convert a URL:

```
Can you fetch and summarize https://example.com?
```

Claude will:
1. Call `fetch_markdown` tool with the URL
2. Get back Markdown-converted content
3. Summarize it for you

## Quick Start - HTTP Mode

For clients other than Claude Desktop, start the MCP server in HTTP mode:

```bash
./markdowninthemiddle mcp \
  --transport chromedp \
  --mcp-transport http \
  --mcp-addr :8081
```

Then make HTTP requests to `http://localhost:8081` with JSON-RPC tool calls.

## Docker Deployment

The `docker-compose.yml` includes an MCP service that runs alongside the proxy:

```bash
# Start proxy (8080) + MCP server (8081) + Chrome (9222)
docker compose up -d

# View logs
docker compose logs -f mcp

# Test MCP health
curl http://localhost:8081
```

All services share:
- Same configuration file (`config.yml`)
- Same Chrome instance (`http://chrome:9222`)
- Same output directories

## Configuration

### Transport Options

**HTTP Transport (default):**
```bash
./markdowninthemiddle mcp --transport http
```
Uses standard HTTP client. Fast, but no JavaScript rendering.

**chromedp Transport:**
```bash
./markdowninthemiddle mcp --transport chromedp
```
Uses headless Chrome for JavaScript rendering. Slower, but handles dynamic content.

### MCP Server Modes

**Stdio Mode (for Claude Desktop):**
```bash
./markdowninthemiddle mcp --mcp-transport stdio
```
Reads from stdin, writes to stdout. Use with Claude Desktop integration.

**HTTP Mode (for other clients):**
```bash
./markdowninthemiddle mcp --mcp-transport http --mcp-addr :8081
```
Listens on HTTP port. Use with REST clients or custom integrations.

### Chrome Configuration

When using `--transport chromedp`:

```bash
# Custom Chrome URL (default: http://localhost:9222)
./markdowninthemiddle mcp --transport chromedp --chrome-url http://chrome-host:9222

# Custom pool size (default: 5)
./markdowninthemiddle mcp --transport chromedp --chrome-pool-size 10
```

### Config File

Configure defaults in `config.yml`:

```yaml
transport:
  type: chromedp              # or http
  chromedp:
    url: http://localhost:9222
    pool_size: 5

output:
  enabled: true               # Save converted files
  dir: ./markdown

conversion:
  tiktokenEncoding: cl100k_base
```

## Tools

### fetch_markdown

Fetch a URL and convert response to Markdown.

**Input:**
```json
{
  "url": "https://example.com"
}
```

**Output:**
```json
{
  "url": "https://example.com",
  "markdown": "# Example\n\nContent here...",
  "tokens": 245,
  "status_code": 200
}
```

**Supported content types:**
- `text/html` - Converted to Markdown
- `application/json` - Formatted as Markdown (with optional Mustache template)
- Other types - Returned as-is

### fetch_raw

Fetch a URL and return the raw response body.

**Input:**
```json
{
  "url": "https://api.example.com/data"
}
```

**Output:**
```json
{
  "url": "https://api.example.com/data",
  "status_code": 200,
  "content_type": "application/json",
  "body": "{\"key\": \"value\"}"
}
```

## Troubleshooting

### Claude Desktop not seeing the tool

1. Verify the MCP server starts without errors:
   ```bash
   ./markdowninthemiddle mcp --transport chromedp
   # Should show: "✅ chromedp browser pool ready for MCP requests"
   ```

2. Check the command path in `claude_desktop_config.json` is absolute and correct

3. Restart Claude Desktop completely (quit and reopen)

4. Check Claude's system logs:
   ```bash
   # macOS
   tail -f ~/Library/Logs/Claude/debug.log
   ```

### Chrome not connecting (chromedp transport)

1. Verify Chrome DevTools is running:
   ```bash
   curl http://localhost:9222/json/version
   ```

2. If using Docker, ensure Chrome service is running:
   ```bash
   docker compose ps chrome
   ```

3. Check Chrome URL matches your setup:
   ```bash
   ./markdowninthemiddle mcp --chrome-url http://localhost:9222
   ```

### Slow fetch_markdown calls

This is normal for chromedp transport (JavaScript rendering). To speed up:

1. Increase pool size for concurrent requests:
   ```bash
   ./markdowninthemiddle mcp --transport chromedp --chrome-pool-size 20
   ```

2. Or switch to HTTP transport (no JavaScript):
   ```bash
   ./markdowninthemiddle mcp --transport http
   ```

## Advanced Usage

### Running Proxy and MCP Together

Both can run simultaneously on different ports:

```bash
# Terminal 1: Proxy on 8080
./markdowninthemiddle --transport chromedp

# Terminal 2: MCP server on 8081
./markdowninthemiddle mcp --transport chromedp --mcp-transport http --mcp-addr :8081
```

Or use Docker Compose (recommended):
```bash
docker compose up -d
```

### Custom JSON-to-Markdown Templates

For APIs that return JSON, use Mustache templates to customize conversion:

In `config.yml`:
```yaml
templates:
  store:
    patterns:
      - pattern: "^https://api\.example\.com/"
        template_file: ./templates/example-api.mustache
```

### Running Multiple MCP Instances

For high-concurrency workloads:

```bash
# Instance 1 - High concurrency for chromedp
./markdowninthemiddle mcp \
  --transport chromedp \
  --chrome-pool-size 20 \
  --mcp-transport http \
  --mcp-addr :8081 &

# Instance 2 - Light load, no JavaScript
./markdowninthemiddle mcp \
  --transport http \
  --mcp-transport http \
  --mcp-addr :8082 &
```

Then load-balance between both instances.

## Performance

### Benchmarks

Typical response times (single concurrent request):

| Transport | Content | Time |
|-----------|---------|------|
| HTTP | Plain HTML (50KB) | 20ms |
| HTTP | JSON (10KB) | 5ms |
| chromedp | SPA with JS (200KB) | 1500ms |
| chromedp | Plain HTML (50KB) | 800ms |

chromedp adds significant latency due to JavaScript rendering. Use HTTP transport for APIs and static content, chromedp only for dynamic SPAs.

### Optimization Tips

1. **Use HTTP transport** for APIs and static content
2. **Increase pool size** if you have multiple concurrent requests
3. **Cache results** - Enable output writing to avoid re-fetching
4. **Filter domains** - Use `--allow` patterns to restrict scope
5. **Run multiple instances** - One for chromedp, one for HTTP

## Integration Examples

### Python Client

```python
import subprocess
import json

# Start MCP server
proc = subprocess.Popen(
    ["./markdowninthemiddle", "mcp", "--transport", "chromedp"],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
)

# Call fetch_markdown
request = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
        "name": "fetch_markdown",
        "arguments": {"url": "https://example.com"}
    }
}

proc.stdin.write(json.dumps(request).encode() + b"\n")
response = json.loads(proc.stdout.readline())
print(response["result"]["content"][0]["text"])
```

### Node.js Client

```javascript
const { spawn } = require('child_process');

const mcp = spawn('./markdowninthemiddle', ['mcp', '--transport', 'chromedp']);

const request = {
  jsonrpc: "2.0",
  id: 1,
  method: "tools/call",
  params: {
    name: "fetch_markdown",
    arguments: { url: "https://example.com" }
  }
};

mcp.stdin.write(JSON.stringify(request) + '\n');
mcp.stdout.on('data', (data) => {
  console.log(JSON.parse(data).result);
});
```

## See Also

- [README.md](./README.md) - Main documentation
- [CHROMEDP.md](./CHROMEDP.md) - JavaScript rendering setup
- [DOCKER.md](./DOCKER.md) - Docker deployment
