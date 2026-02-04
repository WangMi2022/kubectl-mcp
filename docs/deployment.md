# kubectl-mcp Server 部署指南

本文档提供 kubectl-mcp 服务器的完整部署指南，包括本地部署、Docker 部署、与 admin-mit-backend 集成以及详细的配置参数说明。

## 目录

- [前置要求](#前置要求)
- [本地部署](#本地部署)
- [Docker 部署](#docker-部署)
- [与 admin-mit-backend 集成](#与-admin-mit-backend-集成)
- [配置参数说明](#配置参数说明)
- [安全最佳实践](#安全最佳实践)
- [故障排查](#故障排查)

---

## 前置要求

### 系统要求

- **操作系统**：Linux、macOS 或 Windows
- **Go 版本**：1.21 或更高版本（仅本地部署需要）
- **Docker**：20.10 或更高版本（Docker 部署需要）
- **Kubernetes 集群**：任何版本的 Kubernetes 集群
- **网络**：能够访问 Kubernetes API Server

### 必需文件

- **kubeconfig 文件**：有效的 Kubernetes 配置文件
  - 默认路径：`~/.kube/config`
  - 必须包含至少一个有效的 context
  - 必须具有适当的集群访问权限

### 权限要求

kubectl-mcp 服务器需要的 Kubernetes RBAC 权限取决于您要执行的操作：

- **查询操作**：需要 `get`、`list`、`watch` 权限
- **创建操作**：需要 `create` 权限
- **修改操作**：需要 `update`、`patch` 权限
- **删除操作**：需要 `delete` 权限

建议为 kubectl-mcp 创建专用的 ServiceAccount 和 Role/ClusterRole。

---

## 本地部署

### 1. 克隆代码仓库

```bash
git clone <repository-url>
cd kubectl-mcp
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置服务器

创建配置文件（可选）：

```bash
cp config.yaml.example config.yaml
```

编辑 `config.yaml` 根据您的需求修改配置：

```yaml
# HTTP 服务器配置
host: "0.0.0.0"
port: 8080

# Kubeconfig 配置
kubeconfigPath: "~/.kube/config"
defaultContext: ""  # 留空则使用 kubeconfig 中的 current-context

# 日志配置
logLevel: "info"
logFormat: "json"
logFile: "./logs/kubectl-mcp.log"

# 性能配置
maxConcurrentRequests: 100
requestTimeout: "30s"

# 安全配置
apiToken: "your-secure-token-here"
allowedOrigins:
  - "*"

# 缓存配置
enableCache: true
cacheTTL: "5m"
```

### 4. 编译服务器

```bash
go build -o kubectl-mcp cmd/server/main.go
```

### 5. 运行服务器

#### 方式 1：使用配置文件

```bash
./kubectl-mcp --config config.yaml
```

#### 方式 2：使用环境变量

```bash
export KUBECONFIG=~/.kube/config
export KUBECTL_MCP_PORT=8080
export KUBECTL_MCP_LOGLEVEL=info
export KUBECTL_MCP_APITOKEN=your-secure-token-here

./kubectl-mcp
```

#### 方式 3：使用命令行参数

```bash
./kubectl-mcp \
  --port 8080 \
  --kubeconfig ~/.kube/config \
  --loglevel info \
  --apitoken your-secure-token-here
```

### 6. 验证部署

检查服务器是否正常运行：

```bash
# 健康检查
curl http://localhost:8080/health

# 获取工具列表
curl http://localhost:8080/tools

# 获取 context 列表
curl http://localhost:8080/contexts
```

预期响应：

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "contexts": ["prod-cluster", "dev-cluster"]
}
```

---

## Docker 部署

### 1. 构建 Docker 镜像

```bash
cd kubectl-mcp
docker build -t kubectl-mcp:latest .
```

构建参数说明：
- 使用多阶段构建，最终镜像基于 Alpine Linux
- 镜像大小约 20-30 MB
- 以非 root 用户运行（用户 ID: 1000）

### 2. 准备 kubeconfig 文件

确保您的 kubeconfig 文件可用：

```bash
# 检查 kubeconfig 文件
ls -la ~/.kube/config

# 验证 kubeconfig 有效性
kubectl config view
```

### 3. 运行容器

#### 方式 1：使用 docker run

```bash
docker run -d \
  --name kubectl-mcp-server \
  -p 8080:8080 \
  -v ~/.kube/config:/app/kubeconfig/config:ro \
  -e KUBECONFIG=/app/kubeconfig/config \
  -e KUBECTL_MCP_LOGLEVEL=info \
  -e KUBECTL_MCP_APITOKEN=your-secure-token-here \
  kubectl-mcp:latest
```

参数说明：
- `-d`：后台运行
- `--name`：容器名称
- `-p 8080:8080`：端口映射
- `-v ~/.kube/config:/app/kubeconfig/config:ro`：挂载 kubeconfig（只读）
- `-e`：环境变量配置

#### 方式 2：使用 docker-compose

创建 `docker-compose.yml` 文件（项目已提供）：

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f kubectl-mcp

# 停止服务
docker-compose down
```

### 4. 验证容器运行

```bash
# 查看容器状态
docker ps | grep kubectl-mcp

# 查看容器日志
docker logs kubectl-mcp-server

# 检查健康状态
curl http://localhost:8080/health
```

### 5. 容器管理

```bash
# 停止容器
docker stop kubectl-mcp-server

# 启动容器
docker start kubectl-mcp-server

# 重启容器
docker restart kubectl-mcp-server

# 删除容器
docker rm -f kubectl-mcp-server

# 查看容器资源使用
docker stats kubectl-mcp-server
```

---

## 与 admin-mit-backend 集成

kubectl-mcp 可以作为独立的 MCP 服务器与 admin-mit-backend 系统集成，为 AI 运维助手提供 Kubernetes 操作能力。

### 集成架构

```
┌─────────────────────────────────────────────────────────────┐
│                    admin-mit-backend                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           AI Assistant / MCP Client                   │   │
│  │  - 接收用户请求                                        │   │
│  │  - 调用 kubectl-mcp 工具                              │   │
│  │  - 处理响应并展示                                      │   │
│  └────────────────────┬─────────────────────────────────┘   │
│                       │ HTTP POST/GET                        │
└───────────────────────┼──────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                  kubectl-mcp Server                          │
│  - 接收 MCP 工具调用请求                                      │
│  - 执行 Kubernetes 操作                                      │
│  - 返回结构化响应                                            │
│  - 记录审计日志                                              │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Kubernetes Clusters                             │
└─────────────────────────────────────────────────────────────┘
```

### 1. 部署 kubectl-mcp 服务器

#### 选项 A：在 admin-mit-backend 同一主机上部署

```bash
# 使用 Docker 部署
docker run -d \
  --name kubectl-mcp-server \
  --network admin-mit-network \
  -p 8080:8080 \
  -v ~/.kube/config:/app/kubeconfig/config:ro \
  -e KUBECONFIG=/app/kubeconfig/config \
  -e KUBECTL_MCP_APITOKEN=your-secure-token-here \
  kubectl-mcp:latest
```

#### 选项 B：独立主机部署

```bash
# 在独立主机上部署
docker run -d \
  --name kubectl-mcp-server \
  -p 8080:8080 \
  -v ~/.kube/config:/app/kubeconfig/config:ro \
  -e KUBECONFIG=/app/kubeconfig/config \
  -e KUBECTL_MCP_APITOKEN=your-secure-token-here \
  kubectl-mcp:latest
```

### 2. 配置 admin-mit-backend

在 admin-mit-backend 中配置 kubectl-mcp 服务器连接信息。

#### 修改 MCP 配置文件

编辑 `admin-mit-backend/config/mcp.json`：

```json
{
  "mcpServers": {
    "kubectl-mcp": {
      "type": "http",
      "url": "http://localhost:8080",
      "apiToken": "your-secure-token-here",
      "timeout": 30,
      "enabled": true,
      "description": "Kubernetes 运维工具服务器"
    }
  }
}
```

如果 kubectl-mcp 部署在独立主机：

```json
{
  "mcpServers": {
    "kubectl-mcp": {
      "type": "http",
      "url": "http://kubectl-mcp-host:8080",
      "apiToken": "your-secure-token-here",
      "timeout": 30,
      "enabled": true,
      "description": "Kubernetes 运维工具服务器"
    }
  }
}
```

### 3. 在 admin-mit-backend 中调用 kubectl-mcp

#### Python 调用示例

创建 MCP 客户端模块 `admin-mit-backend/app/services/kubectl_mcp_client.py`：

```python
import requests
from typing import Dict, Any, Optional
import logging

logger = logging.getLogger(__name__)

class KubectlMCPClient:
    """kubectl-mcp 服务器客户端"""
    
    def __init__(self, base_url: str, api_token: str, timeout: int = 30):
        self.base_url = base_url.rstrip('/')
        self.api_token = api_token
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {api_token}',
            'Content-Type': 'application/json'
        })
    
    def call_tool(self, tool_name: str, arguments: Dict[str, Any], 
                  user_info: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
        """
        调用 kubectl-mcp 工具
        
        Args:
            tool_name: 工具名称，如 'get_pods'
            arguments: 工具参数，如 {'namespace': 'default'}
            user_info: 用户信息，如 {'id': 'user123', 'name': 'admin'}
        
        Returns:
            工具执行结果
        
        Raises:
            requests.RequestException: 请求失败
        """
        url = f"{self.base_url}/tool"
        payload = {
            "tool": tool_name,
            "arguments": arguments,
            "user": user_info or {}
        }
        
        try:
            response = self.session.post(url, json=payload, timeout=self.timeout)
            response.raise_for_status()
            return response.json()
        except requests.RequestException as e:
            logger.error(f"kubectl-mcp 调用失败: {e}")
            raise
    
    def list_tools(self) -> Dict[str, Any]:
        """获取所有可用工具列表"""
        url = f"{self.base_url}/tools"
        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()
        return response.json()
    
    def list_contexts(self) -> Dict[str, Any]:
        """获取所有可用的 Kubernetes context"""
        url = f"{self.base_url}/contexts"
        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()
        return response.json()
    
    def health_check(self) -> Dict[str, Any]:
        """健康检查"""
        url = f"{self.base_url}/health"
        response = self.session.get(url, timeout=self.timeout)
        response.raise_for_status()
        return response.json()

# 使用示例
def example_usage():
    # 初始化客户端
    client = KubectlMCPClient(
        base_url="http://localhost:8080",
        api_token="your-secure-token-here"
    )
    
    # 查询 Pods
    result = client.call_tool(
        tool_name="get_pods",
        arguments={
            "namespace": "default",
            "context": "prod-cluster"
        },
        user_info={
            "id": "user123",
            "name": "admin",
            "role": "admin"
        }
    )
    
    if result['success']:
        pods = result['data']['pods']
        print(f"找到 {len(pods)} 个 Pod")
    else:
        print(f"错误: {result['error']['message']}")
```

#### 集成到 AI 助手 API

在 `admin-mit-backend/app/api/ai_ops_assistant.py` 中集成：

```python
from flask import Blueprint, request, jsonify
from app.services.kubectl_mcp_client import KubectlMCPClient
from app.core.auth_middleware import require_auth

ai_ops_bp = Blueprint('ai_ops', __name__)

# 初始化 kubectl-mcp 客户端
kubectl_client = KubectlMCPClient(
    base_url="http://localhost:8080",
    api_token="your-secure-token-here"
)

@ai_ops_bp.route('/k8s/execute', methods=['POST'])
@require_auth
def execute_k8s_operation():
    """执行 Kubernetes 操作"""
    data = request.get_json()
    
    tool_name = data.get('tool')
    arguments = data.get('arguments', {})
    
    # 获取当前用户信息
    user_info = {
        'id': request.user.id,
        'name': request.user.username,
        'role': request.user.role
    }
    
    try:
        # 调用 kubectl-mcp
        result = kubectl_client.call_tool(
            tool_name=tool_name,
            arguments=arguments,
            user_info=user_info
        )
        
        return jsonify(result), 200
    except Exception as e:
        return jsonify({
            'success': False,
            'error': {
                'code': 'INTERNAL_ERROR',
                'message': str(e)
            }
        }), 500

@ai_ops_bp.route('/k8s/tools', methods=['GET'])
@require_auth
def list_k8s_tools():
    """获取可用的 Kubernetes 工具列表"""
    try:
        tools = kubectl_client.list_tools()
        return jsonify(tools), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@ai_ops_bp.route('/k8s/contexts', methods=['GET'])
@require_auth
def list_k8s_contexts():
    """获取可用的 Kubernetes context"""
    try:
        contexts = kubectl_client.list_contexts()
        return jsonify(contexts), 200
    except Exception as e:
        return jsonify({'error': str(e)}), 500
```

### 4. 验证集成

#### 测试连接

```bash
# 从 admin-mit-backend 容器内测试
docker exec -it admin-mit-backend bash
curl http://kubectl-mcp-server:8080/health
```

#### 测试工具调用

```python
# 在 admin-mit-backend 中运行测试脚本
from app.services.kubectl_mcp_client import KubectlMCPClient

client = KubectlMCPClient(
    base_url="http://localhost:8080",
    api_token="your-secure-token-here"
)

# 测试健康检查
health = client.health_check()
print(f"健康状态: {health}")

# 测试获取工具列表
tools = client.list_tools()
print(f"可用工具数量: {len(tools['tools'])}")

# 测试查询 Pods
result = client.call_tool(
    tool_name="get_pods",
    arguments={"namespace": "default"}
)
print(f"查询结果: {result}")
```

### 5. 网络配置

#### 同一 Docker 网络

如果 admin-mit-backend 和 kubectl-mcp 都使用 Docker 部署，建议使用同一网络：

```bash
# 创建共享网络
docker network create admin-mit-network

# 启动 kubectl-mcp
docker run -d \
  --name kubectl-mcp-server \
  --network admin-mit-network \
  -p 8080:8080 \
  -v ~/.kube/config:/app/kubeconfig/config:ro \
  -e KUBECONFIG=/app/kubeconfig/config \
  kubectl-mcp:latest

# 在 admin-mit-backend 中使用容器名访问
# URL: http://kubectl-mcp-server:8080
```

#### 防火墙配置

如果部署在不同主机，确保防火墙允许访问：

```bash
# 允许 8080 端口
sudo ufw allow 8080/tcp

# 或者仅允许特定 IP
sudo ufw allow from <admin-backend-ip> to any port 8080
```

---

## 配置参数说明

kubectl-mcp 支持三种配置方式，优先级为：**命令行参数 > 环境变量 > 配置文件**

### HTTP 服务器配置

| 参数 | 配置文件 | 环境变量 | 命令行参数 | 默认值 | 说明 |
|------|---------|---------|-----------|--------|------|
| 监听地址 | `host` | `KUBECTL_MCP_HOST` | `--host` | `0.0.0.0` | HTTP 服务器监听地址 |
| 监听端口 | `port` | `KUBECTL_MCP_PORT` | `--port` | `8080` | HTTP 服务器监听端口 |

### Kubeconfig 配置

| 参数 | 配置文件 | 环境变量 | 命令行参数 | 默认值 | 说明 |
|------|---------|---------|-----------|--------|------|
| Kubeconfig 路径 | `kubeconfigPath` | `KUBECONFIG` 或 `KUBECTL_MCP_KUBECONFIGPATH` | `--kubeconfig` | `~/.kube/config` | Kubernetes 配置文件路径 |
| 默认 Context | `defaultContext` | `KUBECTL_MCP_DEFAULTCONTEXT` | `--context` | 空（使用 kubeconfig 中的 current-context） | 默认使用的 Kubernetes context |

### 日志配置

| 参数 | 配置文件 | 环境变量 | 命令行参数 | 默认值 | 说明 |
|------|---------|---------|-----------|--------|------|
| 日志级别 | `logLevel` | `KUBECTL_MCP_LOGLEVEL` | `--loglevel` | `info` | 日志级别：debug, info, warn, error |
| 日志格式 | `logFormat` | `KUBECTL_MCP_LOGFORMAT` | `--logformat` | `json` | 日志格式：json, text |
| 日志文件 | `logFile` | `KUBECTL_MCP_LOGFILE` | `--logfile` | 空（输出到 stdout） | 日志文件路径 |

### 性能配置

| 参数 | 配置文件 | 环境变量 | 命令行参数 | 默认值 | 说明 |
|------|---------|---------|-----------|--------|------|
| 最大并发请求数 | `maxConcurrentRequests` | `KUBECTL_MCP_MAXCONCURRENTREQUESTS` | `--max-concurrent` | `100` | 同时处理的最大请求数 |
| 请求超时时间 | `requestTimeout` | `KUBECTL_MCP_REQUESTTIMEOUT` | `--timeout` | `30s` | 单个请求的超时时间 |

### 安全配置

| 参数 | 配置文件 | 环境变量 | 命令行参数 | 默认值 | 说明 |
|------|---------|---------|-----------|--------|------|
| API Token | `apiToken` | `KUBECTL_MCP_APITOKEN` | `--apitoken` | 空（不启用认证） | API 认证 Token |
| 允许的来源 | `allowedOrigins` | `KUBECTL_MCP_ALLOWEDORIGINS` | `--allowed-origins` | `["*"]` | CORS 允许的来源列表 |

### 缓存配置

| 参数 | 配置文件 | 环境变量 | 命令行参数 | 默认值 | 说明 |
|------|---------|---------|-----------|--------|------|
| 启用缓存 | `enableCache` | `KUBECTL_MCP_ENABLECACHE` | `--enable-cache` | `true` | 是否启用查询结果缓存 |
| 缓存 TTL | `cacheTTL` | `KUBECTL_MCP_CACHETTL` | `--cache-ttl` | `5m` | 缓存过期时间 |

### 配置示例

#### 完整配置文件示例

```yaml
# config.yaml
# HTTP 服务器配置
host: "0.0.0.0"
port: 8080

# Kubeconfig 配置
kubeconfigPath: "/app/kubeconfig/config"
defaultContext: "prod-cluster"

# 日志配置
logLevel: "info"
logFormat: "json"
logFile: "/app/logs/kubectl-mcp.log"

# 性能配置
maxConcurrentRequests: 100
requestTimeout: "30s"

# 安全配置
apiToken: "your-secure-token-here"
allowedOrigins:
  - "https://admin.example.com"
  - "http://localhost:3000"

# 缓存配置
enableCache: true
cacheTTL: "5m"
```

#### 环境变量配置示例

```bash
# .env 文件
KUBECTL_MCP_HOST=0.0.0.0
KUBECTL_MCP_PORT=8080
KUBECONFIG=/app/kubeconfig/config
KUBECTL_MCP_DEFAULTCONTEXT=prod-cluster
KUBECTL_MCP_LOGLEVEL=info
KUBECTL_MCP_LOGFORMAT=json
KUBECTL_MCP_LOGFILE=/app/logs/kubectl-mcp.log
KUBECTL_MCP_MAXCONCURRENTREQUESTS=100
KUBECTL_MCP_REQUESTTIMEOUT=30s
KUBECTL_MCP_APITOKEN=your-secure-token-here
KUBECTL_MCP_ALLOWEDORIGINS=https://admin.example.com,http://localhost:3000
KUBECTL_MCP_ENABLECACHE=true
KUBECTL_MCP_CACHETTL=5m
```

#### 命令行参数示例

```bash
./kubectl-mcp \
  --host 0.0.0.0 \
  --port 8080 \
  --kubeconfig /app/kubeconfig/config \
  --context prod-cluster \
  --loglevel info \
  --logformat json \
  --logfile /app/logs/kubectl-mcp.log \
  --max-concurrent 100 \
  --timeout 30s \
  --apitoken your-secure-token-here \
  --allowed-origins "https://admin.example.com,http://localhost:3000" \
  --enable-cache \
  --cache-ttl 5m
```

---

## 安全最佳实践

### 1. Kubeconfig 安全

- **文件权限**：确保 kubeconfig 文件权限为 600
  ```bash
  chmod 600 ~/.kube/config
  ```

- **只读挂载**：在 Docker 中以只读方式挂载 kubeconfig
  ```bash
  -v ~/.kube/config:/app/kubeconfig/config:ro
  ```

- **避免明文凭证**：不要在配置文件或环境变量中存储明文凭证

### 2. API Token 认证

- **启用 API Token**：在生产环境中始终启用 API Token 认证
  ```bash
  export KUBECTL_MCP_APITOKEN=$(openssl rand -hex 32)
  ```

- **定期轮换**：定期更换 API Token

- **安全传输**：使用 HTTPS 传输 API Token

### 3. 网络安全

- **限制访问来源**：配置 `allowedOrigins` 限制 CORS 来源
  ```yaml
  allowedOrigins:
    - "https://admin.example.com"
  ```

- **使用防火墙**：限制只有授权 IP 可以访问
  ```bash
  sudo ufw allow from <trusted-ip> to any port 8080
  ```

- **使用反向代理**：通过 Nginx 或 Traefik 添加 SSL/TLS
  ```nginx
  server {
      listen 443 ssl;
      server_name kubectl-mcp.example.com;
      
      ssl_certificate /etc/ssl/certs/kubectl-mcp.crt;
      ssl_certificate_key /etc/ssl/private/kubectl-mcp.key;
      
      location / {
          proxy_pass http://localhost:8080;
          proxy_set_header Host $host;
          proxy_set_header X-Real-IP $remote_addr;
      }
  }
  ```

### 4. RBAC 权限控制

为 kubectl-mcp 创建专用的 ServiceAccount 和 Role：

```yaml
# kubectl-mcp-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubectl-mcp
  namespace: default

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kubectl-mcp-role
rules:
  # 查询权限
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets", "namespaces", "nodes", "events"]
    verbs: ["get", "list", "watch"]
  
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets", "daemonsets"]
    verbs: ["get", "list", "watch"]
  
  # 创建权限
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets", "namespaces"]
    verbs: ["create"]
  
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["create"]
  
  # 修改权限
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets"]
    verbs: ["update", "patch"]
  
  # 删除权限
  - apiGroups: [""]
    resources: ["pods", "services", "configmaps", "secrets"]
    verbs: ["delete"]
  
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["delete"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubectl-mcp-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kubectl-mcp-role
subjects:
  - kind: ServiceAccount
    name: kubectl-mcp
    namespace: default
```

应用 RBAC 配置：

```bash
kubectl apply -f kubectl-mcp-rbac.yaml
```

### 5. 审计日志

- **启用审计日志**：确保所有操作都被记录
  ```yaml
  logLevel: "info"
  logFile: "/app/logs/kubectl-mcp.log"
  ```

- **日志轮转**：配置日志轮转避免磁盘占满
  ```bash
  # 使用 logrotate
  cat > /etc/logrotate.d/kubectl-mcp << EOF
  /app/logs/kubectl-mcp.log {
      daily
      rotate 7
      compress
      delaycompress
      missingok
      notifempty
  }
  EOF
  ```

- **集中日志管理**：将日志发送到集中式日志系统（如 ELK、Loki）

---

## 故障排查

### 常见问题

#### 1. 服务器无法启动

**症状**：服务器启动失败或立即退出

**可能原因**：
- Kubeconfig 文件不存在或无效
- 端口被占用
- 配置文件格式错误

**解决方法**：

```bash
# 检查 kubeconfig 文件
ls -la ~/.kube/config
kubectl config view

# 检查端口占用
netstat -tuln | grep 8080
lsof -i :8080

# 检查配置文件
cat config.yaml

# 查看详细日志
./kubectl-mcp --loglevel debug
```

#### 2. 无法连接到 Kubernetes 集群

**症状**：工具调用返回连接错误

**可能原因**：
- Kubeconfig 配置错误
- 网络不通
- 集群证书过期
- 权限不足

**解决方法**：

```bash
# 测试 kubectl 连接
kubectl cluster-info
kubectl get nodes

# 检查 context
kubectl config current-context
kubectl config get-contexts

# 测试网络连通性
ping <k8s-api-server-ip>
telnet <k8s-api-server-ip> 6443

# 检查证书有效期
openssl x509 -in ~/.kube/config -text -noout | grep "Not After"
```

#### 3. API 调用返回 401 Unauthorized

**症状**：HTTP 请求返回 401 错误

**可能原因**：
- API Token 未配置或错误
- Authorization Header 格式错误

**解决方法**：

```bash
# 检查 API Token 配置
echo $KUBECTL_MCP_APITOKEN

# 测试不带 Token 的请求（应该失败）
curl http://localhost:8080/health

# 测试带 Token 的请求
curl -H "Authorization: Bearer your-token-here" http://localhost:8080/health

# 检查服务器日志
docker logs kubectl-mcp-server | grep "401"
```

#### 4. 工具调用超时

**症状**：工具调用长时间无响应或返回超时错误

**可能原因**：
- Kubernetes API 响应慢
- 查询数据量过大
- 超时时间设置过短

**解决方法**：

```bash
# 增加超时时间
export KUBECTL_MCP_REQUESTTIMEOUT=60s

# 或在配置文件中修改
requestTimeout: "60s"

# 检查 Kubernetes API 响应时间
time kubectl get pods --all-namespaces

# 使用分页查询大量数据
# 在工具参数中添加 limit 参数
```

#### 5. Docker 容器无法访问 kubeconfig

**症状**：容器内无法读取 kubeconfig 文件

**可能原因**：
- Volume 挂载路径错误
- 文件权限问题
- SELinux 阻止访问

**解决方法**：

```bash
# 检查 Volume 挂载
docker inspect kubectl-mcp-server | grep Mounts -A 10

# 进入容器检查文件
docker exec -it kubectl-mcp-server ls -la /app/kubeconfig/

# 检查 SELinux（如果启用）
getenforce
# 临时禁用 SELinux 测试
sudo setenforce 0

# 正确的挂载方式
docker run -d \
  -v ~/.kube/config:/app/kubeconfig/config:ro \
  -e KUBECONFIG=/app/kubeconfig/config \
  kubectl-mcp:latest
```

#### 6. 内存或 CPU 使用过高

**症状**：服务器资源占用过高

**可能原因**：
- 并发请求过多
- 缓存数据过大
- 内存泄漏

**解决方法**：

```bash
# 限制并发请求数
export KUBECTL_MCP_MAXCONCURRENTREQUESTS=50

# 禁用缓存或减少 TTL
export KUBECTL_MCP_ENABLECACHE=false
# 或
export KUBECTL_MCP_CACHETTL=1m

# 设置 Docker 资源限制
docker run -d \
  --memory="512m" \
  --cpus="1.0" \
  kubectl-mcp:latest

# 监控资源使用
docker stats kubectl-mcp-server
```

### 日志分析

#### 查看实时日志

```bash
# Docker 容器日志
docker logs -f kubectl-mcp-server

# 本地部署日志
tail -f /app/logs/kubectl-mcp.log

# 使用 jq 格式化 JSON 日志
docker logs kubectl-mcp-server | jq '.'
```

#### 过滤特定日志

```bash
# 查看错误日志
docker logs kubectl-mcp-server | grep "error"

# 查看特定工具的调用
docker logs kubectl-mcp-server | grep "get_pods"

# 查看特定用户的操作
docker logs kubectl-mcp-server | jq 'select(.user.id == "user123")'

# 查看失败的操作
docker logs kubectl-mcp-server | jq 'select(.success == false)'
```

### 健康检查

```bash
# 基本健康检查
curl http://localhost:8080/health

# 检查工具列表
curl http://localhost:8080/tools | jq '.tools | length'

# 检查 context 列表
curl http://localhost:8080/contexts | jq '.contexts'

# 测试工具调用
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token-here" \
  -d '{
    "tool": "get_namespaces",
    "arguments": {},
    "user": {"id": "test", "name": "test"}
  }' | jq '.'
```

---

## 性能优化建议

### 1. 启用缓存

对于频繁查询的数据，启用缓存可以显著提升性能：

```yaml
enableCache: true
cacheTTL: "5m"  # 根据数据更新频率调整
```

### 2. 调整并发限制

根据服务器资源和集群负载调整并发请求数：

```yaml
maxConcurrentRequests: 100  # 根据实际情况调整
```

### 3. 使用连接池

kubectl-mcp 内置了 Kubernetes 客户端连接池，无需额外配置。

### 4. 优化查询参数

- 使用 namespace 过滤减少查询范围
- 使用 label selector 精确查询
- 避免查询所有 namespace 的资源

```json
{
  "tool": "get_pods",
  "arguments": {
    "namespace": "default",  // 指定 namespace
    "labelSelector": "app=nginx"  // 使用 label 过滤
  }
}
```

### 5. 监控和告警

使用 Prometheus 监控 kubectl-mcp 性能指标：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'kubectl-mcp'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

---

## 生产环境部署清单

### 部署前检查

- [ ] Kubeconfig 文件已准备并验证有效
- [ ] API Token 已生成并安全存储
- [ ] 网络连通性已测试
- [ ] RBAC 权限已配置
- [ ] 日志目录已创建并有写权限
- [ ] 防火墙规则已配置
- [ ] SSL/TLS 证书已准备（如使用 HTTPS）
- [ ] 监控和告警已配置

### 部署步骤

1. **准备环境**
   ```bash
   # 创建必要的目录
   mkdir -p /app/kubectl-mcp/logs
   mkdir -p /app/kubectl-mcp/config
   
   # 复制 kubeconfig
   cp ~/.kube/config /app/kubectl-mcp/config/kubeconfig
   chmod 600 /app/kubectl-mcp/config/kubeconfig
   ```

2. **配置服务**
   ```bash
   # 创建配置文件
   cat > /app/kubectl-mcp/config/config.yaml << EOF
   host: "0.0.0.0"
   port: 8080
   kubeconfigPath: "/app/config/kubeconfig"
   logLevel: "info"
   logFormat: "json"
   logFile: "/app/logs/kubectl-mcp.log"
   maxConcurrentRequests: 100
   requestTimeout: "30s"
   apiToken: "$(openssl rand -hex 32)"
   enableCache: true
   cacheTTL: "5m"
   EOF
   ```

3. **启动服务**
   ```bash
   # 使用 docker-compose
   cd /app/kubectl-mcp
   docker-compose up -d
   
   # 或使用 docker run
   docker run -d \
     --name kubectl-mcp-server \
     --restart unless-stopped \
     -p 8080:8080 \
     -v /app/kubectl-mcp/config/kubeconfig:/app/kubeconfig/config:ro \
     -v /app/kubectl-mcp/logs:/app/logs \
     -e KUBECONFIG=/app/kubeconfig/config \
     -e KUBECTL_MCP_APITOKEN=$(cat /app/kubectl-mcp/config/config.yaml | grep apiToken | cut -d'"' -f2) \
     kubectl-mcp:latest
   ```

4. **验证部署**
   ```bash
   # 检查容器状态
   docker ps | grep kubectl-mcp
   
   # 检查健康状态
   curl http://localhost:8080/health
   
   # 检查日志
   docker logs kubectl-mcp-server
   ```

5. **配置监控**
   ```bash
   # 配置 Prometheus 监控
   # 配置日志收集
   # 配置告警规则
   ```

### 部署后验证

- [ ] 健康检查接口正常响应
- [ ] 工具列表接口正常响应
- [ ] Context 列表接口正常响应
- [ ] 测试工具调用成功
- [ ] 审计日志正常记录
- [ ] 监控指标正常采集
- [ ] 告警规则正常触发

---

## 升级和维护

### 升级步骤

1. **备份配置**
   ```bash
   cp config.yaml config.yaml.backup
   cp ~/.kube/config ~/.kube/config.backup
   ```

2. **拉取新镜像**
   ```bash
   docker pull kubectl-mcp:latest
   ```

3. **停止旧容器**
   ```bash
   docker stop kubectl-mcp-server
   ```

4. **启动新容器**
   ```bash
   docker run -d \
     --name kubectl-mcp-server \
     -p 8080:8080 \
     -v ~/.kube/config:/app/kubeconfig/config:ro \
     -e KUBECONFIG=/app/kubeconfig/config \
     kubectl-mcp:latest
   ```

5. **验证升级**
   ```bash
   curl http://localhost:8080/health
   docker logs kubectl-mcp-server
   ```

### 日常维护

- **日志清理**：定期清理旧日志文件
- **监控检查**：定期检查监控指标和告警
- **性能优化**：根据使用情况调整配置参数
- **安全更新**：及时更新 kubectl-mcp 和依赖库
- **备份配置**：定期备份配置文件和 kubeconfig

---

## 支持和反馈

如果在部署过程中遇到问题，请：

1. 查看本文档的故障排查部分
2. 检查服务器日志获取详细错误信息
3. 访问项目 GitHub 仓库提交 Issue
4. 联系技术支持团队

---

**文档版本**：1.0.0  
**最后更新**：2025-01-23
