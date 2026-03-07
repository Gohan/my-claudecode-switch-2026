#!/bin/bash
# Claude Switch 构建脚本
# 用法: ./build.sh [windows|linux|darwin] [amd64|arm64]

set -e

# 默认值
OS="${1:-windows}"
ARCH="${2:-amd64}"
VERSION="${3:-dev}"
OUTPUT_DIR="dist"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 确保mise环境已加载
if ! command -v go &> /dev/null; then
    if [ -f ~/.zshrc ]; then
        source ~/.zshrc 2>/dev/null
    elif [ -f ~/.bashrc ]; then
        source ~/.bashrc 2>/dev/null
    fi
fi

# 再次检查go
if ! command -v go &> /dev/null; then
    log_error "Go not found. Please install Go via mise: mise install go"
    exit 1
fi

log_info "Using Go: $(go version)"

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 设置输出文件名
BINARY_NAME="claude-switch"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="claude-switch.exe"
fi

OUTPUT_PATH="$OUTPUT_DIR/${OS}_${ARCH}/$BINARY_NAME"

log_info "Building for $OS/$ARCH..."

# 构建参数
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME"

# 执行编译
CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH go build \
    -ldflags "$LDFLAGS" \
    -o "$OUTPUT_PATH" \
    .

if [ $? -eq 0 ]; then
    log_info "Build successful: $OUTPUT_PATH"
    ls -lh "$OUTPUT_PATH"
else
    log_error "Build failed!"
    exit 1
fi

# 可选：构建所有平台
build_all() {
    log_info "Building for all platforms..."
    
    # Linux
    GOOS=linux GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/linux_amd64/claude-switch" .
    GOOS=linux GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/linux_arm64/claude-switch" .
    
    # Windows
    GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/windows_amd64/claude-switch.exe" .
    GOOS=windows GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/windows_arm64/claude-switch.exe" .
    
    # macOS
    GOOS=darwin GOARCH=amd64 go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/darwin_amd64/claude-switch" .
    GOOS=darwin GOARCH=arm64 go build -ldflags "$LDFLAGS" -o "$OUTPUT_DIR/darwin_arm64/claude-switch" .
    
    log_info "All builds complete!"
}

# 如果传入 "all" 作为参数，构建所有平台
if [ "$1" = "all" ]; then
    build_all
fi
