#!/bin/bash

# Saddy Git Tag and Release Helper
# 帮助创建 Git tag 并准备发布

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 检查 Git 仓库
check_git() {
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        error "当前目录不是 Git 仓库"
    fi
}

# 检查工作区状态
check_working_tree() {
    if ! git diff-index --quiet HEAD --; then
        warning "工作区有未提交的更改"
        git status --short
        echo ""
        read -p "是否继续？(y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            error "请先提交或暂存更改"
        fi
    fi
}

# 获取最新的 tag
get_latest_tag() {
    git describe --tags --abbrev=0 2>/dev/null || echo "无"
}

# 验证版本号格式
validate_version() {
    local version=$1
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        error "版本号格式错误，应为 vX.Y.Z (例如: v1.0.0)"
    fi
}

# 检查 tag 是否存在
check_tag_exists() {
    local tag=$1
    if git rev-parse "$tag" >/dev/null 2>&1; then
        error "Tag $tag 已存在"
    fi
}

# 更新 CHANGELOG
update_changelog() {
    local version=$1
    local date=$(date +%Y-%m-%d)
    
    info "更新 CHANGELOG.md..."
    
    # 检查是否有 CHANGELOG.md
    if [ ! -f "CHANGELOG.md" ]; then
        warning "CHANGELOG.md 不存在，跳过更新"
        return
    fi
    
    # 提示用户手动更新
    warning "请手动更新 CHANGELOG.md，将 [Unreleased] 改为 [$version] - $date"
    read -p "已更新？按回车继续..."
}

# 创建 tag
create_tag() {
    local version=$1
    local message=$2
    
    info "创建 Git tag: $version"
    git tag -a "$version" -m "$message"
    success "Tag 创建成功: $version"
}

# 推送 tag
push_tag() {
    local version=$1
    
    echo ""
    read -p "是否推送 tag 到远程仓库？(y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        info "推送 tag: $version"
        git push origin "$version"
        success "Tag 推送成功"
        
        echo ""
        info "GitHub Actions 将自动构建发布包"
        info "请访问 GitHub Actions 查看构建进度"
    else
        warning "跳过推送，稍后可手动推送: git push origin $version"
    fi
}

# 显示发布步骤
show_next_steps() {
    local version=$1
    
    echo ""
    echo "=========================================="
    success "Tag 创建完成！"
    echo "=========================================="
    echo ""
    echo "📋 接下来的步骤："
    echo ""
    echo "1. ✅ Git tag 已创建: ${GREEN}${version}${NC}"
    echo ""
    echo "2. 📦 等待 GitHub Actions 自动构建（如果已推送）"
    echo "   访问: https://github.com/chentyke/saddy/actions"
    echo ""
    echo "3. 📝 在 GitHub 创建 Release"
    echo "   - 访问: https://github.com/chentyke/saddy/releases/new"
    echo "   - 选择 tag: ${version}"
    echo "   - 填写 Release notes"
    echo "   - 发布！"
    echo ""
    echo "4. 🐳 （可选）手动构建本地发布包"
    echo "   运行: ${BLUE}./scripts/release.sh${NC}"
    echo ""
    echo "=========================================="
}

# 主函数
main() {
    echo ""
    echo "=========================================="
    echo "    Saddy Release Tagger"
    echo "=========================================="
    echo ""
    
    # 检查环境
    check_git
    check_working_tree
    
    # 显示当前状态
    local current_branch=$(git branch --show-current)
    local latest_tag=$(get_latest_tag)
    local commit_count=$(git rev-list --count HEAD)
    
    info "当前分支: ${current_branch}"
    info "最新 tag: ${latest_tag}"
    info "提交总数: ${commit_count}"
    echo ""
    
    # 询问版本号
    read -p "请输入新版本号 (格式: vX.Y.Z): " version
    
    # 验证版本号
    validate_version "$version"
    check_tag_exists "$version"
    
    echo ""
    info "版本号: ${GREEN}${version}${NC}"
    
    # 询问 tag 消息
    read -p "请输入 tag 描述（默认: Release ${version}）: " tag_message
    if [ -z "$tag_message" ]; then
        tag_message="Release ${version}"
    fi
    
    echo ""
    info "Tag 描述: ${tag_message}"
    echo ""
    
    # 确认
    warning "即将创建 tag: ${version}"
    read -p "确认创建？(y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        error "取消创建"
    fi
    
    # 更新 CHANGELOG（可选）
    echo ""
    read -p "是否需要更新 CHANGELOG.md？(y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        update_changelog "$version"
        
        # 提交 CHANGELOG 更改
        if ! git diff-index --quiet HEAD --; then
            git add CHANGELOG.md
            git commit -m "docs: update CHANGELOG for ${version}"
            success "CHANGELOG 更新已提交"
        fi
    fi
    
    # 创建 tag
    echo ""
    create_tag "$version" "$tag_message"
    
    # 推送 tag
    push_tag "$version"
    
    # 显示后续步骤
    show_next_steps "$version"
}

# 运行主函数
main

