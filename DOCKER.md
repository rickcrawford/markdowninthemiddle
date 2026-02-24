# Docker Deployment Guide

## Quick Start

### Using the Helper Script (Recommended)

```bash
# Start services
./scripts/docker-compose.sh start

# View logs
./scripts/docker-compose.sh logs proxy

# Check status
./scripts/docker-compose.sh status

# Test the proxy
./scripts/docker-compose.sh test

# Stop services
./scripts/docker-compose.sh stop
```

### Manual Docker Compose

```bash
# Start the proxy + Chrome services
docker compose up -d

# View logs
docker compose logs -f proxy

# Stop services
docker compose down
```

The proxy will be available at:
- HTTP: `http://localhost:8080`
- Chrome DevTools: `http://localhost:9222` (internal)

### 2. Helper Script Commands

The `docker-compose.sh` script provides convenient shortcuts:

```bash
./scripts/docker-compose.sh start        # Start all services
./scripts/docker-compose.sh stop         # Stop all services
./scripts/docker-compose.sh restart      # Restart services
./scripts/docker-compose.sh status       # Show service status
./scripts/docker-compose.sh logs proxy   # View proxy logs
./scripts/docker-compose.sh chrome-logs  # View Chrome logs
./scripts/docker-compose.sh test         # Test with sample request
./scripts/docker-compose.sh shell        # Open shell in proxy container
./scripts/docker-compose.sh build        # Rebuild Docker image
./scripts/docker-compose.sh clean        # Remove all containers/volumes
```

### 3. Configuration

**Note:** When using Docker Compose, Chrome is automatically available at `http://chrome:9222` internally, so you don't need to start it separately.

#### Via config.yml (Recommended)

Create or edit `config.yml` in the project root:

```yaml
transport:
  type: "chromedp"
  chromedp:
    url: "http://chrome:9222"
    pool_size: 5

filter:
  allowed:
    - "^https://api\\.example\\.com"
    - "^https://example\\.com/docs"

conversion:
  enabled: true
  negotiate_only: false
  convert_json: false

tls:
  enabled: false  # Set to true for HTTPS
```

The container automatically reads `/etc/markdowninthemiddle/config.yml` (mapped to your local `config.yml`).

#### Via Environment Variables

Create a `.env` file (copy from `.env.example`):

```bash
# Copy template
cp .env.example .env

# Edit as needed
nano .env

# Start with env file
docker compose --env-file .env up -d
```

### 4. Certificate Generation

Certificates are automatically generated on first startup:

```
Generated self-signed TLS certificates:
  /etc/markdowninthemiddle/certs/cert.pem
  /etc/markdowninthemiddle/certs/key.pem
```

**To use your own certificates:**

```bash
# Place your certs in the certs directory
cp /path/to/your/cert.pem certs/
cp /path/to/your/key.pem certs/

# Update config.yml
tls:
  enabled: true
  cert_file: /etc/markdowninthemiddle/certs/cert.pem
  key_file: /etc/markdowninthemiddle/certs/key.pem
  auto_cert: false

docker compose restart proxy
```

## Port Configuration

The proxy exposes:

| Port | Protocol | Service | Purpose |
|------|----------|---------|---------|
| 8080 | TCP | Proxy | HTTP or HTTPS (depends on TLS setting) |
| 9222 | TCP | Chrome | DevTools Protocol (internal only) |

To change the proxy port, edit `docker-compose.yml`:

```yaml
proxy:
  ports:
    - "3128:8080"      # Use port 3128 instead
```

Then access the proxy at `http://localhost:3128`

## Volume Mounts

The container uses these volumes:

| Mount | Container Path | Purpose |
|-------|-----------------|---------|
| `./config.yml` | `/etc/markdowninthemiddle/config.yml` | Configuration (read-only) |
| `./certs` | `/etc/markdowninthemiddle/certs` | TLS certificates |
| `./output` | `/var/log/markdowninthemiddle/output` | Converted Markdown files |

To add more volumes, edit `docker-compose.yml`:

```yaml
proxy:
  volumes:
    - ./custom-templates:/var/lib/markdowninthemiddle/templates
```

## Health Checks

Both services include health checks:

```bash
# Check status
docker compose ps

# View health of proxy
docker inspect markdowninthemiddle-proxy | grep -A 10 '"Health"'
```

## Logs

View real-time logs:

```bash
# All services
docker compose logs -f

# Proxy only
docker compose logs -f proxy

# Chrome only
docker compose logs -f chrome
```

## Advanced Usage

### Custom Build

Build with custom settings:

```bash
docker build \
  --build-arg GO_VERSION=1.24.7 \
  --build-arg ALPINE_VERSION=3.20 \
  -t my-mitm:latest .

docker compose up -d
```

### Multi-Stage Debugging

Run container interactively:

```bash
docker compose run --rm proxy /bin/bash
```

### Monitor Resource Usage

```bash
docker stats markdowninthemiddle-proxy markdowninthemiddle-chrome
```

### Persist Chrome Cache

Add to `docker-compose.yml`:

```yaml
chrome:
  volumes:
    - chrome-cache:/root/.cache
    - chrome-data:/root/.local

volumes:
  chrome-cache:
  chrome-data:
```

## Troubleshooting

### Chrome Connection Failed

```bash
# Check Chrome is healthy
docker compose ps

# Restart Chrome
docker compose restart chrome

# Wait 5 seconds, restart proxy
sleep 5
docker compose restart proxy
```

### Certificate Generation Failed

```bash
# Check logs
docker compose logs proxy

# Manual generation
docker compose exec proxy /app/gencert \
  -cert /etc/markdowninthemiddle/certs/cert.pem \
  -key /etc/markdowninthemiddle/certs/key.pem
```

### Config Not Loading

```bash
# Verify config file exists
ls -la config.yml

# Check container's config
docker compose exec proxy cat /etc/markdowninthemiddle/config.yml

# Verify environment variable
docker compose exec proxy printenv MITM_CONFIG
```

### High Memory Usage

The proxy + Chrome can use 500MB - 1GB. To limit:

```yaml
proxy:
  mem_limit: 512m
  memswap_limit: 512m

chrome:
  mem_limit: 1g
```

## Production Deployment

For production:

1. **Use your own certificates** (not auto-generated)
2. **Set MITM_LOG_LEVEL=warn**
3. **Enable caching** for frequently accessed content
4. **Use a reverse proxy** (nginx, Traefik) in front for load balancing
5. **Set resource limits** on both containers
6. **Use named volumes** instead of bind mounts
7. **Enable restart policies** (already set to `unless-stopped`)

Example production compose:

```yaml
services:
  chrome:
    image: chromedp/headless-shell:latest
    restart: always
    mem_limit: 1g
    networks:
      - mitm

  proxy:
    build: .
    restart: always
    mem_limit: 512m
    volumes:
      - config:/etc/markdowninthemiddle
      - certs:/etc/markdowninthemiddle/certs
      - output:/var/log/markdowninthemiddle/output
    depends_on:
      chrome:
        condition: service_healthy
    networks:
      - mitm
    environment:
      MITM_LOG_LEVEL: "warn"
      MITM_TRANSPORT_CHROMEDP_URL: "http://chrome:9222"

volumes:
  config:
  certs:
  output:

networks:
  mitm:
    driver: bridge
```

## Docker Hub / Registry

To push your image:

```bash
docker login
docker tag markdowninthemiddle:latest myregistry/markdowninthemiddle:v1.0
docker push myregistry/markdowninthemiddle:v1.0
```

Then pull in production:

```yaml
proxy:
  image: myregistry/markdowninthemiddle:v1.0
```
