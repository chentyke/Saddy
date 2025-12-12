// Package api provides RESTful API endpoints for managing Saddy configuration and monitoring.
package api

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"saddy/pkg/cache"
	"saddy/pkg/config"
	"saddy/pkg/https"
	"saddy/pkg/stats"

	"github.com/gin-gonic/gin"
)

// AdminAPI provides administrative API endpoints for configuration and monitoring.
type AdminAPI struct {
	config *config.Config
	cache  cache.Storage
	tls    *https.AutoTLS
	stats  *stats.TrafficTracker
}

// NewAdminAPI creates a new AdminAPI instance with the given configuration and services.
func NewAdminAPI(cfg *config.Config, cacheStorage cache.Storage, tls *https.AutoTLS, tracker *stats.TrafficTracker) *AdminAPI {
	return &AdminAPI{
		config: cfg,
		cache:  cacheStorage,
		tls:    tls,
		stats:  tracker,
	}
}

// SetupRoutes configures all API routes under the given router group.
func (a *AdminAPI) SetupRoutes(router *gin.RouterGroup) {
	// Check if web UI is enabled and has valid credentials
	if !a.config.WebUI.Enabled || a.config.WebUI.Username == "" || a.config.WebUI.Password == "" {
		// If no valid auth, skip authentication
		return
	}

	// Authentication middleware
	auth := gin.BasicAuth(gin.Accounts{
		a.config.WebUI.Username: a.config.WebUI.Password,
	})

	// Configuration endpoints
	configGroup := router.Group("/config")
	configGroup.Use(auth)
	{
		configGroup.GET("/", a.getConfig)
		configGroup.PUT("/", a.updateConfig)
		configGroup.GET("/proxy", a.getProxyRules)
		configGroup.POST("/proxy", a.addProxyRule)
		configGroup.PUT("/proxy/:domain", a.updateProxyRule)
		configGroup.DELETE("/proxy/:domain", a.deleteProxyRule)
	}

	// Cache endpoints
	cacheGroup := router.Group("/cache")
	cacheGroup.Use(auth)
	{
		cacheGroup.GET("/stats", a.getCacheStats)
		cacheGroup.DELETE("/", a.clearCache)
		cacheGroup.POST("/invalidate", a.invalidateCacheByURL)
		cacheGroup.DELETE("/:key", a.deleteCacheKey)
	}

	// TLS/SSL endpoints
	tlsGroup := router.Group("/tls")
	tlsGroup.Use(auth)
	{
		tlsGroup.GET("/domains", a.getTLSDomains)
		tlsGroup.GET("/domains/:domain", a.getTLSCertInfo)
		tlsGroup.GET("/domains/:domain/check", a.checkDomainStatus)
		tlsGroup.POST("/domains/:domain/renew", a.renewTLSDomain)
		tlsGroup.POST("/domains/:domain", a.addTLSDomain)
		tlsGroup.DELETE("/domains/:domain", a.removeTLSDomain)
	}

	// System endpoints
	systemGroup := router.Group("/system")
	systemGroup.Use(auth)
	{
		systemGroup.GET("/status", a.getSystemStatus)
		systemGroup.GET("/health", a.getHealth)
	}

	// Stats endpoints
	statsGroup := router.Group("/stats")
	statsGroup.Use(auth)
	{
		statsGroup.GET("/traffic", a.getTrafficStats)
	}

	// Auth endpoints (without BasicAuth middleware to avoid browser popup)
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", a.login)
	}
}

func (a *AdminAPI) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, a.config)
}

func (a *AdminAPI) updateConfig(c *gin.Context) {
	var newConfig config.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update current config
	*a.config = newConfig

	// Save to file
	if err := a.config.SaveConfig("config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

func (a *AdminAPI) getProxyRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"rules": a.config.Proxy.Rules})
}

func (a *AdminAPI) addProxyRule(c *gin.Context) {
	var rule config.ProxyRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a.config.AddProxyRule(rule)

	// Save to file
	if err := a.config.SaveConfig("config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Add TLS domain if SSL is enabled
	if rule.SSL.Enabled && a.tls != nil {
		if err := a.tls.AddDomain(rule.Domain); err != nil {
			// Log error but don't fail the operation
			c.Header("X-TLS-Warning", "Failed to obtain TLS certificate: "+err.Error())
		}
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Proxy rule added successfully"})
}

func (a *AdminAPI) updateProxyRule(c *gin.Context) {
	domain := c.Param("domain")
	var rule config.ProxyRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ensure domain matches
	rule.Domain = domain

	a.config.AddProxyRule(rule)

	// Save to file
	if err := a.config.SaveConfig("config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Proxy rule updated successfully"})
}

func (a *AdminAPI) getTrafficStats(c *gin.Context) {
	if a.stats == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "traffic tracking not available",
		})
		return
	}

	duration := 24 * time.Hour
	if rangeParam := c.Query("range"); rangeParam != "" {
		if parsed, err := parseDurationWithDays(rangeParam); err == nil && parsed > 0 {
			duration = parsed
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range parameter"})
			return
		}
	}

	summary := a.stats.GetSummary(duration)
	c.JSON(http.StatusOK, summary)
}

func parseDurationWithDays(input string) (time.Duration, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	if strings.HasSuffix(input, "d") {
		daysStr := strings.TrimSuffix(input, "d")
		days, err := time.ParseDuration(daysStr + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	return time.ParseDuration(input)
}

func (a *AdminAPI) deleteProxyRule(c *gin.Context) {
	domain := c.Param("domain")

	if !a.config.RemoveProxyRule(domain) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proxy rule not found"})
		return
	}

	// Save to file
	if err := a.config.SaveConfig("config.yaml"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Remove TLS domain
	if a.tls != nil {
		a.tls.RemoveDomain(domain)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Proxy rule deleted successfully"})
}

func (a *AdminAPI) getCacheStats(c *gin.Context) {
	if a.cache == nil {
		c.JSON(http.StatusOK, gin.H{"error": "Cache not available"})
		return
	}

	stats := a.cache.Stats()
	c.JSON(http.StatusOK, stats)
}

func (a *AdminAPI) clearCache(c *gin.Context) {
	if a.cache == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache not available"})
		return
	}

	a.cache.Clear()
	c.JSON(http.StatusOK, gin.H{"message": "Cache cleared successfully"})
}

func (a *AdminAPI) invalidateCacheByURL(c *gin.Context) {
	if a.cache == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache not available"})
		return
	}

	var payload struct {
		URL     string `json:"url" binding:"required"`
		Refresh bool   `json:"refresh"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	targetURL, err := parseInvalidateURL(payload.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	domain := strings.ToLower(targetURL.Hostname())
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL must include a valid domain"})
		return
	}

	rule := a.config.GetProxyRule(domain)
	if rule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("No proxy rule found for domain %s", domain)})
		return
	}

	cachePath := targetURL.EscapedPath()
	if cachePath == "" {
		cachePath = "/"
	}

	prefix := fmt.Sprintf("%s:%s:%s", domain, http.MethodGet, cachePath)
	removed := a.cache.DeleteByPrefix(prefix)

	response := gin.H{
		"message":          fmt.Sprintf("Cleared %d cache entries", removed),
		"cleared":          removed,
		"domain":           domain,
		"path":             cachePath,
		"cache_key_prefix": prefix,
	}

	if payload.Refresh {
		if !rule.Cache.Enabled {
			response["refreshed"] = false
			response["refresh_error"] = "cache is disabled for this domain"
			response["message"] = fmt.Sprintf("Cleared %d cache entries; refresh skipped because caching is disabled", removed)
		} else {
			if err := a.refreshCacheForURL(targetURL, domain); err != nil {
				response["refreshed"] = false
				response["refresh_error"] = err.Error()
				response["message"] = fmt.Sprintf("Cleared %d cache entries; refresh failed", removed)
			} else {
				response["refreshed"] = true
				response["message"] = fmt.Sprintf("Cleared %d cache entries and refreshed content", removed)
			}
		}
	}

	c.JSON(http.StatusOK, response)
}

func (a *AdminAPI) deleteCacheKey(c *gin.Context) {
	if a.cache == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache not available"})
		return
	}

	key := c.Param("key")
	a.cache.Delete(key)
	c.JSON(http.StatusOK, gin.H{"message": "Cache key deleted successfully"})
}

func parseInvalidateURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("url is required")
	}

	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("url must include a host")
	}

	if parsed.Path == "" {
		parsed.Path = "/"
	}

	return parsed, nil
}

func (a *AdminAPI) refreshCacheForURL(target *url.URL, host string) error {
	loopbackHost := a.proxyLoopbackHost()
	if loopbackHost == "" {
		return fmt.Errorf("unable to resolve loopback host")
	}

	if a.config.Server.Port <= 0 {
		return fmt.Errorf("proxy server port is not configured")
	}

	path := target.EscapedPath()
	if path == "" {
		path = "/"
	}

	proxyURL := &url.URL{
		Scheme:   "http",
		Host:     net.JoinHostPort(loopbackHost, fmt.Sprintf("%d", a.config.Server.Port)),
		Path:     path,
		RawQuery: target.RawQuery,
	}

	req, err := http.NewRequest(http.MethodGet, proxyURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to build refresh request: %w", err)
	}

	req.Host = host
	req.Header.Set("User-Agent", "Saddy-Cache-Refresh/1.0")
	req.Header.Set("X-Saddy-Cache-Refresh", "1")

	client := &http.Client{
		Timeout: 20 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("refresh request returned status %d", resp.StatusCode)
	}

	return nil
}

func (a *AdminAPI) proxyLoopbackHost() string {
	host := strings.TrimSpace(a.config.Server.Host)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" || host == "*" {
		return "127.0.0.1"
	}
	return host
}

func (a *AdminAPI) getTLSDomains(c *gin.Context) {
	if a.tls == nil {
		c.JSON(http.StatusOK, gin.H{"domains": []string{}})
		return
	}

	domains := a.tls.ListDomains()
	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

func (a *AdminAPI) getTLSCertInfo(c *gin.Context) {
	if a.tls == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TLS not available"})
		return
	}

	domain := c.Param("domain")
	info, err := a.tls.GetCertInfo(domain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

func (a *AdminAPI) renewTLSDomain(c *gin.Context) {
	if a.tls == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TLS not available"})
		return
	}

	domain := c.Param("domain")
	if err := a.tls.ForceRenewal(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Certificate renewed successfully"})
}

func (a *AdminAPI) addTLSDomain(c *gin.Context) {
	if a.tls == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TLS not available"})
		return
	}

	domain := c.Param("domain")
	if err := a.tls.AddDomain(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "TLS domain added successfully"})
}

func (a *AdminAPI) removeTLSDomain(c *gin.Context) {
	if a.tls == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TLS not available"})
		return
	}

	domain := c.Param("domain")
	a.tls.RemoveDomain(domain)
	c.JSON(http.StatusOK, gin.H{"message": "TLS domain removed successfully"})
}

func (a *AdminAPI) getSystemStatus(c *gin.Context) {
	status := gin.H{
		"server": gin.H{
			"host":       a.config.Server.Host,
			"port":       a.config.Server.Port,
			"admin_port": a.config.Server.AdminPort,
			"auto_https": a.config.Server.AutoHTTPS,
		},
		"proxy_rules_count": len(a.config.Proxy.Rules),
		"cache_enabled":     a.cache != nil,
		"tls_enabled":       a.tls != nil,
		"web_ui_enabled":    a.config.WebUI.Enabled,
	}

	// Add cache stats if available
	if a.cache != nil {
		status["cache_stats"] = a.cache.Stats()
	}

	// Add TLS domains if available
	if a.tls != nil {
		status["tls_domains"] = a.tls.ListDomains()
	}

	c.JSON(http.StatusOK, status)
}

func (a *AdminAPI) getHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
	})
}

func (a *AdminAPI) checkDomainStatus(c *gin.Context) {
	domain := c.Param("domain")

	status := gin.H{
		"domain": domain,
		"checks": gin.H{},
	}

	// Check DNS resolution
	status["checks"].(gin.H)["dns"] = checkDNS(domain) //nolint:errcheck

	// Check HTTP accessibility
	status["checks"].(gin.H)["http"] = checkHTTP(domain) //nolint:errcheck

	// Check HTTPS accessibility
	status["checks"].(gin.H)["https"] = checkHTTPS(domain) //nolint:errcheck

	// Check if domain is in proxy rules
	rule := a.config.GetProxyRule(domain)
	status["checks"].(gin.H)["proxy_configured"] = rule != nil //nolint:errcheck

	// Check if SSL is configured for this domain
	if rule != nil {
		status["checks"].(gin.H)["ssl_configured"] = rule.SSL.Enabled //nolint:errcheck
		status["checks"].(gin.H)["force_https"] = rule.SSL.ForceHTTPS //nolint:errcheck
	}

	// Check TLS certificate if available
	if a.tls != nil {
		certInfo, err := a.tls.GetCertInfo(domain)
		if err == nil {
			status["checks"].(gin.H)["certificate"] = gin.H{ //nolint:errcheck
				"valid":          !certInfo.IsExpired,
				"days_remaining": certInfo.DaysRemaining,
				"issuer":         certInfo.Issuer,
				"not_after":      certInfo.NotAfter,
			}
		} else {
			status["checks"].(gin.H)["certificate"] = gin.H{ //nolint:errcheck
				"valid": false,
				"error": "Certificate not found",
			}
		}
	}

	c.JSON(http.StatusOK, status)
}

func checkDNS(domain string) gin.H {
	addrs, err := net.LookupHost(domain)
	if err != nil {
		return gin.H{
			"resolved": false,
			"error":    err.Error(),
		}
	}

	return gin.H{
		"resolved": true,
		"ips":      addrs,
	}
}

func checkHTTP(domain string) gin.H {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://" + domain)
	if err != nil {
		return gin.H{
			"accessible": false,
			"error":      err.Error(),
		}
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	return gin.H{
		"accessible":  true,
		"status_code": resp.StatusCode,
	}
}

func checkHTTPS(domain string) gin.H {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://" + domain)
	if err != nil {
		return gin.H{
			"accessible": false,
			"error":      err.Error(),
		}
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck

	return gin.H{
		"accessible":  true,
		"status_code": resp.StatusCode,
		"tls_version": resp.TLS.Version,
	}
}

// login handles user authentication without triggering browser's HTTP Basic Auth popup.
func (a *AdminAPI) login(c *gin.Context) {
	var credentials struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&credentials); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check credentials
	if credentials.Username == a.config.WebUI.Username && credentials.Password == a.config.WebUI.Password {
		c.JSON(http.StatusOK, gin.H{"success": true})
	} else {
		// Return 401 without WWW-Authenticate header to prevent browser popup
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
	}
}
