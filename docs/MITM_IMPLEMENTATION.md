# MITM (Man-in-the-Middle) Implementation Guide

This guide explains how to implement HTTPS interception with self-signed certificates to process CONNECT tunnel responses.

## Overview

**Current behavior:** CONNECT tunnels pass through unprocessed (can't convert HTML/count tokens)
**Goal:** Decrypt HTTPS traffic, process it, then re-encrypt

## How It Works

```
Standard CONNECT (no processing):
Client --[TLS]---> Proxy --[TLS]---> Server
                  (relay only)

With MITM (with processing):
Client --[TLS]---> Proxy (decrypts) --[TLS]---> Server
        (trusts proxy cert)  ↓ process  ↓ re-encrypt
                        [Convert HTML]
                        [Count tokens]
```

## Architecture

1. **CA Certificate** - Self-signed root CA, client must trust it
2. **Domain Certificates** - Generated on-demand, signed by CA
3. **TLS Interception** - Proxy presents domain cert to client, presents its own cert to server
4. **Request Processing** - Full access to plaintext requests/responses

## Implementation Steps

### Step 1: Create MITM Certificate Manager

**File:** `internal/mitm/mitm.go`

```go
package mitm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// Manager handles CA and domain certificate generation
type Manager struct {
	caCert   *tls.Certificate
	caKey    *rsa.PrivateKey
	caX509   *x509.Certificate
	cacheDir string
	cache    map[string]*tls.Certificate
}

// New creates or loads a CA certificate
func New(cacheDir string) (*Manager, error) {
	m := &Manager{
		cacheDir: cacheDir,
		cache:    make(map[string]*tls.Certificate),
	}

	// Create cache directory if needed
	if cacheDir != "" {
		os.MkdirAll(cacheDir, 0700)
	}

	// Try to load existing CA
	caCertPath := filepath.Join(cacheDir, "ca-cert.pem")
	caKeyPath := filepath.Join(cacheDir, "ca-key.pem")

	if _, err := os.Stat(caCertPath); err == nil {
		// Load existing CA
		certPEM, _ := os.ReadFile(caCertPath)
		keyPEM, _ := os.ReadFile(caKeyPath)

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("loading CA: %w", err)
		}

		x509Cert, _ := x509.ParseCertificate(cert.Certificate[0])
		m.caCert = &cert
		m.caX509 = x509Cert

		// Extract private key
		m.caKey = cert.PrivateKey.(*rsa.PrivateKey)
	} else {
		// Generate new CA
		if err := m.generateCA(); err != nil {
			return nil, err
		}

		// Save CA
		if cacheDir != "" {
			m.saveCA(caCertPath, caKeyPath)
		}
	}

	return m, nil
}

// generateCA creates a self-signed root CA
func (m *Manager) generateCA() error {
	// Generate RSA key
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Country:      []string{"US"},
			Organization: []string{"Markdown in the Middle"},
			CommonName:   "MITM CA",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(10, 0, 0),
		IsCA:      true,
		BasicConstraintsValid: true,
		MaxPathLen: 0,
		KeyUsage:   x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	// Self-sign
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	x509Cert, _ := x509.ParseCertificate(certBytes)
	m.caX509 = x509Cert
	m.caKey = key
	m.caCert = &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  key,
	}

	return nil
}

// GetCertForDomain returns a certificate for the given domain
func (m *Manager) GetCertForDomain(domain string) (*tls.Certificate, error) {
	// Check cache
	if cert, ok := m.cache[domain]; ok {
		return cert, nil
	}

	// Check disk cache
	certPath := filepath.Join(m.cacheDir, domain+"-cert.pem")
	keyPath := filepath.Join(m.cacheDir, domain+"-key.pem")

	if _, err := os.Stat(certPath); err == nil {
		certPEM, _ := os.ReadFile(certPath)
		keyPEM, _ := os.ReadFile(keyPath)

		cert, _ := tls.X509KeyPair(certPEM, keyPEM)
		m.cache[domain] = &cert
		return &cert, nil
	}

	// Generate new certificate
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().Unix()),
		Subject: pkix.Name{
			CommonName: domain,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		DNSNames: []string{domain, "*." + domain},
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		m.caX509,
		&key.PublicKey,
		m.caKey,
	)
	if err != nil {
		return nil, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  key,
	}

	m.cache[domain] = &cert

	// Save to disk
	if m.cacheDir != "" {
		m.saveCert(domain, certBytes, key)
	}

	return &cert, nil
}

// saveCA saves the CA certificate and key to disk
func (m *Manager) saveCA(certPath, keyPath string) error {
	// Save cert
	certFile, _ := os.Create(certPath)
	defer certFile.Close()
	// Write PEM encoded cert...

	// Save key
	keyFile, _ := os.Create(keyPath)
	defer keyFile.Close()
	// Write PEM encoded key...

	return nil
}

// saveCert saves a domain certificate to disk
func (m *Manager) saveCert(domain string, certBytes []byte, key *rsa.PrivateKey) error {
	// Similar to saveCA...
	return nil
}

// GetCA returns the CA certificate for distribution to clients
func (m *Manager) GetCA() *tls.Certificate {
	return m.caCert
}
```

### Step 2: Modify CONNECT Handler

**File:** `internal/proxy/proxy.go`

```go
func handleConnect(w http.ResponseWriter, req *http.Request, mitm *mitm.Manager) {
	// Get certificate for domain
	domain := req.Host
	cert, err := mitm.GetCertForDomain(domain)
	if err != nil {
		http.Error(w, "cert generation failed", http.StatusInternalServerError)
		return
	}

	// Accept CONNECT request
	w.WriteHeader(http.StatusOK)

	// Hijack connection
	hijacker := w.(http.Hijacker)
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		return
	}
	defer clientConn.Close()

	// Start TLS with client (present our cert)
	tlsConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})
	defer tlsConn.Close()

	// Read HTTPS requests from client
	reader := bufio.NewReader(tlsConn)
	for {
		clientReq, err := http.ReadRequest(reader)
		if err != nil {
			break
		}

		// Connect to upstream (as normal TLS)
		upstreamConn, err := tls.Dial("tcp", domain, nil)
		if err != nil {
			break
		}

		// Forward request
		clientReq.Write(upstreamConn)

		// Read response
		upstreamReader := bufio.NewReader(upstreamConn)
		resp, _ := http.ReadResponse(upstreamReader, clientReq)

		// NOW WE CAN PROCESS THE RESPONSE!
		// - Convert HTML to Markdown
		// - Count tokens
		// - Add X-Transport header
		// - Write to output files
		// - Cache responses

		// Write response back to client
		resp.Write(tlsConn)
		upstreamConn.Close()
	}
}
```

### Step 3: Update Config

**File:** `config.yml`

```yaml
mitm:
  enabled: false              # Set to true to enable MITM
  ca_cert_dir: "./certs/mitm"  # Where to store CA and domain certs
```

**File:** `cmd/root.go`

Add flag:
```go
rootCmd.Flags().Bool("mitm", false, "enable HTTPS MITM interception")
```

### Step 4: Client Setup - Trust the CA

#### macOS

```bash
# Extract CA cert from proxy
openssl s_client -connect localhost:8080 -showcerts </dev/null | \
  openssl x509 -outform PEM > ~/mitm-ca.pem

# Add to Keychain
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ~/mitm-ca.pem

# Verify
security dump-trust-settings -d | grep "MITM CA"
```

#### Linux

```bash
# Copy CA cert to trusted store
sudo cp ~/mitm-ca.pem /usr/local/share/ca-certificates/mitm-ca.crt
sudo update-ca-certificates
```

#### Windows (PowerShell)

```powershell
# Import CA cert to trusted root store
Import-Certificate -FilePath "C:\mitm-ca.pem" `
  -CertStoreLocation "Cert:\LocalMachine\Root"
```

### Step 5: Usage

```bash
# Start proxy with MITM enabled
./markdowninthemiddle --mitm

# Now HTTPS traffic is processed!
curl -x http://localhost:8080 https://www.example.com

# Should see X-Transport and X-Token-Count headers
curl -x http://localhost:8080 https://www.example.com -sD - | grep X-
```

## Limitations & Considerations

1. **Certificate Pinning** - Sites with certificate pinning will fail (they validate the cert fingerprint)
2. **Certificate Warnings** - Users will see "untrusted certificate" until they trust the CA
3. **Browser Issues** - Some browsers cache certificate trust decisions
4. **Performance** - TLS encryption/decryption adds overhead
5. **Logging** - You can now log all HTTPS traffic (privacy consideration)

## Security Notes

- **CA Private Key** - Keep safe! Anyone with it can impersonate any domain
- **Certificate Caching** - Reuse domain certs to avoid regenerating each request
- **Cleanup** - Generated certificates are short-lived (24 hours)
- **Transparent** - Always log that MITM is enabled

## Testing

```bash
# Test standard HTTPS site
curl -x http://localhost:8080 https://www.example.com -sD - | head -20

# Test site with JavaScript (should now render)
curl -x http://localhost:8080 https://www.soapbucket.com -sD - | grep -A 5 "<!DOCTYPE"

# Check headers
curl -x http://localhost:8080 https://www.example.com -sD - | grep "X-"
```

## Troubleshooting

**"certificate verify failed"** - Client doesn't trust CA, follow client setup steps

**"cannot write to certificate directory"** - Ensure certs/mitm directory is writable

**"certificate pinning error"** - Site validates cert fingerprint, MITM won't work for that site

**Performance slow** - MITM + HTML conversion is slower; consider disabling for large responses
