# kubectl-mcp Server API 文档

## 概述

kubectl-mcp Server 提供基于 HTTP RESTful API 的 Kubernetes 运维工具服务。所有 API 端点都遵循 MCP (Model Context Protocol) 协议规范，支持通过标准 HTTP 请求调用 Kubernetes 操作。

**基础信息：**
- **Base URL**: `http://localhost:8080` (默认)
- **支持的 HTTP 方法**: `POST`, `GET`
- **Content-Type**: `application/json`
- **认证方式**: Bearer Token (通过 `Authorization` Header)

## 认证

所有 API 请求都需要在 HTTP Header 中携带 API Token：

```http
Authorization: Bearer <your-api-token>
```

如果 Token 无效或缺失，服务器将返回 `401 Unauthorized` 错误。

---

## API 端点

### 1. 执行工具调用

执行指定的 Kubernetes 操作工具。

**端点**: `POST /tool`

**请求头**:
```http
Content-Type: application/json
Authorization: Bearer <api-token>
```

**请求体**:
```json
{
  "tool": "string",           // 工具名称（必填）
  "arguments": {              // 工具参数（必填）
    "context": "string",      // K8S Context（可选，默认使用当前 context）
    "namespace": "string",    // 命名空间（可选，默认为 default）
    ...                       // 其他工具特定参数
  },
  "user": {                   // 用户信息（可选）
    "id": "string",
    "name": "string",
    "role": "string"
  }
}
```

**响应体**:
```json
{
  "success": true,            // 操作是否成功
  "data": {                   // 返回数据（成功时）
    ...
  },
  "error": null               // 错误信息（失败时）
}
```

**示例 1: 查询 Pods**

请求:
```bash
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "tool": "get_pods",
    "arguments": {
      "namespace": "default",
      "context": "prod-cluster"
    },
    "user": {
      "id": "user123",
      "name": "admin"
    }
  }'
```

响应:
```json
{
  "success": true,
  "data": {
    "pods": [
      {
        "name": "nginx-deployment-7d64c8f5d9-abc12",
        "namespace": "default",
        "status": "Running",
        "ip": "10.244.1.5",
        "node": "worker-node-1",
        "labels": {
          "app": "nginx"
        },
        "createdAt": "2024-01-20T10:30:00Z"
      }
    ]
  },
  "error": null
}
```

**示例 2: 创建 Deployment**

请求:
```bash
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "tool": "create_deployment",
    "arguments": {
      "name": "nginx-app",
      "namespace": "default",
      "image": "nginx:1.21",
      "replicas": 3,
      "port": 80,
      "labels": {
        "app": "nginx",
        "env": "prod"
      }
    }
  }'
```

响应:
```json
{
  "success": true,
  "data": {
    "message": "Deployment nginx-app 创建成功",
    "deployment": {
      "name": "nginx-app",
      "namespace": "default",
      "replicas": 3,
      "image": "nginx:1.21"
    }
  },
  "error": null
}
```

**示例 3: 扩缩容 Deployment**

请求:
```bash
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{
    "tool": "scale_deployment",
    "arguments": {
      "name": "nginx-app",
      "namespace": "default",
      "replicas": 5
    }
  }'
```

响应:
```json
{
  "success": true,
  "data": {
    "message": "Deployment nginx-app 扩缩容成功，副本数: 5"
  },
  "error": null
}
```

**错误响应示例**:

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "CLIENT_ERROR",
    "code": "RESOURCE_NOT_FOUND",
    "message": "Deployment not found",
    "details": "deployment.apps \"nginx-app\" not found in namespace \"default\"",
    "suggestion": "请检查 Deployment 名称和命名空间是否正确"
  }
}
```

---

### 2. 获取工具列表

获取所有可用的 Kubernetes 操作工具列表。

**端点**: `GET /tools`

**请求头**:
```http
Authorization: Bearer <api-token>
```

**响应体**:
```json
{
  "tools": [
    {
      "name": "get_pods",
      "description": "查询 Pod 列表，支持按命名空间、名称、标签过滤",
      "category": "query",
      "requiresConfirmation": false,
      "inputSchema": {
        "type": "object",
        "properties": {
          "namespace": {
            "type": "string",
            "description": "命名空间"
          },
          "name": {
            "type": "string",
            "description": "Pod 名称"
          },
          "labelSelector": {
            "type": "string",
            "description": "标签选择器"
          }
        }
      }
    }
  ]
}
```

**示例**:

```bash
curl -X GET http://localhost:8080/tools \
  -H "Authorization: Bearer your-token"
```

---

### 3. 健康检查

检查服务器健康状态和可用的 Kubernetes contexts。

**端点**: `GET /health`

**请求头**:
```http
Authorization: Bearer <api-token>
```

**响应体**:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "contexts": ["prod-cluster", "dev-cluster", "test-cluster"]
}
```

**示例**:

```bash
curl -X GET http://localhost:8080/health \
  -H "Authorization: Bearer your-token"
```

---

### 4. 获取 Context 列表

获取所有可用的 Kubernetes contexts 详细信息。

**端点**: `GET /contexts`

**请求头**:
```http
Authorization: Bearer <api-token>
```

**响应体**:
```json
{
  "contexts": [
    {
      "name": "prod-cluster",
      "cluster": "prod-k8s",
      "user": "admin",
      "namespace": "default",
      "current": true
    },
    {
      "name": "dev-cluster",
      "cluster": "dev-k8s",
      "user": "developer",
      "namespace": "development",
      "current": false
    }
  ]
}
```

**示例**:

```bash
curl -X GET http://localhost:8080/contexts \
  -H "Authorization: Bearer your-token"
```

---

## 可用工具列表

kubectl-mcp Server 提供以下 Kubernetes 操作工具，按功能分类：

### 查询类工具 (Query - READ)

这些工具用于查询 Kubernetes 资源信息，不会修改集群状态。

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `get_nodes` | 查询集群节点列表 | 否 |
| `get_namespaces` | 查询命名空间列表 | 否 |
| `get_pods` | 查询 Pod 列表，支持过滤 | 否 |
| `get_deployments` | 查询 Deployment 列表 | 否 |
| `get_statefulsets` | 查询 StatefulSet 列表 | 否 |
| `get_daemonsets` | 查询 DaemonSet 列表 | 否 |
| `get_services` | 查询 Service 列表 | 否 |
| `get_configmaps` | 查询 ConfigMap 列表 | 否 |
| `get_secrets` | 查询 Secret 列表（脱敏） | 否 |
| `describe_resource` | 获取资源详细信息 | 否 |
| `get_pod_logs` | 获取 Pod 日志 | 否 |
| `get_events` | 查询事件列表 | 否 |

### 创建类工具 (Create - CREATE)

这些工具用于创建新的 Kubernetes 资源。

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `create_namespace` | 创建命名空间 | 是 |
| `create_pod` | 创建 Pod | 是 |
| `create_deployment` | 创建 Deployment | 是 |
| `create_service` | 创建 Service | 是 |
| `create_configmap` | 创建 ConfigMap | 是 |
| `create_secret` | 创建 Secret | 是 |
| `create_from_yaml` | 通过 YAML 创建资源 | 是 |

### 修改类工具 (Update - UPDATE)

这些工具用于修改现有的 Kubernetes 资源。

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `scale_deployment` | 扩缩容 Deployment | 是 |
| `scale_statefulset` | 扩缩容 StatefulSet | 是 |
| `update_deployment_image` | 更新 Deployment 镜像 | 是 |
| `restart_deployment` | 重启 Deployment | 是 |
| `apply_yaml` | 通过 YAML 应用资源 | 是 |
| `patch_resource` | 使用 JSON Patch 修改资源 | 是 |

### 删除类工具 (Delete - DELETE)

这些工具用于删除 Kubernetes 资源，属于高危操作。

| 工具名称 | 描述 | 需要确认 |
|---------|------|---------|
| `delete_pod` | 删除 Pod | 是 |
| `delete_deployment` | 删除 Deployment | 是 |
| `delete_service` | 删除 Service | 是 |
| `delete_configmap` | 删除 ConfigMap | 是 |
| `delete_secret` | 删除 Secret | 是 |
| `delete_namespace` | 删除 Namespace（高危） | 是 |
| `delete_resources` | 批量删除资源（高危） | 是 |

---

## 工具参数详解

### 通用参数

所有工具都支持以下通用参数：

| 参数名 | 类型 | 必填 | 默认值 | 描述 |
|-------|------|------|--------|------|
| `context` | string | 否 | 当前 context | 指定要操作的 Kubernetes context |
| `namespace` | string | 否 | default | 指定命名空间 |

### 查询工具参数

**get_pods**:
```json
{
  "namespace": "default",        // 命名空间（可选）
  "name": "nginx-pod",          // Pod 名称（可选）
  "labelSelector": "app=nginx"  // 标签选择器（可选）
}
```

**get_pod_logs**:
```json
{
  "namespace": "default",       // 命名空间（必填）
  "name": "nginx-pod",         // Pod 名称（必填）
  "container": "nginx",        // 容器名称（可选）
  "tailLines": 100,            // 显示最后 N 行（可选）
  "previous": false            // 是否查看前一个容器的日志（可选）
}
```

**describe_resource**:
```json
{
  "resourceType": "pod",       // 资源类型（必填）
  "namespace": "default",      // 命名空间（必填）
  "name": "nginx-pod"         // 资源名称（必填）
}
```

### 创建工具参数

**create_deployment**:
```json
{
  "name": "nginx-app",         // Deployment 名称（必填）
  "namespace": "default",      // 命名空间（可选）
  "image": "nginx:1.21",      // 容器镜像（必填）
  "replicas": 3,              // 副本数（可选，默认 1）
  "port": 80,                 // 容器端口（可选）
  "labels": {                 // 标签（可选）
    "app": "nginx"
  },
  "env": [                    // 环境变量（可选）
    {
      "name": "ENV",
      "value": "prod"
    }
  ]
}
```

**create_service**:
```json
{
  "name": "nginx-service",     // Service 名称（必填）
  "namespace": "default",      // 命名空间（可选）
  "type": "ClusterIP",        // Service 类型（可选：ClusterIP, NodePort, LoadBalancer）
  "selector": {               // Pod 选择器（必填）
    "app": "nginx"
  },
  "ports": [                  // 端口映射（必填）
    {
      "name": "http",
      "port": 80,
      "targetPort": 8080,
      "protocol": "TCP"
    }
  ]
}
```

**create_from_yaml**:
```json
{
  "yaml": "apiVersion: v1\nkind: Pod\n...",  // YAML 内容（必填）
  "namespace": "default"                      // 命名空间（可选）
}
```

### 修改工具参数

**scale_deployment**:
```json
{
  "name": "nginx-app",         // Deployment 名称（必填）
  "namespace": "default",      // 命名空间（可选）
  "replicas": 5               // 目标副本数（必填）
}
```

**update_deployment_image**:
```json
{
  "name": "nginx-app",         // Deployment 名称（必填）
  "namespace": "default",      // 命名空间（可选）
  "container": "nginx",        // 容器名称（必填）
  "image": "nginx:1.22"       // 新镜像（必填）
}
```

**apply_yaml**:
```json
{
  "yaml": "apiVersion: apps/v1\nkind: Deployment\n...",  // YAML 内容（必填）
  "namespace": "default"                                  // 命名空间（可选）
}
```

**patch_resource**:
```json
{
  "resourceType": "deployment",  // 资源类型（必填）
  "name": "nginx-app",          // 资源名称（必填）
  "namespace": "default",       // 命名空间（可选）
  "patch": "{\"spec\":{\"replicas\":5}}"  // JSON Patch（必填）
}
```

### 删除工具参数

**delete_pod**:
```json
{
  "name": "nginx-pod",         // Pod 名称（必填）
  "namespace": "default",      // 命名空间（可选）
  "force": false,             // 是否强制删除（可选）
  "gracePeriod": 30           // 优雅删除时间（秒，可选）
}
```

**delete_deployment**:
```json
{
  "name": "nginx-app",         // Deployment 名称（必填）
  "namespace": "default",      // 命名空间（可选）
  "cascade": true             // 是否级联删除（可选）
}
```

**delete_namespace**:
```json
{
  "name": "my-namespace"       // Namespace 名称（必填）
}
```

**注意**: 系统命名空间（default、kube-system、kube-public、kube-node-lease）不允许删除。

**delete_resources**:
```json
{
  "resourceType": "pod",       // 资源类型（必填）
  "namespace": "default",      // 命名空间（可选）
  "names": ["pod1", "pod2"],  // 资源名称列表（可选）
  "labelSelector": "app=nginx" // 标签选择器（可选）
}
```

---

## 错误处理

### 错误响应格式

所有错误响应都遵循统一的格式：

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "string",        // 错误类型
    "code": "string",        // 错误码
    "message": "string",     // 错误消息
    "details": "string",     // 详细信息
    "suggestion": "string"   // 修复建议（可选）
  }
}
```

### 错误类型

| 错误类型 | HTTP 状态码 | 描述 |
|---------|------------|------|
| `CLIENT_ERROR` | 400 | 客户端错误（参数错误、资源冲突等） |
| `AUTH_ERROR` | 401 | 认证错误（Token 无效或缺失） |
| `NOT_FOUND` | 404 | 资源未找到 |
| `CONFLICT` | 409 | 资源冲突（资源已存在） |
| `SERVER_ERROR` | 500 | 服务端错误 |
| `NETWORK_ERROR` | 503 | 网络错误（集群不可达） |
| `TIMEOUT` | 504 | 超时错误 |

### 错误码列表

#### Kubeconfig 相关错误

| 错误码 | 描述 | 建议 |
|-------|------|------|
| `KUBECONFIG_NOT_FOUND` | Kubeconfig 文件不存在 | 检查 KUBECONFIG 环境变量或配置文件路径 |
| `KUBECONFIG_INVALID` | Kubeconfig 文件格式无效 | 验证 kubeconfig 文件格式是否正确 |
| `CONTEXT_NOT_FOUND` | 指定的 context 不存在 | 使用 GET /contexts 查看可用的 context |

#### 连接相关错误

| 错误码 | 描述 | 建议 |
|-------|------|------|
| `CLUSTER_UNREACHABLE` | 集群不可达 | 检查网络连接和集群状态 |
| `CONNECTION_TIMEOUT` | 连接超时 | 检查网络或增加超时时间 |
| `AUTH_FAILED` | 认证失败 | 检查 kubeconfig 中的认证信息 |

#### 资源相关错误

| 错误码 | 描述 | 建议 |
|-------|------|------|
| `RESOURCE_NOT_FOUND` | 资源不存在 | 检查资源名称和命名空间是否正确 |
| `RESOURCE_ALREADY_EXISTS` | 资源已存在 | 使用不同的名称或使用 apply_yaml 更新资源 |
| `INVALID_RESOURCE` | 资源定义无效 | 检查资源定义是否符合 Kubernetes 规范 |

#### 参数相关错误

| 错误码 | 描述 | 建议 |
|-------|------|------|
| `INVALID_ARGUMENTS` | 参数无效 | 检查参数类型和格式是否正确 |
| `MISSING_ARGUMENTS` | 缺少必填参数 | 查看工具文档补充必填参数 |

#### 权限相关错误

| 错误码 | 描述 | 建议 |
|-------|------|------|
| `PERMISSION_DENIED` | 权限不足 | 检查 kubeconfig 中用户的 RBAC 权限 |
| `UNAUTHORIZED` | 未授权 | 提供有效的 API Token |

### 错误示例

**示例 1: 资源未找到**

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "NOT_FOUND",
    "code": "RESOURCE_NOT_FOUND",
    "message": "Deployment not found",
    "details": "deployment.apps \"nginx-app\" not found in namespace \"default\"",
    "suggestion": "请检查 Deployment 名称和命名空间是否正确，使用 get_deployments 工具查看可用的 Deployment"
  }
}
```

**示例 2: 参数错误**

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "CLIENT_ERROR",
    "code": "INVALID_ARGUMENTS",
    "message": "Invalid replicas value",
    "details": "replicas must be a positive integer, got: -1",
    "suggestion": "请提供有效的副本数（正整数）"
  }
}
```

**示例 3: 权限不足**

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "AUTH_ERROR",
    "code": "PERMISSION_DENIED",
    "message": "Permission denied",
    "details": "User \"developer\" cannot delete deployments in namespace \"production\"",
    "suggestion": "请联系集群管理员授予相应的 RBAC 权限"
  }
}
```

**示例 4: 集群连接失败**

```json
{
  "success": false,
  "data": null,
  "error": {
    "type": "NETWORK_ERROR",
    "code": "CLUSTER_UNREACHABLE",
    "message": "Unable to connect to cluster",
    "details": "dial tcp 192.168.1.100:6443: i/o timeout",
    "suggestion": "请检查集群网络连接，确认 API Server 地址是否正确，或检查防火墙设置"
  }
}
```

---

## 最佳实践

### 1. 使用 Context 参数

在多集群环境中，始终明确指定 `context` 参数，避免误操作：

```json
{
  "tool": "get_pods",
  "arguments": {
    "context": "prod-cluster",
    "namespace": "production"
  }
}
```

### 2. 使用标签选择器

使用标签选择器可以更精确地筛选资源：

```json
{
  "tool": "get_pods",
  "arguments": {
    "namespace": "default",
    "labelSelector": "app=nginx,env=prod"
  }
}
```

### 3. 危险操作前确认

对于删除、扩缩容等危险操作，建议先查询资源状态再执行：

```bash
# 1. 先查询
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{"tool": "get_deployments", "arguments": {"namespace": "default"}}'

# 2. 确认后再删除
curl -X POST http://localhost:8080/tool \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-token" \
  -d '{"tool": "delete_deployment", "arguments": {"name": "nginx-app", "namespace": "default"}}'
```

### 4. 使用 YAML 进行复杂配置

对于复杂的资源配置，使用 `create_from_yaml` 或 `apply_yaml` 工具：

```json
{
  "tool": "apply_yaml",
  "arguments": {
    "yaml": "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: nginx-app\n..."
  }
}
```

### 5. 查看日志时限制行数

查看 Pod 日志时，使用 `tailLines` 参数限制返回的日志行数，避免数据量过大：

```json
{
  "tool": "get_pod_logs",
  "arguments": {
    "namespace": "default",
    "name": "nginx-pod",
    "tailLines": 100
  }
}
```

### 6. 处理错误响应

始终检查响应中的 `success` 字段，并根据错误类型和错误码进行相应处理：

```python
response = requests.post(
    "http://localhost:8080/tool",
    json={"tool": "get_pods", "arguments": {"namespace": "default"}},
    headers={"Authorization": "Bearer your-token"}
)

data = response.json()
if data["success"]:
    # 处理成功响应
    pods = data["data"]["pods"]
else:
    # 处理错误响应
    error = data["error"]
    print(f"错误类型: {error['type']}")
    print(f"错误码: {error['code']}")
    print(f"错误消息: {error['message']}")
    print(f"建议: {error.get('suggestion', '无')}")
```

---

## 安全注意事项

### 1. API Token 保护

- **不要**在代码中硬编码 API Token
- **不要**在日志中输出 API Token
- 使用环境变量或密钥管理系统存储 Token
- 定期轮换 API Token

### 2. Kubeconfig 安全

- kubectl-mcp Server 仅通过 kubeconfig 文件访问集群
- 不接受明文 token、证书或直接的 API server 地址
- kubeconfig 文件应设置适当的文件权限（600）
- 不要在日志中输出 kubeconfig 内容

### 3. 操作审计

- 所有操作都会记录审计日志
- 审计日志包含用户信息、操作类型、参数、结果
- 定期检查审计日志，发现异常操作

### 4. 权限控制

- 使用 Kubernetes RBAC 控制用户权限
- 遵循最小权限原则
- 不同环境使用不同的 kubeconfig 和权限

### 5. 网络安全

- 在生产环境中使用 HTTPS
- 配置防火墙规则，限制访问来源
- 使用 VPN 或专用网络访问 kubectl-mcp Server

---

## 性能优化建议

### 1. 使用过滤参数

查询资源时，使用 `namespace`、`name`、`labelSelector` 等过滤参数，减少返回的数据量：

```json
{
  "tool": "get_pods",
  "arguments": {
    "namespace": "production",
    "labelSelector": "app=nginx"
  }
}
```

### 2. 限制日志行数

查看 Pod 日志时，使用 `tailLines` 参数限制返回的日志行数：

```json
{
  "tool": "get_pod_logs",
  "arguments": {
    "namespace": "default",
    "name": "nginx-pod",
    "tailLines": 100
  }
}
```

### 3. 批量操作

对于需要操作多个资源的场景，使用批量操作工具（如 `delete_resources`）而不是多次单独调用：

```json
{
  "tool": "delete_resources",
  "arguments": {
    "resourceType": "pod",
    "namespace": "default",
    "labelSelector": "app=old-version"
  }
}
```

### 4. 合理设置超时

根据操作的复杂度和网络状况，设置合理的请求超时时间。

### 5. 使用缓存

kubectl-mcp Server 内置了查询结果缓存机制，频繁查询相同资源时可以利用缓存提高性能。

---

## 故障排查

### 问题 1: 连接超时

**症状**: 请求返回 `CONNECTION_TIMEOUT` 错误

**可能原因**:
- 集群网络不可达
- API Server 负载过高
- 防火墙阻止连接

**解决方法**:
1. 检查集群网络连接：`ping <api-server-ip>`
2. 检查 API Server 状态
3. 检查防火墙规则
4. 增加请求超时时间

### 问题 2: 认证失败

**症状**: 请求返回 `AUTH_FAILED` 错误

**可能原因**:
- Kubeconfig 中的认证信息过期
- 证书或 token 无效
- 用户权限不足

**解决方法**:
1. 验证 kubeconfig 文件是否有效
2. 检查证书或 token 是否过期
3. 使用 `kubectl` 命令测试连接
4. 检查 RBAC 权限配置

### 问题 3: 资源未找到

**症状**: 请求返回 `RESOURCE_NOT_FOUND` 错误

**可能原因**:
- 资源名称或命名空间错误
- 资源已被删除
- 使用了错误的 context

**解决方法**:
1. 使用 `get_*` 工具查询资源列表
2. 检查命名空间是否正确
3. 确认使用了正确的 context

### 问题 4: 权限不足

**症状**: 请求返回 `PERMISSION_DENIED` 错误

**可能原因**:
- 用户缺少相应的 RBAC 权限
- ServiceAccount 权限不足

**解决方法**:
1. 检查用户的 RBAC 权限：`kubectl auth can-i <verb> <resource>`
2. 联系集群管理员授予权限
3. 使用具有足够权限的 kubeconfig

---

## 附录

### A. HTTP 状态码

| 状态码 | 描述 |
|-------|------|
| 200 | 请求成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（Token 无效或缺失） |
| 404 | 资源未找到 |
| 405 | 不支持的 HTTP 方法 |
| 409 | 资源冲突 |
| 500 | 服务器内部错误 |
| 503 | 服务不可用（集群不可达） |
| 504 | 请求超时 |

### B. 支持的资源类型

| 资源类型 | API 版本 | 支持的操作 |
|---------|---------|-----------|
| Node | v1 | GET, DESCRIBE |
| Namespace | v1 | GET, CREATE, DELETE, DESCRIBE |
| Pod | v1 | GET, CREATE, DELETE, DESCRIBE, LOGS |
| Service | v1 | GET, CREATE, DELETE, DESCRIBE |
| ConfigMap | v1 | GET, CREATE, DELETE, DESCRIBE |
| Secret | v1 | GET, CREATE, DELETE, DESCRIBE |
| Deployment | apps/v1 | GET, CREATE, UPDATE, DELETE, DESCRIBE, SCALE |
| StatefulSet | apps/v1 | GET, CREATE, UPDATE, DELETE, DESCRIBE, SCALE |
| DaemonSet | apps/v1 | GET, CREATE, UPDATE, DELETE, DESCRIBE |
| Event | v1 | GET |

### C. 标签选择器语法

标签选择器支持以下语法：

| 语法 | 示例 | 描述 |
|-----|------|------|
| 等于 | `app=nginx` | 选择 app 标签值为 nginx 的资源 |
| 不等于 | `app!=nginx` | 选择 app 标签值不为 nginx 的资源 |
| 存在 | `app` | 选择具有 app 标签的资源 |
| 不存在 | `!app` | 选择不具有 app 标签的资源 |
| 多个条件 | `app=nginx,env=prod` | 选择同时满足多个条件的资源 |
| In | `env in (prod,staging)` | 选择 env 标签值为 prod 或 staging 的资源 |
| NotIn | `env notin (dev,test)` | 选择 env 标签值不为 dev 或 test 的资源 |

### D. 环境变量配置

kubectl-mcp Server 支持以下环境变量：

| 环境变量 | 描述 | 默认值 |
|---------|------|--------|
| `KUBECTL_MCP_HOST` | 服务器监听地址 | 0.0.0.0 |
| `KUBECTL_MCP_PORT` | 服务器监听端口 | 8080 |
| `KUBECONFIG` | Kubeconfig 文件路径 | ~/.kube/config |
| `KUBECTL_MCP_DEFAULT_CONTEXT` | 默认 context | kubeconfig 中的 current-context |
| `KUBECTL_MCP_LOG_LEVEL` | 日志级别 | info |
| `KUBECTL_MCP_LOG_FORMAT` | 日志格式 | json |
| `KUBECTL_MCP_LOG_FILE` | 日志文件路径 | stdout |
| `KUBECTL_MCP_API_TOKEN` | API Token | 无 |

### E. 版本信息

- **当前版本**: 1.0.0
- **API 版本**: v1
- **MCP 协议版本**: 1.0
- **支持的 Kubernetes 版本**: 1.20+

---

## 联系与支持

如有问题或建议，请通过以下方式联系：

- **项目地址**: [GitHub Repository]
- **问题反馈**: [GitHub Issues]
- **文档**: [Documentation]

---

**最后更新**: 2024-01-23
