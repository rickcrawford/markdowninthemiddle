package browser

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
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
// Returns an error if Chrome is unreachable.
func New(ctx context.Context, chromeURL string, poolSize int, timeout time.Duration) (*Pool, error) {
	if poolSize <= 0 {
		poolSize = 1
	}

	allocCtx, cancel := chromedp.NewRemoteAllocator(ctx, chromeURL)

	// Quick check: verify Chrome DevTools endpoint is reachable
	client := &http.Client{Timeout: 2 * time.Second}
	versionURL := chromeURL + "/json/version"

	resp, err := client.Get(versionURL)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to Chrome at %s: %w", chromeURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cancel()
		return nil, fmt.Errorf("Chrome DevTools endpoint returned status %d", resp.StatusCode)
	}

	log.Printf("Connected to Chrome at %s", chromeURL)
	return &Pool{
		allocCtx:    allocCtx,
		allocCancel: cancel,
		sem:         semaphore.NewWeighted(int64(poolSize)),
		timeout:     timeout,
		wsURL:       chromeURL,
		healthy:     true,
	}, nil
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
