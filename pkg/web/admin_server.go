// Package web provides the web administration interface for Saddy.
package web

import (
	"net/http"
	"strings"
	"time"

	"saddy/pkg/api"

	"github.com/gin-gonic/gin"
)

const (
	defaultReadHeaderTimeout = 10 * time.Second
)

// AdminServer manages the web admin interface and API endpoints.
type AdminServer struct {
	engine *gin.Engine
	api    *api.AdminAPI
}

// NewAdminServer creates a new admin server instance with the given API.
func NewAdminServer(adminAPI *api.AdminAPI) *AdminServer {
	gin.SetMode(gin.ReleaseMode)

	server := &AdminServer{
		engine: gin.New(),
		api:    adminAPI,
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
