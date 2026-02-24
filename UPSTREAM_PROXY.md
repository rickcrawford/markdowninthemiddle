# Upstream Proxy Support

The proxy respects Go's standard environment variables for upstream proxy configuration. If your proxy needs to reach the internet through another proxy, set these variables.

---

## Environment Variables

The proxy automatically respects these standard Go proxy environment variables:

| Variable | Purpose | Example |
|----------|---------|---------|
| `HTTP_PROXY` | HTTP upstream proxy | `http://proxy.company.com:8080` |
| `HTTPS_PROXY` | HTTPS upstream proxy | `http://proxy.company.com:8080` |
| `NO_PROXY` | Comma-separated domains to bypass | `localhost,127.0.0.1,.internal.com` |

**Note:** Variable names are case-insensitive (can use `http_proxy`, `HTTP_PROXY`, etc.)

---

## Usage Examples

### Set Upstream Proxy

```bash
# All upstream requests go through a corporate proxy
export HTTP_PROXY=http://corporate-proxy.company.com:3128
export HTTPS_PROXY=http://corporate-proxy.company.com:3128

# Start proxy
./markdowninthemiddle
```

### Exclude Local Hosts

```bash
# Don't proxy requests to internal services
export HTTP_PROXY=http://corporate-proxy.company.com:3128
export NO_PROXY=localhost,127.0.0.1,.internal.com,.local

./markdowninthemiddle
```

### With Authentication

```bash
# Proxy with username/password (URL-encoded)
export HTTP_PROXY=http://username:password@proxy.company.com:3128
export HTTPS_PROXY=http://username:password@proxy.company.com:3128

./markdowninthemiddle
```

### Docker

```bash
# In docker-compose.yml
proxy:
  environment:
    HTTP_PROXY: http://corporate-proxy.company.com:3128
    HTTPS_PROXY: http://corporate-proxy.company.com:3128
    NO_PROXY: localhost,127.0.0.1,.internal.com
```

Or pass via command line:

```bash
docker compose -e HTTP_PROXY=http://proxy.company.com:3128 up
```

---

## How It Works

When the proxy makes requests to upstream servers, it:

1. **Checks environment variables** - `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`
2. **Matches request URL** - Determines if request should go through upstream proxy
3. **Routes accordingly:**
   - If URL matches `NO_PROXY` → Direct connection
   - Otherwise → Routes through upstream proxy specified in `HTTP_PROXY`/`HTTPS_PROXY`

---

## Testing

Verify upstream proxy is being used:

```bash
# Enable verbose logging
./markdowninthemiddle --log-level debug

# Make a request
curl -x http://localhost:8080 http://example.com

# Look for logs showing proxy connection
# (varies by implementation, but should show routing through upstream)
```

Or use tcpdump/Wireshark to verify traffic goes through the upstream proxy.

---

## Common Issues

### Proxy Connection Refused

```
error: dial tcp proxy.company.com:3128: connection refused
```

**Solution:**
- Verify upstream proxy is running
- Check firewall allows connection
- Verify proxy address and port are correct
- Test directly: `curl -x http://proxy.company.com:3128 http://example.com`

### Authentication Failed

```
error: 407 Proxy Authentication Required
```

**Solution:**
- Include credentials in URL: `http://user:pass@proxy.company.com:3128`
- URL-encode special characters: `user@domain:p%40ssword` for `p@ssword`

### NO_PROXY Not Working

```
# This won't match
export NO_PROXY=company.com

# Use correct format
export NO_PROXY=.company.com,internal.company.com
```

**Solution:**
- Use `.company.com` (with dot) for domain suffix matching
- Use full domain for exact match
- Separate multiple entries with commas

### "Proxy Cycle Detected"

If upstream proxy is also this proxy:

```
export HTTP_PROXY=http://localhost:8080  # DON'T DO THIS!
```

**Solution:**
- Never set proxy to point to itself
- Ensure upstream proxy is different service/host

---

## Proxy Chaining

You can chain multiple proxies:

```
Client → Proxy 1 (localhost:8080) → Corporate Proxy → Internet
```

Setup:
```bash
# On corporate proxy machine
corporate-proxy-service &

# On proxy machine
export HTTP_PROXY=http://corporate-proxy.company.com:3128
./markdowninthemiddle --addr :8080
```

Clients connect to Proxy 1, which routes through Corporate Proxy.

---

## Performance Notes

- **Upstream proxy latency** - Each request adds latency of upstream proxy
- **Connection pooling** - Reuses connections to upstream proxy when possible
- **Memory** - No additional memory overhead for proxy setup

---

## See Also

- **Go httpproxy package:** https://pkg.go.dev/golang.org/x/net/http/httpproxy
- **README.md** - Main proxy documentation
- **HTTPS_SETUP.md** - TLS configuration (different from upstream proxy)
