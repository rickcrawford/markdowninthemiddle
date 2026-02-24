package proxy

import (
	"bufio"
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"golang.org/x/net/http/httpproxy"

	"github.com/rickcrawford/markdowninthemiddle/internal/cache"
	"github.com/rickcrawford/markdowninthemiddle/internal/filter"
	"github.com/rickcrawford/markdowninthemiddle/internal/middleware"
	"github.com/rickcrawford/markdowninthemiddle/internal/mitm"
	"github.com/rickcrawford/markdowninthemiddle/internal/output"
	"github.com/rickcrawford/markdowninthemiddle/internal/templates"
	"github.com/rickcrawford/markdowninthemiddle/internal/tokens"
)

// Options configures the proxy server.
type Options struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	TLSConfig *tls.Config

	ConvertHTML   bool
	ConvertJSON   bool
	NegotiateOnly bool
	MaxBodySize   int64
	TLSInsecure   bool

	TokenCounter  *tokens.Counter
	Cache         *cache.DiskCache
	OutputWriter  *output.Writer
	TemplateStore *templates.Store
	Filter        *filter.Filter
	Transport     http.RoundTripper
	TransportType string // "http" or "chrome"
	MITM          *mitm.Manager
}

// New creates an *http.Server configured as a forward proxy.
// It uses Chi for routing and middleware, and a custom RoundTripper to
// post-process responses (decompress, convert HTML to Markdown, count tokens).
func New(opts Options) *http.Server {
	r := chi.NewRouter()

	// Chi middleware for the proxy's own request handling.
	r.Use(chimw.RealIP)
	r.Use(middleware.LoggerMiddleware)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)

	// Inject filter middleware if configured
	if opts.Filter != nil {
		r.Use(opts.Filter.Middleware)
	}

	// Select inner transport
	var innerTransport http.RoundTripper
	if opts.Transport != nil {
		innerTransport = opts.Transport
	} else {
		// Create HTTP transport respecting environment proxy variables
		// (HTTP_PROXY, HTTPS_PROXY, NO_PROXY)
		proxyFuncFromEnv := httpproxy.FromEnvironment().ProxyFunc()
		proxyFunc := func(req *http.Request) (*url.URL, error) {
			return proxyFuncFromEnv(req.URL)
		}

		innerTransport = &http.Transport{
			Proxy: proxyFunc,
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: opts.TLSInsecure,
			},
			DisableCompression: false,
			IdleConnTimeout:    90 * time.Second,
		}
	}

	// The response-processing transport wraps the selected transport.
	transport := &middleware.ResponseProcessor{
		MaxBodySize:   opts.MaxBodySize,
		ConvertHTML:   opts.ConvertHTML,
		ConvertJSON:   opts.ConvertJSON,
		NegotiateOnly: opts.NegotiateOnly,
		TokenCounter:  opts.TokenCounter,
		Cache:         opts.Cache,
		OutputWriter:  opts.OutputWriter,
		TemplateStore: opts.TemplateStore,
		Inner:         innerTransport,
		TransportType: opts.TransportType,
	}

	// CONNECT handler for HTTPS tunneling.
	r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			if opts.MITM != nil {
				handleConnectMITM(w, r, opts.MITM, transport)
			} else {
				handleConnect(w, r)
			}
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

// handleConnectMITM implements HTTPS tunneling with MITM interception.
// This decrypts HTTPS traffic, allowing responses to be processed (converted,
// cached, token counted, etc.), then re-encrypts before sending to client.
func handleConnectMITM(w http.ResponseWriter, req *http.Request, mitmMgr *mitm.Manager, transport http.RoundTripper) {
	// Get or generate a certificate for this domain
	cert, err := mitmMgr.GetCertForDomain(req.Host)
	if err != nil {
		log.Printf("MITM cert generation failed for %s: %v", req.Host, err)
		http.Error(w, "certificate generation failed", http.StatusInternalServerError)
		return
	}

	// Accept the CONNECT request
	w.WriteHeader(http.StatusOK)

	// Hijack the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		log.Printf("hijack error: %v", err)
		http.Error(w, "hijack error", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Wrap client connection with TLS (present our cert)
	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})
	defer tlsConn.Close()

	// Read HTTPS requests from the client (decrypted via our cert)
	reader := bufio.NewReader(tlsConn)
	for {
		clientReq, err := http.ReadRequest(reader)
		if err != nil {
			if err != io.EOF {
				log.Printf("MITM read error: %v", err)
			}
			break
		}

		// Rewrite request to upstream
		clientReq.RequestURI = ""
		clientReq.URL.Scheme = "https"
		clientReq.URL.Host = req.Host

		// Remove hop-by-hop headers
		removeHopByHop(clientReq.Header)

		// Send request to upstream using our transport (with full TLS handshake)
		upstreamResp, err := transport.RoundTrip(clientReq)
		if err != nil {
			log.Printf("MITM upstream error: %v", err)
			break
		}

		// Write response back to client (will be encrypted via our TLS)
		upstreamResp.Write(tlsConn)
		upstreamResp.Body.Close()
	}
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
