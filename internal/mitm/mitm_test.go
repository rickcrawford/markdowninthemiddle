package mitm

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_GenerateCA(t *testing.T) {
	m, err := New("")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if m.caCert == nil {
		t.Fatal("CA certificate not generated")
	}

	if m.caX509 == nil {
		t.Fatal("CA X509 certificate not parsed")
	}

	if m.caKey == nil {
		t.Fatal("CA private key not set")
	}
}

func TestNew_LoadCA(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// First, generate and save CA
	m1, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Verify CA files were created
	certPath := filepath.Join(tmpDir, "ca-cert.pem")
	keyPath := filepath.Join(tmpDir, "ca-key.pem")

	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("CA cert file not created: %v", err)
	}
	if _, err := os.Stat(keyPath); err != nil {
		t.Fatalf("CA key file not created: %v", err)
	}

	// Now load the same CA
	m2, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed loading existing CA: %v", err)
	}

	// Verify certificates match
	if m1.caCert.Leaf.SerialNumber.Cmp(m2.caCert.Leaf.SerialNumber) != 0 {
		t.Fatal("Loaded CA does not match saved CA")
	}
}

func TestGetCertForDomain(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	domain := "example.com"
	cert, err := m.GetCertForDomain(domain)
	if err != nil {
		t.Fatalf("GetCertForDomain() failed: %v", err)
	}

	if cert == nil {
		t.Fatal("Certificate is nil")
	}

	if len(cert.Certificate) == 0 {
		t.Fatal("Certificate chain is empty")
	}

	if cert.PrivateKey == nil {
		t.Fatal("Private key is nil")
	}
}

func TestGetCertForDomain_Caching(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	domain := "example.com"

	// Get certificate first time
	cert1, err := m.GetCertForDomain(domain)
	if err != nil {
		t.Fatalf("GetCertForDomain() failed: %v", err)
	}

	// Get certificate second time (should be cached)
	cert2, err := m.GetCertForDomain(domain)
	if err != nil {
		t.Fatalf("GetCertForDomain() failed: %v", err)
	}

	// Verify same certificate returned
	if cert1 != cert2 {
		t.Fatal("Cached certificate not returned")
	}
}

func TestGetCertForDomain_DiskCache(t *testing.T) {
	tmpDir := t.TempDir()

	// First manager: generate and cache cert
	m1, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	domain := "example.com"
	cert1, err := m1.GetCertForDomain(domain)
	if err != nil {
		t.Fatalf("GetCertForDomain() failed: %v", err)
	}

	// Verify cert was saved to disk
	certPath := filepath.Join(tmpDir, domain+"-cert.pem")
	if _, err := os.Stat(certPath); err != nil {
		t.Fatalf("Domain cert not saved to disk: %v", err)
	}

	// Second manager: should load from disk cache
	m2, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	cert2, err := m2.GetCertForDomain(domain)
	if err != nil {
		t.Fatalf("GetCertForDomain() failed: %v", err)
	}

	// Verify same serial number (same cert)
	if cert1.Leaf.SerialNumber.Cmp(cert2.Leaf.SerialNumber) != 0 {
		t.Fatal("Disk cached certificate does not match")
	}
}

func TestGetCertForDomain_WithPort(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	domain := "example.com:443"
	cert, err := m.GetCertForDomain(domain)
	if err != nil {
		t.Fatalf("GetCertForDomain() failed: %v", err)
	}

	if cert == nil {
		t.Fatal("Certificate is nil")
	}
}

func TestGetCACertPEM(t *testing.T) {
	m, err := New("")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	pem, err := m.GetCACertPEM()
	if err != nil {
		t.Fatalf("GetCACertPEM() failed: %v", err)
	}

	if len(pem) == 0 {
		t.Fatal("PEM output is empty")
	}

	// Should contain BEGIN CERTIFICATE marker
	if !contains(string(pem), "BEGIN CERTIFICATE") {
		t.Fatal("PEM does not contain certificate marker")
	}
}

func TestCACertPath(t *testing.T) {
	tmpDir := t.TempDir()
	m, err := New(tmpDir)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	path := m.CACertPath()
	expected := filepath.Join(tmpDir, "ca-cert.pem")
	if path != expected {
		t.Fatalf("CACertPath() = %s, want %s", path, expected)
	}
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
