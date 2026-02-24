# Optional: JavaScript Rendering with Chromedp

> **Default Transport:** The proxy uses standard HTTP by default, which is simpler and has no external dependencies. Chromedp is an **optional** feature for advanced use cases.

## Overview

**chromedp** is a headless browser automation library that lets the proxy render JavaScript-heavy websites by controlling a real Chrome/Chromium browser instance instead of making raw HTTP requests.

### When to Use chromedp

**Use chromedp when:**
- ✅ Websites require JavaScript to render content
- ✅ You need to handle dynamic/SPA (Single Page Application) content
- ✅ Target sites load data via client-side JavaScript
- ✅ You need accurate visual rendering

**Use standard HTTP when (default):**
- ✅ Simple HTML sites (no JavaScript) - **RECOMMENDED**
- ✅ Performance is critical (chromedp is slower)
- ✅ Resource constraints (chromedp uses more memory)
- ✅ Quick proxying without rendering overhead

## Architecture

```
┌──────────────────────────────────────────────┐
│ Proxy Request                                │
│  GET http://example.com → HTTP Client       │
└──────────────┬───────────────────────────────┘
               │
         ┌─────▼──────────────┐
         │ Transport Layer    │
         │ (http or chromedp) │
         └─────┬──────────────┘
               │
      ┌────────┴────────────────────────────┐
      │                                     │
  ┌───▼─────┐  (standard HTTP)  ┌──────────▼────┐
  │ Standard│ ◄─────────────────► │ Chrome DevTools
  │ Transport   tcp/ip            │ Protocol (CDP)
  └───┬─────┘                    └──────────┬────┘
      │                                    │
      │                          ┌─────────▼─────────┐
      │                          │ Chrome Instance   │
      │                          │ (Headless Shell)  │
      │                          │                   │
      │                          │ [Tab1] [Tab2] ... │
      │                          │                   │
      │                          │ Renders page,     │
      │                          │ executes JS,      │
      │                          │ extracts HTML     │
      │                          └─────────┬─────────┘
      │                                    │
      │  ┌────────────────────────────────┘
      │  │
  ┌───▼──▼───────┐
  │ Response     │
  │ (HTML/JSON)  │
  └──────────────┘
```

## How It Works

### chromedp Flow

1. **Allocator** - Establishes WebSocket connection to Chrome DevTools Protocol
   ```
   Chrome listens on: http://localhost:9222
   chromedp connects to the remote debugging protocol
   ```

2. **Browser Context** - Each request gets a temporary browser tab
   ```go
   ctx, cancel := chromedp.NewContext(allocCtx)
   defer cancel() // closes tab when done
   ```

3. **Navigation & Rendering**
   ```go
   chromedp.Navigate(url)          // Go to URL
   chromedp.WaitReady("body")       // Wait for body element
   chromedp.OuterHTML("html", &html) // Extract rendered HTML
   ```

4. **Semaphore Pool** - Limits concurrent tabs to avoid browser crashes
   ```
   pool_size=5 means max 5 tabs open at once
   Requests queue if limit is reached
   ```

### Request Lifecycle (with chromedp)

```
┌─────────────────────────────────────────────────────────────┐
│ Client Request: GET http://example.com/page                 │
└────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ Proxy receives request, checks semaphore                     │
│ If all 5 tabs busy: wait for one to free up                │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ Acquire slot (semaphore -1)                                 │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ chromedp.NewContext() → Open new tab in Chrome             │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ chromedp.Navigate(url) → Visit the page                    │
│ • Chrome loads HTML                                         │
│ • Browser parses HTML                                       │
│ • Executes <script> tags                                   │
│ • Loads resources (CSS, JS, images)                        │
│ • Fires events (DOMContentLoaded, load)                    │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ chromedp.WaitReady("body") → Wait until body element ready  │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ chromedp.OuterHTML("html", &html) → Extract rendered HTML │
│ • Gets current DOM tree                                     │
│ • Includes all JS modifications                            │
│ • CSS is applied                                           │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ Return HTML as response                                     │
│ (HTML-to-Markdown conversion happens next)                 │
└─────────────────────────────────────────────────────────────┬┘
                                                              │
┌─────────────────────────────────────────────────────────────▼┐
│ Close tab & release semaphore slot (+1)                    │
└─────────────────────────────────────────────────────────────┘
```

## Setup Options

### Option 0: Default (HTTP Transport)

**The proxy works out-of-the-box with standard HTTP transport (no Chrome needed):**

```bash
# Docker Compose (HTTP only)
docker compose up -d

# Local binary
./markdowninthemiddle
```

Both start with HTTP transport and work immediately. To enable chromedp, follow options below.

---

### Option 1: Docker Compose with Chromedp

**To enable JavaScript rendering in Docker:**

```bash
# In docker-compose.yml, uncomment the Chrome service dependency:
# depends_on:
#   chrome:
#     condition: service_healthy

# Set transport to chromedp:
export MITM_TRANSPORT_TYPE=chromedp
export MITM_TRANSPORT_CHROMEDP_URL=http://chrome:9222

# Start services
docker compose up -d
```

**What happens:**
- `chrome` service starts with DevTools Protocol enabled
- `proxy` service waits for Chrome health check to pass
- Proxy connects to `http://chrome:9222` and uses chromedp for rendering

### Option 2: Local Chrome (Development)

**Run Chrome locally on your machine:**

**macOS:**
```bash
# Install Chrome if needed
brew install google-chrome

# Start Chrome with remote debugging
google-chrome --headless --disable-gpu --remote-debugging-port=9222

# In another terminal, start proxy
./markdowninthemiddle --transport chromedp

# Proxy connects to http://localhost:9222
```

**Linux:**
```bash
# Install Chrome if needed
sudo apt-get install chromium-browser

# Start Chrome
chromium-browser --headless --disable-gpu --remote-debugging-port=9222

# Start proxy
./markdowninthemiddle --transport chromedp
```

**Windows:**
```bash
# Start Chrome with DevTools enabled
"C:\Program Files\Google\Chrome\Application\chrome.exe" ^
  --headless --disable-gpu --remote-debugging-port=9222

# Start proxy (in PowerShell)
.\markdowninthemiddle.exe --transport chromedp
```

### Option 3: Remote Chrome (Production)

**Point to Chrome running elsewhere:**

```bash
# Chrome running on another server at 192.168.1.100:9222
export MITM_TRANSPORT_CHROMEDP_URL=http://192.168.1.100:9222
./markdowninthemiddle --transport chromedp
```

Or in `config.yml`:

```yaml
transport:
  type: chromedp
  chromedp:
    url: http://192.168.1.100:9222
    pool_size: 10
```

### Option 4: Cloud Chrome (Browserless, Puppeteer Services)

**Using Browserless.io or similar service:**

```bash
export MITM_TRANSPORT_CHROMEDP_URL=https://chrome.browserless.io
./markdowninthemiddle --transport chromedp
```

**Note:** chromedp expects a WebSocket CDP endpoint. Verify your service exposes `/json/version`.

## Configuration

### config.yml

```yaml
transport:
  # Type of transport: "http" (default) or "chromedp"
  type: "chromedp"

  chromedp:
    # Chrome DevTools Protocol URL
    # Docker Compose: http://chrome:9222
    # Local: http://localhost:9222
    # Remote: http://other-host:9222
    url: "http://localhost:9222"

    # Maximum concurrent browser tabs
    # Higher = more parallelism but more memory usage
    # Typical values: 3-10 depending on available resources
    pool_size: 5
```

### Environment Variables

```bash
# Enable chromedp transport
export MITM_TRANSPORT_TYPE=chromedp

# Point to your Chrome instance
export MITM_TRANSPORT_CHROMEDP_URL=http://localhost:9222

# Limit concurrent tabs
export MITM_TRANSPORT_CHROMEDP_POOL_SIZE=5
```

### CLI Flags

```bash
# Only transport type can be set via CLI
./markdowninthemiddle --transport chromedp

# Chrome URL and pool size must be in config.yml or env vars
```

## Troubleshooting

### Error: "connection refused"

**Problem:** Chrome is not running or not accessible

**Solutions:**

1. **Docker Compose:**
   ```bash
   docker compose ps  # Check if chrome service is running
   docker compose logs chrome  # View Chrome logs
   docker compose restart chrome  # Restart Chrome
   ```

2. **Local Chrome:**
   ```bash
   # Check if Chrome is listening
   nc -zv localhost 9222

   # If not, start Chrome:
   google-chrome --headless --disable-gpu --remote-debugging-port=9222
   ```

3. **Check URL:**
   ```bash
   # Verify Chrome URL is correct
   curl http://localhost:9222/json/version
   # Should return JSON like: {"Browser":"Chrome/...","Version":...}
   ```

### Error: "too many open files"

**Problem:** Pool size too high, hitting OS limit

**Solution:**
```bash
# Increase OS limit (Linux/macOS)
ulimit -n 2048

# Or reduce pool size
export MITM_TRANSPORT_CHROMEDP_POOL_SIZE=3
```

### Error: "BUS_ADRERR" or "out of memory"

**Problem:** Chrome running out of shared memory (Docker)

**Solution:**
Docker Compose already sets `shm_size: "2gb"`. If still failing:

```bash
docker compose down
docker system prune  # Clean up
docker compose up -d
```

### Slow responses

**Problem:** Browser rendering is slow

**Causes & Solutions:**

1. **Complex JavaScript:** Normal - chromedp waits for JS to execute
   - Consider using HTTP transport for simple HTML
   - Increase timeout: `chromedp.timeout: 60s` (in code)

2. **Pool too small:** Requests queuing
   - Increase `pool_size` (use more memory)

3. **Chrome overloaded:** Many tabs competing for resources
   - Reduce concurrent load
   - Add more Chrome instances (run multiple Docker containers)

### If Chrome Connection Fails

**Important:** Chromedp is optional. If Chrome isn't available:

1. **Use HTTP transport (default)** - No Chrome needed, proxy works immediately
2. **Don't enable chromedp** - Only set `--transport chromedp` if Chrome is running
3. **Graceful degradation** - Use multiple environment configs:

```bash
# Only enable chromedp if Chrome is available
if [ "$(curl -s http://localhost:9222/json/version)" ]; then
  export MITM_TRANSPORT_TYPE=chromedp
else
  export MITM_TRANSPORT_TYPE=http
fi

./markdowninthemiddle
```

Or just use HTTP transport for most use cases.

## Best Practices

1. **Use Docker Compose for production**
   - Orchestrates Chrome + Proxy
   - Health checks ensure reliability
   - Easy scaling

2. **Set appropriate pool_size**
   ```
   Available RAM / 50MB per tab ≈ optimal pool_size

   Example:
   512MB available → pool_size: 3-5
   2GB available → pool_size: 10-15
   8GB available → pool_size: 50+
   ```

3. **Monitor Chrome health**
   ```bash
   docker stats markdowninthemiddle-chrome
   ```

4. **Use HTTP transport as default**
   - Only enable chromedp for domains requiring JS
   - Use filter to target specific URLs:
   ```yaml
   filter:
     allowed:
       - "^https://spa-app\\.com"  # chromedp
       - "^https://api\\.example\\.com"  # HTTP
   ```

5. **Set reasonable timeouts**
   ```yaml
   proxy:
     read_timeout: 60s  # Increase for chromedp (JS rendering)
     write_timeout: 60s
   ```

## Advanced Usage

### Custom Chrome Arguments

To customize Chrome behavior, modify the docker-compose.yml:

```yaml
chrome:
  environment:
    HEADLESS_SHELL_ARGS: "--disable-gpu --disable-dev-shm-usage --no-sandbox"
```

### Multiple Chrome Instances

For high load, run multiple Chrome instances:

```yaml
chrome-1:
  image: chromedp/headless-shell:latest
  ports:
    - "9222:9222"

chrome-2:
  image: chromedp/headless-shell:latest
  ports:
    - "9223:9222"

proxy:
  # Configure to use first Chrome, or implement load balancing
  environment:
    MITM_TRANSPORT_CHROMEDP_URL: http://chrome-1:9222
```

## References

- **chromedp GitHub:** https://github.com/chromedp/chromedp
- **Chrome DevTools Protocol:** https://chromedevtools.github.io/devtools-protocol/
- **Headless Chrome:** https://developers.google.com/web/updates/2017/04/headless-chrome
