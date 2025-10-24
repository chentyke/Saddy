# Changelog

All important changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- WebSocket proxy support
- Load balancing functionality
- Access logs and statistics
- Prometheus monitoring interface

## [1.0.1] - 2025-10-24

### Fixed
- 🐛 Fixed browser popup issue when entering wrong password on login page
  - Replaced direct BasicAuth endpoint with dedicated login API
  - Login errors now display on the page instead of triggering browser's system authentication dialog
  - Improved user experience with proper error messaging

### Changed
- 🔧 Enhanced authentication handling for login page
- ⚡ Optimized login flow to avoid WWW-Authenticate header response

## [1.0.0] - 2025-10-24

### Added
- ✨ Multi-domain reverse proxy functionality
- 🔒 Automatic HTTPS/TLS certificate management (Let's Encrypt)
- 💾 Built-in CDN caching system with TTL support
- 🎛️ Web-based configuration management interface
- 📡 Complete REST API for all operations
- ⚡ High-performance HTTP server based on Gin framework
- 🔧 Hot configuration reload support
- 🐳 Docker deployment support
- 📊 Real-time system monitoring and statistics
- 🔐 HTTP Basic Authentication for admin interface
- 📋 Domain status checking functionality
- 🔄 Cache management and cleanup mechanisms
- 📝 Comprehensive logging system

### Features
- **Proxy Management**
  - Add, edit, and delete proxy rules
  - Domain-based routing
  - Target URL configuration
  - SSL/TLS settings per domain
  - Cache configuration per rule

- **Cache System**
  - Memory-based caching with optional persistence
  - Configurable TTL (Time To Live)
  - Size limits and cleanup intervals
  - Per-domain cache management
  - Cache statistics and monitoring

- **SSL/TLS Management**
  - Automatic Let's Encrypt certificate issuance
  - Certificate renewal and management
  - HTTPS enforcement options
  - Domain validation and status checking

- **Web Interface**
  - Clean, responsive design using Shadcn UI principles
  - Real-time status monitoring
  - Interactive configuration management
  - System statistics dashboard
  - Mobile-friendly interface

- **Security**
  - HTTP Basic Authentication
  - Secure credential management
  - TLS certificate validation
  - HTTPS redirect support

### Technical Details
- Built with Go 1.21+
- Gin HTTP framework for performance
- Modular architecture with clean separation of concerns
- Configuration via YAML files and environment variables
- Comprehensive error handling and logging
- Cross-platform support (Linux, macOS, Windows)

### Documentation
- Complete README with installation and usage guides
- API documentation with examples
- Docker deployment instructions
- Configuration reference
- Troubleshooting guide

## [0.9.0] - 2025-10-24

### Added
- Initial project structure
- Basic reverse proxy functionality
- Configuration management system
- Docker support
- Basic web interface

### Known Issues
- Limited caching capabilities
- Manual SSL certificate management
- Basic authentication system

---

## Version History

### Version 1.0.0 (Current)
- **Status**: Stable Release
- **Release Date**: October 24, 2025
- **Compatibility**: Go 1.21+, Docker 20.03+
- **Supported Platforms**: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)

### Migration Guide

#### From 0.9.0 to 1.0.0
- Configuration file format has changed - see `configs/config.yaml.example`
- Web interface authentication is now required
- Docker deployment method recommended for production
- Environment variables can now override configuration file settings

#### Breaking Changes
- Default admin port changed from 8080 to 8081
- Configuration file structure updated
- Authentication is now mandatory for web interface and API

---

## Support

For questions about these changes or for support, please:
- Check the [documentation](README.md)
- Create an [issue](https://github.com/chentyke/saddy/issues)
- Start a [discussion](https://github.com/chentyke/saddy/discussions)

---

## Contributing

See the [Contributing Guide](CONTRIBUTING.md) for information on how to contribute to this project.

---

**Note**: This changelog follows the principles of [Keep a Changelog](https://keepachangelog.com/). Only major changes that affect users are documented here. For a complete list of all changes, please refer to the git commit history.