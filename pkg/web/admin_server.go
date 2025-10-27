// Package web provides the web administration interface for Saddy.
package web

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"saddy/pkg/api"
	"saddy/pkg/config"

	"github.com/gin-gonic/gin"
)

const (
	defaultReadHeaderTimeout = 10 * time.Second
)

// AdminServer manages the web admin interface and API endpoints.
type AdminServer struct {
	engine *gin.Engine
	api    *api.AdminAPI
	config *config.Config
}

// NewAdminServer creates a new admin server instance with the given API.
func NewAdminServer(adminAPI *api.AdminAPI, cfg *config.Config) *AdminServer {
	gin.SetMode(gin.ReleaseMode)

	server := &AdminServer{
		engine: gin.New(),
		api:    adminAPI,
		config: cfg,
	}

	server.setupRoutes()
	return server
}

func (s *AdminServer) setupRoutes() {
	// Middleware
	s.engine.Use(gin.Logger())
	s.engine.Use(gin.Recovery())

	// Serve static files - look in current directory first, then web/
	s.engine.Static("/static", "./web/static")
	s.engine.LoadHTMLGlob("web/templates/*")

	// Login page
	s.engine.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	// Main page (with auth check)
	s.engine.GET("/", func(c *gin.Context) {
		// Check for basic auth in header first
		auth := c.GetHeader("Authorization")

		// If no auth header, check for cookie
		if auth == "" {
			cookie, err := c.Cookie("saddy_auth")
			if err == nil && cookie != "" {
				auth = "Basic " + cookie
			}
		}

		if auth == "" {
			// No auth header or cookie, check if accessing from browser (not API)
			if c.GetHeader("Accept") == "" || strings.Contains(c.GetHeader("Accept"), "text/html") {
				c.Redirect(http.StatusFound, "/login")
				return
			}
		}

		// For API calls without proper auth, return 401
		if auth == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}

		// Validate credentials against config
		username, password, err := decodeBasicAuth(auth)
		if err != nil || !s.validateCredentials(username, password) {
			// Invalid credentials, redirect to login for browser requests
			if c.GetHeader("Accept") == "" || strings.Contains(c.GetHeader("Accept"), "text/html") {
				c.Redirect(http.StatusFound, "/login")
				return
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authentication"})
			return
		}

		c.HTML(http.StatusOK, "index.html", nil)
	})

	// API routes with versioning
	v1 := s.engine.Group("/api/v1")
	s.api.SetupRoutes(v1)
}

// Start starts the admin server on the specified address.
func (s *AdminServer) Start(addr string) error {
	server := &http.Server{
		Addr:              addr,
		Handler:           s.engine,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}
	return server.ListenAndServe()
}

// decodeBasicAuth decodes a Basic Auth header and returns username and password.
func decodeBasicAuth(auth string) (string, string, error) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return "", "", fmt.Errorf("invalid authorization header format")
	}

	// Decode the base64 part
	encoded := strings.TrimPrefix(auth, prefix)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", "", err
	}

	// Split username:password
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid authorization header content")
	}

	return parts[0], parts[1], nil
}

// validateCredentials checks if the provided username and password match the configured ones.
func (s *AdminServer) validateCredentials(username, password string) bool {
	if s.config == nil {
		return false
	}
	return username == s.config.WebUI.Username && password == s.config.WebUI.Password
}
