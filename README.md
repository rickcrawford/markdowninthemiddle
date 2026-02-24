# Markdown in the Middle

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

> An HTTPS/HTTP forward proxy that intercepts HTML responses and converts them to Markdown on the fly, with optional JavaScript rendering via headless Chrome.

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-brightgreen?style=flat-square)](/)

## Why Markdown in the Middle?

Inspired by [Cloudflare's Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/), this proxy brings HTML-to-Markdown conversion to **local networks, internal services, and private APIs** where Cloudflare isn't available.

### What It Solves

- 📄 **Automatic HTML → Markdown conversion** - All HTML responses become clean Markdown
- 🔐 **Works with self-signed certificates** - Perfect for internal/staging environments
- 🌐 **Forward proxy architecture** - Route traffic through it without code changes
- 💬 **Token counting** - Get TikToken counts for LLM usage planning
- 📦 **Optional JavaScript rendering** - Use headless Chrome for SPA/dynamic content
- 💾 **Response caching** - RFC 7234 compliant cache with disk persistence
- 🎯 **Content negotiation** - Only convert when clients request `Accept: text/markdown`
- 🔍 **Request filtering** - Restrict proxy to specific domain patterns

### Use Cases

- **Proxying internal APIs** for use with Claude or other LLMs
- **Converting web content** for processing by AI agents
- **Token counting** before feeding HTML to language models
- **Caching HTML** responses for repeatable processing
- **Rendering SPAs** with JavaScript execution before conversion

---

## Quick Start

### Docker (Recommended)

```bash
# Start proxy + Chrome
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

## Configuration

**Default behavior:**
- HTTP listener (no TLS)
- HTML → Markdown conversion enabled
- Token counting enabled
- JavaScript rendering via chromedp (Docker) or disabled (local)

**Common options:**
```bash
./markdowninthemiddle --addr :9090              # Custom port
./markdowninthemiddle --cache-dir ./cache       # Enable caching
./markdowninthemiddle --output-dir ./markdown   # Save converted files
./markdowninthemiddle --negotiate-only          # Convert only when requested
./markdowninthemiddle --mitm                    # HTTPS interception (see docs)
```

**See configuration docs:**
- **[HTTPS_SETUP.md](./HTTPS_SETUP.md)** - Add TLS/HTTPS, MITM mode, certificate setup
- **[MITM_SETUP.md](./MITM_SETUP.md)** - Client setup for MITM interception
- **[UPSTREAM_PROXY.md](./UPSTREAM_PROXY.md)** - Route through another proxy
- **[CODE_DETAILS.md](./CODE_DETAILS.md)** - Full CLI reference and architecture

---

---

## Common Use Cases

**Check response was converted:**
```bash
curl -x http://localhost:8080 http://example.com -sD - | grep "X-Token-Count"
```

**Only convert when requested (Accept header):**
```bash
./markdowninthemiddle --negotiate-only
curl -x http://localhost:8080 -H "Accept: text/markdown" http://example.com
```

**Restrict to specific domains:**
```bash
./markdowninthemiddle --allow "^https://api\.example\.com/" --allow "^https://docs\.example\.com/"
```

**Save converted Markdown files:**
```bash
./markdowninthemiddle --output-dir ./markdown
```

**Cache responses for repeatable processing:**
```bash
./markdowninthemiddle --cache-dir ./cache
```

---

## Advanced Configuration

**CLI flags, environment variables, or config.yml:**

```bash
# CLI
./markdowninthemiddle --addr :9090 --cache-dir ./cache

# Environment
MITM_PROXY_ADDR=":9090" MITM_CACHE_DIR="./cache" ./markdowninthemiddle

# config.yml
proxy:
  addr: ":9090"
cache:
  dir: "./cache"
```

**Full configuration reference:** See [CODE_DETAILS.md](./CODE_DETAILS.md)

---

## Features

| Feature | Status | Notes |
|---------|--------|-------|
| HTML → Markdown | ✅ | Automatic for all `text/html` responses |
| Token Counting | ✅ | TikToken `cl100k_base` encoding |
| Content Negotiation | ✅ | `Accept: text/markdown` header support |
| TLS/HTTPS | ✅ | Auto-generated or custom certificates |
| Response Caching | ✅ | RFC 7234 compliant with disk persistence |
| JavaScript Rendering | ✅ | Via headless Chrome (chromedp) |
| Request Filtering | ✅ | Regex-based domain allowlists |
| Forward Proxy | ✅ | Standard HTTP CONNECT tunneling |
| Markdown Output | ✅ | Save converted files to disk |
| Multi-platform | ✅ | macOS, Linux, Windows |

---

## Response Headers

| Header | Example | Description |
|--------|---------|-------------|
| `X-Token-Count` | `1234` | TikToken count of converted Markdown |
| `X-Transport` | `chromedp` or `http` | Transport used: `chromedp` (headless Chrome) or `http` (standard HTTP) |
| `Vary` | `accept` | Cache variant header for downstream caches |
| `Content-Type` | `text/markdown; charset=utf-8` | Converted response type |

**Test response headers:**
```bash
curl -x http://localhost:8080 http://example.com -sD - | grep "X-"
```

---

## Architecture

```
┌─────────────┐
│ HTTP Client │
└──────┬──────┘
       │
       ▼
┌──────────────────────┐
│  Proxy Listener      │
│  :8080 (HTTP/HTTPS)  │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Request Filter       │ ◄─ Optional: regex allow-list
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Transport Layer      │ ◄─ http OR chromedp
├──────────────────────┤
│ • Standard HTTP      │
│ • Headless Chrome    │
└──────┬───────────────┘
       │
       ▼
┌──────────────────────┐
│ Response Processing  │
├──────────────────────┤
│ • Decompress body    │
│ • Cache HTML (opt)   │
│ • Convert HTML→MD    │
│ • Count tokens       │
│ • Write files (opt)  │
└──────┬───────────────┘
       │
       ▼
┌─────────────────┐
│ HTTP Response   │
│ (Markdown)      │
└─────────────────┘
```

---

## Docker

The `docker-compose.yml` includes both proxy and Chrome services:

```bash
docker compose up -d          # Start
docker compose logs -f proxy  # View logs
docker compose down           # Stop
```

See [CODE_DETAILS.md](./CODE_DETAILS.md) for full Docker configuration options.

---

## Troubleshooting

**Proxy won't start:**
- Check port 8080 is available: `lsof -i :8080`
- Check logs: `docker compose logs proxy` or see console output

**Chrome not connecting (Docker):**
- Verify Chrome service is running: `docker compose ps`
- Restart Chrome: `docker compose restart chrome`

**HTTPS/TLS issues:**
- See [HTTPS_SETUP.md](./HTTPS_SETUP.md) for certificate setup

**MITM certificate not trusted:**
- See [MITM_SETUP.md](./MITM_SETUP.md) for client certificate setup

**Upstream proxy issues:**
- See [UPSTREAM_PROXY.md](./UPSTREAM_PROXY.md) for proxy environment variables

---

## Inspiration

This project was inspired by [Cloudflare's Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/) — an excellent solution for converting web content to Markdown at the edge. Markdown in the Middle brings similar benefits to local networks, internal services, and private APIs that can't use Cloudflare, with the added bonus of optional JavaScript rendering via headless Chrome.

---

## See Also

- **[CODE_DETAILS.md](./CODE_DETAILS.md)** - Technical architecture, CLI reference, and implementation details
- **[CHROMEDP.md](./CHROMEDP.md)** - Detailed chromedp/JavaScript rendering setup
- **[DOCKER.md](./DOCKER.md)** - Docker deployment guide

---

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

---

## License

MIT - See [LICENSE](LICENSE) for details

---

## Author

Created by [Rick Crawford](https://github.com/rickcrawford)

## Support

For issues, questions, or feature requests: [GitHub Issues](https://github.com/rickcrawford/markdowninthemiddle/issues)
