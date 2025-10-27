## 🎉 Saddy v1.0.1 - Bug Fix Release

Thank you for using Saddy! This is a minor bug fix release that improves the login experience.

### 🐛 Bug Fixes

**Fixed Browser Popup Issue on Login**
- Resolved an issue where entering a wrong password would trigger the browser's system authentication dialog
- The login page now properly displays error messages without causing browser popups
- Enhanced user experience with cleaner error handling

**Technical Details:**
- Added dedicated `/api/v1/auth/login` endpoint without BasicAuth middleware
- Login authentication no longer sends `WWW-Authenticate` header that triggers browser dialogs
- Error messages now display directly on the login page

### 📦 Downloads

Choose the appropriate version for your system:

- **Linux AMD64**: `saddy-v1.0.1-linux-amd64.tar.gz`
- **Linux ARM64**: `saddy-v1.0.1-linux-arm64.tar.gz`
- **macOS Intel**: `saddy-v1.0.1-darwin-amd64.tar.gz`
- **macOS Apple Silicon**: `saddy-v1.0.1-darwin-arm64.tar.gz`
- **Windows AMD64**: `saddy-v1.0.1-windows-amd64.zip`

### 🚀 Quick Start

#### Linux/macOS

```bash
# Download and extract
tar -xzf saddy-v1.0.1-*.tar.gz

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
docker pull chentyke/saddy:v1.0.1
# or use :latest for the latest version
docker pull chentyke/saddy:latest

docker run -d \
  -p 80:80 \
  -p 443:443 \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/configs:/app/configs:ro \
  -v saddy-certs:/app/certs \
  chentyke/saddy:v1.0.1
```

### ✨ Key Features

- 🚀 Multi-domain reverse proxy
- 🔒 Let's Encrypt automatic HTTPS
- 💾 Built-in CDN caching (memory/file)
- 🎛️ Web management interface
- 📡 Complete REST API
- 🐳 Docker support
- ⚡ High performance (Gin-based)

### 📋 Full Changelog

See [CHANGELOG.md](https://github.com/chentyke/saddy/blob/main/CHANGELOG.md) for all changes.

### ⬆️ Upgrade from v1.0.0

This is a drop-in replacement for v1.0.0. No configuration changes are required.

Simply replace the binary and restart the service.

### 📚 Documentation

- [Usage Guide](https://github.com/chentyke/saddy#readme)
- [Configuration Reference](https://github.com/chentyke/saddy/blob/main/configs/config.yaml.example)
- [Contributing Guide](https://github.com/chentyke/saddy/blob/main/CONTRIBUTING.md)

### 🐛 Issue Reporting

If you encounter issues, please:
- Check existing [Issues](https://github.com/chentyke/saddy/issues)
- Submit a new [Issue](https://github.com/chentyke/saddy/issues/new)

### 🤝 Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](https://github.com/chentyke/saddy/blob/main/CONTRIBUTING.md)

### 📄 License

MIT License - see [LICENSE](https://github.com/chentyke/saddy/blob/main/LICENSE)

---

**SHA256 Checksums**

Please download `checksums.txt` from the release assets to verify file integrity.

---

**Full Changelog**: https://github.com/chentyke/saddy/compare/v1.0.0...v1.0.1


