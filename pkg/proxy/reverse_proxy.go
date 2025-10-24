// Package proxy implements the reverse proxy functionality with caching support.
package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"saddy/pkg/cache"
	"saddy/pkg/config"

	"github.com/gin-gonic/gin"
)

// ReverseProxy manages reverse proxy routing and caching.
type ReverseProxy struct {
	config *config.Config
	cache  cache.Storage
	server *http.Server
	engine *gin.Engine
}

// NewReverseProxy creates a new reverse proxy instance with the given configuration.
func NewReverseProxy(cfg *config.Config, cacheStorage cache.Storage) *ReverseProxy {
	proxy := &ReverseProxy{
		config: cfg,
		cache:  cacheStorage,
		engine: gin.New(),
	}

	proxy.setupRoutes()
	return proxy
}

func (rp *ReverseProxy) setupRoutes() {
	// Middleware
	rp.engine.Use(gin.Logger())
	rp.engine.Use(gin.Recovery())
	rp.engine.Use(rp.corsMiddleware())

	// Health check
	rp.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Proxy routes - must be defined after specific routes
	rp.engine.NoRoute(rp.handleProxy)
}

func (rp *ReverseProxy) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func (rp *ReverseProxy) handleProxy(c *gin.Context) {
	host := c.Request.Host
	// Remove port if present
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	// Find matching proxy rule
	rule := rp.config.GetProxyRule(host)
	if rule == nil {
		c.JSON(404, gin.H{"error": "No proxy rule found for domain: " + host})
		return
	}

	// Check cache if enabled
	if rule.Cache.Enabled && c.Request.Method == "GET" {
		cacheKey := rp.generateCacheKey(c.Request, rule.Domain)
		if cachedItem := rp.cache.GetItem(cacheKey); cachedItem != nil {
			// Restore headers
			for key, value := range cachedItem.Headers {
				c.Header(key, value)
			}
			c.Header("X-Cache", "HIT")
			c.Header("X-Cache-Key", cacheKey)

			// Get Content-Type from cached headers, or use default
			contentType := cachedItem.Headers["Content-Type"]
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			c.Data(cachedItem.StatusCode, contentType, cachedItem.Value)
			return
		}
	}

	// Parse target URL
	targetURL, err := url.Parse(rule.Target)
	if err != nil {
		c.JSON(500, gin.H{"error": "Invalid target URL: " + err.Error()})
		return
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(_ http.ResponseWriter, _ *http.Request, err error) {
		c.JSON(502, gin.H{"error": "Bad Gateway: " + err.Error()})
	}

	// Modify request
	c.Request.URL.Scheme = targetURL.Scheme
	c.Request.URL.Host = targetURL.Host
	c.Request.Host = targetURL.Host

	// Custom director to add headers
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.Host = targetURL.Host
		req.Header.Set("X-Forwarded-Host", c.Request.Host)
		req.Header.Set("X-Forwarded-For", c.ClientIP())
		req.Header.Set("X-Forwarded-Proto", "https")
		req.Header.Set("X-Real-IP", c.ClientIP())
	}

	// Cache response if enabled
	if rule.Cache.Enabled && c.Request.Method == "GET" {
		rp.cacheResponse(c, proxy, rule)
	} else {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (rp *ReverseProxy) cacheResponse(c *gin.Context, proxy *httputil.ReverseProxy, rule *config.ProxyRule) {
	// Intercept response
	writer := &responseWriter{
		ResponseWriter:  c.Writer,
		body:            make([]byte, 0),
		statusCode:      200,
		headers:         make(map[string]string),
		headersCaptured: false,
	}

	proxy.ServeHTTP(writer, c.Request)

	// Cache successful responses
	if writer.statusCode == 200 && len(writer.body) > 0 {
		// Capture headers if not already done
		if !writer.headersCaptured {
			writer.captureHeaders()
		}

		cacheKey := rp.generateCacheKey(c.Request, rule.Domain)
		rp.cache.SetWithHeaders(
			cacheKey,
			writer.body,
			writer.headers,
			writer.statusCode,
			time.Duration(rule.Cache.TTL)*time.Second,
		)
	}
}

func (rp *ReverseProxy) generateCacheKey(req *http.Request, domain string) string {
	// Include query string to differentiate requests like /image?id=1 and /image?id=2
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path = path + "?" + req.URL.RawQuery
	}
	return fmt.Sprintf("%s:%s:%s", domain, req.Method, path)
}

type responseWriter struct {
	http.ResponseWriter
	body            []byte
	headers         map[string]string
	statusCode      int
	headersCaptured bool
}

func (rw *responseWriter) captureHeaders() {
	if rw.headersCaptured {
		return
	}
	// Capture important headers
	for key, values := range rw.ResponseWriter.Header() {
		if len(values) > 0 {
			// Save important headers like Content-Type, Content-Encoding, etc.
			switch key {
			case "Content-Type", "Content-Encoding", "Content-Language", "Cache-Control", "Content-Disposition", "ETag":
				rw.headers[key] = values[0]
			}
		}
	}
	rw.headersCaptured = true
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	// Capture headers before first write
	if !rw.headersCaptured {
		rw.captureHeaders()
	}
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.captureHeaders()
	rw.ResponseWriter.WriteHeader(statusCode)
}

// Start starts the reverse proxy server.
func (rp *ReverseProxy) Start() error {
	rp.server = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", rp.config.Server.Host, rp.config.Server.Port),
		Handler:           rp.engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	return rp.server.ListenAndServe()
}

// GetEngine returns the underlying Gin engine for advanced configuration.
func (rp *ReverseProxy) GetEngine() *gin.Engine {
	return rp.engine
}

// Stop gracefully shuts down the reverse proxy server.
func (rp *ReverseProxy) Stop() error {
	if rp.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return rp.server.Shutdown(ctx)
	}
	return nil
}
