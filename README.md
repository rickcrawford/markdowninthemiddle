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

### Docker (Recommended - includes Chrome)

```bash
# Start proxy + Chrome in Docker
./scripts/docker-compose.sh start

# Test it
curl -x http://localhost:8080 http://example.com

# View logs
./scripts/docker-compose.sh logs proxy

# Stop everything
./scripts/docker-compose.sh stop
```

**Available on:** `http://localhost:8080`

### macOS/Linux (Binary)

```bash
# Build from source
go build -o markdowninthemiddle .

# Start with defaults (HTTP on :8080)
./markdowninthemiddle

# With TLS (HTTPS on :8080)
./markdowninthemiddle --tls --auto-cert

# With content negotiation only
./markdowninthemiddle --negotiate-only

# With caching
./markdowninthemiddle --cache-dir ./cache

# With JavaScript rendering (requires Chrome)
./scripts/start-chrome.sh &
./markdowninthemiddle --transport chromedp
```

---

## Installation

### Docker Compose (All-in-one)

```bash
docker compose up -d
```

Includes:
- Markdown in the Middle proxy
- Headless Chrome with DevTools enabled
- Health checks and auto-restart
- Certificate generation

### macOS

```bash
# Via Homebrew (if available)
brew install markdowninthemiddle

# Or build from source
git clone https://github.com/rickcrawford/markdowninthemiddle.git
cd markdowninthemiddle
go build -o markdowninthemiddle .
```

### Linux

```bash
# Via package manager (if available)
sudo apt install markdowninthemiddle

# Or build from source
git clone https://github.com/rickcrawford/markdowninthemiddle.git
cd markdowninthemiddle
go build -o markdowninthemiddle .
```

### Windows

```bash
# Via Chocolatey (if available)
choco install markdowninthemiddle

# Or build from source
git clone https://github.com/rickcrawford/markdowninthemiddle.git
cd markdowninthemiddle
go build -o markdowninthemiddle.exe .
```

---

## Usage

### HTTP Proxy (Default)

```bash
# Start proxy on :8080
./markdowninthemiddle

# Use as proxy
curl -x http://localhost:8080 http://example.com

# Get token count
curl -x http://localhost:8080 -sD - http://example.com | grep X-Token-Count
```

### HTTPS Proxy with TLS

#### Quick Start (auto-generated cert)

```bash
./markdowninthemiddle --tls --auto-cert
curl -x https://localhost:8080 --insecure http://example.com
```

#### macOS (Proper Certificate Setup)

**Step 1: Generate certificate**
```bash
./markdowninthemiddle gencert --host localhost --dir ./certs
```

**Step 2: Add to macOS Keychain**
```bash
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./certs/cert.pem
```

**Step 3: Start with TLS**
```bash
./markdowninthemiddle --tls
```

**Step 4: Use with curl (no --insecure needed)**
```bash
curl -x https://localhost:8080 http://example.com
```

### JavaScript Rendering (chromedp)

**Step 1: Start Chrome**
```bash
./scripts/start-chrome.sh
```

**Step 2: Start proxy with chromedp**
```bash
./markdowninthemiddle --transport chromedp
```

**Step 3: Use normally**
```bash
curl -x http://localhost:8080 https://spa-website.com
```

### Content Negotiation Mode

Only convert when client explicitly requests Markdown:

```bash
./markdowninthemiddle --negotiate-only

# Returns HTML as-is
curl -x http://localhost:8080 http://example.com

# Returns Markdown
curl -x http://localhost:8080 -H "Accept: text/markdown" http://example.com
```

### Request Filtering

Restrict proxy to specific domains:

```bash
./markdowninthemiddle \
  --allow "^https://api\.example\.com/" \
  --allow "^https://docs\.example\.com/"

# Returns 403 Forbidden for other domains
curl -x http://localhost:8080 http://other-domain.com
```

### Output Files

Save converted Markdown to disk:

```bash
./markdowninthemiddle --output-dir ./markdown

# Files named: example.com__path__to__page.md
ls ./markdown
```

---

## Configuration

Configuration loads in this order (highest to lowest priority):

1. **CLI flags** - `./markdowninthemiddle --tls --cache-dir ./cache`
2. **Environment variables** - `MITM_TLS_ENABLED=true MITM_CACHE_DIR=./cache`
3. **config.yml** - Local configuration file
4. **Built-in defaults**

### config.yml Example

```yaml
proxy:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

tls:
  enabled: false
  auto_cert: true
  auto_cert_host: "localhost"
  auto_cert_dir: "./certs"

conversion:
  enabled: true
  tiktoken_encoding: "cl100k_base"
  negotiate_only: false
  convert_json: false

cache:
  enabled: false
  dir: "./cache"
  respect_headers: true

output:
  enabled: false
  dir: "./markdown-output"

transport:
  type: "http"  # or "chromedp"
  chromedp:
    url: "http://localhost:9222"
    pool_size: 5

filter:
  allowed: []  # Empty = allow all

log_level: "info"
```

### Environment Variables

Prefix with `MITM_` and use `_` for nested keys:

```bash
MITM_PROXY_ADDR=":9090"
MITM_TLS_ENABLED="true"
MITM_CACHE_ENABLED="true"
MITM_CACHE_DIR="./cache"
MITM_TRANSPORT_TYPE="chromedp"
MITM_TRANSPORT_CHROMEDP_URL="http://localhost:9222"
```

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
| `Vary` | `accept` | Cache variant header for downstream caches |
| `Content-Type` | `text/markdown; charset=utf-8` | Converted response type |

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

## Docker Usage

### Quick Start

```bash
./scripts/docker-compose.sh start        # Start all services
./scripts/docker-compose.sh logs proxy   # View proxy logs
./scripts/docker-compose.sh test         # Run test request
./scripts/docker-compose.sh stop         # Stop all services
```

### Helper Script Commands

```bash
./scripts/docker-compose.sh start        # Start proxy + Chrome
./scripts/docker-compose.sh stop         # Stop all services
./scripts/docker-compose.sh restart      # Restart services
./scripts/docker-compose.sh status       # Show service status
./scripts/docker-compose.sh logs [svc]   # View logs (proxy/chrome)
./scripts/docker-compose.sh test         # Test with sample request
./scripts/docker-compose.sh shell        # Open container shell
./scripts/docker-compose.sh build        # Rebuild image
./scripts/docker-compose.sh clean        # Remove all containers
```

### Ports

| Port | Protocol | Service | Purpose |
|------|----------|---------|---------|
| 8080 | TCP | Proxy | HTTP or HTTPS (depends on TLS setting) |
| 9222 | TCP | Chrome | DevTools (internal only) |

---

## Troubleshooting

### Chrome Connection Issues

```bash
# Check if Chrome is running
docker compose ps

# View Chrome logs
./scripts/docker-compose.sh chrome-logs

# Restart Chrome
docker compose restart chrome
```

### Certificate Trust Issues (macOS)

```bash
# Add certificate to Keychain
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ./certs/cert.pem

# Verify it's installed
security dump-trust-settings -d | grep -i markdown
```

### High Memory Usage

Reduce Chrome pool size or add memory limits:

```bash
# In docker-compose.yml
chrome:
  mem_limit: 512m
```

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
