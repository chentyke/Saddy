<div align="center">

# 🚀 Saddy

**Lightweight Reverse Proxy Server**

[![GitHub release](https://img.shields.io/github/release/yourusername/saddy.svg)](https://github.com/yourusername/saddy/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourusername/saddy)](https://goreportcard.com/report/github.com/yourusername/saddy)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev)

Saddy is a lightweight reverse proxy server written in Go, providing automatic HTTPS, CDN caching, and a web configuration interface.

</div>

---

## ✨ Key Features

- 🚀 **Reverse Proxy** - Support for multi-domain reverse proxy configuration
- 🔒 **Automatic HTTPS** - Automatic acquisition and renewal of Let's Encrypt certificates
- 💾 **CDN Caching** - Built-in caching mechanism to improve response speed
- 🎛️ **Web Interface** - Graphical configuration management interface
- 📡 **REST API** - Complete API interface support
- ⚡ **High Performance** - High-performance HTTP server based on Gin framework
- 🔧 **Hot Reload** - Support for configuration hot reloading
- 🐳 **Docker Support** - Complete Docker deployment solution

## 📦 Quick Start

### Prerequisites

- Go 1.21+ (if building from source)
- Docker (optional, recommended for production environment)

### Using Docker (Recommended)

```bash
# Clone the project
git clone https://github.com/yourusername/saddy.git
cd saddy

# Start with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f saddy
```

Visit `http://localhost:8081` to open the management interface.

### Building from Source

```bash
# 1. Clone the project
git clone https://github.com/yourusername/saddy.git
cd saddy

# 2. Download dependencies
go mod download

# 3. Build
./build.sh
# Or use Make
make build

# 4. Copy configuration file
cp configs/config.yaml.example configs/config.yaml

# 5. Edit configuration file
vim configs/config.yaml

# 6. Run
./saddy -config configs/config.yaml
```

## ⚙️ Configuration Guide

### Basic Configuration

Create or edit `configs/config.yaml`:

```yaml
server:
  host: "0.0.0.0"           # Server listening address
  port: 8080                # Reverse proxy port
  admin_port: 8081          # Management interface port
  auto_https: true          # Enable automatic HTTPS
  tls:
    email: "admin@example.com"  # Let's Encrypt email
    cache_dir: "./certs"        # Certificate cache directory

proxy:
  rules:
    - domain: "example.com"
      target: "http://localhost:3000"
      cache:
        enabled: true
        ttl: 300              # Cache time (seconds)
        max_size: "100MB"     # Maximum cache size
      ssl:
        enabled: true
        force_https: true     # Force HTTPS

cache:
  default_ttl: 300            # Default cache time (seconds)
  max_size: "500MB"           # Maximum cache size
  cleanup_interval: 600       # Cleanup interval (seconds)
  storage_type: "memory"      # Storage type: memory/file
  persistent: false           # Whether to persist

web_ui:
  enabled: true
  username: "admin"           # Web interface username
  password: "admin123"        # Web interface password (please change)
```

### Environment Variables

You can also override configuration through environment variables:

```bash
export SADDY_ADMIN_USERNAME=admin
export SADDY_ADMIN_PASSWORD=your_secure_password
export SADDY_TLS_EMAIL=your@email.com
```

## 🎨 Web Management Interface

Visit `http://localhost:8081` to open the web management interface:

- **Overview** - System status and statistics
- **Proxy Rules** - Manage reverse proxy rules
- **Cache Management** - View and clear cache
- **SSL/TLS** - Manage certificates and domains
- **System Settings** - Server configuration

**Default Login Information**:
- Username: `admin`
- Password: `admin123`

⚠️ **Please change the default password in production environment!**

## 📡 REST API

Saddy provides a complete REST API interface, all requests require HTTP Basic authentication.

### API Endpoints

#### System Status

```bash
curl -u admin:admin123 http://localhost:8081/api/v1/system/status
```

#### Proxy Rule Management

```bash
# Get all proxy rules
curl -u admin:admin123 http://localhost:8081/api/v1/config/proxy

# Add proxy rule
curl -u admin:admin123 -X POST http://localhost:8081/api/v1/config/proxy \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "api.example.com",
    "target": "http://localhost:3000",
    "cache": {"enabled": true, "ttl": 300},
    "ssl": {"enabled": true, "force_https": true}
  }'

# Update proxy rule
curl -u admin:admin123 -X PUT http://localhost:8081/api/v1/config/proxy/api.example.com \
  -H "Content-Type: application/json" \
  -d '{"target": "http://localhost:3001"}'

# Delete proxy rule
curl -u admin:admin123 -X DELETE http://localhost:8081/api/v1/config/proxy/api.example.com
```

#### Cache Management

```bash
# Get cache statistics
curl -u admin:admin123 http://localhost:8081/api/v1/cache/stats

# Clear all cache
curl -u admin:admin123 -X DELETE http://localhost:8081/api/v1/cache/

# Clear specific domain cache
curl -u admin:admin123 -X DELETE http://localhost:8081/api/v1/cache/example.com
```

#### TLS/SSL Management

```bash
# Get TLS domain list
curl -u admin:admin123 http://localhost:8081/api/v1/tls/domains

# Add domain
curl -u admin:admin123 -X POST http://localhost:8081/api/v1/tls/domains \
  -H "Content-Type: application/json" \
  -d '{"domain": "new.example.com"}'

# Delete domain
curl -u admin:admin123 -X DELETE http://localhost:8081/api/v1/tls/domains/new.example.com
```

## 🏗️ Architecture Design

```
┌─────────────────┐
│   Client Request │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│          Saddy Reverse Proxy Server      │
│  ┌─────────────────────────────────┐   │
│  │        TLS/HTTPS Layer            │   │
│  │  (Let's Encrypt Auto Cert Mgmt)   │   │
│  └─────────────┬───────────────────┘   │
│                ▼                        │
│  ┌─────────────────────────────────┐   │
│  │        Cache Layer               │   │
│  │  (Memory/File Cache, TTL Mgmt)   │   │
│  └─────────────┬───────────────────┘   │
│                ▼                        │
│  ┌─────────────────────────────────┐   │
│  │      Reverse Proxy Routing       │   │
│  │  (Multi-domain Rules, LB)        │   │
│  └─────────────┬───────────────────┘   │
└────────────────┼───────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────┐
│           Backend Services              │
│  ┌──────────┐  ┌──────────┐            │
│  │ Service A│  │ Service B│   ...      │
│  └──────────┘  └──────────┘            │
└─────────────────────────────────────────┘

         Web Management Interface (Port 8081)
              │
              ▼
    ┌─────────────────────┐
    │   REST API          │
    │  - Config Mgmt      │
    │  - Cache Control    │
    │  - Certificate Mgmt │
    │  - System Monitoring│
    └─────────────────────┘
```

## 📂 Project Structure

```
saddy/
├── cmd/
│   └── saddy/          # Application entry point
│       └── main.go
├── pkg/                # Core packages
│   ├── api/           # REST API implementation
│   ├── cache/         # Cache module
│   ├── config/        # Configuration management
│   ├── https/         # TLS/HTTPS management
│   ├── proxy/         # Reverse proxy core
│   └── web/           # Web server
├── internal/          # Internal packages
│   ├── middleware/    # Middleware
│   └── server/        # Server implementation
├── web/               # Web interface resources
│   ├── static/        # Static files like CSS, JS
│   └── templates/     # HTML templates
├── configs/           # Configuration files
│   ├── config.yaml.example
│   └── config.yaml
├── examples/          # Examples and demos
├── Dockerfile         # Docker image
├── docker-compose.yml # Docker Compose configuration
├── build.sh           # Build script
├── Makefile           # Make build file
└── README.md          # Project documentation
```

## 🚀 Deployment

### Docker Deployment (Recommended)

#### Using Docker Compose

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Update services
docker-compose pull
docker-compose up -d
```

#### Custom Docker Deployment

```bash
# Build image
docker build -t saddy:latest .

# Run container
docker run -d \
  --name saddy \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/configs:/app/configs:ro \
  -v saddy-certs:/app/certs \
  -v saddy-logs:/app/logs \
  saddy:latest
```

### Systemd Service Deployment

Create service file `/etc/systemd/system/saddy.service`:

```ini
[Unit]
Description=Saddy Reverse Proxy Server
After=network.target

[Service]
Type=simple
User=saddy
Group=saddy
WorkingDirectory=/opt/saddy
ExecStart=/opt/saddy/saddy -config /opt/saddy/configs/config.yaml
Restart=always
RestartSec=5
LimitNOFILE=65536

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/saddy/certs /opt/saddy/logs

[Install]
WantedBy=multi-user.target
```

Enable and start service:

```bash
# Create user
sudo useradd -r -s /bin/false saddy

# Set permissions
sudo chown -R saddy:saddy /opt/saddy

# Enable service
sudo systemctl daemon-reload
sudo systemctl enable saddy
sudo systemctl start saddy

# View status
sudo systemctl status saddy

# View logs
sudo journalctl -u saddy -f
```

### Production Environment Recommendations

1. **Security**
   - Change default admin password
   - Use strong passwords or key authentication
   - Configure firewall rules
   - Enable HTTPS and force redirects

2. **Performance Optimization**
   - Adjust cache size and TTL
   - Choose memory or file cache based on needs
   - Configure appropriate cleanup intervals

3. **Monitoring and Logging**
   - Check logs regularly
   - Configure log rotation
   - Set up alert notifications

4. **Backup**
   - Regularly backup configuration files
   - Backup TLS certificates

## 🔧 Troubleshooting

### Certificate Acquisition Failure

**Problem**: Let's Encrypt certificate acquisition failed

**Solution**:
1. Ensure domain DNS resolution points correctly to the server
2. Check firewall, ensure ports 80 and 443 are open
3. Verify email address validity
4. Check if server time is correct

```bash
# Check DNS resolution
dig example.com

# Test port connectivity
nc -zv your-server-ip 80
nc -zv your-server-ip 443

# Check logs
docker-compose logs saddy | grep -i "certificate"
```

### Proxy Not Working

**Problem**: Reverse proxy cannot access backend services

**Solution**:
1. Check if target service is running
2. Verify proxy rule configuration is correct
3. Check network connectivity
4. View detailed logs

```bash
# Test backend service
curl -v http://localhost:3000

# Check proxy configuration
curl -u admin:admin123 http://localhost:8081/api/v1/config/proxy

# View logs
docker-compose logs saddy -f
```

### Cache Issues

**Problem**: Cache not working or consuming too many resources

**Solution**:
1. Check cache configuration
2. Verify cache size limits
3. Manually clear cache

```bash
# View cache statistics
curl -u admin:admin123 http://localhost:8081/api/v1/cache/stats

# Clear cache
curl -u admin:admin123 -X DELETE http://localhost:8081/api/v1/cache/
```

### Performance Issues

**Problem**: Slow service response or high resource usage

**Solution**:
1. Adjust cache settings
2. Increase system resources
3. Check backend service performance
4. Optimize proxy rules

## 🤝 Contributing

We welcome all forms of contributions!

- 🐛 Report Bugs
- 💡 Propose new features
- 📖 Improve documentation
- 🔧 Submit code

Please read the [Contributing Guide](CONTRIBUTING.md) for detailed information.

### Contributors

Thanks to everyone who contributed to this project!

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🔗 Related Links

- [Changelog](CHANGELOG.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Issue Tracker](https://github.com/yourusername/saddy/issues)
- [Discussions](https://github.com/yourusername/saddy/discussions)

## ⭐ Star History

If this project helps you, please give us a Star!

[![Star History Chart](https://api.star-history.com/svg?repos=yourusername/saddy&type=Date)](https://star-history.com/#yourusername/saddy&Date)

## 📧 Contact

For questions or suggestions, please:

- Create an [Issue](https://github.com/yourusername/saddy/issues)
- Start a [Discussion](https://github.com/yourusername/saddy/discussions)

---

<div align="center">

**[⬆ Back to top](#-saddy)**

Made with ❤️ by the Saddy Team

</div>