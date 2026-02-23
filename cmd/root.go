package cmd

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/rickcrawford/markdowninthemiddle/internal/banner"
	"github.com/rickcrawford/markdowninthemiddle/internal/cache"
	"github.com/rickcrawford/markdowninthemiddle/internal/certs"
	"github.com/rickcrawford/markdowninthemiddle/internal/config"
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
	var tlsCfg *tls.Config
	if cfg.TLS.Enabled {
		cert, err := certs.LoadOrGenerate(
			cfg.TLS.CertFile, cfg.TLS.KeyFile,
			cfg.TLS.AutoCert, cfg.TLS.AutoCertHost, cfg.TLS.AutoCertDir,
		)
		if err != nil {
			return fmt.Errorf("loading TLS certificate: %w", err)
		}
		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		log.Println("TLS enabled on proxy listener")
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
	}

	srv := proxy.New(opts)

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down proxy...")
		srv.Close()
	}()

	log.Printf("starting proxy on %s (TLS: %v, convert: %v, max body: %d bytes)",
		cfg.Proxy.Addr, cfg.TLS.Enabled, cfg.Conversion.Enabled, cfg.MaxBodySize)

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
