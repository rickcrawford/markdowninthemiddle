# Docker Deployment

This directory contains Docker configuration files for running Markdown in the Middle.

## Files

- **Dockerfile** - Multi-stage build for the proxy application
- **docker-compose.yml** - Docker Compose configuration for proxy + Chrome services
- **.dockerignore** - Files to exclude from Docker build context
- **.env.example** - Example environment variables (copy to `.env` to customize)

## Quick Start

From the `docker/` directory:

```bash
# Start services (proxy on port 8080, Chrome on port 9222, MCP on port 8081)
docker compose up -d

# View logs
docker compose logs -f proxy

# Stop services
docker compose down
```

## Configuration

1. **Copy the example config:**
   ```bash
   cp ../examples/config.example.yml config.yml
   ```

2. **Customize environment variables (optional):**
   ```bash
   cp .env.example .env
   # Edit .env as needed
   ```

3. **Update docker-compose.yml volumes** (if using custom config):
   - Replace `../examples/config.example.yml` with `./config.yml` in the volumes section

## Services

### Proxy Service (`proxy`)
- **Image:** Built from root Dockerfile
- **Port:** 8080 (HTTP)
- **Features:**
  - HTML to Markdown conversion
  - Optional JavaScript rendering (chromedp)
  - TLS certificate auto-generation
  - Token counting
  - Response caching

### Chrome Service (`chrome`)
- **Image:** chromedp/headless-shell:latest
- **Port:** 9222 (DevTools Protocol)
- **Features:**
  - Headless browser for JavaScript rendering
  - Shared memory: 2GB

### MCP Service (`mcp`)
- **Image:** Built from root Dockerfile
- **Port:** 8081 (HTTP)
- **Features:**
  - MCP protocol server for Claude integration
  - fetch_markdown and fetch_raw tools
  - Uses chromedp for rendering

## Environment Variables

See `.env.example` for available options. Common ones:

```env
MITM_CONFIG="/etc/markdowninthemiddle/config.yml"
MITM_PROXY_ADDR="0.0.0.0:8080"
MITM_TLS_ENABLED="false"
MITM_TRANSPORT_TYPE="chromedp"
MITM_CONVERSION_ENABLED="true"
MITM_LOG_LEVEL="info"
```

## Building

```bash
# Build/rebuild the image
docker compose build

# View build logs
docker compose build --progress=plain
```

## Troubleshooting

```bash
# Check service health
docker compose ps

# View logs for a specific service
docker compose logs -f proxy    # proxy logs
docker compose logs -f chrome   # chrome logs
docker compose logs -f mcp      # mcp logs

# Open shell in proxy container
docker compose exec proxy sh

# Test the proxy
curl -x http://localhost:8080 http://example.com
```

## Volume Mounts

- `config.yml` - Configuration file (read-only)
- `certs/` - TLS certificates (auto-generated)
- `output/` - Markdown output directory (if enabled)

## Networks

All services communicate via the `mitm` bridge network. Port mappings:
- Proxy: 8080 → :8080
- Chrome DevTools: 9222 → :9222
- MCP: 8081 → :8081

## For more information

See the documentation in the `docs/` folder:
- [DOCKER.md](../docs/DOCKER.md) - Comprehensive deployment guide
- [CODE_DETAILS.md](../docs/CODE_DETAILS.md) - Full configuration reference
