#!/bin/bash

set -euo pipefail  # 更严格的错误处理

# 彩色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

PROJECT="autossh"
VERSION="${1:-v1.1.0}"
BUILD_TIME="$(date +%FT%T%z)"
START_TIME=$(date +%s)

# 支持的目标平台
ALL_TARGETS=(
    "darwin arm64 macOS"
    "darwin amd64 macOS"
    "linux amd64 linux"
    "linux 386 linux"
    "linux arm linux"
)

# 显示帮助
if [[ "${1:-}" == "help" ]]; then
    echo "用法: ./build.sh [version] [all]"
    echo "示例: ./build.sh v1.2.0 all"
    exit 0
fi

# 检查依赖
for cmd in go zip; do
    if ! command -v "$cmd" &>/dev/null; then
        echo -e "${RED}缺少依赖: $cmd，请先安装！${NC}"
        exit 1
    fi
done

export GO111MODULE=on
go mod tidy

chmod +x install

# 构建函数
build() {
    local os="$1"
    local arch="$2"
    local alias_name="$3"
    local package="${PROJECT}-${alias_name}-${arch}_${VERSION}"

    echo -e "${GREEN}==> 构建 ${package} ...${NC}"

    mkdir -p "releases/${package}"
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
        -o "releases/${package}/autossh" \
        -ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD_TIME}" \
        src/main/main.go

    cp config.example.json "releases/${package}/config.json"
    
    cp install "releases/${package}/install"

    (cd releases && zip -rq "${package}.zip" "${package}")

    rm -rf "releases/${package}"
    echo -e "${GREEN}==> 完成 ${package}${NC}"
}

# 选择构建目标
TARGETS=("${ALL_TARGETS[@]}")
if [[ "${2:-}" != "all" ]]; then
    # 默认只构建 amd64 的 mac 和 linux
    TARGETS=(
        "darwin amd64 macOS"
        # "linux amd64 linux"
    )
fi

echo -e "${GREEN}========== 开始构建 version=${VERSION} ==========${NC}"
for target in "${TARGETS[@]}"; do
    build $target
done

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))
echo -e "${GREEN}========== 构建完成，用时 ${ELAPSED}s ==========${NC}"
