
```
 ███╗   ███╗ █████╗ ██████╗ ██╗  ██╗██████╗  ██████╗ ██╗    ██╗███╗   ██╗
 ████╗ ████║██╔══██╗██╔══██╗██║ ██╔╝██╔══██╗██╔═══██╗██║    ██║████╗  ██║
 ██╔████╔██║███████║██████╔╝█████╔╝ ██║  ██║██║   ██║██║ █╗ ██║██╔██╗ ██║
 ██║╚██╔╝██║██╔══██║██╔══██╗██╔═██╗ ██║  ██║██║   ██║██║███╗██║██║╚██╗██║
 ██║ ╚═╝ ██║██║  ██║██║  ██║██║  ██╗██████╔╝╚██████╔╝╚███╔███╔╝██║ ╚████║
 ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚══╝╚══╝ ╚═╝  ╚═══╝
          ██╗███╗   ██╗    ████████╗██╗  ██╗███████╗
          ██║████╗  ██║    ╚══██╔══╝██║  ██║██╔════╝
          ██║██╔██╗ ██║       ██║   ███████║█████╗
          ██║██║╚██╗██║       ██║   ██╔══██║██╔══╝
          ██║██║ ╚████║       ██║   ██║  ██║███████╗
          ╚═╝╚═╝  ╚═══╝       ╚═╝   ╚═╝  ╚═╝╚══════╝
 ███╗   ███╗██╗██████╗ ██████╗ ██╗     ███████╗
 ████╗ ████║██║██╔══██╗██╔══██╗██║     ██╔════╝
 ██╔████╔██║██║██║  ██║██║  ██║██║     █████╗
 ██║╚██╔╝██║██║██║  ██║██║  ██║██║     ██╔══╝
 ██║ ╚═╝ ██║██║██████╔╝██████╔╝███████╗███████╗
 ╚═╝     ╚═╝╚═╝╚═════╝ ╚═════╝ ╚══════╝╚══════╝
```

**An intelligent forward proxy that converts HTML & JSON to Markdown in real-time, with optional JavaScript rendering for dynamic content.**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

---

**Markdown in the Middle** is inspired by [Cloudflare's Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/). While Cloudflare's solution handles the edge, this project brings HTML-to-Markdown conversion to:

- 🏢 Internal networks and private services
- 🔒 Staging/testing environments with self-signed certs
- 🤖 Local LLM deployments
- 🔌 Private APIs without external dependencies

It's production-ready with full Docker support, MCP integration for Claude Desktop, and optional JavaScript rendering for dynamic content.

---

## Author

**Rick Crawford** - [LinkedIn](https://linkedin.com/in/rickcrawford)

---

## ✨ Key Features

| Feature | Details |
|---------|---------|
| 📄 **HTML to Markdown** | Automatic conversion of all HTML responses |
| 📋 **JSON to Markdown** | Convert API responses using Mustache templates |
| 🤖 **MCP Integration** | Expose tools to Claude Desktop and other LLM clients |
| 🔄 **JavaScript Rendering** | Execute SPAs before conversion (headless Chrome) |
| 💬 **Token Counting** | Estimate LLM usage with TikToken encoding |
| 💾 **Caching** | RFC 7234 compliant response caching |
| 🔐 **HTTPS/TLS** | Full MITM support with certificate handling |
| 🎯 **Smart Negotiation** | Convert only when client requests `Accept: text/markdown` |
| 🔍 **URL Filtering** | Restrict proxy to approved domains with regex |
| 🌐 **Forward Proxy** | Standard HTTP CONNECT tunneling |

---

## Use Cases

🧠 **AI & LLMs** - Feed web content to Claude, GPT, and local models. Use MCP for Claude Desktop integration.

🔌 **Internal APIs** - Convert private/staging APIs to Markdown for processing or caching.

💼 **Data Processing** - Extract and normalize JSON responses with custom Mustache templates.

🚀 **SPA Rendering** - Execute JavaScript before conversion for dynamic content.

📊 **Token Planning** - Pre-count tokens before sending content to LLMs for cost estimation.

---

## Quick Start

### Docker (Recommended)

```bash
# Start proxy + Chrome (from docker/ folder)
cd docker
docker compose up -d

# Test it
curl -x http://localhost:8080 http://example.com

# View logs
docker compose logs -f proxy

# Stop
docker compose down
```

**Runs on:** `http://localhost:8080` (HTTP, no TLS)

### macOS/Linux (Binary)

```bash
# Build from source
go build -o markdowninthemiddle .

# Start proxy (HTTP on :8080)
./markdowninthemiddle

# Test
curl -x http://localhost:8080 http://example.com
```

---

## 📋 JSON-to-Markdown with Templates

Convert API responses to clean Markdown using **Mustache templates**:

```bash
# Copy example templates
cp -r examples/mustache-templates ~/my-templates

# Enable JSON conversion
./markdowninthemiddle --convert-json --template-dir ~/my-templates

# Test with an API
curl -x http://localhost:8080 https://api.github.com/users/octocat
```

**Template naming:** Use `__` (double underscore) as path separator:
- `api.example.com__users.mustache` → matches `https://api.example.com/users*`
- `_default.mustache` → fallback for unmatched JSON

**See:** [JSON_CONVERSION.md](./docs/JSON_CONVERSION.md) - Full template guide with examples

---

## 🤖 MCP Server (Model Context Protocol)

Expose HTML-to-Markdown conversion as an **LLM tool** using the Model Context Protocol. Perfect for Claude Desktop or other MCP-compatible clients.

### Quick Start - Claude Desktop

```bash
# Start MCP server in stdio mode (for Claude Desktop)
./markdowninthemiddle mcp --transport chromedp
```

Add to `Claude Desktop > Settings > Extensions`:
```json
{
  "mcpServers": {
    "markdowninthemiddle": {
      "command": "/path/to/markdowninthemiddle",
      "args": ["mcp", "--transport", "chromedp"]
    }
  }
}
```

Then use in Claude conversation:
```
Can you fetch and summarize https://example.com?
```

### Quick Start - HTTP Mode (for Other Clients)

```bash
# Start MCP HTTP server
./markdowninthemiddle mcp --transport chromedp --mcp-transport http --mcp-addr :8081

# Or in Docker with both proxy and MCP
docker compose up -d
```

### Tools

**`fetch_markdown`** - Fetch a URL and convert to Markdown
```json
{
  "url": "https://example.com"
}
```
Returns Markdown, token count, and HTTP status.

**`fetch_raw`** - Fetch a URL and return raw HTML/JSON body
```json
{
  "url": "https://api.example.com/data"
}
```
Returns raw body with content type and status.

### Configuration

**Command-line flags:**
```bash
./markdowninthemiddle mcp \
  --transport chromedp              # http or chromedp
  --chrome-url http://chrome:9222   # Chrome DevTools URL
  --chrome-pool-size 10             # Max concurrent tabs
  --mcp-transport http              # stdio or http
  --mcp-addr :8081                  # HTTP server address
```

**Config file (config.yml):**
```yaml
transport:
  type: chromedp                     # or http
  chromedp:
    url: http://localhost:9222
    pool_size: 5
```

**Docker Compose (both proxy + MCP):**
```bash
docker compose up -d
# Proxy: http://localhost:8080
# MCP:   http://localhost:8081
```

---

## 📚 Complete Documentation

| Guide | Purpose |
|-------|---------|
| **[CONFIGURATION.md](./docs/CONFIGURATION.md)** | All CLI flags, env vars, and config.yml options |
| **[JSON_CONVERSION.md](./docs/JSON_CONVERSION.md)** | JSON templates, Mustache syntax, examples |
| **[CHROMEDP.md](./docs/CHROMEDP.md)** | JavaScript rendering and headless Chrome setup |
| **[HTTPS_SETUP.md](./docs/HTTPS_SETUP.md)** | TLS, certificates, and HTTPS interception |
| **[MITM_SETUP.md](./docs/MITM_SETUP.md)** | Client certificate installation and MITM |
| **[DOCKER.md](./docs/DOCKER.md)** | Docker deployment and multi-service setup |
| **[MCP_SERVER.md](./docs/MCP_SERVER.md)** | Claude Desktop and LLM integration |
| **[TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md)** | Common issues and solutions |
| **[CODE_DETAILS.md](./docs/CODE_DETAILS.md)** | Technical architecture and implementation |

---

## ⚙️ Quick Configuration

See [CONFIGURATION.md](./docs/CONFIGURATION.md) for the full reference.

```bash
# Custom port
./markdowninthemiddle --addr :9090

# Enable caching
./markdowninthemiddle --cache-dir ./cache

# Save Markdown files
./markdowninthemiddle --output-dir ./markdown

# JavaScript rendering
./markdowninthemiddle --transport chromedp

# HTTPS + MITM
./markdowninthemiddle --tls --auto-cert

# Restrict domains
./markdowninthemiddle --allow "^https://api\.example\.com/"
```

All options support CLI flags, environment variables (`MITM_*`), or `config.yml`.

---

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-thing`)
3. Add tests for new functionality
4. Submit a pull request

---

## Support

- 📖 **Documentation** - See the [docs](./docs) folder for comprehensive guides
- 🐛 **Bug Reports** - [GitHub Issues](https://github.com/rickcrawford/markdowninthemiddle/issues)
- 🔧 **Quick Help** - Start with [TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md)

---

## License

MIT License - See [LICENSE](LICENSE) for details
