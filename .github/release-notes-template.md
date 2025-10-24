## 🎉 Saddy Release

Thank you for using Saddy! This is a lightweight reverse proxy server with automatic HTTPS and CDN caching support.

### 📦 Downloads

Choose the appropriate version for your system:

- **Linux AMD64**: `saddy-linux-amd64.tar.gz`
- **Linux ARM64**: `saddy-linux-arm64.tar.gz`
- **macOS Intel**: `saddy-darwin-amd64.tar.gz`
- **macOS Apple Silicon**: `saddy-darwin-arm64.tar.gz`
- **Windows AMD64**: `saddy-windows-amd64.zip`

### 🚀 Quick Start

#### Linux/macOS

```bash
# Download and extract
tar -xzf saddy-*-*.tar.gz

# Copy configuration file
cp config.yaml.example config.yaml

# Edit configuration
vim config.yaml

# Run
./saddy-* -config config.yaml
```

#### Windows

```powershell
# Extract zip file
# Copy and edit configuration file
# Run
saddy-windows-amd64.exe -config config.yaml
```

#### Docker

```bash
docker pull yourusername/saddy:latest
docker run -d \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/configs:/app/configs:ro \
  -v saddy-certs:/app/certs \
  yourusername/saddy:latest
```

### ✨ Key Features

- 🚀 Multi-domain reverse proxy
- 🔒 Let's Encrypt automatic HTTPS
- 💾 Built-in CDN caching (memory/file)
- 🎛️ Web management interface
- 📡 Complete REST API
- 🐳 Docker support
- ⚡ High performance (Gin-based)

### 📋 What's New

Please see [CHANGELOG.md](https://github.com/yourusername/saddy/blob/main/CHANGELOG.md) for detailed changes.

### 📚 Documentation

- [Usage Guide](https://github.com/yourusername/saddy#readme)
- [Configuration Guide](https://github.com/yourusername/saddy/blob/main/docs/README.md)
- [API Documentation](https://github.com/yourusername/saddy/blob/main/docs/README.md)
- [Contributing Guide](https://github.com/yourusername/saddy/blob/main/CONTRIBUTING.md)

### 🐛 Issue Reporting

If you encounter issues, please:
- Check existing [Issues](https://github.com/yourusername/saddy/issues)
- Submit a new [Bug Report](https://github.com/yourusername/saddy/issues/new?template=bug_report.md)

### 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](https://github.com/yourusername/saddy/blob/main/CONTRIBUTING.md)

### 📄 License

MIT License - see [LICENSE](https://github.com/yourusername/saddy/blob/main/LICENSE)

---

**SHA256 Checksums**

Please download `checksums.txt` to verify file integrity.