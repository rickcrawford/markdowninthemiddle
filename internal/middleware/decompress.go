package middleware

import (
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
)

// Decompress returns a reader that decodes the body based on Content-Encoding.
// The caller is responsible for closing the returned reader if it implements
// io.Closer.
func Decompress(body io.Reader, encoding string) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip":
		return gzip.NewReader(body)
	case "deflate":
		return flate.NewReader(body), nil
	case "identity", "":
		return body, nil
	default:
		return nil, fmt.Errorf("unsupported content-encoding: %s", encoding)
	}
}
