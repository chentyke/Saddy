// Package api provides RESTful API endpoints for managing Saddy configuration and monitoring.
package api

import (
	"net"
	"net/http"
	"time"

	"saddy/pkg/cache"
	"saddy/pkg/config"
	"saddy/pkg/https"

	"github.com/gin-gonic/gin"
)

// AdminAPI provides administrative API endpoints for configuration and monitoring.
type AdminAPI struct {
	config *config.Config
	cache  cache.Storage
	tls    *https.AutoTLS
}

// NewAdminAPI creates a new AdminAPI instance with the given configuration and services.
func NewAdminAPI(cfg *config.Config, cacheStorage cache.Storage, tls *https.AutoTLS) *AdminAPI {
	return &AdminAPI{
		config: cfg,
		cache:  cacheStorage,
		tls:    tls,
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

func (a *AdminAPI) deleteCacheKey(c *gin.Context) {
	if a.cache == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cache not available"})
		return
	}

	key := c.Param("key")
	a.cache.Delete(key)
	c.JSON(http.StatusOK, gin.H{"message": "Cache key deleted successfully"})
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
