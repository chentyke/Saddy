#!/bin/bash

# Saddy Release Script
# 用于本地构建和打包发布版本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
APP_NAME="saddy"
BUILD_DIR="build"
RELEASE_DIR="release"

# 函数：打印信息
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 函数：检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        error "$1 未安装，请先安装"
    fi
}

# 函数：获取版本号
get_version() {
    # 尝试从 git tag 获取版本
    if git describe --tags --exact-match 2>/dev/null; then
        VERSION=$(git describe --tags --exact-match)
    else
        # 如果没有 tag，使用最近的 tag + commit hash
        VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    fi
    echo "$VERSION"
}

# 函数：清理旧文件
cleanup() {
    info "清理旧的构建文件..."
    rm -rf "$BUILD_DIR" "$RELEASE_DIR"
    mkdir -p "$BUILD_DIR" "$RELEASE_DIR"
    success "清理完成"
}

# 函数：构建单个平台
build_platform() {
    local goos=$1
    local goarch=$2
    local output_name="${APP_NAME}-${goos}-${goarch}"
    
    if [ "$goos" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    info "构建 ${goos}/${goarch}..."
    
    GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build \
        -ldflags "-s -w -X main.Version=${VERSION} -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
        -o "${BUILD_DIR}/${output_name}" \
        ./cmd/saddy
    
    if [ $? -eq 0 ]; then
        success "构建成功: ${output_name}"
    else
        error "构建失败: ${goos}/${goarch}"
    fi
}

# 函数：创建压缩包
create_archive() {
    local goos=$1
    local goarch=$2
    local binary_name="${APP_NAME}-${goos}-${goarch}"
    local archive_name="${APP_NAME}-${VERSION}-${goos}-${goarch}"
    
    if [ "$goos" = "windows" ]; then
        binary_name="${binary_name}.exe"
        archive_name="${archive_name}.zip"
        
        info "创建 Windows 压缩包..."
        cd "$BUILD_DIR"
        zip -q "../${RELEASE_DIR}/${archive_name}" \
            "$binary_name" \
            ../README.md \
            ../LICENSE \
            ../CHANGELOG.md \
            ../configs/config.yaml.example
        cd ..
    else
        archive_name="${archive_name}.tar.gz"
        
        info "创建 ${goos} 压缩包..."
        tar -czf "${RELEASE_DIR}/${archive_name}" \
            -C "$BUILD_DIR" "$binary_name" \
            -C .. README.md LICENSE CHANGELOG.md \
            -C configs config.yaml.example
    fi
    
    success "压缩包创建成功: ${archive_name}"
}

# 函数：生成校验和
generate_checksums() {
    info "生成 SHA256 校验和..."
    cd "$RELEASE_DIR"
    sha256sum *.tar.gz *.zip 2>/dev/null > checksums.txt || shasum -a 256 *.tar.gz *.zip > checksums.txt
    cd ..
    success "校验和生成完成"
}

# 函数：显示发布信息
show_release_info() {
    echo ""
    echo "=========================================="
    success "发布包构建完成！"
    echo "=========================================="
    echo ""
    echo "版本: ${GREEN}${VERSION}${NC}"
    echo "输出目录: ${BLUE}${RELEASE_DIR}/${NC}"
    echo ""
    echo "文件列表:"
    ls -lh "$RELEASE_DIR" | tail -n +2
    echo ""
    echo "校验和 (checksums.txt):"
    cat "$RELEASE_DIR/checksums.txt"
    echo ""
    echo "=========================================="
    echo "下一步操作:"
    echo "1. 测试构建的二进制文件"
    echo "2. 创建 Git tag: ${GREEN}git tag -a ${VERSION} -m 'Release ${VERSION}'${NC}"
    echo "3. 推送 tag: ${GREEN}git push origin ${VERSION}${NC}"
    echo "4. 在 GitHub 创建 Release 并上传文件"
    echo "=========================================="
}

# 主函数
main() {
    echo ""
    echo "=========================================="
    echo "    Saddy Release Builder"
    echo "=========================================="
    echo ""
    
    # 检查必要的命令
    check_command "go"
    check_command "git"
    
    # 获取版本号
    VERSION=$(get_version)
    info "当前版本: ${VERSION}"
    
    # 询问是否继续
    read -p "是否继续构建发布包？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        warning "取消构建"
        exit 0
    fi
    
    # 清理
    cleanup
    
    # 构建所有平台
    info "开始构建所有平台..."
    echo ""
    
    build_platform "linux" "amd64"
    build_platform "linux" "arm64"
    build_platform "darwin" "amd64"
    build_platform "darwin" "arm64"
    build_platform "windows" "amd64"
    
    echo ""
    info "所有平台构建完成"
    echo ""
    
    # 创建压缩包
    info "创建发布压缩包..."
    echo ""
    
    create_archive "linux" "amd64"
    create_archive "linux" "arm64"
    create_archive "darwin" "amd64"
    create_archive "darwin" "arm64"
    create_archive "windows" "amd64"
    
    echo ""
    
    # 生成校验和
    generate_checksums
    
    # 显示发布信息
    show_release_info
}

# 运行主函数
main

