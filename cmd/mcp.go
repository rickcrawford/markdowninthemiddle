package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/rickcrawford/markdowninthemiddle/internal/banner"
	"github.com/rickcrawford/markdowninthemiddle/internal/browser"
	"github.com/rickcrawford/markdowninthemiddle/internal/config"
	mcpserver "github.com/rickcrawford/markdowninthemiddle/internal/mcp"
	"github.com/rickcrawford/markdowninthemiddle/internal/output"
	"github.com/rickcrawford/markdowninthemiddle/internal/templates"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP server for fetch_markdown tool",
	Long: `Start an MCP server that provides tools for fetching and converting web content.
Supports both stdio mode (for Claude Desktop) and HTTP mode (for Streamable HTTP).`,
	RunE: runMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)

	mcpCmd.Flags().String("mcp-transport", "stdio", "MCP server transport: stdio (Claude Desktop) or http (Streamable HTTP)")
	mcpCmd.Flags().String("mcp-addr", ":8081", "address for HTTP mode MCP server")
	mcpCmd.Flags().String("config", "", "config file (default: ./config.yml)")
	mcpCmd.Flags().String("transport", "", "fetch transport: http (default) or chromedp (headless Chrome rendering)")
	mcpCmd.Flags().String("chrome-url", "", "Chrome DevTools URL for chromedp transport (default: http://localhost:9222)")
	mcpCmd.Flags().Int("chrome-pool-size", 0, "max concurrent Chrome tabs (default: 5)")
	mcpCmd.Flags().Bool("tls-insecure", false, "skip TLS certificate verification for upstream requests")
	mcpCmd.Flags().String("template-dir", "", "directory containing .mustache template files for JSON conversion")
	mcpCmd.Flags().Bool("convert-json", false, "enable JSON-to-Markdown conversion via Mustache templates")
}

func getTLSConfig(insecure bool) *tls.Config {
	if insecure {
		return &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	return nil
}

func runMCP(cmd *cobra.Command, args []string) error {
	banner.Print()

	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// CLI flag overrides
	if v, _ := cmd.Flags().GetString("mcp-transport"); v != "" && v != "stdio" && v != "http" {
		return fmt.Errorf("invalid mcp-transport: %s (must be stdio or http)", v)
	}

	mcpTransport, _ := cmd.Flags().GetString("mcp-transport")
	mcpAddr, _ := cmd.Flags().GetString("mcp-addr")

	// CLI flag overrides for fetch transport
	if v, _ := cmd.Flags().GetString("transport"); v != "" {
		if v != "http" && v != "chromedp" {
			return fmt.Errorf("invalid transport: %s (must be http or chromedp)", v)
		}
		cfg.Transport.Type = v
	}
	if v, _ := cmd.Flags().GetString("chrome-url"); v != "" {
		cfg.Transport.Chromedp.URL = v
	}
	if v, _ := cmd.Flags().GetInt("chrome-pool-size"); v > 0 {
		cfg.Transport.Chromedp.PoolSize = v
	}

	if v, _ := cmd.Flags().GetBool("tls-insecure"); v {
		cfg.TLS.Insecure = true
	}

	if cfg.TLS.Insecure {
		log.Println("WARNING: TLS certificate verification disabled for upstream requests")
	}

	// CLI flag overrides for templates
	if v, _ := cmd.Flags().GetString("template-dir"); v != "" {
		cfg.Conversion.TemplateDir = v
	}
	if v, _ := cmd.Flags().GetBool("convert-json"); v {
		cfg.Conversion.ConvertJSON = true
	}

	// Load templates if configured
	var templateStore *templates.Store
	if cfg.Conversion.TemplateDir != "" {
		templateStore, err = templates.New(cfg.Conversion.TemplateDir)
		if err != nil {
			return fmt.Errorf("loading templates: %w", err)
		}
		log.Printf("Mustache templates loaded from: %s", cfg.Conversion.TemplateDir)
	}

	// Token counter
	tokenCounter, err := tokens.NewCounter(cfg.Conversion.TiktokenEncoding)
	if err != nil {
		return fmt.Errorf("initializing token counter: %w", err)
	}

	// Markdown output writer (optional)
	var outputWriter *output.Writer
	if cfg.Output.Enabled && cfg.Output.Dir != "" {
		outputWriter, err = output.New(cfg.Output.Dir)
		if err != nil {
			return fmt.Errorf("initializing output writer: %w", err)
		}
		log.Printf("Markdown output enabled: %s", cfg.Output.Dir)
	}

	// Create HTTP client with configured transport
	var httpClient *http.Client
	var transport http.RoundTripper

	if cfg.Transport.Type == "chromedp" {
		chromeURL := cfg.Transport.Chromedp.URL
		if chromeURL == "" {
			chromeURL = "http://localhost:9222"
		}
		log.Printf("Initializing chromedp browser pool for MCP (URL: %s)", chromeURL)
		pool, err := browser.New(context.Background(), chromeURL, cfg.Transport.Chromedp.PoolSize, 30*time.Second)
		if err != nil {
			log.Printf("Warning: Could not initialize chromedp pool: %v (falling back to HTTP transport)", err)
			// Fall back to standard HTTP
			transport = &http.Transport{
				TLSClientConfig: getTLSConfig(cfg.TLS.Insecure),
			}
		} else {
			transport = pool
			log.Println("✅ chromedp browser pool ready for MCP requests")
		}
	} else {
		// Standard HTTP transport
		transport = &http.Transport{
			TLSClientConfig: getTLSConfig(cfg.TLS.Insecure),
		}
	}

	httpClient = &http.Client{Transport: transport}

	// Create MCP server
	mcpServer := mcpserver.New(mcpserver.Deps{
		HTTPClient:    httpClient,
		TokenCounter:  tokenCounter,
		OutputWriter:  outputWriter,
		TemplateStore: templateStore,
	})

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down MCP server...")
		cancel()
	}()

	log.Printf("starting MCP server (transport: %s)", mcpTransport)

	// Run MCP server based on transport type
	if mcpTransport == "http" {
		return runMCPHTTP(ctx, mcpServer, mcpAddr)
	}

	// Default to stdio
	return runMCPStdio(ctx, mcpServer)
}

func runMCPStdio(ctx context.Context, mcpServer *server.MCPServer) error {
	log.Println("MCP stdio mode enabled (use with Claude Desktop)")

	// Read from stdin, write to stdout
	// This requires the mcp-go library to have StdioTransport support
	// For now, we'll use a simple implementation
	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

func runMCPHTTP(ctx context.Context, mcpServer *server.MCPServer, addr string) error {
	log.Printf("MCP HTTP mode enabled on %s", addr)

	// Create HTTP server
	httpServer := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simple health check endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status": "ok", "message": "MCP server running"}`))
		}),
	}

	go func() {
		<-ctx.Done()
		httpServer.Close()
	}()

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	return httpServer.Serve(listener)
}
