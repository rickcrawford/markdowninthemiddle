package middleware

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/rickcrawford/markdowninthemiddle/internal/cache"
	"github.com/rickcrawford/markdowninthemiddle/internal/converter"
	"github.com/rickcrawford/markdowninthemiddle/internal/output"
	"github.com/rickcrawford/markdowninthemiddle/internal/templates"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

// ResponseProcessor holds the dependencies needed by the response-rewriting
// transport layer.
type ResponseProcessor struct {
	// MaxBodySize is the maximum response body in bytes to process. 0 = unlimited.
	MaxBodySize int64
	// ConvertHTML controls whether HTML responses are converted to Markdown.
	ConvertHTML bool
	// ConvertJSON controls whether JSON responses are converted to Markdown via Mustache.
	ConvertJSON bool
	// NegotiateOnly when true only converts when the client sends Accept: text/markdown.
	NegotiateOnly bool
	// TokenCounter counts tokens on converted markdown responses.
	TokenCounter *tokens.Counter
	// Cache stores HTML responses to disk.
	Cache *cache.DiskCache
	// OutputWriter writes converted Markdown files to a directory.
	OutputWriter *output.Writer
	// TemplateStore holds user-defined Mustache templates for JSON conversion.
	TemplateStore *templates.Store
	// Inner is the actual transport used to make requests.
	Inner http.RoundTripper
}

// wantsMarkdown checks if the request Accept header includes text/markdown.
func wantsMarkdown(req *http.Request) bool {
	accept := req.Header.Get("Accept")
	for _, part := range strings.Split(accept, ",") {
		mediaType := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if strings.EqualFold(mediaType, "text/markdown") {
			return true
		}
	}
	return false
}

// RoundTrip implements http.RoundTripper. It delegates to the inner transport,
// then post-processes the response: decompresses encoded bodies, enforces size
// limits, caches HTML, converts HTML to Markdown, and counts tokens.
// When JSON conversion is enabled, JSON responses are also converted to
// Markdown using Mustache templates (user-defined or auto-generated).
func (rp *ResponseProcessor) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rp.Inner.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	ct := resp.Header.Get("Content-Type")
	isHTML := converter.IsHTMLContentType(ct)
	isJSON := converter.IsJSONContentType(ct)

	if !isHTML && !isJSON {
		return resp, nil
	}

	// Determine whether to convert this response.
	shouldConvertHTML := isHTML && rp.ConvertHTML
	shouldConvertJSON := isJSON && rp.ConvertJSON
	if rp.NegotiateOnly {
		wants := wantsMarkdown(req)
		shouldConvertHTML = isHTML && wants
		shouldConvertJSON = isJSON && wants
	}

	// If neither conversion applies and it's not HTML (which we still decompress), bail early.
	if !isHTML && !shouldConvertJSON {
		return resp, nil
	}

	// Decompress encoded body.
	encoding := resp.Header.Get("Content-Encoding")
	body, err := Decompress(resp.Body, encoding)
	if err != nil {
		log.Printf("decompress error: %v", err)
		return resp, nil
	}

	// Enforce body size limit.
	var reader io.Reader = body
	if rp.MaxBodySize > 0 {
		reader = io.LimitReader(body, rp.MaxBodySize)
	}

	rawBytes, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("reading response body: %v", err)
		return resp, nil
	}

	// Close the original body now that we've consumed it.
	resp.Body.Close()

	rawStr := string(rawBytes)

	// Cache the original HTML if caching is enabled and response is cacheable.
	if isHTML && rp.Cache != nil && cache.IsCacheable(resp) {
		ttl := cache.TTL(resp)
		if err := rp.Cache.Put(req.URL.String(), rawBytes, ttl); err != nil {
			log.Printf("cache put error: %v", err)
		}
	}

	// Convert JSON to Markdown via Mustache templates.
	if shouldConvertJSON {
		// Look up a user-defined template for this URL.
		var tpl string
		if rp.TemplateStore != nil {
			tpl = rp.TemplateStore.Match(req.URL.String())
		}

		md, err := converter.JSONToMarkdown(rawBytes, tpl)
		if err != nil {
			log.Printf("json-to-markdown conversion error: %v", err)
			// Fall through with original JSON.
			resp.Body = io.NopCloser(strings.NewReader(rawStr))
			resp.ContentLength = int64(len(rawStr))
			return resp, nil
		}

		return rp.finalizeMarkdown(resp, req, md), nil
	}

	// Convert HTML to Markdown.
	if shouldConvertHTML {
		md, err := converter.HTMLToMarkdown(rawStr)
		if err != nil {
			log.Printf("html-to-markdown conversion error: %v", err)
			// Fall through with original HTML.
			resp.Body = io.NopCloser(strings.NewReader(rawStr))
			resp.ContentLength = int64(len(rawStr))
			return resp, nil
		}

		return rp.finalizeMarkdown(resp, req, md), nil
	}

	// Not converting — return the decompressed body.
	resp.Body = io.NopCloser(strings.NewReader(rawStr))
	resp.ContentLength = int64(len(rawStr))
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(rawStr)))
	return resp, nil
}

// finalizeMarkdown sets the response body to the converted Markdown, counts
// tokens, writes output, and updates response headers.
func (rp *ResponseProcessor) finalizeMarkdown(resp *http.Response, req *http.Request, md string) *http.Response {
	// Count tokens on the converted Markdown and set header.
	if rp.TokenCounter != nil {
		count := rp.TokenCounter.Count(md)
		resp.Header.Set("X-Token-Count", strconv.Itoa(count))
	}

	// Write converted Markdown to output directory if configured.
	if rp.OutputWriter != nil {
		if err := rp.OutputWriter.Write(req.URL.String(), []byte(md)); err != nil {
			log.Printf("output write error: %v", err)
		}
	}

	// Replace response body with Markdown.
	resp.Body = io.NopCloser(strings.NewReader(md))
	resp.ContentLength = int64(len(md))
	resp.Header.Set("Content-Type", "text/markdown; charset=utf-8")
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(md)))
	// Signal that the response varies based on the Accept header,
	// consistent with Cloudflare's Markdown for Agents approach.
	resp.Header.Set("Vary", "accept")

	return resp
}
