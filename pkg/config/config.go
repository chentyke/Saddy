// Package config provides configuration management for the Saddy reverse proxy server.
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// ServerConfig defines the main server configuration settings.
type ServerConfig struct {
	Host      string    `yaml:"host" json:"host"`
	Port      int       `yaml:"port" json:"port"`
	AdminPort int       `yaml:"admin_port" json:"admin_port"`
	AutoHTTPS bool      `yaml:"auto_https" json:"auto_https"`
	TLS       TLSConfig `yaml:"tls" json:"tls"`
}

// TLSConfig defines TLS/SSL configuration for automatic HTTPS.
type TLSConfig struct {
	Email    string `yaml:"email" json:"email"`
	CacheDir string `yaml:"cache_dir" json:"cache_dir"`
}

// CacheRule defines caching behavior for a specific proxy rule.
type CacheRule struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	TTL     int    `yaml:"ttl" json:"ttl"`
	MaxSize string `yaml:"max_size" json:"max_size"`
}

// SSLRule defines SSL/TLS settings for a specific proxy rule.
type SSLRule struct {
	Enabled    bool `yaml:"enabled" json:"enabled"`
	ForceHTTPS bool `yaml:"force_https" json:"force_https"`
}

// ProxyRule defines a single reverse proxy routing rule.
type ProxyRule struct {
	Domain string    `yaml:"domain" json:"domain"`
	Target string    `yaml:"target" json:"target"`
	Cache  CacheRule `yaml:"cache" json:"cache"`
	SSL    SSLRule   `yaml:"ssl" json:"ssl"`
}

// CacheConfig defines global cache configuration settings.
type CacheConfig struct {
	DefaultTTL      int    `yaml:"default_ttl" json:"default_ttl"`
	MaxSize         string `yaml:"max_size" json:"max_size"`
	CleanupInterval int    `yaml:"cleanup_interval" json:"cleanup_interval"`
	StorageType     string `yaml:"storage_type" json:"storage_type"`
	CacheDir        string `yaml:"cache_dir" json:"cache_dir"`   // Directory for file-based cache
	Persistent      bool   `yaml:"persistent" json:"persistent"` // If true, cache never expires
}

// WebUIConfig defines configuration for the web admin interface.
type WebUIConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`
}

// ProxyConfig contains all proxy routing rules.
type ProxyConfig struct {
	Rules []ProxyRule `yaml:"rules" json:"rules"`
}

// Config represents the complete application configuration.
type Config struct {
	Server ServerConfig `yaml:"server" json:"server"`
	Proxy  ProxyConfig  `yaml:"proxy" json:"proxy"`
	Cache  CacheConfig  `yaml:"cache" json:"cache"`
	WebUI  WebUIConfig  `yaml:"web_ui" json:"web_ui"`
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Set defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.AdminPort == 0 {
		config.Server.AdminPort = 8081
	}

	return &config, nil
}

// SaveConfig saves the current configuration to a YAML file.
func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// GetProxyRule retrieves a proxy rule for a specific domain.
func (c *Config) GetProxyRule(domain string) *ProxyRule {
	for _, rule := range c.Proxy.Rules {
		if rule.Domain == domain {
			return &rule
		}
	}
	return nil
}

// AddProxyRule adds or updates a proxy rule for a domain.
func (c *Config) AddProxyRule(rule ProxyRule) {
	// Remove existing rule for this domain if exists
	for i, r := range c.Proxy.Rules {
		if r.Domain == rule.Domain {
			c.Proxy.Rules = append(c.Proxy.Rules[:i], c.Proxy.Rules[i+1:]...)
			break
		}
	}
	c.Proxy.Rules = append(c.Proxy.Rules, rule)
}

// RemoveProxyRule removes a proxy rule for a specific domain.
func (c *Config) RemoveProxyRule(domain string) bool {
	for i, rule := range c.Proxy.Rules {
		if rule.Domain == domain {
			c.Proxy.Rules = append(c.Proxy.Rules[:i], c.Proxy.Rules[i+1:]...)
			return true
		}
	}
	return false
}
