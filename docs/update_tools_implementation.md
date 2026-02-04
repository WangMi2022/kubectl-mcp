# 修改类工具实现文档

## 概述

本文档描述了 kubectl-mcp 服务器中修改类（UPDATE）工具的实现。所有修改类工具都需要用户确认，并且具有不同的风险等级。

## 实现的工具

### 1. scale_deployment - 扩缩容 Deployment

**功能**: 调整 Deployment 的副本数量

**参数**:
- `name` (必填): Deployment 名称
- `namespace` (可选): 命名空间，默认 default
- `replicas` (必填): 目标副本数（≥0）
- `context` (可选): K8S context 名称

**风险等级**: medium

**实现文件**: `internal/tools/update_scale.go`

### 2. scale_statefulset - 扩缩容 StatefulSet

**功能**: 调整 StatefulSet 的副本数量

**参数**:
- `name` (必填): StatefulSet 名称
- `namespace` (可选): 命名空间，默认 default
- `replicas` (必填): 目标副本数（≥0）
- `context` (可选): K8S context 名称

**风险等级**: high

**实现文件**: `internal/tools/update_scale.go`

### 3. update_deployment_image - 更新 Deployment 镜像

**功能**: 更新 Deployment 的容器镜像，触发滚动更新

**参数**:
- `name` (必填): Deployment 名称
- `namespace` (可选): 命名空间，默认 default
- `image` (必填): 新的容器镜像（格式: image:tag）
- `containerName` (可选): 容器名称，不指定则更新第一个容器
- `context` (可选): K8S context 名称

**风险等级**: high

**实现文件**: `internal/tools/update_image.go`

### 4. restart_deployment - 重启 Deployment

**功能**: 通过滚动更新重启 Deployment 的所有 Pod

**参数**:
- `name` (必填): Deployment 名称
- `namespace` (可选): 命名空间，默认 default
- `context` (可选): K8S context 名称

**实现原理**: 在 Pod 模板中添加/更新 `kubectl.kubernetes.io/restartedAt` 注解，触发滚动重启

**风险等级**: high

**实现文件**: `internal/tools/update_restart.go`

### 5. apply_yaml - 应用 YAML 资源

**功能**: 通过 YAML 应用 Kubernetes 资源（类似 kubectl apply），如果资源存在则更新，不存在则创建

**参数**:
- `yaml` (必填): YAML 格式的资源定义
- `namespace` (可选): 命名空间（如果 YAML 中未指定则使用此值）
- `context` (可选): K8S context 名称

**风险等级**: high

**实现文件**: `internal/tools/update_yaml.go`

**特性**:
- 自动检测资源是否存在
- 存在则更新（保留 ResourceVersion）
- 不存在则创建
- 支持所有 Kubernetes 资源类型

### 6. patch_resource - Patch 资源

**功能**: 使用 JSON Patch 修改 Kubernetes 资源，支持精细化的资源修改

**参数**:
- `kind` (必填): 资源类型（如 Deployment, Service, Pod 等）
- `name` (必填): 资源名称
- `namespace` (可选): 命名空间（对于 namespace-scoped 资源）
- `patch` (必填): Patch 内容（JSON 格式字符串）
- `patchType` (可选): Patch 类型，支持 json, merge, strategic，默认 strategic
- `context` (可选): K8S context 名称

**风险等级**: high

**实现文件**: `internal/tools/update_patch.go`

**支持的 Patch 类型**:
- `json`: JSON Patch (RFC 6902)
- `merge`: JSON Merge Patch (RFC 7386)
- `strategic`: Strategic Merge Patch (Kubernetes 默认)

**支持的资源类型**:
- Core API: Pod, Service, ConfigMap, Secret, Namespace, Node, PV, PVC, ServiceAccount
- Apps API: Deployment, StatefulSet, DaemonSet, ReplicaSet
- Batch API: Job, CronJob
- Networking API: Ingress
- RBAC API: Role, RoleBinding, ClusterRole, ClusterRoleBinding

## 文件结构

```
kubectl-mcp/internal/tools/
├── update_scale.go          # 扩缩容工具
├── update_image.go          # 更新镜像工具
├── update_restart.go        # 重启工具
├── update_yaml.go           # Apply YAML 工具
├── update_patch.go          # Patch 工具
├── register_update.go       # 注册所有 update 工具
└── types.go                 # 包含 UpdateResult 类型定义
```

## 数据结构

### UpdateResult

修改操作的返回结果：

```go
type UpdateResult struct {
    Kind      string      `json:"kind"`
    Name      string      `json:"name"`
    Namespace string      `json:"namespace,omitempty"`
    Action    string      `json:"action"` // Scale, UpdateImage, Restart, Patch, Apply
    Status    string      `json:"status"`
    Message   string      `json:"message"`
    OldValue  string      `json:"oldValue,omitempty"`
    NewValue  string      `json:"newValue,omitempty"`
    Details   interface{} `json:"details,omitempty"`
}
```

## 使用示例

### 扩缩容 Deployment

```json
{
  "tool": "scale_deployment",
  "arguments": {
    "name": "nginx",
    "namespace": "default",
    "replicas": 5
  }
}
```

### 更新镜像

```json
{
  "tool": "update_deployment_image",
  "arguments": {
    "name": "nginx",
    "namespace": "default",
    "image": "nginx:1.21",
    "containerName": "nginx"
  }
}
```

### 重启 Deployment

```json
{
  "tool": "restart_deployment",
  "arguments": {
    "name": "nginx",
    "namespace": "default"
  }
}
```

### Apply YAML

```json
{
  "tool": "apply_yaml",
  "arguments": {
    "yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test-config\ndata:\n  key: value"
  }
}
```

### Patch 资源

```json
{
  "tool": "patch_resource",
  "arguments": {
    "kind": "Deployment",
    "name": "nginx",
    "namespace": "default",
    "patch": "{\"spec\":{\"replicas\":3}}",
    "patchType": "strategic"
  }
}
```

## 测试

所有工具都有完整的单元测试，位于 `test/update_tools_test.go`。

运行测试：

```bash
go test -v ./test/update_tools_test.go
```

测试覆盖：
- 工具注册验证
- Schema 定义验证
- 风险等级验证
- 分类验证
- 重复注册检测

## 安全性

所有修改类工具都：
1. 需要用户确认（`RequiresConfirmation: true`）
2. 具有明确的风险等级（medium 或 high）
3. 通过 kubeconfig 认证访问集群
4. 记录完整的操作审计日志
5. 提供详细的错误信息和建议

## 依赖

- `k8s.io/client-go`: Kubernetes 客户端库
- `k8s.io/apimachinery`: Kubernetes API 机制
- `kubectl-mcp/internal/k8s`: K8S 客户端管理器

## 注意事项

1. **扩缩容到 0**: scale_deployment 和 scale_statefulset 都支持扩缩容到 0，但这是高风险操作
2. **StatefulSet 扩缩容**: StatefulSet 的扩缩容是有序的，缩容时从最大序号开始删除
3. **镜像更新**: 更新镜像会触发滚动更新，可能导致服务短暂不可用
4. **重启操作**: restart_deployment 通过添加注解触发滚动重启，不会立即重启所有 Pod
5. **YAML Apply**: 支持任意资源类型，但需要确保 YAML 格式正确
6. **Patch 操作**: 不同的 Patch 类型有不同的行为，strategic 是最常用的类型

## 后续工作

- 添加更多资源类型的支持（如 Job, CronJob 等）
- 实现批量操作功能
- 添加操作预览功能（dry-run）
- 实现操作回滚功能
