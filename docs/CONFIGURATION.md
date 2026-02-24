# Configuration Guide

Configure Markdown in the Middle via CLI flags, environment variables, or `config.yml`.

---

## Configuration Priority

Settings are applied in this order (highest to lowest):

1. **CLI Flags** - Command-line arguments
2. **Environment Variables** - `MITM_` prefixed (use `_` for nesting)
3. **config.yml** - Local YAML configuration file
4. **Built-in Defaults** - Hardcoded defaults in code

Example: CLI flag overrides environment variable, which overrides config.yml

```bash
# CLI takes priority
./markdowninthemiddle --addr :9090 \
  # overrides MITM_PROXY_ADDR env var \
  # overrides proxy.addr in config.yml
```

---

## Common Configuration Options

### Proxy

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| Listen Address | `--addr` | `MITM_PROXY_ADDR` | `proxy.addr` | `:8080` | Port/address to listen on |
| TLS | `--tls` | `MITM_TLS_ENABLED` | `tls.enabled` | `false` | Enable HTTPS on proxy |
| Auto-Certificate | `--auto-cert` | `MITM_TLS_AUTO_CERT` | `tls.auto_cert` | `false` | Auto-generate self-signed cert |

### Conversion

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| HTML→MD | `--convert` | `MITM_CONVERSION_ENABLED` | `conversion.enabled` | `true` | Convert HTML to Markdown |
| JSON→MD | `--convert-json` | `MITM_CONVERSION_CONVERT_JSON` | `conversion.convert_json` | `false` | Convert JSON to Markdown |
| Negotiate Only | `--negotiate-only` | `MITM_CONVERSION_NEGOTIATE_ONLY` | `conversion.negotiate_only` | `false` | Only convert when requested |
| Template Dir | `--template-dir` | `MITM_CONVERSION_TEMPLATE_DIR` | `conversion.template_dir` | `` | Directory with `.mustache` files |

### Transport

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| Type | `--transport` | `MITM_TRANSPORT_TYPE` | `transport.type` | `http` | `http` or `chromedp` |
| Chrome URL | `--chrome-url` | `MITM_TRANSPORT_CHROMEDP_URL` | `transport.chromedp.url` | `http://localhost:9222` | Chrome DevTools endpoint |
| Pool Size | `--chrome-pool-size` | `MITM_TRANSPORT_CHROMEDP_POOL_SIZE` | `transport.chromedp.pool_size` | `5` | Max concurrent Chrome tabs |

### Caching

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| Cache Dir | `--cache-dir` | `MITM_CACHE_DIR` | `cache.dir` | `` | Enable caching in directory |
| Respect Headers | N/A | `MITM_CACHE_RESPECT_HEADERS` | `cache.respect_headers` | `true` | Follow RFC cache-control |

### Output

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| Output Dir | `--output-dir` | `MITM_OUTPUT_DIR` | `output.dir` | `` | Save Markdown files to directory |

### Filtering

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| Allow Patterns | `--allow` | N/A | `filter.allowed` | `` | Regex patterns for allowed URLs |

### Security

| Option | CLI Flag | Env Var | Config | Default | Description |
|--------|----------|---------|--------|---------|-------------|
| TLS Insecure | `--tls-insecure` | `MITM_TLS_INSECURE` | `tls.insecure` | `false` | Skip upstream TLS verification |

---

## CLI Examples

### Proxy on custom port with caching

```bash
./markdowninthemiddle --addr :9090 --cache-dir ./cache
```

### Enable JSON conversion with templates

```bash
./markdowninthemiddle --convert-json --template-dir ./my-templates
```

### HTTPS with auto-generated certificate

```bash
./markdowninthemiddle --tls --auto-cert
```

### JavaScript rendering + caching

```bash
./markdowninthemiddle --transport chromedp --cache-dir ./cache
```

### Restrict to specific domains

```bash
./markdowninthemiddle \
  --allow "^https://api\.example\.com/" \
  --allow "^https://docs\.example\.com/"
```

### Save converted Markdown files

```bash
./markdowninthemiddle --output-dir ./markdown
```

### MCP server with JSON templates

```bash
./markdowninthemiddle mcp \
  --convert-json \
  --template-dir ./my-templates \
  --transport chromedp \
  --mcp-transport http \
  --mcp-addr :8081
```

---

## Environment Variables

All configuration can be set via environment variables using the `MITM_` prefix:

```bash
# Proxy
MITM_PROXY_ADDR=":9090"
MITM_PROXY_READ_TIMEOUT="60s"
MITM_PROXY_WRITE_TIMEOUT="60s"

# TLS
MITM_TLS_ENABLED="true"
MITM_TLS_AUTO_CERT="true"
MITM_TLS_CERT_FILE="/path/to/cert.pem"
MITM_TLS_KEY_FILE="/path/to/key.pem"
MITM_TLS_INSECURE="false"

# Conversion
MITM_CONVERSION_ENABLED="true"
MITM_CONVERSION_CONVERT_JSON="true"
MITM_CONVERSION_TEMPLATE_DIR="./my-templates"
MITM_CONVERSION_NEGOTIATE_ONLY="false"
MITM_CONVERSION_TIKTOKEN_ENCODING="cl100k_base"

# Transport
MITM_TRANSPORT_TYPE="chromedp"
MITM_TRANSPORT_CHROMEDP_URL="http://localhost:9222"
MITM_TRANSPORT_CHROMEDP_POOL_SIZE="5"

# Cache
MITM_CACHE_ENABLED="true"
MITM_CACHE_DIR="./cache"
MITM_CACHE_RESPECT_HEADERS="true"

# Output
MITM_OUTPUT_ENABLED="true"
MITM_OUTPUT_DIR="./markdown"

# Logging
MITM_LOG_LEVEL="info"

# Limits
MITM_MAX_BODY_SIZE="10485760"
```

Run with environment variables:

```bash
MITM_PROXY_ADDR=":9090" \
MITM_CACHE_DIR="./cache" \
MITM_CONVERSION_CONVERT_JSON="true" \
./markdowninthemiddle
```

---

## config.yml

Full configuration file example (see `examples/config.example.yml` for defaults):

```yaml
# Proxy listener settings
proxy:
  addr: ":8080"
  read_timeout: 30s
  write_timeout: 30s

# TLS settings
tls:
  enabled: false
  cert_file: ""
  key_file: ""
  auto_cert: true
  auto_cert_host: "localhost"
  auto_cert_dir: "./certs"
  insecure: false

# Conversion settings
conversion:
  enabled: true
  convert_json: false
  template_dir: ""
  tiktoken_encoding: "cl100k_base"
  negotiate_only: false

# Cache settings
cache:
  enabled: false
  dir: ""
  respect_headers: true

# Output settings
output:
  enabled: false
  dir: ""

# Transport settings
transport:
  type: "http"  # or "chromedp"
  chromedp:
    url: "http://localhost:9222"
    pool_size: 5

# Request filtering
filter:
  allowed: []

# Body size limit
max_body_size: 10485760  # 10 MB

# Logging
log_level: "info"
```

Load config file:

```bash
./markdowninthemiddle --config ./config.yml
```

Or with MITM_CONFIG env var:

```bash
MITM_CONFIG="./config.yml" ./markdowninthemiddle
```

---

## Docker Configuration

Docker services read from `examples/config.example.yml` mounted as read-only.

```bash
cd docker
docker compose up -d
```

To customize, mount a custom config:

Edit `docker/docker-compose.yml`:
```yaml
volumes:
  - ./config.yml:/etc/markdowninthemiddle/config.yml:ro
```

Then place `config.yml` in the `docker/` folder.

---

## MCP Server Configuration

MCP server configuration supports the same options as the proxy.

### Via CLI

```bash
./markdowninthemiddle mcp \
  --config ./config.yml \
  --transport chromedp \
  --convert-json \
  --template-dir ./my-templates
```

### Via config.yml

Same `config.yml` structure applies to MCP server:

```yaml
transport:
  type: "chromedp"
  chromedp:
    url: "http://localhost:9222"
    pool_size: 5

conversion:
  convert_json: true
  template_dir: "./my-templates"
```

---

## See Also

- [JSON_CONVERSION.md](./JSON_CONVERSION.md) - Configure JSON templates
- [CHROMEDP.md](./CHROMEDP.md) - Configure JavaScript rendering
- [HTTPS_SETUP.md](./HTTPS_SETUP.md) - TLS/HTTPS configuration
- [CODE_DETAILS.md](./CODE_DETAILS.md) - Full CLI reference
