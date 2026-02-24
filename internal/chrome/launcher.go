package chrome

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/chromedp/chromedp"
)

// Launcher manages a headless Chrome process.
type Launcher struct {
	cmd    *exec.Cmd
	port   int
	binary string
}

// New creates a new Chrome launcher for the given port.
func New(port int) *Launcher {
	binary := findChromeBinary()
	return &Launcher{
		port:   port,
		binary: binary,
	}
}

// Start launches Chrome in headless mode with debugging enabled.
// Returns the Chrome URL for chromedp to connect to.
func (l *Launcher) Start() (string, error) {
	if l.binary == "" {
		return "", fmt.Errorf("Chrome/Chromium not found. Install it or start manually:\n" +
			"  macOS: brew install google-chrome\n" +
			"  Linux: sudo apt-get install chromium-browser\n" +
			"  Windows: https://www.google.com/chrome/\n" +
			"  Or use Docker: docker compose up -d")
	}

	log.Printf("Starting Chrome (%s) on port %d...", l.binary, l.port)

	// Build command arguments
	args := []string{
		"--headless",
		"--disable-gpu",
		fmt.Sprintf("--remote-debugging-port=%d", l.port),
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--disable-background-networking",
		"--disable-client-side-phishing-detection",
		"--disable-component-extensions-with-background-pages",
	}

	l.cmd = exec.Command(l.binary, args...)

	// Suppress Chrome's own output (it's verbose)
	l.cmd.Stdout = nil
	l.cmd.Stderr = nil

	// Start the process
	if err := l.cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start Chrome: %w", err)
	}

	log.Printf("Chrome process started (PID: %d)", l.cmd.Process.Pid)

	// Wait for Chrome to be ready (increased timeout for first launch)
	// First launch can take 30-60 seconds on some systems
	url := fmt.Sprintf("http://localhost:%d", l.port)
	if err := waitForChrome(url, 60*time.Second); err != nil {
		l.Stop()
		return "", fmt.Errorf("Chrome failed to start: %w", err)
	}

	log.Printf("Chrome is ready at %s", url)
	return url, nil
}

// Stop terminates the Chrome process.
func (l *Launcher) Stop() error {
	if l.cmd == nil || l.cmd.Process == nil {
		return nil
	}

	log.Printf("Stopping Chrome (PID: %d)...", l.cmd.Process.Pid)

	// Try graceful kill first
	if err := l.cmd.Process.Signal(os.Interrupt); err != nil {
		// If that doesn't work, force kill
		if err := l.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill Chrome: %w", err)
		}
	}

	// Wait for process to exit
	if err := l.cmd.Wait(); err != nil {
		// Process already exited, ignore
		return nil
	}

	return nil
}

// IsRunning checks if the Chrome process is still running.
func (l *Launcher) IsRunning() bool {
	if l.cmd == nil || l.cmd.Process == nil {
		return false
	}

	// Try to get process state
	if runtime.GOOS == "windows" {
		// On Windows, use a different approach
		return l.cmd.ProcessState == nil || !l.cmd.ProcessState.Exited()
	}

	// On Unix, try to send signal 0 (test if process exists)
	return l.cmd.Process.Signal(os.Signal(nil)) == nil
}

// findChromeBinary finds the Chrome executable on the system.
func findChromeBinary() string {
	var candidates []string

	switch runtime.GOOS {
	case "darwin":
		// macOS
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
	case "linux":
		// Linux
		candidates = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		}
	case "windows":
		// Windows
		candidates = []string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files\\Chromium\\Application\\chrome.exe",
		}
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// waitForChrome polls the Chrome debugging endpoint until it's ready.
// It checks both the version endpoint AND tries to open a tab to ensure full readiness.
func waitForChrome(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	endpoint := url + "/json/version"

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	devtoolsReady := false
	attempts := 0

	for {
		attempts++
		if time.Now().After(deadline) {
			if devtoolsReady {
				return fmt.Errorf("Chrome DevTools ready but browser process not responding after %v", timeout)
			}
			return fmt.Errorf("Chrome did not start within %v", timeout)
		}

		// Check if DevTools endpoint is responding
		resp, err := client.Get(endpoint)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// DevTools endpoint is now responding
		if !devtoolsReady {
			devtoolsReady = true
			log.Println("Chrome DevTools endpoint is responding, waiting for browser to initialize...")
		}

		// DevTools is responding, but Chrome might not be fully ready yet.
		// Try to actually connect and create a context to verify browser is ready.
		testCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		allocCtx, allocCancel := chromedp.NewRemoteAllocator(testCtx, url)
		browserCtx, browserCancel := chromedp.NewContext(allocCtx)

		// Try to create a tab - this will fail if browser isn't ready
		err = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			// Just opening the context is enough to verify browser is ready
			return nil
		}))

		browserCancel()
		allocCancel()
		cancel()

		if err == nil {
			// Chrome is fully ready
			return nil
		}

		// Chrome DevTools is responding but browser isn't ready yet
		// This is normal during startup - wait and retry
		if attempts%10 == 0 {
			log.Printf("Still waiting for Chrome browser to initialize... (attempt %d)", attempts)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
