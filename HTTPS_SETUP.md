# HTTPS Setup Guide

This guide covers running the proxy with and without TLS (HTTPS) on the proxy listener itself, and explains the difference between proxy TLS and MITM interception.

---

## Quick Comparison

| Mode | TLS | Use Case | Setup |
|------|-----|----------|-------|
| **HTTP (Default)** | No | Local testing, internal networks | No setup needed |
| **HTTPS (TLS)** | Yes | Public internet, shared networks | Generate/provide certificate |
| **MITM** | Special | Process HTTPS upstream traffic | Trust CA certificate |

---

## Mode 1: HTTP (Default - Easiest)

The proxy runs on plain HTTP without encryption. This is the default and requires no setup.

```bash
# Start proxy on http://localhost:8080
./markdowninthemiddle

# Test
curl -x http://localhost:8080 http://example.com
```

**When to use:**
- Local testing
- Internal networks (firewalled)
- Development environments
- Behind VPN

**When NOT to use:**
- Public internet
- Untrusted networks
- Shared WiFi
- Production with sensitive data

---

## Mode 2: HTTPS (TLS on Proxy Listener)

The proxy listens on HTTPS instead of HTTP. Clients connect to the proxy over encrypted TLS.

```bash
# Start with auto-generated certificate
./markdowninthemiddle --tls --auto-cert

# Or with custom certificate
./markdowninthemiddle --tls --cert-file ./certs/cert.pem --key-file ./certs/key.pem
```

**When to use:**
- Proxy exposed on internet
- Shared networks
- Enterprise security requirements
- Any untrusted network path

### Generate a Certificate

#### Option A: Auto-generated (Quick, but self-signed)

```bash
./markdowninthemiddle --tls --auto-cert
# Generates: ./certs/cert.pem, ./certs/key.pem
```

Then clients trust it:
```bash
# macOS/Linux
curl -x https://localhost:8080 --insecure http://example.com

# Better: Import to system
# See MITM_SETUP.md for macOS/Linux/Windows cert trust steps
```

#### Option B: Bring Your Own Certificate

```bash
# Use existing certificate and key
./markdowninthemiddle --tls \
  --cert-file /path/to/cert.pem \
  --key-file /path/to/key.pem
```

#### Option C: Generate with openssl (Self-signed)

```bash
# Generate RSA key
openssl genrsa -out key.pem 2048

# Generate self-signed certificate (10 year validity)
openssl req -new -x509 -key key.pem -out cert.pem -days 3650 \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"

# Use with proxy
./markdowninthemiddle --tls --cert-file ./cert.pem --key-file ./key.pem
```

#### Option D: Buy a Real Certificate (Production)

1. Get certificate from CA (e.g., Let's Encrypt, DigiCert)
2. Place in `./certs/` directory
3. Run: `./markdowninthemiddle --tls --cert-file ./cert.pem --key-file ./key.pem`

### Configuration

**config.yml:**
```yaml
proxy:
  addr: ":8080"  # or ":8443" for HTTPS
  read_timeout: 30s
  write_timeout: 30s

tls:
  enabled: true
  cert_file: "./certs/cert.pem"
  key_file: "./certs/key.pem"
  # Auto-generate if files missing:
  auto_cert: true
  auto_cert_host: "localhost"
  auto_cert_dir: "./certs"
```

**Environment variables:**
```bash
MITM_TLS_ENABLED=true
MITM_TLS_CERT_FILE=./certs/cert.pem
MITM_TLS_KEY_FILE=./certs/key.pem
./markdowninthemiddle
```

### Testing HTTPS Mode

```bash
# Without certificate trust (insecure)
curl -x https://localhost:8080 --insecure http://example.com

# With certificate trust (secure)
curl -x https://localhost:8080 http://example.com
# (requires installing cert, see below)
```

---

## Mode 3: MITM Interception (Decrypt HTTPS Upstream)

This is different from proxy TLS. While proxy TLS encrypts the connection between client and proxy, MITM allows the proxy to decrypt and process HTTPS traffic to upstream servers.

**Example:**
```
HTTPS UPSTREAM TRAFFIC:
Client → Proxy → Upstream HTTPS Server
                (can't decrypt)

MITM ENABLED:
Client → Proxy (decrypts via fake cert) → Upstream HTTPS Server
                (can read/modify)
```

See **[MITM_SETUP.md](./MITM_SETUP.md)** for full instructions on MITM mode.

Quick test:
```bash
# Enable MITM (requires client to trust CA)
./markdowninthemiddle --mitm

# Test (client must have CA cert)
curl -x http://localhost:8080 https://www.example.com -sD - | grep X-
# Should show: X-Token-Count, X-Transport headers
```

---

## Docker Setup

By default, Docker runs with chromedp transport and HTTP (no TLS):

```bash
docker compose up -d
curl -x http://localhost:8080 http://example.com
```

### Enable TLS in Docker

Edit `docker-compose.yml`:
```yaml
proxy:
  environment:
    MITM_TLS_ENABLED: "true"
```

Or use auto-generated:
```yaml
proxy:
  environment:
    MITM_TLS_ENABLED: "true"
    MITM_TLS_AUTO_CERT: "true"
```

Then:
```bash
docker compose up -d
# Test with insecure flag (or import cert)
curl -x https://localhost:8080 --insecure http://example.com
```

### Enable MITM in Docker

Edit `docker-compose.yml`:
```yaml
proxy:
  environment:
    MITM_ENABLED: "true"
    MITM_CERT_DIR: "/app/certs/mitm"
```

Then extract and trust CA:
```bash
docker compose exec proxy cat /app/certs/mitm/ca-cert.pem > ~/mitm-ca.pem
# Follow MITM_SETUP.md to trust certificate on your OS
```

---

## Troubleshooting

### "certificate verify failed" with HTTPS proxy

**Cause:** Client doesn't trust proxy certificate

**Solution:**

**Option 1: Ignore (insecure, testing only)**
```bash
curl -x https://localhost:8080 --insecure http://example.com
```

**Option 2: Trust the certificate (secure)**
- See MITM_SETUP.md (same process)
- Or use `--cacert`:
```bash
curl -x https://localhost:8080 --cacert ./certs/cert.pem http://example.com
```

### macOS curl won't trust self-signed cert

macOS curl uses Apple SecTrust which is strict about self-signed certs:

```bash
# Option 1: Use --insecure (testing only)
curl --insecure ...

# Option 2: Use homebrew curl (uses OpenSSL)
brew install curl
/usr/local/opt/curl/bin/curl ...

# Option 3: Add to Keychain
# See MITM_SETUP.md for macOS section
```

### "address already in use" error

Another service is using the port:

```bash
# Find what's using :8080
lsof -i :8080

# Or use different port
./markdowninthemiddle --addr :9090

# Or in docker-compose.yml
proxy:
  ports:
    - "9090:8080"
```

### Certificate generation failed

```bash
# Check directory permissions
ls -la ./certs/

# Fix permissions
chmod 755 ./certs

# Regenerate
rm ./certs/cert.pem ./certs/key.pem
./markdowninthemiddle --tls --auto-cert
```

---

## Summary Table

| Need | Use This | Why |
|------|----------|-----|
| Local testing | HTTP (default) | Easiest, no setup |
| Internal network | HTTP + firewall | Firewall provides security |
| Public internet | HTTPS (TLS) | Encrypt client-proxy link |
| Process HTTPS sites | MITM mode | Decrypt upstream traffic |
| Both proxy TLS + MITM | TLS + MITM | Full encryption + processing |

---

## See Also

- **[MITM_SETUP.md](./MITM_SETUP.md)** - Client setup for MITM mode
- **[MITM_IMPLEMENTATION.md](./MITM_IMPLEMENTATION.md)** - Technical details on MITM
- **[README.md](./README.md)** - Main documentation
