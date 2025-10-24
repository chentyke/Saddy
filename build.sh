#!/bin/bash

# Build script for Saddy

set -e

echo "Building Saddy..."

# Create build directory
mkdir -p build

# Build for current platform
echo "Building for $(go env GOOS)/$(go env GOARCH)..."
go build -o build/saddy ./cmd/saddy

# Build for multiple platforms
echo "Building for multiple platforms..."

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o build/saddy-linux-amd64 ./cmd/saddy

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o build/saddy-linux-arm64 ./cmd/saddy

# macOS AMD64
GOOS=darwin GOARCH=amd64 go build -o build/saddy-darwin-amd64 ./cmd/saddy

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o build/saddy-darwin-arm64 ./cmd/saddy

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o build/saddy-windows-amd64.exe ./cmd/saddy

echo "Build completed successfully!"
echo "Binaries available in build/ directory:"
ls -la build/

# Create a simple start script
cat > build/start.sh << 'EOF'
#!/bin/bash

# Simple start script for Saddy

# Check if config file exists
if [ ! -f "configs/config.yaml" ]; then
    echo "Configuration file not found. Please copy configs/config.yaml.example to configs/config.yaml"
    exit 1
fi

# Start Saddy
echo "Starting Saddy..."
./saddy

EOF

chmod +x build/start.sh

echo "Created start.sh script in build/ directory"