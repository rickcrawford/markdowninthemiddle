package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rickcrawford/markdowninthemiddle/internal/banner"
	"github.com/rickcrawford/markdowninthemiddle/internal/browser"
	"github.com/rickcrawford/markdowninthemiddle/internal/cache"
	"github.com/rickcrawford/markdowninthemiddle/internal/certs"
	"github.com/rickcrawford/markdowninthemiddle/internal/config"
	"github.com/rickcrawford/markdowninthemiddle/internal/filter"
	"github.com/rickcrawford/markdowninthemiddle/internal/mitm"
	"github.com/rickcrawford/markdowninthemiddle/internal/output"
	"github.com/rickcrawford/markdowninthemiddle/internal/proxy"
	"github.com/rickcrawford/markdowninthemiddle/internal/templates"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

var cfgFile string

// rootCmd is the top-level command for the proxy.
var rootCmd = &cobra.Command{
	Use:   "markdowninthemiddle",
	Short: "An HTTPS forward proxy that converts HTML responses to Markdown",
	Long: `Markdown in the Middle is an HTTPS forward proxy that intercepts HTTP
responses with Content-Type text/html, converts them to Markdown, counts
tokens using TikToken, and optionally caches the original HTML to disk.

Configure via config.yml, environment variables (MITM_ prefix), or CLI flags.`,
	RunE: run,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config.yml)")
	rootCmd.Flags().String("addr", "", "proxy listen address (overrides config)")
	rootCmd.Flags().Bool("tls", false, "enable TLS on proxy listener (overrides config)")
	rootCmd.Flags().Bool("auto-cert", false, "auto-generate self-signed certificate (overrides config)")
	rootCmd.Flags().String("cache-dir", "", "cache directory for HTML responses (overrides config)")
	rootCmd.Flags().Int64("max-body-size", 0, "max response body size in bytes (overrides config)")
	rootCmd.Flags().Bool("tls-insecure", false, "skip TLS certificate verification for upstream requests")
	rootCmd.Flags().String("output-dir", "", "directory to write converted Markdown files")
	rootCmd.Flags().Bool("negotiate-only", false, "only convert when client sends Accept: text/markdown")
	rootCmd.Flags().Bool("convert-json", false, "enable JSON-to-Markdown conversion via Mustache templates")
	rootCmd.Flags().String("template-dir", "", "directory containing .mustache template files for JSON conversion")
	rootCmd.Flags().String("transport", "", "transport type: http (standard reverse proxy) or chromedp (headless Chrome rendering)")
	rootCmd.Flags().StringSlice("allow", []string{}, "regex patterns for allowed URLs (repeatable)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	banner.Print()

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// CLI flag overrides.
	if v, _ := cmd.Flags().GetString("addr"); v != "" {
		cfg.Proxy.Addr = v
	}
	if v, _ := cmd.Flags().GetBool("tls"); v {
		cfg.TLS.Enabled = true
	}
	if v, _ := cmd.Flags().GetBool("auto-cert"); v {
		cfg.TLS.AutoCert = true
	}
	if v, _ := cmd.Flags().GetString("cache-dir"); v != "" {
		cfg.Cache.Dir = v
		cfg.Cache.Enabled = true
	}
	if v, _ := cmd.Flags().GetInt64("max-body-size"); v > 0 {
		cfg.MaxBodySize = v
	}
	if v, _ := cmd.Flags().GetBool("tls-insecure"); v {
		cfg.TLS.Insecure = true
	}
	if v, _ := cmd.Flags().GetString("output-dir"); v != "" {
		cfg.Output.Dir = v
		cfg.Output.Enabled = true
	}
	if v, _ := cmd.Flags().GetBool("negotiate-only"); v {
		cfg.Conversion.NegotiateOnly = true
	}
	if v, _ := cmd.Flags().GetBool("convert-json"); v {
		cfg.Conversion.ConvertJSON = true
	}
	if v, _ := cmd.Flags().GetString("template-dir"); v != "" {
		cfg.Conversion.TemplateDir = v
	}
	if v, _ := cmd.Flags().GetString("transport"); v != "" {
		cfg.Transport.Type = v
	}
	if v, _ := cmd.Flags().GetStringSlice("allow"); len(v) > 0 {
		cfg.Filter.Allowed = v
	}

	// Auto-enable MITM if TLS is enabled (no need for separate flag)
	if cfg.TLS.Enabled {
		cfg.MITM.Enabled = true
	}

	// Token counter.
	tokenCounter, err := tokens.NewCounter(cfg.Conversion.TiktokenEncoding)
	if err != nil {
		return fmt.Errorf("initializing token counter: %w", err)
	}

	// Cache.
	var diskCache *cache.DiskCache
	if cfg.Cache.Enabled && cfg.Cache.Dir != "" {
		diskCache, err = cache.New(cfg.Cache.Dir)
		if err != nil {
			return fmt.Errorf("initializing cache: %w", err)
		}
		log.Printf("HTML cache enabled: %s", cfg.Cache.Dir)
	}

	// TLS config for the proxy listener.
	// If both TLS and MITM are enabled, use a unified CA certificate that works for both.
	var tlsCfg *tls.Config
	var sharedCAPath, sharedKeyPath string // Shared certificate for TLS and MITM

	if cfg.TLS.Enabled {
		var cert tls.Certificate
		var err error

		// If MITM is also enabled, use a unified CA certificate for both TLS and MITM
		if cfg.MITM.Enabled && cfg.TLS.CertFile == "" && cfg.TLS.KeyFile == "" && cfg.TLS.AutoCert {
			// Generate a unified CA certificate that works for both TLS and MITM
			certDir := cfg.TLS.AutoCertDir
			sharedCAPath, sharedKeyPath, err = certs.GenerateCA(cfg.TLS.AutoCertHost, certDir)
			if err != nil {
				return fmt.Errorf("generating unified CA certificate: %w", err)
			}
			cert, err = tls.LoadX509KeyPair(sharedCAPath, sharedKeyPath)
			if err != nil {
				return fmt.Errorf("loading unified CA certificate: %w", err)
			}
			log.Println("TLS enabled on proxy listener with unified CA certificate (also used for MITM)")
			log.Println("⚠️  Clients: Trust the CA certificate in " + certDir + " for both TLS and MITM")
		} else {
			// Use separate TLS certificate
			cert, err = certs.LoadOrGenerate(
				cfg.TLS.CertFile, cfg.TLS.KeyFile,
				cfg.TLS.AutoCert, cfg.TLS.AutoCertHost, cfg.TLS.AutoCertDir,
			)
			if err != nil {
				return fmt.Errorf("loading TLS certificate: %w", err)
			}
			log.Println("TLS enabled on proxy listener")
		}

		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	// Markdown output writer.
	var outputWriter *output.Writer
	if cfg.Output.Enabled && cfg.Output.Dir != "" {
		outputWriter, err = output.New(cfg.Output.Dir)
		if err != nil {
			return fmt.Errorf("initializing output writer: %w", err)
		}
		log.Printf("Markdown output enabled: %s", cfg.Output.Dir)
	}

	// Template store for JSON-to-Markdown conversion.
	var templateStore *templates.Store
	if cfg.Conversion.TemplateDir != "" {
		templateStore, err = templates.New(cfg.Conversion.TemplateDir)
		if err != nil {
			return fmt.Errorf("loading templates: %w", err)
		}
		log.Printf("Mustache templates loaded from: %s", cfg.Conversion.TemplateDir)
	}

	if cfg.Conversion.ConvertJSON {
		log.Println("JSON-to-Markdown conversion enabled")
	}

	if cfg.TLS.Insecure {
		log.Println("WARNING: TLS certificate verification disabled for upstream requests")
	}

	// Compile request filter if patterns are specified
	var reqFilter *filter.Filter
	if len(cfg.Filter.Allowed) > 0 {
		reqFilter, err = filter.New(cfg.Filter.Allowed)
		if err != nil {
			return fmt.Errorf("compiling request filter: %w", err)
		}
		log.Printf("Request filter enabled with %d pattern(s)", len(cfg.Filter.Allowed))
	}

	// Initialize MITM manager if enabled
	var mitmMgr *mitm.Manager
	if cfg.MITM.Enabled {
		// If we have a shared CA certificate (from unified TLS+MITM), use that directory
		mitmCertDir := cfg.MITM.CertDir
		if sharedCAPath != "" {
			// Use the TLS certificate directory which now contains the shared CA
			mitmCertDir = cfg.TLS.AutoCertDir
		}

		mitmMgr, err = mitm.New(mitmCertDir)
		if err != nil {
			return fmt.Errorf("initializing MITM: %w", err)
		}
		log.Println("HTTPS MITM interception enabled")
		log.Printf("CA certificate: %s", mitmMgr.CACertPath())
		if sharedCAPath != "" {
			log.Println("✅ Using unified CA certificate (shared with TLS listener)")
		}
		log.Println("⚠️  IMPORTANT: Clients must trust this CA certificate to use MITM mode")
		log.Println("   See MITM_SETUP.md for client setup instructions")
	}

	// Initialize browser pool if chromedp transport is configured
	ctx := context.Background()
	var chromePool http.RoundTripper

	if cfg.Transport.Type == "chromedp" {
		log.Println("chromedp transport enabled. Connecting to Chrome...")
		chromeURL := cfg.Transport.Chromedp.URL
		if chromeURL == "" {
			chromeURL = "http://localhost:9222"
		}

		chromePool, err = browser.New(ctx, chromeURL, cfg.Transport.Chromedp.PoolSize, 30*time.Second)
		if err != nil {
			log.Printf("ERROR: Failed to connect to Chrome at %s: %v", chromeURL, err)
			log.Println("\nTo use chromedp transport, start Chrome with:")
			log.Println("  macOS:   /Applications/Google\\ Chrome.app/Contents/MacOS/Google\\ Chrome --headless --disable-gpu --remote-debugging-port=9222")
			log.Println("  Linux:   chromium-browser --headless --disable-gpu --remote-debugging-port=9222")
			log.Println("  Windows: chrome.exe --headless --disable-gpu --remote-debugging-port=9222")
			log.Println("  Docker:  docker compose up -d")
			return fmt.Errorf("chromedp transport enabled but Chrome is not running at %s", chromeURL)
		}
		log.Printf("✅ chromedp browser pool ready (size: %d, URL: %s)", cfg.Transport.Chromedp.PoolSize, chromeURL)
	}

	transportType := "http"
	if chromePool != nil {
		transportType = "chrome"
	}

	opts := proxy.Options{
		Addr:         cfg.Proxy.Addr,
		ReadTimeout:  cfg.Proxy.ReadTimeout,
		WriteTimeout: cfg.Proxy.WriteTimeout,
		TLSConfig:    tlsCfg,
		ConvertHTML:   cfg.Conversion.Enabled,
		ConvertJSON:   cfg.Conversion.ConvertJSON,
		NegotiateOnly: cfg.Conversion.NegotiateOnly,
		MaxBodySize:   cfg.MaxBodySize,
		TLSInsecure:  cfg.TLS.Insecure,
		TokenCounter: tokenCounter,
		Cache:         diskCache,
		OutputWriter:  outputWriter,
		TemplateStore: templateStore,
		Filter:        reqFilter,
		Transport:     chromePool,
		TransportType: transportType,
		MITM:          mitmMgr,
	}

	srv := proxy.New(opts)

	// Schedule cleanup of browser pool on shutdown
	var browserPoolCleanup func()
	if chromePool != nil {
		if pool, ok := chromePool.(*browser.Pool); ok {
			browserPoolCleanup = func() { pool.Close() }
		}
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("starting proxy on %s (TLS: %v, convert: %v, max body: %d bytes)",
		cfg.Proxy.Addr, cfg.TLS.Enabled, cfg.Conversion.Enabled, cfg.MaxBodySize)

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		<-quit
		log.Println("shutting down proxy...")
		if browserPoolCleanup != nil {
			log.Println("closing browser pool...")
			browserPoolCleanup()
		}
		srv.Close()
	}()

	if cfg.TLS.Enabled {
		// TLS cert/key are already loaded into TLSConfig; use empty strings.
		err = srv.ListenAndServeTLS("", "")
	} else {
		err = srv.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
