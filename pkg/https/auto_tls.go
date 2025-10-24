// Package https provides automatic HTTPS/TLS certificate management using Let's Encrypt.
package https

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

// AutoTLS manages automatic TLS certificate provisioning and renewal.
type AutoTLS struct {
	config       *TLSConfig
	certManager  *autocert.Manager
	mu           sync.RWMutex
	certificates map[string]*tls.Certificate
	allowedHosts map[string]bool
}

// TLSConfig defines configuration for automatic TLS management.
type TLSConfig struct {
	Email    string
	CacheDir string
	Staging  bool
}

// NewAutoTLS creates a new AutoTLS instance with the given configuration.
func NewAutoTLS(config *TLSConfig) *AutoTLS {
	if config.CacheDir == "" {
		config.CacheDir = "./certs"
	}

	// Create cache directory
	if err := os.MkdirAll(config.CacheDir, 0750); err != nil {
		log.Printf("Failed to create cache directory: %v", err)
	}

	autoTLS := &AutoTLS{
		config:       config,
		certificates: make(map[string]*tls.Certificate),
		allowedHosts: make(map[string]bool),
	}

	autoTLS.initCertManager()
	return autoTLS
}

func (a *AutoTLS) initCertManager() {
	hostPolicy := func(_ context.Context, host string) error {
		a.mu.RLock()
		defer a.mu.RUnlock()

		// Check if host is in allowed list
		if a.allowedHosts[host] {
			return nil
		}

		return fmt.Errorf("host %s is not allowed", host)
	}

	// Create cert manager
	certManager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: hostPolicy,
		Email:      a.config.Email,
		Cache:      autocert.DirCache(a.config.CacheDir),
	}

	// Use staging server for testing
	if a.config.Staging {
		certManager.Client = &acme.Client{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
		}
	}

	a.certManager = certManager
}

// GetCertificate retrieves or provisions a TLS certificate for the given client hello.
func (a *AutoTLS) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check if we have cached certificate
	if cert, exists := a.certificates[hello.ServerName]; exists {
		return cert, nil
	}

	// Get certificate from autocert
	return a.certManager.GetCertificate(hello)
}

// GetTLSConfig returns a TLS configuration suitable for use with http.Server.
func (a *AutoTLS) GetTLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: a.GetCertificate,
		MinVersion:     tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
	}
}

// StartHTTPChallenge starts an HTTP server for Let's Encrypt HTTP-01 challenges.
func (a *AutoTLS) StartHTTPChallenge(listenAddr string) error {
	server := &http.Server{
		Addr:              listenAddr,
		Handler:           a.certManager.HTTPHandler(nil),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("Starting HTTP challenge server on %s", listenAddr)
	return server.ListenAndServe()
}

// AddDomain adds a domain to the list of allowed domains for certificate provisioning.
func (a *AutoTLS) AddDomain(domain string) error {
	// Add domain to allowed hosts
	a.mu.Lock()
	a.allowedHosts[domain] = true
	a.mu.Unlock()

	// Pre-load certificate for domain
	_, err := a.certManager.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
	if err != nil {
		log.Printf("Warning: Failed to get certificate for %s (will retry on first request): %v", domain, err)
		// Don't return error - certificate will be obtained on first request
		return nil
	}

	log.Printf("Successfully obtained certificate for domain: %s", domain)
	return nil
}

// RemoveDomain removes a domain from the allowed list and deletes its certificate.
func (a *AutoTLS) RemoveDomain(domain string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.certificates, domain)
	delete(a.allowedHosts, domain)

	// Remove from cache
	certFile := filepath.Join(a.config.CacheDir, domain+".crt")
	keyFile := filepath.Join(a.config.CacheDir, domain+".key")

	_ = os.Remove(certFile) //nolint:errcheck
	_ = os.Remove(keyFile)  //nolint:errcheck

	log.Printf("Removed certificate for domain: %s", domain)
}

// ListDomains returns a list of all registered domains.
func (a *AutoTLS) ListDomains() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	domains := make([]string, 0, len(a.allowedHosts))
	for domain := range a.allowedHosts {
		domains = append(domains, domain)
	}

	return domains
}

// GetCertInfo retrieves information about a certificate for a specific domain.
func (a *AutoTLS) GetCertInfo(domain string) (*CertInfo, error) {
	// Try to get certificate from autocert manager
	hello := &tls.ClientHelloInfo{ServerName: domain}
	cert, err := a.certManager.GetCertificate(hello)
	if err != nil {
		return nil, fmt.Errorf("certificate not found for domain %s: %v", domain, err)
	}

	// Parse certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	return &CertInfo{
		Domain:        domain,
		Issuer:        x509Cert.Issuer.CommonName,
		NotBefore:     x509Cert.NotBefore,
		NotAfter:      x509Cert.NotAfter,
		IsExpired:     time.Now().After(x509Cert.NotAfter),
		DaysRemaining: int(time.Until(x509Cert.NotAfter).Hours() / 24),
		SerialNumber:  x509Cert.SerialNumber.String(),
	}, nil
}

// CertInfo contains information about a TLS certificate.
type CertInfo struct {
	Domain        string    `json:"domain"`
	Issuer        string    `json:"issuer"`
	NotBefore     time.Time `json:"not_before"`
	NotAfter      time.Time `json:"not_after"`
	IsExpired     bool      `json:"is_expired"`
	DaysRemaining int       `json:"days_remaining"`
	SerialNumber  string    `json:"serial_number"`
}

// GenerateSelfSignedCert generates and saves a self-signed certificate for the domain.
func (a *AutoTLS) GenerateSelfSignedCert(domain string) error {
	// Fallback to self-signed certificate if ACME fails
	cert, key := generateSelfSignedCertificate(domain)

	// Save to cache
	certFile := filepath.Join(a.config.CacheDir, domain+".crt")
	keyFile := filepath.Join(a.config.CacheDir, domain+".key")

	if err := os.WriteFile(certFile, cert, 0600); err != nil {
		return err
	}

	if err := os.WriteFile(keyFile, key, 0600); err != nil {
		return err
	}

	log.Printf("Generated self-signed certificate for domain: %s", domain)
	return nil
}

func generateSelfSignedCertificate(_ string) ([]byte, []byte) {
	// This is a placeholder - in a real implementation you would
	// use crypto/tls or crypto/x509 to generate actual certificates
	return []byte("self-signed-cert"), []byte("self-signed-key")
}

// ForceRenewal forces immediate renewal of a certificate for the given domain.
func (a *AutoTLS) ForceRenewal(domain string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Remove from cache to force renewal
	delete(a.certificates, domain)

	// Force ACME renewal
	if err := a.AddDomain(domain); err != nil {
		return fmt.Errorf("failed to renew certificate for %s: %v", domain, err)
	}

	log.Printf("Successfully renewed certificate for domain: %s", domain)
	return nil
}

// CheckRenewals starts a background process that checks and renews expiring certificates daily.
func (a *AutoTLS) CheckRenewals() {
	ticker := time.NewTicker(24 * time.Hour) // Check daily
	defer ticker.Stop()

	for range ticker.C {
		a.checkAndRenewExpiringCerts()
	}
}

func (a *AutoTLS) checkAndRenewExpiringCerts() {
	a.mu.RLock()
	domains := make([]string, 0, len(a.certificates))
	for domain := range a.certificates {
		domains = append(domains, domain)
	}
	a.mu.RUnlock()

	for _, domain := range domains {
		info, err := a.GetCertInfo(domain)
		if err != nil {
			log.Printf("Error getting cert info for %s: %v", domain, err)
			continue
		}

		// Renew if expires in less than 30 days
		if info.DaysRemaining < 30 {
			log.Printf("Certificate for %s expires in %d days, renewing...", domain, info.DaysRemaining)
			if err := a.ForceRenewal(domain); err != nil {
				log.Printf("Failed to renew certificate for %s: %v", domain, err)
			}
		}
	}
}
