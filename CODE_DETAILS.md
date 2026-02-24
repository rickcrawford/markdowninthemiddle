# Code Details - Technical Reference

This document provides technical details about the codebase, CLI reference, architecture, and implementation specifics.

---

## Table of Contents

1. [CLI Reference](#cli-reference)
2. [Architecture](#architecture)
3. [Project Structure](#project-structure)
4. [Configuration System](#configuration-system)
5. [Transport Layer](#transport-layer)
6. [Conversion Pipeline](#conversion-pipeline)
7. [Building from Source](#building-from-source)

---

## CLI Reference

### Root Command

```bash
./markdowninthemiddle [flags]
```

**Description:** Start the HTTP/HTTPS forward proxy with optional HTML-to-Markdown conversion.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `./config.yml` | Path to config file |
| `--addr` | string | `:8080` | Proxy listen address |
| `--tls` | bool | `false` | Enable TLS on proxy listener |
| `--auto-cert` | bool | `false` | Auto-generate self-signed certificate |
| `--tls-insecure` | bool | `false` | Skip upstream TLS verification |
| `--negotiate-only` | bool | `false` | Only convert when client requests Markdown |
| `--cache-dir` | string | `` | Enable HTML caching to directory |
| `--output-dir` | string | `` | Write Markdown files to directory |
| `--max-body-size` | int64 | `10485760` | Max response body size (bytes) |
| `--transport` | string | `http` | Transport type: `http` or `chromedp` |
| `--convert-json` | bool | `false` | Enable JSON-to-Markdown conversion |
| `--template-dir` | string | `` | Directory with Mustache templates |
| `--allow` | []string | `` | Regex patterns for allowed URLs (repeatable) |

### Subcommands

#### gencert

Generate a self-signed TLS certificate.

```bash
./markdowninthemiddle gencert [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--host` | string | `localhost` | Hostname/IP for certificate |
| `--dir` | string | `./certs` | Output directory for cert/key |

**Example:**

```bash
./markdowninthemiddle gencert --host myhost.local --dir ./certs
# Creates: ./certs/cert.pem and ./certs/key.pem
```

---

## Architecture

### High-Level Flow

```
Request → Filter → Transport → Response Processor → Client
          (regex)  (http/cdp)  (decompress, convert,
                               cache, count tokens)
```

### Request Processing

1. **Filter Middleware** - Check if URL matches allow-list (if configured)
   - Returns 403 Forbidden if blocked
   - Continues to transport if allowed or no filter

2. **Transport Selection** - Choose between HTTP or chromedp
   - **HTTP** - Standard `net.Dial` + TLS for upstream
   - **chromedp** - Headless Chrome browser pool with semaphore

3. **Response Processing** - PostProcessor middleware
   - Decompress body (`gzip`, `deflate`)
   - Check content-type and size limits
   - Cache HTML to disk (if enabled)
   - Convert HTML → Markdown (if applicable)
   - Count tokens via TikToken
   - Write Markdown to files (if enabled)
   - Add `X-Token-Count` header
   - Add `Vary: accept` header

### Concurrency Model

- **Proxy listener** - Single goroutine per connection (Go HTTP server)
- **Browser pool** - Semaphore-bounded concurrent tabs (chromedp)
  - Default: 5 concurrent tabs
  - Each tab: ~20-50MB memory
  - Configurable via `pool_size`

---

## Project Structure

```
markdowninthemiddle/
├── main.go                           # Entry point
├── config.yml                        # Default configuration
├── Dockerfile                        # Multi-stage Docker build
├── docker-compose.yml                # Services (proxy + Chrome)
├── go.mod, go.sum                    # Dependencies
│
├── scripts/
│   ├── start-chrome.sh               # macOS/Linux Chrome launcher
│   ├── start-chrome.bat              # Windows Chrome launcher
│   └── docker-compose.sh             # Docker helper script
│
├── cmd/
│   ├── main.go                       # Cobra root command
│   ├── root.go                       # Proxy startup logic
│   └── gencert.go                    # Certificate generation
│
└── internal/
    ├── banner/
    │   └── banner.go                 # ASCII art startup banner
    │
    ├── browser/
    │   ├── browser.go                # chromedp pool (http.RoundTripper)
    │   └── browser_test.go           # Browser pool tests
    │
    ├── cache/
    │   ├── cache.go                  # Disk cache with RFC 7234
    │   └── cache_test.go             # Cache tests
    │
    ├── certs/
    │   ├── certs.go                  # TLS cert generation/loading
    │   └── certs_test.go             # Cert tests
    │
    ├── config/
    │   ├── config.go                 # Viper config loader
    │   └── config_test.go            # Config tests
    │
    ├── converter/
    │   ├── converter.go              # HTML→Markdown conversion
    │   └── converter_test.go         # Converter tests
    │
    ├── filter/
    │   ├── filter.go                 # Regex URL filtering
    │   └── filter_test.go            # Filter tests
    │
    ├── middleware/
    │   ├── logger.go                 # Custom request logging
    │   ├── decompress.go             # Content-Encoding decompression
    │   └── middleware.go             # Response processing RoundTripper
    │
    ├── output/
    │   ├── output.go                 # Markdown file writer
    │   └── output_test.go            # Output tests
    │
    ├── proxy/
    │   ├── proxy.go                  # Chi HTTP server/router
    │   └── proxy_test.go             # Proxy tests
    │
    ├── templates/
    │   ├── templates.go              # Mustache template loader
    │   └── templates_test.go         # Template tests
    │
    └── tokens/
        ├── tokens.go                 # TikToken counter wrapper
        └── tokens_test.go            # Token counter tests
```

---

## Configuration System

### Load Order (Highest to Lowest Priority)

1. **CLI Flags** - Command-line arguments
2. **Environment Variables** - `MITM_` prefixed (underscore for nesting)
3. **config.yml** - Local YAML file
4. **Built-in Defaults** - Hardcoded defaults in code

### Configuration Structure

```go
type Config struct {
    Proxy       ProxyConfig      // Listener settings
    TLS         TLSConfig        // TLS/HTTPS settings
    Conversion  ConversionConfig // HTML conversion settings
    MaxBodySize int64            // Body size limit
    Cache       CacheConfig      // Disk cache settings
    Output      OutputConfig     // Markdown file output
    LogLevel    string           // Log level (info, warn, error, debug)
    Transport   TransportConfig  // HTTP vs chromedp
    Filter      FilterConfig     // Request filtering
}
```

### Environment Variable Mapping

```
MITM_PROXY_ADDR              → proxy.addr
MITM_TLS_ENABLED             → tls.enabled
MITM_TLS_AUTO_CERT           → tls.auto_cert
MITM_CONVERSION_ENABLED      → conversion.enabled
MITM_TRANSPORT_TYPE          → transport.type
MITM_TRANSPORT_CHROMEDP_URL  → transport.chromedp.url
MITM_CACHE_ENABLED           → cache.enabled
MITM_FILTER_ALLOWED          → filter.allowed (comma-separated)
```

---

## Transport Layer

### HTTP Transport (Default)

- **Package**: `net/http`
- **Implementation**: Standard library `http.Transport`
- **Features**:
  - Keep-alive connections
  - Connection pooling
  - Automatic decompression (disabled in proxy)
  - TLS with configurable min version

```go
&http.Transport{
    TLSClientConfig: &tls.Config{
        MinVersion:         tls.VersionTLS12,
        InsecureSkipVerify: opts.TLSInsecure,
    },
    DisableCompression: false,
    IdleConnTimeout:    90 * time.Second,
}
```

### chromedp Transport

- **Package**: `github.com/chromedp/chromedp`
- **Implementation**: `browser.Pool` (implements `http.RoundTripper`)
- **Features**:
  - WebSocket connection to Chrome DevTools Protocol (CDP)
  - Semaphore-bounded concurrent tabs
  - Automatic tab cleanup after request
  - Exponential backoff on startup

**Pool Configuration:**

```go
type Pool struct {
    allocCtx    context.Context           // Parent context
    allocCancel context.CancelFunc        // Cancel function
    sem         *semaphore.Weighted       // Concurrency limiter
    timeout     time.Duration             // Request timeout
    wsURL       string                    // Chrome DevTools URL
    healthy     bool                      // Health flag
}
```

**RoundTrip Flow:**

1. Acquire semaphore slot (wait if at limit)
2. Create child context with timeout
3. Create chromedp context (opens new tab)
4. Navigate to URL
5. Wait for `body` element
6. Extract outer HTML
7. Release semaphore slot
8. Return response

---

## Conversion Pipeline

### HTML to Markdown

**Library**: `github.com/JohannesKaufmann/html-to-markdown`

**Process:**

1. **Check Content-Type**
   - Only process `text/html` responses
   - Respect `negotiate_only` flag (check `Accept: text/markdown`)

2. **Decompress**
   - Handle `gzip` and `deflate` encoding
   - Preserve original for caching

3. **Convert**
   - Parse HTML with `html.Parse`
   - Walk DOM tree and convert to Markdown
   - Handle tables, lists, links, images, etc.

4. **Count Tokens**
   - Use TikToken library to count tokens
   - Default encoding: `cl100k_base` (GPT-4/Claude)
   - Add to `X-Token-Count` header

5. **Cache & Output**
   - Cache original HTML (if enabled, respects RFC 7234)
   - Write Markdown to files (if enabled)
   - Add cache headers to response

### Token Counting

**Library**: `github.com/pkoukk/tiktoken-go`

**Encoding Options:**

- `cl100k_base` - GPT-4, Claude
- `p50k_base` - GPT-3.5
- `r50k_base` - Older models

**Counting Method:**

```go
enc, _ := tiktoken.GetEncoding("cl100k_base")
tokens := enc.EncodeOrdinary(text)
count := len(tokens)
```

---

## Building from Source

### Prerequisites

- **Go** 1.21 or later
- **git**
- (Optional) **Docker** for containerized build

### Build Steps

```bash
# Clone repository
git clone https://github.com/rickcrawford/markdowninthemiddle.git
cd markdowninthemiddle

# Download dependencies
go mod download

# Build binary
go build -o markdowninthemiddle .

# Verify build
./markdowninthemiddle --version
```

### Build with Optimization

```bash
# Strip debug info and reduce binary size
go build -ldflags="-s -w" -o markdowninthemiddle .
```

### Cross-Compilation

```bash
# macOS binary on Linux
GOOS=darwin GOARCH=arm64 go build -o markdowninthemiddle-macos .

# Linux binary
GOOS=linux GOARCH=amd64 go build -o markdowninthemiddle-linux .

# Windows binary
GOOS=windows GOARCH=amd64 go build -o markdowninthemiddle.exe .
```

### Building Docker Image

```bash
# Build local image
docker build -t markdowninthemiddle:latest .

# Build with specific Go version
docker build --build-arg GO_VERSION=1.23 -t markdowninthemiddle:latest .

# Multi-arch build
docker buildx build --platform linux/amd64,linux/arm64 -t markdowninthemiddle:latest .
```

---

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Specific Test

```bash
go test -run TestFilterMiddleware ./internal/filter/...
```

### Benchmark Tests

```bash
go test -bench=. -benchmem ./...
```

---

## Dependencies

### Core Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/go-chi/chi/v5` | Latest | HTTP router & middleware |
| `github.com/spf13/cobra` | Latest | CLI framework |
| `github.com/spf13/viper` | Latest | Configuration |
| `github.com/JohannesKaufmann/html-to-markdown` | Latest | HTML conversion |
| `github.com/pkoukk/tiktoken-go` | Latest | Token counting |
| `github.com/chromedp/chromedp` | Latest | Browser automation |
| `golang.org/x/sync` | Latest | Semaphore |

### Install Dependencies

```bash
go mod tidy         # Update dependencies
go mod download     # Download to cache
go mod verify       # Verify checksums
```

---

## Performance Tuning

### Memory Usage

**Per Request:**
- HTTP: <1MB
- chromedp: 20-50MB (depends on page complexity)

**Reduce chromedp Memory:**
```yaml
transport:
  chromedp:
    pool_size: 3  # Reduce concurrent tabs
```

### Connection Pooling

**HTTP Transport:**
```bash
# Adjust idle connections
MITM_HTTP_MAX_IDLE_CONNS=100
```

### Token Counting

**Avoid re-creating encoder:**
```go
// Good: create once, reuse
enc, _ := tiktoken.GetEncoding("cl100k_base")
// Cache 'enc' for multiple uses
```

---

## Debugging

### Enable Debug Logging

```bash
./markdowninthemiddle --config config.yml --log-level debug
```

### Browser Pool Debugging

```bash
# Add to code
log.Printf("Pool state: %d/%d slots available", p.sem.Release(1), poolSize)
```

### Capture HTTP Traffic

```bash
# Via proxy with verbose logging
./markdowninthemiddle --log-level debug 2>&1 | tee proxy.log

# Monitor with tcpdump
sudo tcpdump -i lo0 -n "port 8080"
```

### Test Certificate Loading

```bash
openssl x509 -in ./certs/cert.pem -text -noout
```

---

## Common Issues & Solutions

### Chrome Connection Failures

**Problem:** "Failed to open new tab - no browser is open"

**Solution:**
- Verify Chrome is running: `curl http://localhost:9222/json/version`
- Check Docker container health: `docker compose ps`
- Increase startup timeout in `browser.go`

### Memory Leaks

**Problem:** Memory usage grows over time with chromedp

**Solution:**
- Reduce `pool_size` to limit concurrent tabs
- Ensure tab contexts are properly closed
- Monitor with: `docker stats markdowninthemiddle-proxy`

### Slow Conversion

**Problem:** HTML to Markdown conversion is slow

**Causes:**
- Very large HTML (>10MB) - increase `max_body_size`
- chromedp overhead - use HTTP transport for non-JS sites
- Network latency - enable caching

---

## Contributing Guidelines

### Code Style

- **Format:** `gofmt` (enforced)
- **Linting:** `golangci-lint`
- **Tests:** Minimum 80% coverage

```bash
go fmt ./...
golangci-lint run ./...
go test -cover ./...
```

### Adding Features

1. Create feature branch: `git checkout -b feature/my-feature`
2. Add tests: `*_test.go`
3. Document: Update README if user-facing
4. Submit PR with description

---

## References

- **Go Documentation**: https://golang.org/doc
- **chromedp**: https://github.com/chromedp/chromedp
- **Chrome DevTools Protocol**: https://chromedevtools.github.io/devtools-protocol/
- **html-to-markdown**: https://github.com/JohannesKaufmann/html-to-markdown
- **TikToken**: https://github.com/pkoukk/tiktoken-go
