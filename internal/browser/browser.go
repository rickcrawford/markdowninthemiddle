package browser

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/sync/semaphore"
)

// Pool manages a semaphore-bounded pool of tabs against a remote Chrome instance.
type Pool struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	sem         *semaphore.Weighted
	timeout     time.Duration
	wsURL       string
	healthy     bool
}

// New connects to Chrome via CDP at chromeURL.
// poolSize caps concurrent tab usage. timeout is per-request page load.
// Returns an error if Chrome is unreachable after retries.
func New(ctx context.Context, chromeURL string, poolSize int, timeout time.Duration) (*Pool, error) {
	if poolSize <= 0 {
		poolSize = 1
	}

	allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, chromeURL)

	// Verify Chrome DevTools endpoint is responding
	// Chrome 66+ rejects Host headers that are DNS names (not IP or localhost)
	// So we resolve the hostname to IP if needed
	parsedURL, err := url.Parse(chromeURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("invalid Chrome URL: %w", err)
	}

	// Resolve hostname to IP address (avoids Chrome's Host header validation issue)
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "80"
		if parsedURL.Scheme == "https" {
			port = "443"
		}
	}

	var resolvedURL string
	if host == "localhost" || net.ParseIP(host) != nil {
		// Already an IP or localhost, use as-is
		resolvedURL = chromeURL
	} else {
		// Resolve DNS name to IP
		addrs, err := net.LookupHost(host)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to resolve Chrome hostname %s: %w", host, err)
		}
		if len(addrs) == 0 {
			cancel()
			return nil, fmt.Errorf("Chrome hostname %s resolved to no addresses", host)
		}
		// Use first resolved IP
		resolvedURL = fmt.Sprintf("%s://%s:%s/json/version", parsedURL.Scheme, addrs[0], port)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	versionURL := resolvedURL
	if net.ParseIP(host) != nil || host == "localhost" {
		versionURL = chromeURL + "/json/version"
	}

	maxRetries := 5
	retryDelay := 1 * time.Second

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("GET", versionURL, nil)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Printf("Chrome connection attempt %d/%d failed: %v (retrying...)", attempt, maxRetries, err)
				time.Sleep(retryDelay)
			}
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				log.Printf("Chrome connection attempt %d/%d failed: %v (retrying...)", attempt, maxRetries, err)
				time.Sleep(retryDelay)
			}
			continue
		}

		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Printf("Connected to Chrome at %s (attempt %d)", chromeURL, attempt)
			return &Pool{
				allocCtx:    allocCtx,
				allocCancel: cancel,
				sem:         semaphore.NewWeighted(int64(poolSize)),
				timeout:     timeout,
				wsURL:       chromeURL,
				healthy:     true,
			}, nil
		}

		lastErr = fmt.Errorf("Chrome DevTools endpoint returned status %d", resp.StatusCode)
		if attempt < maxRetries {
			log.Printf("Chrome connection attempt %d/%d failed: %v (retrying...)", attempt, maxRetries, lastErr)
			time.Sleep(retryDelay)
		}
	}

	cancel()
	return nil, fmt.Errorf("failed to connect to Chrome at %s after %d retries: %w", chromeURL, maxRetries, lastErr)
}

// Close releases the allocator context and all tabs.
func (p *Pool) Close() error {
	p.allocCancel()
	return nil
}

// IsAlive checks if a Chrome instance is responding without doing retries.
// Returns immediately true/false without waiting.
func IsAlive(chromeURL string) bool {
	testCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(testCtx, chromeURL)
	defer allocCancel()

	testCtx2, cancel2 := chromedp.NewContext(allocCtx)
	defer cancel2()

	err := chromedp.Run(testCtx2)
	return err == nil
}

// RoundTrip implements http.RoundTripper.
// Acquires a tab slot, navigates to req.URL, waits for body, returns rendered HTML.
func (p *Pool) RoundTrip(req *http.Request) (*http.Response, error) {
	// Acquire a semaphore slot
	if err := p.sem.Acquire(req.Context(), 1); err != nil {
		return nil, fmt.Errorf("failed to acquire pool slot: %w", err)
	}
	defer p.sem.Release(1)

	// Create a new tab context with timeout
	tabCtx, cancel := context.WithTimeout(p.allocCtx, p.timeout)
	defer cancel()

	// Create a new chromedp context
	tabCtx, chromedpCancel := chromedp.NewContext(tabCtx)
	defer chromedpCancel()

	var html string
	statusCode := http.StatusOK

	// Navigate to the URL and capture the rendered HTML
	err := chromedp.Run(tabCtx,
		chromedp.Navigate(req.URL.String()),
		chromedp.WaitReady("body"),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp navigation failed for %s: %w", req.URL.String(), err)
	}

	// Create response with piped HTML content
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		pw.Write([]byte(html))
	}()

	resp := &http.Response{
		Status:        http.StatusText(statusCode),
		StatusCode:    statusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          pr,
		ContentLength: int64(len(html)),
		Request:       req,
	}

	resp.Header.Set("Content-Type", "text/html; charset=utf-8")

	return resp, nil
}

// retry attempts a function up to maxRetries times with exponential backoff.
func retry(ctx context.Context, maxRetries int, initialDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for i := 0; i < maxRetries; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < maxRetries-1 {
				select {
				case <-time.After(delay):
					delay *= 2
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("retry failed after %d attempts: %w", maxRetries, lastErr)
}
