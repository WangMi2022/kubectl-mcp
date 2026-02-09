# kubectl-mcp Dockerfile - 多阶段构建，使用国内镜像源优化
# Stage 1: 构建阶段
FROM hub.1panel.dev/library/golang:1.23-alpine AS builder

# 设置工作目录
WORKDIR /build

# 配置 Go 国内镜像源
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 配置 Alpine 国内镜像源并安装构建工具
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates tzdata upx

# 复制 go.mod 和 go.sum 并下载依赖（优化缓存层）
COPY go.mod go.sum ./
RUN go clean -modcache && go mod download && go mod verify

# 复制源代码
COPY . .

# 构建二进制文件
# -ldflags="-w -s" 去除调试信息和符号表，减小体积
# -v 显示详细构建信息
RUN go build -v -ldflags="-w -s" -trimpath -o kubectl-mcp ./cmd/server/main.go


# 使用 UPX 进一步压缩二进制文件（可选，减少约 50% 体积）
RUN upx --best --lzma kubectl-mcp || true

# Stage 2: 运行阶段
FROM hub.1panel.dev/library/alpine:latest

# 配置 Alpine 国内镜像源并安装运行时依赖
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories \
    && apk --no-cache add ca-certificates tzdata wget curl \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata

# 创建非 root 用户
RUN addgroup -g 1000 kubectl-mcp \
    && adduser -D -u 1000 -G kubectl-mcp kubectl-mcp

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/kubectl-mcp /app/kubectl-mcp

# 创建必要的目录并设置权限
RUN mkdir -p /app/logs /app/config /app/data \
    && chown -R kubectl-mcp:kubectl-mcp /app \
    && chmod +x /app/kubectl-mcp

# 复制配置文件（在切换用户之前）
COPY --chown=kubectl-mcp:kubectl-mcp config.yaml /app/config/config.yaml

# 切换到非 root 用户
USER kubectl-mcp  

# 暴露端口
EXPOSE 8080

# 健康检查 - 使用 GET 方法而非 HEAD（--spider）
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/health || exit 1

# 设置环境变量
ENV KUBECTL_MCP_HOST=0.0.0.0 \
    KUBECTL_MCP_PORT=8080 \
    KUBECTL_MCP_LOG_LEVEL=info \
    KUBECTL_MCP_LOG_FORMAT=json \
    TZ=Asia/Shanghai

# 启动服务器
ENTRYPOINT ["/app/kubectl-mcp"]
CMD []

