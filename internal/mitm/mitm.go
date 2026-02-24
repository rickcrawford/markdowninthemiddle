package mitm

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Manager handles CA and domain certificate generation for MITM interception.
type Manager struct {
	caCert   *tls.Certificate
	caX509   *x509.Certificate
	caKey    *rsa.PrivateKey
	cacheDir string
	cache    map[string]*tls.Certificate
	mu       sync.RWMutex
}

// New creates or loads a CA certificate from disk.
// If cacheDir is empty, certificates are kept in memory only.
func New(cacheDir string) (*Manager, error) {
	m := &Manager{
		cacheDir: cacheDir,
		cache:    make(map[string]*tls.Certificate),
	}

	// Create cache directory if needed
	if cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			return nil, fmt.Errorf("creating cache dir: %w", err)
		}
	}

	// Try to load existing CA
	caCertPath := filepath.Join(cacheDir, "ca-cert.pem")
	caKeyPath := filepath.Join(cacheDir, "ca-key.pem")

	if _, err := os.Stat(caCertPath); err == nil {
		// Load existing CA
		if err := m.loadCA(caCertPath, caKeyPath); err != nil {
			return nil, fmt.Errorf("loading CA: %w", err)
		}
	} else {
		// Generate new CA
		if err := m.generateCA(); err != nil {
			return nil, fmt.Errorf("generating CA: %w", err)
		}

		// Save CA if cache dir provided
		if cacheDir != "" {
			if err := m.saveCA(caCertPath, caKeyPath); err != nil {
				return nil, fmt.Errorf("saving CA: %w", err)
			}
		}
	}

	return m, nil
}

// loadCA loads a CA certificate and key from PEM files.
func (m *Manager) loadCA(certPath, keyPath string) error {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return err
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return err
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return err
	}

	m.caCert = &cert
	m.caX509 = x509Cert

	// Extract private key
	privKey, ok := cert.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return fmt.Errorf("CA private key is not RSA")
	}
	m.caKey = privKey

	return nil
}

// generateCA creates a new self-signed root CA certificate.
func (m *Manager) generateCA() error {
	// Generate RSA key (2048-bit for MITM CA)
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
			CommonName:   "Markdown in the Middle MITM CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	// Self-sign
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	x509Cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return err
	}

	m.caX509 = x509Cert
	m.caKey = key
	m.caCert = &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  key,
		Leaf:        x509Cert,
	}

	return nil
}

// GetCertForDomain returns a TLS certificate for the given domain.
// Certificates are generated on-demand, cached in memory, and optionally persisted to disk.
func (m *Manager) GetCertForDomain(domain string) (*tls.Certificate, error) {
	m.mu.RLock()
	if cert, ok := m.cache[domain]; ok {
		m.mu.RUnlock()
		return cert, nil
	}
	m.mu.RUnlock()

	// Check disk cache
	if m.cacheDir != "" {
		certPath := filepath.Join(m.cacheDir, domain+"-cert.pem")
		keyPath := filepath.Join(m.cacheDir, domain+"-key.pem")

		if _, err := os.Stat(certPath); err == nil {
			certPEM, _ := os.ReadFile(certPath)
			keyPEM, _ := os.ReadFile(keyPath)

			cert, err := tls.X509KeyPair(certPEM, keyPEM)
			if err == nil {
				m.mu.Lock()
				m.cache[domain] = &cert
				m.mu.Unlock()
				return &cert, nil
			}
		}
	}

	// Generate new certificate
	cert, err := m.generateDomainCert(domain)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.cache[domain] = cert
	m.mu.Unlock()

	// Save to disk
	if m.cacheDir != "" {
		certPath := filepath.Join(m.cacheDir, domain+"-cert.pem")
		keyPath := filepath.Join(m.cacheDir, domain+"-key.pem")
		_ = m.saveDomainCert(domain, cert, certPath, keyPath)
	}

	return cert, nil
}

// generateDomainCert creates a new certificate for a domain, signed by the CA.
func (m *Manager) generateDomainCert(domain string) (*tls.Certificate, error) {
	// Generate RSA key (2048-bit for domain certs)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Parse domain (remove port if present)
	host := domain
	if h, _, err := net.SplitHostPort(domain); err == nil {
		host = h
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: host,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	// Add DNS names
	template.DNSNames = append(template.DNSNames, host)
	if host != "" && host[0] != '*' {
		template.DNSNames = append(template.DNSNames, "*."+host)
	}

	// Sign with CA
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

	x509Cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}

	return &tls.Certificate{
		Certificate: [][]byte{certBytes},
		PrivateKey:  key,
		Leaf:        x509Cert,
	}, nil
}

// saveCA saves the CA certificate and key to PEM files.
func (m *Manager) saveCA(certPath, keyPath string) error {
	// Save certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()

	pem.Encode(certOut, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: m.caCert.Certificate[0],
	})

	// Save private key
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(m.caKey)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return nil
}

// saveDomainCert saves a domain certificate and key to PEM files.
func (m *Manager) saveDomainCert(domain string, cert *tls.Certificate, certPath, keyPath string) error {
	// Save certificate
	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer certOut.Close()

	for _, certBytes := range cert.Certificate {
		pem.Encode(certOut, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		})
	}

	// Save private key
	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	privKeyBytes, err := x509.MarshalPKCS8PrivateKey(cert.PrivateKey)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return nil
}

// GetCACert returns the CA certificate for distribution to clients.
func (m *Manager) GetCACert() *tls.Certificate {
	return m.caCert
}

// GetCACertPEM returns the CA certificate in PEM format for exporting to clients.
func (m *Manager) GetCACertPEM() ([]byte, error) {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: m.caCert.Certificate[0],
	}), nil
}

// CACertPath returns the path to the CA certificate file.
func (m *Manager) CACertPath() string {
	return filepath.Join(m.cacheDir, "ca-cert.pem")
}
