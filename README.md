# Markdown in the Middle

An HTTPS forward proxy that intercepts HTML responses and converts them to Markdown on the fly. Token counts are calculated with TikToken and returned via a response header.

Inspired by Cloudflare's [Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/) — which converts HTML to Markdown at the CDN edge for public sites — **Markdown in the Middle** brings the same capability to **local, internal, and private networks** where Cloudflare isn't an option. Run it as a forward proxy on your local machine or internal infrastructure and get Markdown conversion, token counting, and response caching for any HTTP traffic.

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

## Why This Exists

Cloudflare's [Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/) is a great solution for public websites behind Cloudflare's CDN. But many use cases fall outside that scope:

- **Internal applications** behind corporate firewalls
- **Local development** servers and staging environments
- **Self-hosted services** not using Cloudflare
- **Private APIs** returning HTML that you want to consume as Markdown
- **Air-gapped or restricted networks** with no external CDN

Markdown in the Middle fills that gap as a self-hosted, configurable forward proxy.

## Features

- **HTML to Markdown conversion** — Proxied `text/html` responses are automatically converted to Markdown using [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown)
- **`Accept: text/markdown` content negotiation** — Optional mode where conversion only happens when the client sends `Accept: text/markdown`, matching the [Cloudflare Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/) content negotiation pattern
- **Token counting** — Converted Markdown responses include an `X-Token-Count` header with the TikToken token count (default encoding: `cl100k_base`)
- **Markdown output directory** — Optionally write every converted Markdown response to disk as `.md` files with URL-derived, file-safe names
- **Content-Encoding handling** — Transparently decompresses `gzip` and `deflate` encoded response bodies via Chi middleware before processing
- **Response body size limit** — Configurable maximum body size (default 10 MB) to prevent memory exhaustion
- **Self-signed TLS certificates** — Auto-generate ECDSA P-256 self-signed certs, or bring your own
- **TLS insecure mode** — Skip upstream TLS certificate verification for development/testing with self-signed certs
- **RFC 7234 response caching** — Respects `Cache-Control` (`no-store`, `private`, `max-age`, `s-maxage`) and `Expires` headers; optionally writes HTML to a disk cache directory
- **`Vary: accept` response header** — Converted responses include `Vary: accept` so downstream caches store separate HTML and Markdown variants
- **HTTPS CONNECT tunneling** — Standard HTTP CONNECT support for proxying HTTPS traffic
- **YAML + env configuration** — Configure via `config.yml`, environment variables (`MITM_` prefix), or CLI flags

## Installation

### Quick Install (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/rickcrawford/markdowninthemiddle/main/install.sh | sh
```

### Quick Install (Windows PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/rickcrawford/markdowninthemiddle/main/install.ps1 | iex
```

### Install with Go

```bash
go install github.com/rickcrawford/markdowninthemiddle@latest
```

### Build from source

```bash
git clone https://github.com/rickcrawford/markdowninthemiddle.git
cd markdowninthemiddle
go build -o markdowninthemiddle .
```

## Quick Start

```bash
# Start with defaults (listens on :8080, converts all HTML)
./markdowninthemiddle

# Only convert when client requests Markdown (content negotiation)
./markdowninthemiddle --negotiate-only

# Start with TLS listener using auto-generated self-signed cert
./markdowninthemiddle --tls --auto-cert

# Allow connections to upstream servers with invalid/self-signed certs
./markdowninthemiddle --tls-insecure

# Enable HTML disk caching
./markdowninthemiddle --cache-dir ./cache

# Write converted Markdown files to a directory
./markdowninthemiddle --output-dir ./markdown-output
```

## Usage

### As an HTTP proxy

Point your HTTP client at the proxy. HTML responses will be returned as Markdown:

```bash
# Plain HTTP proxy — all HTML is converted
curl -x http://localhost:8080 http://example.com

# Check the token count header
curl -x http://localhost:8080 -sD - http://example.com | grep X-Token-Count
```

### Content negotiation mode

In negotiate-only mode, the proxy only converts HTML when the client explicitly requests Markdown via the `Accept` header — the same pattern used by [Cloudflare's Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/):

```bash
# Start in negotiate-only mode
./markdowninthemiddle --negotiate-only

# This returns HTML as-is (no Accept: text/markdown header)
curl -x http://localhost:8080 http://example.com

# This returns converted Markdown
curl -x http://localhost:8080 -H "Accept: text/markdown" http://example.com
```

### With TLS on the proxy listener

```bash
# Generate a self-signed certificate
./markdowninthemiddle gencert --host localhost --dir ./certs

# Start with TLS
./markdowninthemiddle --tls --auto-cert

# Connect through the TLS proxy (trust the self-signed cert)
curl -x https://localhost:8080 --proxy-cacert ./certs/cert.pem http://example.com
```

### Skipping upstream TLS verification

When upstream servers use self-signed or invalid certificates (common in development), use the `--tls-insecure` flag:

```bash
./markdowninthemiddle --tls-insecure
```

Or set it in `config.yml`:

```yaml
tls:
  insecure: true
```

Or via environment variable:

```bash
MITM_TLS_INSECURE=true ./markdowninthemiddle
```

### Writing Markdown output to disk

Save every converted Markdown response to a directory. Files are named using a URL-derived, file-safe naming convention with `.md` extension:

```bash
./markdowninthemiddle --output-dir ./markdown-output
```

Example output filenames:
- `http://example.com/blog/my-post` -> `example.com__blog__my-post.md`
- `http://example.com/search?q=test` -> `example.com__search__q_test.md`

Or set it in `config.yml`:

```yaml
output:
  enabled: true
  dir: "./markdown-output"
```

### Generate a self-signed certificate

```bash
./markdowninthemiddle gencert --host myhost.local --dir ./certs
```

This creates `cert.pem` and `key.pem` in the specified directory.

## Configuration

Configuration is loaded in this order of precedence (highest to lowest):

1. CLI flags
2. Environment variables (`MITM_` prefix, e.g. `MITM_PROXY_ADDR`)
3. `config.yml` file
4. Built-in defaults

### config.yml

```yaml
proxy:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

tls:
  enabled: false
  cert_file: ""
  key_file: ""
  auto_cert: true
  auto_cert_host: "localhost"
  auto_cert_dir: "./certs"
  insecure: false

conversion:
  enabled: true
  tiktoken_encoding: "cl100k_base"
  negotiate_only: false

max_body_size: 10485760  # 10 MB

cache:
  enabled: false
  dir: ""
  respect_headers: true

output:
  enabled: false
  dir: ""

log_level: "info"
```

### CLI Flags

| Flag | Description |
|---|---|
| `--config` | Path to config file (default: `./config.yml`) |
| `--addr` | Proxy listen address |
| `--tls` | Enable TLS on the proxy listener |
| `--auto-cert` | Auto-generate a self-signed certificate |
| `--tls-insecure` | Skip TLS certificate verification for upstream requests |
| `--negotiate-only` | Only convert when client sends `Accept: text/markdown` |
| `--cache-dir` | Directory to cache HTML responses |
| `--output-dir` | Directory to write converted Markdown `.md` files |
| `--max-body-size` | Maximum response body size in bytes |

### Subcommands

| Command | Description |
|---|---|
| `gencert` | Generate a self-signed TLS certificate |

#### gencert flags

| Flag | Description |
|---|---|
| `--host` | Hostname or IP for the certificate (default: `localhost`) |
| `--dir` | Output directory for cert/key files (default: `./certs`) |

## How It Works

1. Client configures `markdowninthemiddle` as its HTTP proxy
2. For **HTTP** requests, the proxy forwards the request and inspects the response:
   - If `Content-Type` contains `text/html` (and content negotiation passes, if enabled):
     - Decompresses the body if `Content-Encoding` is `gzip` or `deflate`
     - Enforces the configured body size limit
     - Caches the original HTML to disk (if caching is enabled and RFC headers allow it)
     - Converts the HTML to Markdown
     - Writes the Markdown to the output directory as a `.md` file (if output is enabled)
     - Counts tokens using TikToken and sets the `X-Token-Count` response header
     - Sets `Content-Type` to `text/markdown; charset=utf-8`
     - Sets `Vary: accept` for proper cache separation
   - Other content types pass through unmodified
3. For **HTTPS** requests (`CONNECT` method), a raw TCP tunnel is established — traffic is encrypted end-to-end and not processed

## Response Headers

| Header | Description |
|---|---|
| `X-Token-Count` | Number of TikToken tokens in the converted Markdown body. Only present on converted responses. |
| `Vary` | Set to `accept` on converted responses, enabling downstream caches to store separate HTML/Markdown variants. |

## Comparison with Cloudflare Markdown for Agents

| Feature | Cloudflare | Markdown in the Middle |
|---|---|---|
| Deployment | CDN edge (public sites only) | Self-hosted proxy (local/internal/private) |
| Content negotiation | `Accept: text/markdown` | `Accept: text/markdown` (via `--negotiate-only`) or convert all HTML |
| Token counting header | `x-markdown-tokens` | `X-Token-Count` |
| Cache variant header | `Vary: accept` | `Vary: accept` |
| HTML caching | Cloudflare CDN cache | Disk cache with RFC 7234 compliance |
| Markdown file output | N/A | Write `.md` files to disk with URL-derived names |
| Self-signed TLS | N/A | Built-in cert generation |
| Configuration | Cloudflare dashboard | YAML / env vars / CLI flags |
| Works on internal networks | No | Yes |

## Project Structure

```
markdowninthemiddle/
├── main.go                            # Entry point
├── config.yml                         # Default configuration
├── install.sh                         # Install script (macOS/Linux)
├── install.ps1                        # Install script (Windows PowerShell)
├── cmd/
│   ├── root.go                        # Cobra root command, proxy startup
│   └── gencert.go                     # Certificate generation subcommand
└── internal/
    ├── banner/banner.go               # ASCII art startup banner
    ├── cache/cache.go                 # Disk cache with RFC 7234 compliance
    ├── certs/certs.go                 # Self-signed certificate generation
    ├── config/config.go               # Viper config loader
    ├── converter/converter.go         # HTML-to-Markdown conversion
    ├── middleware/
    │   ├── decompress.go              # Content-Encoding decompression
    │   └── middleware.go              # Response processing RoundTripper
    ├── output/output.go               # Markdown file output writer
    ├── proxy/proxy.go                 # Chi-based forward proxy server
    └── tokens/tokens.go              # TikToken token counter
```

## Dependencies

- [chi](https://github.com/go-chi/chi) — HTTP router and middleware
- [cobra](https://github.com/spf13/cobra) — CLI framework
- [viper](https://github.com/spf13/viper) — Configuration management
- [html-to-markdown](https://github.com/JohannesKaufmann/html-to-markdown) — HTML to Markdown conversion
- [tiktoken-go](https://github.com/pkoukk/tiktoken-go) — TikToken token counting

## Author

Created by [Rick Crawford](https://www.linkedin.com/in/rickcrawford/)

## License

MIT
