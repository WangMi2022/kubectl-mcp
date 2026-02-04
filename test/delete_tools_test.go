package test

import (
	"testing"

	"kubectl-mcp/internal/tools"

	"github.com/stretchr/testify/assert"
)

// TestRegisterDeleteTools 测试注册所有删除类工具
func TestRegisterDeleteTools(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 注册删除类工具
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err, "注册删除类工具应该成功")

	// 验证工具数量
	expectedTools := []string{
		"delete_pod",
		"delete_deployment",
		"delete_service",
		"delete_configmap",
		"delete_secret",
		"delete_namespace",
		"delete_resources",
	}

	for _, toolName := range expectedTools {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "工具 '%s' 应该存在", toolName)
		assert.NotNil(t, tool, "工具 '%s' 不应为 nil", toolName)
		assert.Equal(t, tools.CategoryDelete, tool.Category, "工具 '%s' 应该属于删除类别", toolName)
		assert.True(t, tool.RequiresConfirmation, "删除工具 '%s' 应该需要确认", toolName)
	}
}

// TestDeleteToolsSchema 测试删除类工具的 Schema 定义
func TestDeleteToolsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err)

	// 测试 delete_pod 工具的 Schema
	tool, exists := registry.GetTool("delete_pod")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Equal(t, "object", tool.InputSchema.Type)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "force")
	assert.Contains(t, tool.InputSchema.Properties, "gracePeriod")

	// 测试 delete_deployment 工具的 Schema
	tool, exists = registry.GetTool("delete_deployment")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "cascade")

	// 测试 delete_resources 批量删除工具的 Schema
	tool, exists = registry.GetTool("delete_resources")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "kind")
	assert.Contains(t, tool.InputSchema.Properties, "kind")
	assert.Contains(t, tool.InputSchema.Properties, "names")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
}

// TestDeleteToolsRiskLevel 测试删除类工具的风险等级
func TestDeleteToolsRiskLevel(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err)

	// 测试不同工具的风险等级
	riskLevels := map[string]string{
		"delete_pod":        "medium",
		"delete_deployment": "high",
		"delete_service":    "medium",
		"delete_configmap":  "medium",
		"delete_secret":     "high",
		"delete_namespace":  "critical",
		"delete_resources":  "high",
	}

	for toolName, expectedRisk := range riskLevels {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "工具 '%s' 应该存在", toolName)
		assert.Equal(t, expectedRisk, tool.RiskLevel, "工具 '%s' 的风险等级应该是 %s", toolName, expectedRisk)
	}
}

// TestDeleteToolsCategory 测试删除类工具的分类
func TestDeleteToolsCategory(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err)

	deleteTools := registry.GetToolsByCategory(tools.CategoryDelete)
	assert.Len(t, deleteTools, 7, "应该有 7 个删除类工具")

	for _, tool := range deleteTools {
		assert.Equal(t, tools.CategoryDelete, tool.Category)
		assert.True(t, tool.RequiresConfirmation, "所有删除工具都应该需要确认")
	}
}

// TestDuplicateDeleteToolRegistration 测试重复注册删除工具
func TestDuplicateDeleteToolRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 第一次注册应该成功
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err)

	// 第二次注册应该失败
	err = tools.RegisterDeleteTools(registry)
	assert.Error(t, err, "重复注册工具应该失败")
}

// TestDeleteNamespaceSystemProtection 测试删除 namespace 的系统保护
func TestDeleteNamespaceSystemProtection(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err)

	tool, exists := registry.GetTool("delete_namespace")
	assert.True(t, exists)
	assert.Equal(t, "critical", tool.RiskLevel, "删除 namespace 应该是最高风险等级")
	assert.Contains(t, tool.Description, "高危操作", "描述中应该包含高危警告")
	assert.Contains(t, tool.Description, "系统 namespace", "描述中应该提到系统 namespace 保护")
}

// TestDeleteResourcesKindEnum 测试批量删除工具的资源类型枚举
func TestDeleteResourcesKindEnum(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterDeleteTools(registry)
	assert.NoError(t, err)

	tool, exists := registry.GetTool("delete_resources")
	assert.True(t, exists)

	kindSchema := tool.InputSchema.Properties["kind"]
	assert.NotNil(t, kindSchema)
	assert.NotEmpty(t, kindSchema.Enum, "kind 参数应该有枚举值")

	expectedKinds := []interface{}{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "ConfigMap", "Secret"}
	assert.ElementsMatch(t, expectedKinds, kindSchema.Enum, "kind 枚举值应该包含所有支持的资源类型")
}

// TestDeletePodSuccess 测试成功删除 Pod
func TestDeletePodSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	// 对于单元测试，我们主要测试工具的注册、Schema 定义和参数验证。
	//
	// 测试场景：
	// 1. 创建一个测试 Pod
	// 2. 调用 DeletePod 删除该 Pod
	// 3. 验证返回的 DeleteResult 包含正确的信息
	// 4. 验证 Pod 已被删除
}

// TestDeletePodNotFound 测试删除不存在的 Pod
func TestDeletePodNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源不存在测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除一个不存在的 Pod
	// 2. 验证返回错误信息包含 "获取 Pod" 和资源名称
	// 3. 验证错误类型为 NotFound
}

// TestDeletePodForce 测试强制删除 Pod
func TestDeletePodForce(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：强制删除测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 创建一个测试 Pod
	// 2. 使用 force=true 参数调用 DeletePod
	// 3. 验证返回的 DeleteResult 中 Force 字段为 true
	// 4. 验证消息中包含 "强制删除" 和 "grace period: 0s"
	// 5. 验证 Pod 被立即删除（grace period 为 0）
}

// TestDeletePodWithGracePeriod 测试使用自定义优雅删除时间删除 Pod
func TestDeletePodWithGracePeriod(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：自定义 grace period 测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 创建一个测试 Pod
	// 2. 使用 gracePeriod=60 参数调用 DeletePod
	// 3. 验证返回的消息中包含 "grace period: 60s"
	// 4. 验证 Pod 在指定时间后被删除
}

// TestDeletePodMissingName 测试缺少必填参数 name
func TestDeletePodMissingName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
	//
	// 测试场景：
	// 1. 调用 DeletePod 但不提供 name 参数
	// 2. 验证返回错误信息包含 "参数 'name' 是必填的"
}

// TestDeleteDeploymentSuccess 测试成功删除 Deployment
func TestDeleteDeploymentSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	//
	// 测试场景：
	// 1. 创建一个测试 Deployment
	// 2. 调用 DeleteDeployment 删除该 Deployment
	// 3. 验证返回的 DeleteResult 包含正确的信息
	// 4. 验证 Deployment 已被删除
	// 5. 验证关联的 ReplicaSet 和 Pod 也被删除（cascade=true）
}

// TestDeleteDeploymentNotFound 测试删除不存在的 Deployment
func TestDeleteDeploymentNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源不存在测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除一个不存在的 Deployment
	// 2. 验证返回错误信息包含 "获取 Deployment" 和资源名称
	// 3. 验证错误类型为 NotFound
}

// TestDeleteDeploymentNoCascade 测试不级联删除 Deployment
func TestDeleteDeploymentNoCascade(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：不级联删除测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 创建一个测试 Deployment
	// 2. 使用 cascade=false 参数调用 DeleteDeployment
	// 3. 验证返回的 DeleteResult 中 Cascade 字段为 false
	// 4. 验证消息中包含 "不级联删除"
	// 5. 验证 Deployment 被删除但 ReplicaSet 和 Pod 仍然存在
}

// TestDeleteDeploymentMissingName 测试缺少必填参数 name
func TestDeleteDeploymentMissingName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
	//
	// 测试场景：
	// 1. 调用 DeleteDeployment 但不提供 name 参数
	// 2. 验证返回错误信息包含 "参数 'name' 是必填的"
}

// TestDeleteServiceSuccess 测试成功删除 Service
func TestDeleteServiceSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	//
	// 测试场景：
	// 1. 创建一个测试 Service
	// 2. 调用 DeleteService 删除该 Service
	// 3. 验证返回的 DeleteResult 包含正确的信息
	// 4. 验证 Service 已被删除
}

// TestDeleteServiceNotFound 测试删除不存在的 Service
func TestDeleteServiceNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源不存在测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除一个不存在的 Service
	// 2. 验证返回错误信息包含 "获取 Service" 和资源名称
	// 3. 验证错误类型为 NotFound
}

// TestDeleteServiceMissingName 测试缺少必填参数 name
func TestDeleteServiceMissingName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
	//
	// 测试场景：
	// 1. 调用 DeleteService 但不提供 name 参数
	// 2. 验证返回错误信息包含 "参数 'name' 是必填的"
}

// TestDeleteConfigMapSuccess 测试成功删除 ConfigMap
func TestDeleteConfigMapSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	//
	// 测试场景：
	// 1. 创建一个测试 ConfigMap
	// 2. 调用 DeleteConfigMap 删除该 ConfigMap
	// 3. 验证返回的 DeleteResult 包含正确的信息
	// 4. 验证 ConfigMap 已被删除
}

// TestDeleteConfigMapNotFound 测试删除不存在的 ConfigMap
func TestDeleteConfigMapNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源不存在测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除一个不存在的 ConfigMap
	// 2. 验证返回错误信息包含 "获取 ConfigMap" 和资源名称
	// 3. 验证错误类型为 NotFound
}

// TestDeleteSecretSuccess 测试成功删除 Secret
func TestDeleteSecretSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	//
	// 测试场景：
	// 1. 创建一个测试 Secret
	// 2. 调用 DeleteSecret 删除该 Secret
	// 3. 验证返回的 DeleteResult 包含正确的信息
	// 4. 验证 Secret 已被删除
}

// TestDeleteSecretNotFound 测试删除不存在的 Secret
func TestDeleteSecretNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源不存在测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除一个不存在的 Secret
	// 2. 验证返回错误信息包含 "获取 Secret" 和资源名称
	// 3. 验证错误类型为 NotFound
}

// TestDeleteNamespaceSuccess 测试成功删除 Namespace
func TestDeleteNamespaceSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	//
	// 测试场景：
	// 1. 创建一个测试 Namespace
	// 2. 调用 DeleteNamespace 删除该 Namespace
	// 3. 验证返回的 DeleteResult 包含正确的信息
	// 4. 验证 Namespace 已被删除
	// 5. 验证 Namespace 下的所有资源也被删除
}

// TestDeleteNamespaceNotFound 测试删除不存在的 Namespace
func TestDeleteNamespaceNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源不存在测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除一个不存在的 Namespace
	// 2. 验证返回错误信息包含 "获取 Namespace" 和资源名称
	// 3. 验证错误类型为 NotFound
}

// TestDeleteNamespaceSystemProtected 测试删除系统 Namespace 被保护
func TestDeleteNamespaceSystemProtected(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：系统 Namespace 保护测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 尝试删除系统 Namespace（default、kube-system、kube-public、kube-node-lease）
	// 2. 验证返回错误信息包含 "系统 namespace 不允许删除"
	// 3. 验证 Namespace 未被删除
}

// TestDeleteResourcesBatchSuccess 测试批量删除资源成功
func TestDeleteResourcesBatchSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	//
	// 测试场景：
	// 1. 创建多个测试 Pod
	// 2. 使用 names 参数批量删除这些 Pod
	// 3. 验证返回的 BatchDeleteResult 包含正确的统计信息
	// 4. 验证所有 Pod 都被删除
}

// TestDeleteResourcesByLabelSelector 测试通过标签选择器批量删除资源
func TestDeleteResourcesByLabelSelector(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：标签选择器测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 创建多个带有相同标签的测试 Pod
	// 2. 使用 labelSelector 参数批量删除这些 Pod
	// 3. 验证返回的 BatchDeleteResult 包含正确的统计信息
	// 4. 验证所有匹配标签的 Pod 都被删除
	// 5. 验证不匹配标签的 Pod 未被删除
}

// TestDeleteResourcesPartialFailure 测试批量删除部分失败
func TestDeleteResourcesPartialFailure(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：部分失败测试需要在集成测试环境中运行
	//
	// 测试场景：
	// 1. 创建一些测试 Pod
	// 2. 尝试批量删除包含存在和不存在的 Pod
	// 3. 验证返回的 BatchDeleteResult 包含正确的成功和失败统计
	// 4. 验证 FailureList 包含失败的资源名称和错误信息
	// 5. 验证存在的 Pod 被成功删除
}

// TestDeleteResourcesMissingKind 测试缺少必填参数 kind
func TestDeleteResourcesMissingKind(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
	//
	// 测试场景：
	// 1. 调用 DeleteResources 但不提供 kind 参数
	// 2. 验证返回错误信息包含 "参数 'kind' 是必填的"
}

// TestDeleteResourcesInvalidKind 测试使用无效的 kind 参数
func TestDeleteResourcesInvalidKind(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
	//
	// 测试场景：
	// 1. 调用 DeleteResources 使用不支持的 kind（如 "InvalidKind"）
	// 2. 验证返回错误信息包含 "不支持的资源类型"
}
