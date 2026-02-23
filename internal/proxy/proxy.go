package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/rickcrawford/markdowninthemiddle/internal/cache"
	"github.com/rickcrawford/markdowninthemiddle/internal/middleware"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

// Options configures the proxy server.
type Options struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	TLSConfig *tls.Config

	ConvertHTML bool
	MaxBodySize int64
	TLSInsecure bool

	TokenCounter *tokens.Counter
	Cache        *cache.DiskCache
}

// New creates an *http.Server configured as a forward proxy.
// It uses Chi for routing and middleware, and a custom RoundTripper to
// post-process responses (decompress, convert HTML to Markdown, count tokens).
func New(opts Options) *http.Server {
	r := chi.NewRouter()

	// Chi middleware for the proxy's own request handling.
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	// The response-processing transport wraps the default transport.
	transport := &middleware.ResponseProcessor{
		MaxBodySize:  opts.MaxBodySize,
		ConvertHTML:  opts.ConvertHTML,
		TokenCounter: opts.TokenCounter,
		Cache:        opts.Cache,
		Inner: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: opts.TLSInsecure,
			},
			DisableCompression: false,
			IdleConnTimeout:    90 * time.Second,
		},
	}

	// CONNECT handler for HTTPS tunneling.
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			handleConnect(w, r)
			return
		}
		handleHTTP(w, r, transport)
	})

	srv := &http.Server{
		Addr:         opts.Addr,
		Handler:      r,
		ReadTimeout:  opts.ReadTimeout,
		WriteTimeout: opts.WriteTimeout,
		TLSConfig:    opts.TLSConfig,
	}

	return srv
}

// handleHTTP handles non-CONNECT proxy requests (plain HTTP).
func handleHTTP(w http.ResponseWriter, req *http.Request, transport http.RoundTripper) {
	// Ensure the request URL is absolute.
	if !req.URL.IsAbs() {
		http.Error(w, "request URI must be absolute for proxy", http.StatusBadRequest)
		return
	}

	// Remove hop-by-hop headers.
	req.RequestURI = ""
	removeHopByHop(req.Header)

	resp, err := transport.RoundTrip(req)
	if err != nil {
		log.Printf("proxy roundtrip error: %v", err)
		http.Error(w, "proxy error: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers.
	copyHeaders(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// handleConnect implements the HTTP CONNECT method for HTTPS tunneling.
// This creates a raw TCP tunnel — responses passing through CONNECT tunnels
// are encrypted end-to-end and NOT processed by our middleware.
func handleConnect(w http.ResponseWriter, req *http.Request) {
	destConn, err := net.DialTimeout("tcp", req.Host, 10*time.Second)
	if err != nil {
		http.Error(w, "unable to connect to destination", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		destConn.Close()
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "hijack error", http.StatusInternalServerError)
		destConn.Close()
		return
	}

	go transfer(destConn, clientConn)
	go transfer(clientConn, destConn)
}

func transfer(dst io.WriteCloser, src io.ReadCloser) {
	defer dst.Close()
	defer src.Close()
	io.Copy(dst, src)
}

func removeHopByHop(h http.Header) {
	hopByHop := []string{
		"Connection",
		"Keep-Alive",
		"Proxy-Authenticate",
		"Proxy-Authorization",
		"TE",
		"Trailer",
		"Transfer-Encoding",
		"Upgrade",
	}
	for _, key := range hopByHop {
		h.Del(key)
	}
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
