package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/rickcrawford/markdowninthemiddle/internal/converter"
	"github.com/rickcrawford/markdowninthemiddle/internal/output"
	"github.com/rickcrawford/markdowninthemiddle/internal/templates"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

// Deps holds dependencies for MCP handlers
type Deps struct {
	HTTPClient    *http.Client
	TokenCounter  *tokens.Counter
	OutputWriter  *output.Writer
	TemplateStore *templates.Store
}

// Handler handles MCP tool calls
type Handler struct {
	httpClient    *http.Client
	tokenCounter  *tokens.Counter
	outputWriter  *output.Writer
	templateStore *templates.Store
}

// New creates an MCP server with registered tools
func New(deps Deps) *server.MCPServer {
	s := server.NewMCPServer(
		"markdowninthemiddle",
		"1.0.0",
	)

	// Register tools
	handler := &Handler{
		httpClient:    deps.HTTPClient,
		tokenCounter:  deps.TokenCounter,
		outputWriter:  deps.OutputWriter,
		templateStore: deps.TemplateStore,
	}

	RegisterTools(s, handler)

	return s
}

// RegisterTools registers fetch_markdown and fetch_raw tools
func RegisterTools(s *server.MCPServer, handler *Handler) {
	// fetch_markdown tool
	s.AddTool(
		mcp.Tool{
			Name:        "fetch_markdown",
			Description: "Fetch a URL and convert to Markdown",
			InputSchema: mcp.ToolInputSchema(mcp.ToolArgumentsSchema{
				Type: "object",
				Properties: map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch",
					},
				},
				Required: []string{"url"},
			}),
		},
		handler.handleFetchMarkdown,
	)

	// fetch_raw tool
	s.AddTool(
		mcp.Tool{
			Name:        "fetch_raw",
			Description: "Fetch a URL and return raw HTML/JSON body",
			InputSchema: mcp.ToolInputSchema(mcp.ToolArgumentsSchema{
				Type: "object",
				Properties: map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch",
					},
				},
				Required: []string{"url"},
			}),
		},
		handler.handleFetchRaw,
	)
}

// handleFetchMarkdown implements the fetch_markdown tool
func (h *Handler) handleFetchMarkdown(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := request.GetString("url", "")
	if url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	// Fetch the content
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error fetching URL: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error reading response: %v", err)), nil
	}

	// Determine content type
	contentType := resp.Header.Get("Content-Type")

	// Convert to markdown
	var markdown string
	switch {
	case isJSON(contentType):
		// Convert JSON to Markdown
		template := ""
		if h.templateStore != nil {
			template = h.templateStore.Match(url)
		}
		md, err := converter.JSONToMarkdown(body, template)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error converting JSON: %v", err)), nil
		}
		markdown = md
	case isHTML(contentType):
		// Convert HTML to Markdown
		md, err := converter.HTMLToMarkdown(string(body))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error converting HTML: %v", err)), nil
		}
		markdown = md
	default:
		// Return as-is
		markdown = string(body)
	}

	// Count tokens if available
	tokenCount := 0
	if h.tokenCounter != nil {
		tokenCount = h.tokenCounter.Count(markdown)
	}

	// Write output if enabled
	if h.outputWriter != nil {
		if err := h.outputWriter.Write(url, []byte(markdown)); err != nil {
			log.Printf("error writing output: %v", err)
		}
	}

	result := map[string]interface{}{
		"url":         url,
		"markdown":    markdown,
		"tokens":      tokenCount,
		"status_code": resp.StatusCode,
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleFetchRaw implements the fetch_raw tool
func (h *Handler) handleFetchRaw(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	url := request.GetString("url", "")
	if url == "" {
		return mcp.NewToolResultError("url is required"), nil
	}

	// Fetch the content
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error fetching URL: %v", err)), nil
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error reading response: %v", err)), nil
	}

	result := map[string]interface{}{
		"url":          url,
		"status_code":  resp.StatusCode,
		"content_type": resp.Header.Get("Content-Type"),
		"body":         string(body),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// isHTML checks if content type is HTML
func isHTML(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "text/html") || strings.Contains(ct, "xhtml")
}

// isJSON checks if content type is JSON
func isJSON(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.Contains(ct, "application/json") || strings.Contains(ct, "ld+json")
}
