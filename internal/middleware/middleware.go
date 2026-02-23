package middleware

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/rickcrawford/markdowninthemiddle/internal/cache"
	"github.com/rickcrawford/markdowninthemiddle/internal/converter"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

// ResponseProcessor holds the dependencies needed by the response-rewriting
// transport layer.
type ResponseProcessor struct {
	// MaxBodySize is the maximum response body in bytes to process. 0 = unlimited.
	MaxBodySize int64
	// ConvertHTML controls whether HTML responses are converted to Markdown.
	ConvertHTML bool
	// TokenCounter counts tokens on converted markdown responses.
	TokenCounter *tokens.Counter
	// Cache stores HTML responses to disk.
	Cache *cache.DiskCache
	// Inner is the actual transport used to make requests.
	Inner http.RoundTripper
}

// RoundTrip implements http.RoundTripper. It delegates to the inner transport,
// then post-processes the response: decompresses encoded bodies, enforces size
// limits, caches HTML, converts HTML to Markdown, and counts tokens.
func (rp *ResponseProcessor) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rp.Inner.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	ct := resp.Header.Get("Content-Type")
	if !converter.IsHTMLContentType(ct) {
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

	htmlBytes, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("reading response body: %v", err)
		return resp, nil
	}

	// Close the original body now that we've consumed it.
	resp.Body.Close()

	htmlStr := string(htmlBytes)

	// Cache the original HTML if caching is enabled and response is cacheable.
	if rp.Cache != nil && cache.IsCacheable(resp) {
		ttl := cache.TTL(resp)
		if err := rp.Cache.Put(req.URL.String(), htmlBytes, ttl); err != nil {
			log.Printf("cache put error: %v", err)
		}
	}

	// Convert HTML to Markdown.
	if rp.ConvertHTML {
		md, err := converter.HTMLToMarkdown(htmlStr)
		if err != nil {
			log.Printf("html-to-markdown conversion error: %v", err)
			// Fall through with original HTML.
			resp.Body = io.NopCloser(strings.NewReader(htmlStr))
			resp.ContentLength = int64(len(htmlStr))
			return resp, nil
		}

		// Count tokens on the converted Markdown and set header.
		if rp.TokenCounter != nil {
			count := rp.TokenCounter.Count(md)
			resp.Header.Set("X-Token-Count", strconv.Itoa(count))
		}

		// Replace response body with Markdown.
		resp.Body = io.NopCloser(strings.NewReader(md))
		resp.ContentLength = int64(len(md))
		resp.Header.Set("Content-Type", "text/markdown; charset=utf-8")
		resp.Header.Del("Content-Encoding")
		resp.Header.Set("Content-Length", strconv.Itoa(len(md)))

		return resp, nil
	}

	// Not converting — return the decompressed HTML.
	resp.Body = io.NopCloser(strings.NewReader(htmlStr))
	resp.ContentLength = int64(len(htmlStr))
	resp.Header.Del("Content-Encoding")
	resp.Header.Set("Content-Length", strconv.Itoa(len(htmlStr)))
	return resp, nil
}
