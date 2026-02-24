# Troubleshooting Guide

Common issues and solutions.

---

## Proxy Won't Start

### Port Already in Use

**Error:** `listen tcp :8080: bind: address already in use`

**Solution:**
1. Check what's using the port:
   ```bash
   lsof -i :8080
   ```

2. Kill the existing process:
   ```bash
   kill -9 <PID>
   ```

3. Or use a different port:
   ```bash
   ./markdowninthemiddle --addr :9090
   ```

### Permission Denied

**Error:** `listen tcp :80: bind: permission denied`

**Solution:**
- Use a port > 1024 (don't need sudo):
  ```bash
  ./markdowninthemiddle --addr :8080
  ```

- Or run with sudo (not recommended):
  ```bash
  sudo ./markdowninthemiddle --addr :80
  ```

---

## Chrome Connection Issues

### Chrome Not Connecting (Docker)

**Error:**
```
chromedp: failed to connect to Chrome at http://chrome:9222
```

**Solution:**

1. Check Chrome service is running:
   ```bash
   docker compose ps chrome
   ```
   Should show `Up` and `healthy`

2. Restart Chrome:
   ```bash
   docker compose restart chrome
   ```

3. Check Chrome logs:
   ```bash
   docker compose logs -f chrome
   ```

4. Verify Chrome port is open:
   ```bash
   curl http://localhost:9222/json/version
   ```

### Chrome URL Wrong (Local)

**Error:**
```
chromedp: failed to connect to Chrome at http://localhost:9222
```

**Solution:**

1. Start Chrome with debugging protocol:
   ```bash
   # macOS
   /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome \
     --remote-debugging-port=9222 &

   # Linux
   google-chrome --remote-debugging-port=9222 &

   # Windows
   "C:\Program Files\Google\Chrome\Application\chrome.exe" --remote-debugging-port=9222
   ```

2. Or specify the Chrome URL:
   ```bash
   ./markdowninthemiddle --chrome-url http://localhost:9222
   ```

3. Or use helper script:
   ```bash
   ./scripts/start-chrome.sh &
   ./markdowninthemiddle
   ```

---

## HTTPS/TLS Issues

### Certificate Generation Failed

**Error:**
```
error generating certificate: open ./certs: permission denied
```

**Solution:**

1. Create certs directory:
   ```bash
   mkdir -p certs
   chmod 755 certs
   ```

2. Generate certificate manually:
   ```bash
   ./markdowninthemiddle gencert --dir ./certs
   ```

3. Or let Docker handle it:
   ```bash
   cd docker
   docker compose up -d
   ```

### TLS Handshake Error

**Error:**
```
http: TLS handshake error from [::1]:12345: tls: first record does not look like a TLS handshake
```

**Solution:**

This usually means sending HTTP to an HTTPS port.

- Use HTTPS with the proxy:
  ```bash
  curl -k -x https://localhost:8080 http://example.com
  ```

- Or disable TLS:
  ```bash
  ./markdowninthemiddle --tls=false
  ```

### Client Certificate Not Trusted

**Error:** Browser shows "certificate not trusted" warning

**Solution:**

For MITM mode, you need to trust the CA certificate:

1. See [HTTPS_SETUP.md](./HTTPS_SETUP.md) for TLS setup
2. See [MITM_SETUP.md](./MITM_SETUP.md) for certificate installation

---

## Conversion Not Working

### HTML Not Converting to Markdown

**Check:**
1. Verify conversion is enabled:
   ```bash
   ./markdowninthemiddle --convert=true
   ```

2. Check response Content-Type:
   ```bash
   curl -x http://localhost:8080 http://example.com -sD - | grep "Content-Type"
   ```
   Should be `text/html` or `text/markdown`

3. Check response headers:
   ```bash
   curl -x http://localhost:8080 http://example.com -sD - | grep "X-"
   ```
   Look for `X-Token-Count` (shows it was converted)

### JSON Not Converting to Markdown

**Check:**
1. Enable JSON conversion:
   ```bash
   ./markdowninthemiddle --convert-json --template-dir ./my-templates
   ```

2. Verify Content-Type is `application/json`:
   ```bash
   curl -x http://localhost:8080 https://api.example.com/data -sD - | grep "Content-Type"
   ```

3. Check if template matches:
   - Enable debug logging:
     ```bash
     ./markdowninthemiddle --convert-json --template-dir ./my-templates --log-level debug
     ```
   - Look for "template matched" in logs

4. Verify template naming:
   - Template: `api.example.com__users.mustache`
   - URL: `https://api.example.com/users`
   - Scheme is stripped before matching

### Negotiate-Only Mode Not Working

**Check:**
1. Enable negotiate-only mode:
   ```bash
   ./markdowninthemiddle --negotiate-only
   ```

2. Send Accept header:
   ```bash
   curl -x http://localhost:8080 \
     -H "Accept: text/markdown" \
     http://example.com
   ```

   Without header, HTML is returned as-is.

---

## Caching Issues

### Cache Not Working

**Check:**
1. Enable cache and verify directory:
   ```bash
   ./markdowninthemiddle --cache-dir ./cache
   ```

2. Check cache directory exists and is writable:
   ```bash
   ls -la cache/
   ```

3. Make the same request twice (should be faster second time):
   ```bash
   curl -x http://localhost:8080 http://example.com
   curl -x http://localhost:8080 http://example.com  # Should be cached
   ```

4. Check cache files were created:
   ```bash
   ls -la cache/
   ```

### Cache Control Not Respected

**Check:**
1. Verify respect-headers is enabled (default):
   ```bash
   # In config.yml
   cache:
     respect_headers: true
   ```

2. Check the API response Cache-Control header:
   ```bash
   curl http://example.com/api -sD - | grep "Cache-Control"
   ```

3. Some APIs use `no-store` or `no-cache` (won't be cached)

---

## Token Counting Issues

### Token Count Not Appearing

**Check:**
1. Verify encoding is correct:
   ```bash
   ./markdowninthemiddle --config ./config.yml
   # Check: conversion.tiktoken_encoding in config
   ```

2. Check response has X-Token-Count header:
   ```bash
   curl -x http://localhost:8080 http://example.com -sD - | grep "X-Token-Count"
   ```

3. Invalid encoding will fall back to estimating:
   ```bash
   # Valid: cl100k_base, p50k_base
   # Invalid: gibberish → token count will be estimated
   ```

---

## Request Filtering Issues

### URL Not Being Filtered

**Check:**
1. Verify filter pattern:
   ```bash
   ./markdowninthemiddle \
     --allow "^https://api\.example\.com/" \
     --allow "^https://docs\.example\.com/"
   ```

2. Test with a non-matching URL:
   ```bash
   curl -x http://localhost:8080 http://other.com
   ```
   Should return 403 Forbidden

3. Test with matching URL:
   ```bash
   curl -x http://localhost:8080 https://api.example.com/data
   ```
   Should work (200 OK)

4. Check regex pattern is valid:
   - Use [regex101.com](https://regex101.com) to test
   - Use Go regex dialect
   - `^` = start, `$` = end, `\.` = literal dot

---

## Docker Issues

### Services Won't Start

**Check:**
1. Verify docker-compose.yml is valid:
   ```bash
   cd docker
   docker compose config
   ```

2. Check logs:
   ```bash
   docker compose logs -f
   ```

3. Look for common errors:
   - Port conflicts: `docker ps | grep 8080`
   - Image build failures: `docker compose build --progress=plain`
   - Volume mount issues: `docker compose ps` and check volumes

### Container Exits Immediately

**Check:**
1. View detailed logs:
   ```bash
   docker compose logs proxy
   ```

2. Common causes:
   - Config file missing (check volume mounts)
   - Port already in use
   - Cert generation failed

3. Rebuild and restart:
   ```bash
   docker compose down -v
   docker compose build
   docker compose up -d
   ```

### Chrome Service Not Healthy

**Check:**
1. View Chrome logs:
   ```bash
   docker compose logs chrome
   ```

2. Test Chrome health manually:
   ```bash
   docker compose exec chrome curl http://localhost:9222/json/version
   ```

3. Increase health check timeout (if needed):
   Edit `docker/docker-compose.yml`:
   ```yaml
   healthcheck:
     timeout: 15s  # Increase from 10s
   ```

---

## Performance Issues

### Proxy Slow

**Check:**
1. Enable caching:
   ```bash
   ./markdowninthemiddle --cache-dir ./cache
   ```

2. Reduce Chrome pool size if low memory:
   ```bash
   ./markdowninthemiddle --chrome-pool-size 2
   ```

3. Check system resources:
   ```bash
   top -o %MEM  # Memory usage
   ```

4. Use HTTP transport instead of chromedp:
   ```bash
   ./markdowninthemiddle --transport http
   ```

### Memory Usage High

**Check:**
1. Reduce Chrome pool size:
   ```bash
   --chrome-pool-size 2  # Default is 5
   ```

2. Limit response body size:
   ```bash
   --max-body-size 5242880  # 5 MB instead of 10 MB
   ```

3. Disable caching if large:
   ```bash
   # Don't use --cache-dir or --output-dir
   ```

---

## Docker Compose on Mac/Windows

### Docker Desktop Issues

**Problem:** `docker compose` command not found

**Solution:**
1. Install Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop)
2. Or use `docker-compose` (with dash) on older versions:
   ```bash
   docker-compose up -d  # Old syntax
   docker compose up -d  # New syntax
   ```

### File Permissions Issues

**Problem:** Volume mounts show permission denied

**Solution:**
1. On macOS, Docker Desktop handles permissions automatically
2. On Windows with WSL2:
   ```bash
   cd /mnt/c/Users/YourUser/Projects/markdowninthemiddle
   docker compose up -d
   ```

---

## Still Having Issues?

1. **Check logs:**
   ```bash
   ./markdowninthemiddle 2>&1 | tee proxy.log
   ```

2. **Enable debug logging:**
   ```bash
   ./markdowninthemiddle --log-level debug
   ```

3. **Check configuration:**
   ```bash
   ./markdowninthemiddle --help
   cat config.yml | grep -A5 conversion
   ```

4. **Report on GitHub:**
   - [GitHub Issues](https://github.com/rickcrawford/markdowninthemiddle/issues)
   - Include logs, configuration, error messages
   - Describe what you expected vs what happened

---

## See Also

- [CONFIGURATION.md](./CONFIGURATION.md) - Configuration options
- [HTTPS_SETUP.md](./HTTPS_SETUP.md) - TLS/HTTPS setup
- [MITM_SETUP.md](./MITM_SETUP.md) - MITM certificate setup
- [CHROMEDP.md](./CHROMEDP.md) - JavaScript rendering issues
