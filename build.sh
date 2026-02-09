#!/bin/bash
#
# kubectl-mcp 编译脚本 (Linux/macOS)
#
# 用法:
#   ./build.sh [windows|linux|all]
#

set -e

# 默认参数
PLATFORM="${1:-all}"
VERSION="1.3.0"
BUILD_TIME=$(date +%Y-%m-%d)
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

echo "========================================"
echo "  kubectl-mcp 编译脚本"
echo "========================================"
echo "版本号: ${VERSION}"
echo "构建日期: ${BUILD_TIME}"
echo "目标平台: ${PLATFORM}"
echo "========================================"
echo ""

# 编译 Windows 版本
build_windows() {
    echo "正在编译 Windows 版本..."
    GOOS=windows GOARCH=amd64 go build -o kubectl-mcp.exe -ldflags "${LDFLAGS}" ./cmd/server
    
    if [ -f kubectl-mcp.exe ]; then
        SIZE=$(du -h kubectl-mcp.exe | cut -f1)
        echo "✅ Windows 版本编译成功: kubectl-mcp.exe (${SIZE})"
    else
        echo "❌ Windows 版本编译失败"
        exit 1
    fi
}

# 编译 Linux 版本
build_linux() {
    echo "正在编译 Linux 版本..."
    GOOS=linux GOARCH=amd64 go build -o kubectl-mcp-linux-amd64 -ldflags "${LDFLAGS}" ./cmd/server
    
    if [ -f kubectl-mcp-linux-amd64 ]; then
        SIZE=$(du -h kubectl-mcp-linux-amd64 | cut -f1)
        echo "✅ Linux 版本编译成功: kubectl-mcp-linux-amd64 (${SIZE})"
    else
        echo "❌ Linux 版本编译失败"
        exit 1
    fi
}

# 根据参数执行编译
case "${PLATFORM}" in
    windows)
        build_windows
        ;;
    linux)
        build_linux
        ;;
    all)
        build_windows
        echo ""
        build_linux
        ;;
    *)
        echo "错误: 未知的平台 '${PLATFORM}'"
        echo "用法: $0 [windows|linux|all]"
        exit 1
        ;;
esac

echo ""
echo "========================================"
echo "  编译完成！"
echo "========================================"
