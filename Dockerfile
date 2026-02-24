# Build stage
FROM golang:1.24.7 AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (includes gencert subcommand)
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/markdowninthemiddle ./main.go

# Runtime stage
FROM alpine:3.20

# Install utilities
RUN apk --no-cache add ca-certificates curl bash

# Create app directory structure
WORKDIR /app
RUN mkdir -p /etc/markdowninthemiddle /var/log/markdowninthemiddle/output /etc/markdowninthemiddle/certs

# Copy binary from builder
COPY --from=builder /build/markdowninthemiddle /app/markdowninthemiddle

# Create entrypoint script
RUN cat > /app/entrypoint.sh << 'EOF'
#!/bin/bash

echo "Starting Markdown in the Middle..."

# If the first argument looks like a command, pass through to the binary directly
if [ $# -gt 0 ] && { [ "${1:0:1}" != "-" ] || [ "$1" = "--help" ] || [ "$1" = "--version" ]; }; then
    exec /app/markdowninthemiddle "$@"
fi

# Otherwise, run as a proxy server

# Ensure directories exist
mkdir -p /etc/markdowninthemiddle/certs
mkdir -p /var/log/markdowninthemiddle/output

# Generate TLS certificates if they don't exist
if [ ! -f /etc/markdowninthemiddle/certs/cert.pem ] || [ ! -f /etc/markdowninthemiddle/certs/key.pem ]; then
    echo "Generating self-signed TLS certificates..."
    /app/markdowninthemiddle gencert \
        --host "localhost,127.0.0.1,0.0.0.0" \
        --dir /etc/markdowninthemiddle/certs 2>/dev/null || true
fi

# Load config from file if specified
CONFIG_ARG=""
if [ -f "${MITM_CONFIG:-/etc/markdowninthemiddle/config.yml}" ]; then
    CONFIG_ARG="--config ${MITM_CONFIG:-/etc/markdowninthemiddle/config.yml}"
    echo "Using config: ${MITM_CONFIG:-/etc/markdowninthemiddle/config.yml}"
fi

# Start the proxy
echo "Starting proxy server on ${MITM_PROXY_ADDR:-0.0.0.0:8080}..."
exec /app/markdowninthemiddle ${CONFIG_ARG} "$@"
EOF

RUN chmod +x /app/entrypoint.sh

# Environment variables with defaults
ENV MITM_CONFIG="/etc/markdowninthemiddle/config.yml"
ENV MITM_PROXY_ADDR="0.0.0.0:8080"
ENV MITM_CONVERSION_ENABLED="true"
ENV MITM_LOG_LEVEL="info"
ENV MITM_TLS_ENABLED="false"
ENV MITM_TLS_CERT_FILE="/etc/markdowninthemiddle/certs/cert.pem"
ENV MITM_TLS_KEY_FILE="/etc/markdowninthemiddle/certs/key.pem"

# Expose ports
EXPOSE 8080 8443 8080/udp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080 || exit 1

# Run entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]
