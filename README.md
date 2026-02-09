# kubectl-mcp Server

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

kubectl-mcp 是一个基于 Model Context Protocol (MCP) 的独立 Kubernetes 运维工具服务器，使用 Go 语言开发。该工具作为 AI 运维助手的后端服务，通过 kubeconfig 文件提供安全、可控、可审计的 Kubernetes 集群访问能力。

## 项目介绍

kubectl-mcp 旨在为 AI 运维助手提供标准化的 Kubernetes 操作接口。通过实现 MCP 协议，AI 助手可以安全地执行各种 K8S 运维操作，包括资源查询、创建、更新和删除。

### 设计理念

- **安全第一**：所有集群访问必须基于 kubeconfig 文件，禁止使用明文 token 或证书
- **操作可控**：危险操作（CREATE/UPDATE/DELETE）需要用户确认，防止误操作
- **完整审计**：记录所有操作的详细日志，包括用户、时间、参数和结果
- **多集群管理**：支持同时管理多个 Kubernetes 集群和 context
- **标准协议**：实现标准 MCP 协议，与任何 MCP 客户端无缝集成
- **高性能设计**：支持并发请求处理、连接池管理和流式数据返回

### 适用场景

- AI 运维助手的 Kubernetes 操作后端
- 多集群 Kubernetes 管理工具
- Kubernetes 操作审计和监控
- 自动化运维平台的 K8S 接口层

## 核心特性

### 🔒 安全性

- ✅ 仅通过 kubeconfig 文件访问集群
- ✅ 禁止明文 token、证书或直接 API 访问
- ✅ API Token 认证保护 HTTP 接口
- ✅ 操作权限基于 kubeconfig 中的 RBAC 配置
- ✅ 敏感数据（如 Secret）自动脱敏处理

### 🎯 功能完整性

- ✅ **查询操作**：Pods, Deployments, Services, ConfigMaps, Secrets 等
- ✅ **创建操作**：支持通过参数或 YAML 创建资源
- ✅ **更新操作**：扩缩容、镜像更新、重启、Patch 等
- ✅ **删除操作**：单个或批量删除资源
- ✅ **日志查看**：实时查看 Pod 日志
- ✅ **事件查询**：查看集群和资源事件

### 🌐 多集群支持

- ✅ 支持加载多个 kubeconfig 文件
- ✅ 支持多个 context 管理
- ✅ 动态切换 context
- ✅ Context 隔离保证操作安全

### 📊 审计与监控

- ✅ 完整的操作审计日志
- ✅ 支持 JSON 和文本格式日志
- ✅ 记录用户、时间、参数、结果
- ✅ 性能指标收集（Prometheus 格式）

### ⚡ 高性能

- ✅ 并发请求处理
- ✅ K8S 客户端连接池
- ✅ 查询结果缓存
- ✅ 大数据流式返回
- ✅ 可配置的并发限制和超时

## 快速开始

### 前置要求

- **Go 1.21+**：用于编译和运行服务器
- **Kubernetes 集群**：需要有可访问的 K8S 集群
- **kubeconfig 文件**：有效的集群访问配置
- **kubectl**（可选）：用于验证集群连接

### 安装

#### 方式 1：从源码编译

```bash
# 克隆仓库
git clone https://github.com/your-org/kubectl-mcp.git
cd kubectl-mcp

# 安装依赖
go mod download

# 编译
go build -o kubectl-mcp cmd/server/main.go

# 运行
export KUBECONFIG=~/.kube/config
./kubectl-mcp --port 8080
```

#### 方式 2：使用 Docker

```bash
# 构建镜像
docker build -t kubectl-mcp:latest .

# 运行容器
docker run -d \
  --name kubectl-mcp \
  -p 8080:8080 \
  -v ~/.kube/config:/app/kubeconfig:ro \
  -e KUBECONFIG=/app/kubeconfig \
  -e KUBECTL_MCP_LOG_LEVEL=info \
  kubectl-mcp:latest
```

#### 方式 3：使用 docker-compose

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 验证安装

```bash
# 检查健康状态
curl http://localhost:8080/health

# 查看可用工具列表
curl http://localhost:8080/tools

# 查看可用的 context
curl http://localhost:8080/contexts
```

### 第一个工具调用

```bash
# 查询 default namespace 的 Pods
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-token" \
  -d '{
    "tool": "get_pods",
    "arguments": {
      "namespace": "default"
    },
    "user": {
      "id": "user123",
      "name": "admin"
    }
  }'
```

### 反向查询示例：已知 NodePort 端口号查询 Service 和 Deployment

**场景**：你知道某个服务暴露在 NodePort 30080，需要找到对应的 Service 和 Deployment。

**步骤 1：通过 NodePort 反向查找 Service**

```bash
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "find_service_by_nodeport",
    "arguments": {
      "nodePort": 30080,
      "includeEndpoints": true
    }
  }'
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "service": {
      "name": "nginx-service",
      "namespace": "production",
      "type": "NodePort",
      "port": 80,
      "targetPort": 8080,
      "nodePort": 30080,
      "selector": {
        "app": "nginx"
      }
    },
    "endpoints": [
      {
        "podName": "nginx-deployment-7d64c8d5f9-abc12",
        "podIP": "10.244.1.5",
        "port": 8080
      },
      {
        "podName": "nginx-deployment-7d64c8d5f9-def45",
        "podIP": "10.244.2.10",
        "port": 8080
      }
    ]
  }
}
```

**步骤 2：通过 Service selector 反向查找 Deployment**

```bash
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "find_workload_by_service",
    "arguments": {
      "serviceName": "nginx-service",
      "namespace": "production",
      "includeEndpoints": true
    }
  }'
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "workloads": [
      {
        "kind": "Deployment",
        "name": "nginx-deployment",
        "namespace": "production",
        "replicas": 2,
        "readyReplicas": 2,
        "image": "nginx:1.21",
        "selector": {
          "app": "nginx"
        }
      }
    ],
    "endpoints": [
      {
        "podName": "nginx-deployment-7d64c8d5f9-abc12",
        "podIP": "10.244.1.5",
        "port": 8080
      },
      {
        "podName": "nginx-deployment-7d64c8d5f9-def45",
        "podIP": "10.244.2.10",
        "port": 8080
      }
    ]
  }
}
```

**一步到位：完整链路追踪**

或者使用 `trace_by_nodeport` 工具一次性获取完整链路（NodePort → Service → Workload）：

```bash
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -d '{
    "tool": "trace_by_nodeport",
    "arguments": {
      "nodePort": 30080
    }
  }'
```

**响应示例：**

```json
{
  "success": true,
  "data": {
    "nodePort": 30080,
    "service": {
      "name": "nginx-service",
      "namespace": "production",
      "type": "NodePort",
      "port": 80,
      "targetPort": 8080
    },
    "workloads": [
      {
        "kind": "Deployment",
        "name": "nginx-deployment",
        "namespace": "production",
        "replicas": 2,
        "readyReplicas": 2,
        "image": "nginx:1.21"
      }
    ],
    "endpoints": [
      {
        "podName": "nginx-deployment-7d64c8d5f9-abc12",
        "podIP": "10.244.1.5",
        "port": 8080
      },
      {
        "podName": "nginx-deployment-7d64c8d5f9-def45",
        "podIP": "10.244.2.10",
        "port": 8080
      }
    ]
  }
}
```

## API 接口

kubectl-mcp 提供 RESTful HTTP API 接口，支持 POST 和 GET 方法。

### POST /tool

执行 Kubernetes 工具调用。

**请求头：**
```
Content-Type: application/json
Authorization: Bearer <api-token>
```

**请求体：**

```json
{
  "tool": "get_pods",
  "arguments": {
    "namespace": "default",
    "context": "prod-cluster",
    "labelSelector": "app=nginx"
  },
  "user": {
    "id": "user123",
    "name": "admin",
    "role": "operator"
  }
}
```

**响应示例（成功）：**

```json
{
  "success": true,
  "data": {
    "pods": [
      {
        "name": "nginx-deployment-7d64c8d5f9-abc12",
        "namespace": "default",
        "status": "Running",
        "ip": "10.244.1.5",
        "node": "worker-node-1",
        "labels": {
          "app": "nginx"
        },
        "createdAt": "2026-02-04T10:30:00Z"
      }
    ]
  },
  "error": null
}
```

**响应示例（失败）：**

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "CLIENT_ERROR",
    "code": "RESOURCE_NOT_FOUND",
    "message": "Namespace 'invalid-ns' not found",
    "details": "The specified namespace does not exist in the cluster",
    "suggestion": "Use 'get_namespaces' tool to list available namespaces"
  }
}
```

### GET /tools

获取所有可用工具列表及其元数据。

**响应示例：**

```json
{
  "tools": [
    {
      "name": "get_pods",
      "description": "查询 Pod 列表，支持按 namespace、name、label 过滤",
      "category": "query",
      "requiresConfirmation": false,
      "inputSchema": {
        "type": "object",
        "properties": {
          "namespace": {
            "type": "string",
            "description": "命名空间，默认为 default"
          },
          "name": {
            "type": "string",
            "description": "Pod 名称，支持模糊匹配"
          },
          "labelSelector": {
            "type": "string",
            "description": "标签选择器，例如 app=nginx"
          },
          "context": {
            "type": "string",
            "description": "目标 context，默认使用当前 context"
          }
        }
      }
    },
    {
      "name": "delete_pod",
      "description": "删除指定的 Pod",
      "category": "delete",
      "requiresConfirmation": true,
      "inputSchema": {
        "type": "object",
        "required": ["namespace", "name"],
        "properties": {
          "namespace": {
            "type": "string",
            "description": "命名空间"
          },
          "name": {
            "type": "string",
            "description": "Pod 名称"
          },
          "force": {
            "type": "boolean",
            "description": "是否强制删除（grace period = 0）"
          },
          "context": {
            "type": "string",
            "description": "目标 context"
          }
        }
      }
    }
  ]
}
```

### GET /health

健康检查接口，用于容器编排和监控。

**响应示例：**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": "2h30m15s",
  "contexts": ["prod-cluster", "dev-cluster", "test-cluster"],
  "currentContext": "prod-cluster"
}
```

### GET /contexts

获取所有可用的 Kubernetes context。

**响应示例：**

```json
{
  "contexts": [
    {
      "name": "prod-cluster",
      "cluster": "prod-k8s-cluster",
      "user": "admin",
      "namespace": "default",
      "current": true
    },
    {
      "name": "dev-cluster",
      "cluster": "dev-k8s-cluster",
      "user": "developer",
      "namespace": "development",
      "current": false
    }
  ],
  "currentContext": "prod-cluster"
}
```

## 可用工具列表

kubectl-mcp 提供以下 Kubernetes 操作工具：

### 查询类工具（READ）

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `get_nodes` | 查询 Node 列表 | ❌ |
| `get_namespaces` | 查询 Namespace 列表 | ❌ |
| `get_pods` | 查询 Pod 列表 | ❌ |
| `get_deployments` | 查询 Deployment 列表 | ❌ |
| `get_statefulsets` | 查询 StatefulSet 列表 | ❌ |
| `get_daemonsets` | 查询 DaemonSet 列表 | ❌ |
| `get_services` | 查询 Service 列表 | ❌ |
| `get_configmaps` | 查询 ConfigMap 列表 | ❌ |
| `get_secrets` | 查询 Secret 列表（自动脱敏） | ❌ |
| `describe_resource` | 查看资源详细信息 | ❌ |
| `get_pod_logs` | 查看 Pod 日志 | ❌ |
| `get_events` | 查看集群事件 | ❌ |

### 创建类工具（CREATE）

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `create_namespace` | 创建 Namespace | ✅ |
| `create_pod` | 创建 Pod | ✅ |
| `create_deployment` | 创建 Deployment | ✅ |
| `create_service` | 创建 Service | ✅ |
| `create_configmap` | 创建 ConfigMap | ✅ |
| `create_secret` | 创建 Secret | ✅ |
| `create_from_yaml` | 通过 YAML 创建资源 | ✅ |

### 更新类工具（UPDATE）

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `scale_deployment` | 扩缩容 Deployment | ✅ |
| `scale_statefulset` | 扩缩容 StatefulSet | ✅ |
| `update_deployment_image` | 更新 Deployment 镜像 | ✅ |
| `restart_deployment` | 重启 Deployment | ✅ |
| `apply_yaml` | 通过 YAML 更新资源 | ✅ |
| `patch_resource` | Patch 资源 | ✅ |

### 删除类工具（DELETE）

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `delete_pod` | 删除 Pod | ✅ |
| `delete_deployment` | 删除 Deployment | ✅ |
| `delete_service` | 删除 Service | ✅ |
| `delete_configmap` | 删除 ConfigMap | ✅ |
| `delete_secret` | 删除 Secret | ✅ |
| `delete_namespace` | 删除 Namespace（高危） | ✅ |
| `delete_resources` | 批量删除资源 | ✅ |

### Context 管理工具

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `list_contexts` | 列出所有 context | ❌ |
| `switch_context` | 切换当前 context | ✅ |
| `get_current_context` | 获取当前 context | ❌ |

## 配置说明

kubectl-mcp 支持多种配置方式，优先级从高到低为：**命令行参数 > 环境变量 > 配置文件**

### 配置文件

创建 `config.yaml` 文件：

```yaml
# HTTP 服务器配置
host: "0.0.0.0"
port: 8080

# Kubeconfig 配置
kubeconfigPath: "~/.kube/config"
defaultContext: "prod-cluster"

# 日志配置
logLevel: "info"          # debug, info, warn, error
logFormat: "json"         # json, text
logFile: "/var/log/kubectl-mcp/app.log"

# 性能配置
maxConcurrentRequests: 100
requestTimeout: "30s"

# 安全配置
apiToken: "your-secure-api-token"
allowedOrigins:
  - "*"

# 缓存配置
enableCache: true
cacheTTL: "5m"
```

### 环境变量

所有配置项都可以通过环境变量覆盖，使用 `KUBECTL_MCP_` 前缀：

```bash
# HTTP 服务器
export KUBECTL_MCP_HOST="0.0.0.0"
export KUBECTL_MCP_PORT="8080"

# Kubeconfig
export KUBECONFIG="~/.kube/config"
export KUBECTL_MCP_DEFAULT_CONTEXT="prod-cluster"

# 日志
export KUBECTL_MCP_LOG_LEVEL="info"
export KUBECTL_MCP_LOG_FORMAT="json"
export KUBECTL_MCP_LOG_FILE="/var/log/kubectl-mcp/app.log"

# 安全
export KUBECTL_MCP_API_TOKEN="your-secure-api-token"
```

### 命令行参数

```bash
kubectl-mcp \
  --host 0.0.0.0 \
  --port 8080 \
  --kubeconfig ~/.kube/config \
  --default-context prod-cluster \
  --log-level info \
  --log-format json \
  --api-token your-secure-api-token
```

### 配置项说明

#### HTTP 服务器配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|-------|---------|-------|------|
| `host` | `KUBECTL_MCP_HOST` | `0.0.0.0` | 监听地址 |
| `port` | `KUBECTL_MCP_PORT` | `8080` | 监听端口 |

#### Kubeconfig 配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|-------|---------|-------|------|
| `kubeconfigPath` | `KUBECONFIG` | `~/.kube/config` | kubeconfig 文件路径 |
| `defaultContext` | `KUBECTL_MCP_DEFAULT_CONTEXT` | - | 默认使用的 context |

#### 日志配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|-------|---------|-------|------|
| `logLevel` | `KUBECTL_MCP_LOG_LEVEL` | `info` | 日志级别：debug/info/warn/error |
| `logFormat` | `KUBECTL_MCP_LOG_FORMAT` | `json` | 日志格式：json/text |
| `logFile` | `KUBECTL_MCP_LOG_FILE` | - | 日志文件路径，为空则输出到 stdout |

#### 性能配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|-------|---------|-------|------|
| `maxConcurrentRequests` | `KUBECTL_MCP_MAX_CONCURRENT` | `100` | 最大并发请求数 |
| `requestTimeout` | `KUBECTL_MCP_REQUEST_TIMEOUT` | `30s` | 单个请求超时时间 |

#### 安全配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|-------|---------|-------|------|
| `apiToken` | `KUBECTL_MCP_API_TOKEN` | - | API 访问 Token（强烈建议设置） |
| `allowedOrigins` | `KUBECTL_MCP_ALLOWED_ORIGINS` | `*` | CORS 允许的来源 |

#### 缓存配置

| 配置项 | 环境变量 | 默认值 | 说明 |
|-------|---------|-------|------|
| `enableCache` | `KUBECTL_MCP_ENABLE_CACHE` | `true` | 是否启用查询缓存 |
| `cacheTTL` | `KUBECTL_MCP_CACHE_TTL` | `5m` | 缓存过期时间 |

### Docker 部署配置

使用 Docker 时，推荐通过环境变量和 Volume 挂载配置：

```bash
docker run -d \
  --name kubectl-mcp \
  -p 8080:8080 \
  -v ~/.kube/config:/app/kubeconfig:ro \
  -v /var/log/kubectl-mcp:/var/log/kubectl-mcp \
  -e KUBECONFIG=/app/kubeconfig \
  -e KUBECTL_MCP_LOG_LEVEL=info \
  -e KUBECTL_MCP_LOG_FILE=/var/log/kubectl-mcp/app.log \
  -e KUBECTL_MCP_API_TOKEN=your-secure-token \
  kubectl-mcp:latest
```

### docker-compose 配置

`docker-compose.yml` 示例：

```yaml
version: '3.8'

services:
  kubectl-mcp:
    build: .
    container_name: kubectl-mcp
    ports:
      - "8080:8080"
    volumes:
      - ~/.kube/config:/app/kubeconfig:ro
      - ./logs:/var/log/kubectl-mcp
      - ./config.yaml:/app/config.yaml:ro
    environment:
      - KUBECONFIG=/app/kubeconfig
      - KUBECTL_MCP_LOG_LEVEL=info
      - KUBECTL_MCP_API_TOKEN=${API_TOKEN}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## 开发指南

### 项目结构

```
kubectl-mcp/
├── cmd/
│   └── server/
│       └── main.go              # 服务器入口，初始化和启动逻辑
├── internal/                     # 内部包，不对外暴露
│   ├── config/
│   │   └── config.go            # 配置管理，支持文件/环境变量/命令行
│   ├── server/
│   │   ├── http.go              # HTTP 服务器实现（Gin）
│   │   └── handler.go           # HTTP 请求处理器
│   ├── mcp/
│   │   ├── protocol.go          # MCP 协议处理逻辑
│   │   └── types.go             # MCP 数据类型定义
│   ├── tools/
│   │   ├── registry.go          # 工具注册表和路由
│   │   ├── query.go             # 查询类工具实现（READ）
│   │   ├── create.go            # 创建类工具实现（CREATE）
│   │   ├── update.go            # 更新类工具实现（UPDATE）
│   │   └── delete.go            # 删除类工具实现（DELETE）
│   ├── k8s/
│   │   ├── client.go            # K8S 客户端管理器
│   │   ├── context.go           # Context 管理和切换
│   │   └── types.go             # K8S 资源数据类型
│   └── audit/
│       ├── logger.go            # 审计日志系统
│       └── metrics.go           # 性能指标收集
├── pkg/                          # 公共包，可对外使用
│   └── utils/
│       └── errors.go            # 错误处理工具
├── test/                         # 测试文件
│   ├── *_test.go                # 单元测试
│   ├── integration/             # 集成测试
│   └── property/                # 属性测试
├── docs/                         # 文档
│   ├── api.md                   # API 详细文档
│   └── deployment.md            # 部署指南
├── config.yaml.example          # 配置文件示例
├── Dockerfile                   # Docker 镜像构建
├── docker-compose.yml           # Docker Compose 配置
├── go.mod                       # Go 模块定义
├── go.sum                       # Go 依赖校验
└── README.md                    # 项目说明文档
```

### 技术栈

- **语言**：Go 1.21+
- **HTTP 框架**：[Gin](https://github.com/gin-gonic/gin) - 高性能 HTTP Web 框架
- **K8S 客户端**：[client-go](https://github.com/kubernetes/client-go) v0.28+ - Kubernetes 官方 Go 客户端
- **日志库**：[zap](https://github.com/uber-go/zap) - 高性能结构化日志库
- **配置管理**：[viper](https://github.com/spf13/viper) - 配置解决方案
- **测试框架**：
  - `testing` - Go 标准测试库
  - [testify](https://github.com/stretchr/testify) - 测试断言库
  - [gopter](https://github.com/leanovate/gopter) - 属性测试库

### 开发环境设置

1. **安装 Go 1.21+**

```bash
# 下载并安装 Go
# https://golang.org/dl/

# 验证安装
go version
```

2. **克隆项目**

```bash
git clone https://github.com/your-org/kubectl-mcp.git
cd kubectl-mcp
```

3. **安装依赖**

```bash
go mod download
```

4. **配置 kubeconfig**

```bash
# 确保有可用的 kubeconfig
export KUBECONFIG=~/.kube/config

# 验证集群连接
kubectl cluster-info
```

5. **运行开发服务器**

```bash
# 直接运行
go run cmd/server/main.go --port 8080

# 或编译后运行
go build -o kubectl-mcp cmd/server/main.go
./kubectl-mcp --port 8080
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行单元测试
go test ./internal/...

# 运行特定包的测试
go test ./internal/k8s/...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行集成测试（需要真实 K8S 集群）
go test ./test/integration/...

# 运行属性测试
go test ./test/*_property_test.go -v

# 运行测试时显示详细输出
go test -v ./...
```

### 代码规范

项目遵循 Go 官方代码规范和最佳实践：

1. **格式化代码**

```bash
# 使用 gofmt 格式化
gofmt -w .

# 使用 goimports（推荐）
go install golang.org/x/tools/cmd/goimports@latest
goimports -w .
```

2. **代码检查**

```bash
# 使用 go vet 检查
go vet ./...

# 使用 golangci-lint（推荐）
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run
```

3. **命名规范**
   - 包名：小写，简短，有意义
   - 导出函数/类型：大写开头（PascalCase）
   - 私有函数/变量：小写开头（camelCase）
   - 常量：大写或 PascalCase

4. **注释规范**
   - 导出的函数、类型、常量必须有注释
   - 注释以名称开头，例如：`// GetPods queries pods from Kubernetes cluster`

### 添加新工具

要添加新的 Kubernetes 操作工具，按以下步骤操作：

1. **在 `internal/tools/` 中实现工具函数**

```go
// internal/tools/query.go

// GetReplicaSets 查询 ReplicaSet 列表
func GetReplicaSets(ctx context.Context, args map[string]interface{}, k8sClient *K8SClientManager) (interface{}, error) {
    // 解析参数
    namespace := getStringArg(args, "namespace", "default")
    
    // 获取客户端
    clientset, err := k8sClient.GetClient()
    if err != nil {
        return nil, err
    }
    
    // 调用 K8S API
    rsList, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
    if err != nil {
        return nil, err
    }
    
    // 格式化返回结果
    return formatReplicaSets(rsList), nil
}
```

2. **在 `internal/tools/registry.go` 中注册工具**

```go
// RegisterTools 注册所有工具
func RegisterTools(registry *ToolRegistry) {
    // ... 其他工具注册
    
    // 注册新工具
    registry.RegisterTool(&Tool{
        Name:        "get_replicasets",
        Description: "查询 ReplicaSet 列表，支持按 namespace 过滤",
        Category:    "query",
        RequiresConfirmation: false,
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "namespace": map[string]interface{}{
                    "type":        "string",
                    "description": "命名空间，默认为 default",
                },
            },
        },
        Handler: GetReplicaSets,
    })
}
```

3. **编写单元测试**

```go
// test/query_tools_test.go

func TestGetReplicaSets(t *testing.T) {
    // 创建 fake clientset
    clientset := fake.NewSimpleClientset()
    
    // 创建测试数据
    rs := &appsv1.ReplicaSet{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "test-rs",
            Namespace: "default",
        },
    }
    clientset.AppsV1().ReplicaSets("default").Create(context.TODO(), rs, metav1.CreateOptions{})
    
    // 测试工具调用
    result, err := GetReplicaSets(context.TODO(), map[string]interface{}{
        "namespace": "default",
    }, mockK8SManager)
    
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### 调试技巧

1. **启用 Debug 日志**

```bash
export KUBECTL_MCP_LOG_LEVEL=debug
./kubectl-mcp
```

2. **使用 Delve 调试器**

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 启动调试
dlv debug cmd/server/main.go -- --port 8080
```

3. **查看审计日志**

```bash
# 实时查看日志
tail -f /var/log/kubectl-mcp/app.log

# 使用 jq 格式化 JSON 日志
tail -f /var/log/kubectl-mcp/app.log | jq .
```

### 性能优化建议

1. **启用缓存**：对频繁查询的数据启用缓存
2. **连接池管理**：合理配置 K8S 客户端连接池大小
3. **并发控制**：根据集群规模调整 `maxConcurrentRequests`
4. **超时设置**：根据网络情况调整 `requestTimeout`
5. **日志级别**：生产环境使用 `info` 或 `warn` 级别

### 贡献指南

欢迎贡献代码！请遵循以下流程：

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

**Pull Request 要求：**
- 代码通过所有测试
- 代码符合 Go 规范（通过 `gofmt` 和 `golangci-lint`）
- 添加必要的单元测试
- 更新相关文档
- 提供清晰的 PR 描述

### 常见问题

**Q: 如何处理多个 kubeconfig 文件？**

A: 可以使用 `KUBECONFIG` 环境变量指定多个文件，用冒号分隔：
```bash
export KUBECONFIG=~/.kube/config1:~/.kube/config2
```

**Q: 如何限制工具的访问权限？**

A: kubectl-mcp 的权限完全基于 kubeconfig 中的 RBAC 配置。通过配置 K8S 的 Role 和 RoleBinding 来限制权限。

**Q: 如何监控服务器性能？**

A: 服务器提供 Prometheus 格式的性能指标接口（计划中），可以集成到监控系统。

**Q: 支持哪些 Kubernetes 版本？**

A: 支持 Kubernetes 1.20+ 版本，推荐使用 1.24+ 版本。

## 与 admin-mit-backend 集成

kubectl-mcp 可以作为独立的 MCP 服务器与 admin-mit-backend 集成。详细集成指南请参考 [docs/deployment.md](docs/deployment.md)

## 故障排查

### 常见错误及解决方案

#### 1. Kubeconfig 加载失败

**错误信息：**
```json
{
  "error": {
    "code": "KUBECONFIG_NOT_FOUND",
    "message": "Failed to load kubeconfig"
  }
}
```

**解决方案：**
- 检查 `KUBECONFIG` 环境变量是否正确设置
- 确认 kubeconfig 文件存在且有读取权限
- 验证 kubeconfig 文件格式是否正确：`kubectl config view`

#### 2. 集群连接失败

**错误信息：**
```json
{
  "error": {
    "code": "CLUSTER_UNREACHABLE",
    "message": "Unable to connect to the cluster"
  }
}
```

**解决方案：**
- 检查网络连接：`ping <cluster-api-server>`
- 验证集群可访问性：`kubectl cluster-info`
- 检查防火墙和网络策略
- 确认 kubeconfig 中的证书未过期

#### 3. 权限不足

**错误信息：**
```json
{
  "error": {
    "code": "PERMISSION_DENIED",
    "message": "User does not have permission to list pods"
  }
}
```

**解决方案：**
- 检查 kubeconfig 中用户的 RBAC 权限
- 使用 `kubectl auth can-i` 验证权限：
  ```bash
  kubectl auth can-i list pods --namespace default
  ```
- 联系集群管理员授予必要权限

#### 4. Context 不存在

**错误信息：**
```json
{
  "error": {
    "code": "CONTEXT_NOT_FOUND",
    "message": "Context 'invalid-context' not found"
  }
}
```

**解决方案：**
- 列出所有可用 context：`kubectl config get-contexts`
- 使用 GET `/contexts` 接口查看服务器识别的 context
- 确认 context 名称拼写正确

#### 5. API Token 认证失败

**错误信息：**
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid or missing API token"
  }
}
```

**解决方案：**
- 确认请求头包含正确的 Authorization：`Bearer <token>`
- 检查服务器配置的 `apiToken` 是否与请求匹配
- 确认 token 未过期（如果使用时效性 token）

### 日志分析

**查看详细日志：**
```bash
# 设置 debug 级别
export KUBECTL_MCP_LOG_LEVEL=debug

# 查看实时日志
tail -f /var/log/kubectl-mcp/app.log | jq .
```

**关键日志字段：**
- `timestamp`: 操作时间
- `level`: 日志级别（debug/info/warn/error）
- `tool`: 调用的工具名称
- `user`: 操作用户信息
- `context`: 使用的 K8S context
- `duration`: 操作耗时
- `error`: 错误信息（如果有）

## 性能调优

### 并发配置

根据集群规模和服务器资源调整并发参数：

```yaml
# 小型集群（< 100 nodes）
maxConcurrentRequests: 50
requestTimeout: "30s"

# 中型集群（100-500 nodes）
maxConcurrentRequests: 100
requestTimeout: "60s"

# 大型集群（> 500 nodes）
maxConcurrentRequests: 200
requestTimeout: "120s"
```

### 缓存优化

```yaml
# 启用缓存以减少 API 调用
enableCache: true

# 根据数据变化频率调整 TTL
cacheTTL: "5m"  # 频繁变化的数据（如 Pod 状态）
cacheTTL: "30m" # 相对稳定的数据（如 ConfigMap）
```

### 资源限制

**Docker 部署时的资源限制：**

```yaml
# docker-compose.yml
services:
  kubectl-mcp:
    # ...
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### 监控指标

kubectl-mcp 提供以下性能指标（Prometheus 格式）：

- `kubectl_mcp_requests_total`: 总请求数
- `kubectl_mcp_requests_duration_seconds`: 请求耗时
- `kubectl_mcp_errors_total`: 错误总数
- `kubectl_mcp_concurrent_requests`: 当前并发请求数
- `kubectl_mcp_k8s_api_calls_total`: K8S API 调用次数

## 安全最佳实践

### 1. Kubeconfig 安全

- ✅ 使用只读挂载 kubeconfig：`-v ~/.kube/config:/app/kubeconfig:ro`
- ✅ 限制 kubeconfig 文件权限：`chmod 600 ~/.kube/config`
- ✅ 定期轮换 kubeconfig 中的凭证
- ✅ 使用最小权限原则配置 RBAC

### 2. API Token 保护

- ✅ 使用强随机 token：`openssl rand -hex 32`
- ✅ 通过环境变量传递 token，避免硬编码
- ✅ 定期轮换 API token
- ✅ 在生产环境必须启用 API token 认证

### 3. 网络安全

- ✅ 使用 HTTPS（通过反向代理如 Nginx）
- ✅ 配置 CORS 限制允许的来源
- ✅ 使用防火墙限制访问来源 IP
- ✅ 启用请求速率限制

### 4. 容器安全

- ✅ 以非 root 用户运行容器
- ✅ 使用最小化的基础镜像
- ✅ 定期更新依赖和镜像
- ✅ 扫描镜像漏洞

### 5. 审计与监控

- ✅ 启用完整的操作审计日志
- ✅ 定期审查审计日志
- ✅ 配置告警规则监控异常操作
- ✅ 集成到 SIEM 系统

## 生产部署建议

### 高可用部署

```yaml
# docker-compose.yml - 多实例部署
version: '3.8'

services:
  kubectl-mcp-1:
    image: kubectl-mcp:latest
    # ... 配置

  kubectl-mcp-2:
    image: kubectl-mcp:latest
    # ... 配置

  kubectl-mcp-3:
    image: kubectl-mcp:latest
    # ... 配置

  nginx:
    image: nginx:alpine
    ports:
      - "8080:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - kubectl-mcp-1
      - kubectl-mcp-2
      - kubectl-mcp-3
```

### 使用 Nginx 反向代理

```nginx
# nginx.conf
upstream kubectl_mcp_backend {
    least_conn;
    server kubectl-mcp-1:8080;
    server kubectl-mcp-2:8080;
    server kubectl-mcp-3:8080;
}

server {
    listen 80;
    server_name kubectl-mcp.example.com;

    # HTTPS 重定向
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name kubectl-mcp.example.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    location / {
        proxy_pass http://kubectl_mcp_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # 健康检查
    location /health {
        proxy_pass http://kubectl_mcp_backend/health;
        access_log off;
    }
}
```

### Kubernetes 部署

```yaml
# kubectl-mcp-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kubectl-mcp
  namespace: ops-tools
spec:
  replicas: 3
  selector:
    matchLabels:
      app: kubectl-mcp
  template:
    metadata:
      labels:
        app: kubectl-mcp
    spec:
      serviceAccountName: kubectl-mcp
      containers:
      - name: kubectl-mcp
        image: kubectl-mcp:latest
        ports:
        - containerPort: 8080
        env:
        - name: KUBECTL_MCP_LOG_LEVEL
          value: "info"
        - name: KUBECTL_MCP_API_TOKEN
          valueFrom:
            secretKeyRef:
              name: kubectl-mcp-secret
              key: api-token
        volumeMounts:
        - name: kubeconfig
          mountPath: /app/kubeconfig
          readOnly: true
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            cpu: 500m
            memory: 512Mi
          limits:
            cpu: 2000m
            memory: 2Gi
      volumes:
      - name: kubeconfig
        secret:
          secretName: kubectl-mcp-kubeconfig

---
apiVersion: v1
kind: Service
metadata:
  name: kubectl-mcp
  namespace: ops-tools
spec:
  selector:
    app: kubectl-mcp
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

## 更新日志

### v1.0.0 (2026-02-04)

**新特性：**
- ✅ 完整的 Kubernetes CRUD 操作支持
- ✅ 多集群和多 context 管理
- ✅ 基于 kubeconfig 的安全认证
- ✅ 完整的操作审计日志
- ✅ HTTP RESTful API 接口
- ✅ Docker 和 docker-compose 部署支持
- ✅ 并发请求处理和连接池管理
- ✅ 查询结果缓存机制
- ✅ 详细的错误处理和建议

**已知限制：**
- 暂不支持 WebSocket 实时推送
- 暂不支持自定义资源（CRD）操作
- 暂不支持 Helm Chart 管理

## 路线图

### v1.1.0（计划中）

- [ ] WebSocket 支持，实时推送资源变化
- [ ] 自定义资源（CRD）操作支持
- [ ] Prometheus 指标导出
- [ ] 更细粒度的权限控制
- [ ] 操作回滚功能

### v1.2.0（计划中）

- [ ] Helm Chart 管理工具
- [ ] 资源模板管理
- [ ] 批量操作优化
- [ ] GraphQL API 支持

### v2.0.0（未来）

- [ ] 多租户支持
- [ ] 资源配额管理
- [ ] 成本分析和优化建议
- [ ] AI 驱动的故障诊断

## 相关资源

- **官方文档**：[docs/](docs/)
- **API 文档**：[docs/api.md](docs/api.md)
- **部署指南**：[docs/deployment.md](docs/deployment.md)
- **Kubernetes 官方文档**：https://kubernetes.io/docs/
- **client-go 文档**：https://github.com/kubernetes/client-go
- **MCP 协议规范**：https://modelcontextprotocol.io/

## 社区与支持

- **问题反馈**：[GitHub Issues](https://github.com/wangyufeng1995/kubectl-mcp/issues)
- **功能建议**：[GitHub Discussions](https://github.com/wangyufeng1995/kubectl-mcp/discussions)
- **安全漏洞**：请发送邮件至 wangyufeng@yunlizhihui.com

## 致谢

感谢以下开源项目：

- [Kubernetes](https://kubernetes.io/) - 容器编排平台
- [client-go](https://github.com/kubernetes/client-go) - Kubernetes Go 客户端
- [Gin](https://github.com/gin-gonic/gin) - HTTP Web 框架
- [Zap](https://github.com/uber-go/zap) - 高性能日志库
- [Viper](https://github.com/spf13/viper) - 配置管理库

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 贡献者

感谢所有为本项目做出贡献的开发者！

---

**Made with ❤️ by the kubectl-mcp team**
