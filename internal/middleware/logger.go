package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// LoggerMiddleware provides proper logging for forward proxy requests.
// It replaces chi's default logger to correctly format URLs for proxy traffic.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer to capture status code and size
		wrapped := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Record the start time
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		// Get request details
		method := r.Method
		path := r.RequestURI
		if path == "" {
			path = r.URL.String()
		}
		proto := r.Proto
		status := wrapped.Status()
		statusStr := ""
		if status > 0 {
			statusStr = http.StatusText(status)
		}
		bytes := wrapped.BytesWritten()
		elapsed := time.Since(start)
		remoteAddr := r.RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			remoteAddr = xff
		}

		// Log in a format similar to HTTP access logs
		log.Printf("%s %d %s %s %s %dB",
			method+" "+path+" "+proto,
			status,
			statusStr,
			remoteAddr,
			elapsed.String(),
			bytes,
		)
	})
}
