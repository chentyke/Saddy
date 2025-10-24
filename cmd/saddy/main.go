// Package main is the entry point for the Saddy reverse proxy server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"saddy/pkg/api"
	"saddy/pkg/cache"
	"saddy/pkg/config"
	"saddy/pkg/https"
	"saddy/pkg/proxy"
	"saddy/pkg/web"
)

const (
	defaultReadHeaderTimeout = 10 * time.Second
)

func main() {
	var configFile = flag.String("config", "config.yaml", "Configuration file path")
	var help = flag.Bool("help", false, "Show help message")
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Starting Saddy with configuration from %s", *configFile)

	// Initialize components
	cacheInstance := initializeCache(cfg)
	tlsInstance := initializeTLS(cfg)

	// Initialize servers
	reverseProxy := proxy.NewReverseProxy(cfg, cacheInstance)
	adminAPI := api.NewAdminAPI(cfg, cacheInstance, tlsInstance)
	adminServer := web.NewAdminServer(adminAPI)

	// Start servers and wait for shutdown
	runServers(cfg, reverseProxy, adminServer, tlsInstance, cacheInstance)
}

func initializeCache(cfg *config.Config) cache.Storage {
	cacheInstance, err := cache.NewCacheStorage(cache.FactoryConfig{
		StorageType:     cfg.Cache.StorageType,
		CacheDir:        cfg.Cache.CacheDir,
		MaxSize:         cfg.Cache.MaxSize,
		DefaultTTL:      cfg.Cache.DefaultTTL,
		CleanupInterval: cfg.Cache.CleanupInterval,
		Persistent:      cfg.Cache.Persistent,
	})
	if err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}

	// Log cache configuration
	if cfg.Cache.Persistent {
		log.Printf("Cache initialized: type=%s, persistent=true, dir=%s",
			cfg.Cache.StorageType, cfg.Cache.CacheDir)
	} else {
		log.Printf("Cache initialized: type=%s, ttl=%ds",
			cfg.Cache.StorageType, cfg.Cache.DefaultTTL)
	}

	return cacheInstance
}

func initializeTLS(cfg *config.Config) *https.AutoTLS {
	if !cfg.Server.AutoHTTPS {
		return nil
	}

	tlsConfig := &https.TLSConfig{
		Email:    cfg.Server.TLS.Email,
		CacheDir: cfg.Server.TLS.CacheDir,
		Staging:  false, // Set to true for development
	}
	tlsInstance := https.NewAutoTLS(tlsConfig)
	log.Printf("Auto HTTPS enabled with email: %s", cfg.Server.TLS.Email)

	// Register domains from proxy rules with SSL enabled
	for _, rule := range cfg.Proxy.Rules {
		if rule.SSL.Enabled {
			log.Printf("Registering domain for HTTPS: %s", rule.Domain)
			if err := tlsInstance.AddDomain(rule.Domain); err != nil {
				log.Printf("Warning: Failed to register domain %s: %v", rule.Domain, err)
			}
		}
	}

	return tlsInstance
}

func runServers(cfg *config.Config, reverseProxy *proxy.ReverseProxy, adminServer *web.AdminServer, tlsInstance *https.AutoTLS, cacheInstance cache.Storage) {
	// Create context for graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start servers in goroutines
	errChan := make(chan error, 2)

	// Start reverse proxy server
	go startReverseProxy(cfg, reverseProxy, tlsInstance, errChan)

	// Start admin server
	go startAdminServer(cfg, adminServer, errChan)

	// Start TLS renewal checker
	if tlsInstance != nil {
		go tlsInstance.CheckRenewals()
	}

	// Wait for interrupt signal or error
	waitForShutdownSignal(errChan, cancel)

	// Graceful shutdown
	shutdownServers(reverseProxy, cacheInstance)
}

func startReverseProxy(cfg *config.Config, reverseProxy *proxy.ReverseProxy, tlsInstance *https.AutoTLS, errChan chan error) {
	if cfg.Server.AutoHTTPS && tlsInstance != nil {
		startHTTPSReverseProxy(cfg, reverseProxy, tlsInstance, errChan)
	} else {
		startHTTPReverseProxy(cfg, reverseProxy, errChan)
	}
}

func startHTTPSReverseProxy(cfg *config.Config, reverseProxy *proxy.ReverseProxy, tlsInstance *https.AutoTLS, errChan chan error) {
	// Start HTTPS server on port 443
	httpsAddr := fmt.Sprintf("%s:443", cfg.Server.Host)
	log.Printf("Starting HTTPS reverse proxy server on %s", httpsAddr)

	httpsServer := &http.Server{
		Addr:              httpsAddr,
		Handler:           reverseProxy.GetEngine(),
		TLSConfig:         tlsInstance.GetTLSConfig(),
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	// Start HTTP challenge server for Let's Encrypt on port 80
	go func() {
		challengeAddr := fmt.Sprintf("%s:80", cfg.Server.Host)
		log.Printf("Starting HTTP challenge server on %s", challengeAddr)
		if err := tlsInstance.StartHTTPChallenge(challengeAddr); err != nil {
			log.Printf("HTTP challenge server error: %v", err)
		}
	}()

	// Also start HTTP redirect server on configured port (if different from 80)
	if cfg.Server.Port != 80 && cfg.Server.Port != 443 {
		go func() {
			httpAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
			log.Printf("Starting HTTP server on %s (for non-HTTPS access)", httpAddr)
			if err := reverseProxy.Start(); err != nil {
				log.Printf("HTTP server error: %v", err)
			}
		}()
	}

	errChan <- httpsServer.ListenAndServeTLS("", "")
}

func startHTTPReverseProxy(cfg *config.Config, reverseProxy *proxy.ReverseProxy, errChan chan error) {
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting HTTP reverse proxy server on %s", addr)
	errChan <- reverseProxy.Start()
}

func startAdminServer(cfg *config.Config, adminServer *web.AdminServer, errChan chan error) {
	adminAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.AdminPort)
	log.Printf("Starting admin server on %s", adminAddr)

	if cfg.WebUI.Enabled {
		log.Printf("Web UI available at http://%s:%d", cfg.Server.Host, cfg.Server.AdminPort)
	}

	errChan <- adminServer.Start(adminAddr)
}

func waitForShutdownSignal(errChan chan error, cancel context.CancelFunc) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Printf("Server error: %v", err)
		cancel()
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
		cancel()
	}
}

func shutdownServers(reverseProxy *proxy.ReverseProxy, cacheInstance cache.Storage) {
	log.Println("Shutting down servers...")

	// Shutdown reverse proxy
	if err := reverseProxy.Stop(); err != nil {
		log.Printf("Error shutting down reverse proxy: %v", err)
	}

	// Shutdown cache
	if cacheInstance != nil {
		cacheInstance.Stop()
	}

	log.Println("Saddy stopped gracefully")
}

func showHelp() {
	fmt.Println(`Saddy - A lightweight reverse proxy with auto HTTPS and CDN caching

Usage:
  saddy [options]

Options:
  -config string
        Configuration file path (default "configs/config.yaml")
  -help
        Show this help message

Configuration:
  The configuration file should be in YAML format. See configs/config.yaml for an example.

Features:
  - Reverse proxy with multiple domain support
  - Automatic HTTPS/TLS certificate management (Let's Encrypt)
  - CDN-like caching with configurable TTL
  - Web-based configuration interface
  - RESTful API for configuration management
  - Graceful shutdown and hot reloading

Web Interface:
  Access the web interface at http://localhost:8081 (default admin port)
  Default credentials: admin / admin123 (change in config)

API:
  RESTful API available at http://localhost:8081/api/v1
  Authentication required (Basic Auth)

Examples:
  saddy                                    # Start with default config
  saddy -config /path/to/config.yaml      # Start with custom config`)
}
