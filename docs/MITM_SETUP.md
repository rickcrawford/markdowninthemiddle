# MITM Setup Guide - Client Configuration

This guide shows how to set up HTTPS interception on your client machine so you can process HTTPS traffic through the proxy.

## Overview

MITM (Man-in-the-Middle) mode allows the proxy to decrypt HTTPS traffic, process it (convert HTML, count tokens, etc.), then re-encrypt it. This requires clients to trust the proxy's self-signed certificate.

**Important:** Only use MITM mode with trusted proxies in controlled environments (local networks, internal services). The proxy can see and modify all traffic passing through it.

---

## Step 1: Extract the CA Certificate

The proxy generates a self-signed CA (Certificate Authority) on first run. You need to export it to your client.

**From Docker:**
```bash
# Copy CA cert from running container
docker compose exec proxy cat /app/certs/mitm/ca-cert.pem > ~/mitm-ca.pem
```

**From local installation:**
```bash
# CA is stored at:
cat ./certs/mitm/ca-cert.pem > ~/mitm-ca.pem
```

Alternatively, configure the path in `config.yml`:
```yaml
mitm:
  enabled: true
  cert_dir: "./certs/mitm"  # Path to CA certificates
```

---

## Step 2: Trust the CA Certificate

Choose your operating system below:

### macOS

```bash
# Add to System Keychain (requires password)
sudo security add-trusted-cert -d -r trustRoot \
  -k /Library/Keychains/System.keychain ~/mitm-ca.pem

# Verify it's installed
security dump-trust-settings -d | grep "Markdown"

# To remove later:
sudo security delete-certificate -c "Markdown in the Middle MITM CA"
```

**Alternative - Add to browser only (Chrome/Firefox):**

1. Open Chrome → Settings → Privacy and security → Security
2. Scroll to "Manage certificates"
3. Go to "Authorities" tab
4. Click "Import" and select `~/mitm-ca.pem`
5. Check "Trust this certificate for identifying websites"

### Linux (Ubuntu/Debian)

```bash
# Copy to trusted CA store
sudo cp ~/mitm-ca.pem /usr/local/share/ca-certificates/mitm-ca.crt

# Update CA database
sudo update-ca-certificates

# Verify
grep -l "Markdown" /etc/ssl/certs/*

# To remove later:
sudo rm /usr/local/share/ca-certificates/mitm-ca.crt
sudo update-ca-certificates
```

### Linux (Fedora/RHEL)

```bash
# Copy to trusted CA store
sudo cp ~/mitm-ca.pem /etc/pki/ca-trust/source/anchors/mitm-ca.crt

# Update CA database
sudo update-ca-trust

# To remove later:
sudo rm /etc/pki/ca-trust/source/anchors/mitm-ca.crt
sudo update-ca-trust
```

### Windows (PowerShell)

```powershell
# Import to trusted root store (requires admin)
Import-Certificate -FilePath "C:\Users\$env:USERNAME\mitm-ca.pem" `
  -CertStoreLocation "Cert:\LocalMachine\Root"

# Verify
Get-ChildItem Cert:\LocalMachine\Root | Select-String "Markdown"

# To remove later:
$cert = Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -match "Markdown"
Remove-Item $cert.PSPath
```

### Windows (Browser only - Chrome/Edge)

1. Open Chrome → Settings → Privacy and security → Security
2. Scroll to "Manage certificates"
3. Go to "Trusted Root Certification Authorities" tab
4. Right-click → "Import"
5. Select `C:\Users\<username>\mitm-ca.pem`
6. Click "Place all certificates in the following store"
7. Select "Trusted Root Certification Authorities"

---

## Step 3: Test MITM Mode

### Start Proxy with MITM Enabled

```bash
# Docker
./scripts/docker-compose.sh start  # Uses chromedp by default
# Or enable MITM in docker-compose.yml:
# MITM_ENABLED=true

# Local
./markdowninthemiddle --mitm
```

### Test the Connection

```bash
# Basic test (should process HTTPS)
curl -x http://localhost:8080 https://www.example.com

# Check headers
curl -x http://localhost:8080 https://www.example.com -sD - | grep -E "X-Transport|X-Token-Count"

# Full response dump
curl -vx http://localhost:8080 https://www.example.com 2>&1 | head -30
```

### Expected Output

With MITM enabled, you should see:
- `X-Transport: http` or `X-Transport: chrome` (depending on transport)
- `X-Token-Count: <number>` (token count of converted Markdown)
- HTML converted to Markdown in response body

---

## Troubleshooting

### Certificate Verify Failed

**Error:** `curl: (60) SSL certificate problem`

**Cause:** Certificate not properly trusted by your OS/browser

**Solution:**
1. Verify CA was imported: See verification commands above
2. Clear browser certificate cache: Restart browser or clear cache
3. Re-import with correct permissions

### curl Still Says "Untrusted Certificate"

macOS curl uses Apple SecTrust, not system CA store:

```bash
# Use --cacert flag
curl -x http://localhost:8080 https://www.example.com \
  --cacert ~/mitm-ca.pem

# Or use homebrew curl (uses OpenSSL)
brew install curl
/usr/local/opt/curl/bin/curl -x http://localhost:8080 https://www.example.com
```

### "No matching root certificate"

**Cause:** Certificate not installed or wrong path

**Solution:**
```bash
# Verify CA exists and is readable
openssl x509 -in ~/mitm-ca.pem -text -noout

# Re-install following OS-specific steps above
```

### Browser Shows "Not Secure"

**If using Chrome/Firefox:**
- Make sure you imported to **Authorities/Trusted CA** tab, not just "Other"
- Restart the browser completely
- Try incognito/private window

**If using Safari (macOS):**
- Add to **System Keychain** (not login keychain)
- Use "sudo security..." commands from macOS section above

### Certificate PIN Violations

Some apps use **certificate pinning** (validate specific cert fingerprint). These will fail with MITM:

- Mobile apps (iOS, Android) - often pin certificates
- Security-sensitive sites - may pin certs
- Banks - typically pin certificates

**Solution:** Only use MITM for sites/services that don't use certificate pinning

---

## Advanced Usage

### Using a Custom CA

If you want to use an existing CA instead of auto-generated:

```bash
# Copy your CA files
cp your-ca-cert.pem ./certs/mitm/ca-cert.pem
cp your-ca-key.pem ./certs/mitm/ca-key.pem

# Start proxy
./markdowninthemiddle --mitm
```

### Exporting for Distribution

To distribute the CA to multiple machines:

```bash
# Export CA in different formats
# PEM (default)
cat ./certs/mitm/ca-cert.pem

# DER (binary)
openssl x509 -in ./certs/mitm/ca-cert.pem -outform DER -out ca-cert.der

# PKCS12 (for Windows)
openssl pkcs12 -export -in ./certs/mitm/ca-cert.pem \
  -inkey ./certs/mitm/ca-key.pem -out ca-bundle.p12
```

### Monitoring MITM Traffic

With MITM enabled, check logs for decrypted connections:

```bash
# View proxy logs
docker compose logs proxy | grep MITM

# Local logs
./markdowninthemiddle --mitm 2>&1 | grep MITM
```

---

## Security Notes

- **Private Key Security:** The CA private key (`ca-key.pem`) can sign any certificate for any domain. Protect it like a password.
- **Certificate Lifetime:** Generated domain certificates are valid for 24 hours
- **Logging:** With MITM enabled, the proxy can see all HTTPS traffic (URLs, headers, bodies)
- **Trust Carefully:** Only use with trusted proxies; malicious MITM can steal credentials

---

## Removing MITM Setup

### Remove from macOS
```bash
sudo security delete-certificate -c "Markdown in the Middle MITM CA"
rm ~/mitm-ca.pem
```

### Remove from Linux (Ubuntu)
```bash
sudo rm /usr/local/share/ca-certificates/mitm-ca.crt
sudo update-ca-certificates
rm ~/mitm-ca.pem
```

### Remove from Windows
```powershell
$cert = Get-ChildItem Cert:\LocalMachine\Root | Where-Object Subject -match "Markdown"
Remove-Item $cert.PSPath
Remove-Item C:\Users\$env:USERNAME\mitm-ca.pem
```

### Disable in Proxy
```bash
# config.yml
mitm:
  enabled: false

# Or CLI flag: remove --mitm
```

---

## Related Documentation

- **[MITM_IMPLEMENTATION.md](./MITM_IMPLEMENTATION.md)** - Technical details on how MITM works
- **[README.md](./README.md)** - Main documentation
- **[HTTPS_SETUP.md](./HTTPS_SETUP.md)** - Proxy TLS listener setup (different from MITM)
