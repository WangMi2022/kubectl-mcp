package test

import (
	"testing"

	"kubectl-mcp/internal/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterUpdateTools 测试注册所有修改类工具
func TestRegisterUpdateTools(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 注册修改类工具
	err := tools.RegisterUpdateTools(registry)
	assert.NoError(t, err, "注册修改类工具应该成功")

	// 验证工具数量和存在性
	expectedTools := []string{
		"scale_deployment",
		"scale_statefulset",
		"update_deployment_image",
		"restart_deployment",
		"apply_yaml",
		"patch_resource",
	}

	for _, toolName := range expectedTools {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "工具 '%s' 应该存在", toolName)
		assert.NotNil(t, tool, "工具 '%s' 不应为 nil", toolName)
		assert.Equal(t, tools.CategoryUpdate, tool.Category, "工具 '%s' 应该属于修改类别", toolName)
		assert.True(t, tool.RequiresConfirmation, "修改工具 '%s' 应该需要确认", toolName)
	}
}

// TestUpdateToolsSchema 测试修改类工具的 Schema 定义
func TestUpdateToolsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterUpdateTools(registry)
	require.NoError(t, err)

	// 测试 scale_deployment 工具的 Schema
	tool, exists := registry.GetTool("scale_deployment")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Equal(t, "object", tool.InputSchema.Type)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "replicas")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "replicas")

	// 测试 scale_statefulset 工具的 Schema
	tool, exists = registry.GetTool("scale_statefulset")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "replicas")

	// 测试 update_deployment_image 工具的 Schema
	tool, exists = registry.GetTool("update_deployment_image")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "image")
	assert.Contains(t, tool.InputSchema.Properties, "containerName")

	// 测试 restart_deployment 工具的 Schema
	tool, exists = registry.GetTool("restart_deployment")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")

	// 测试 apply_yaml 工具的 Schema
	tool, exists = registry.GetTool("apply_yaml")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "yaml")

	// 测试 patch_resource 工具的 Schema
	tool, exists = registry.GetTool("patch_resource")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "kind")
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "patch")
}

// TestUpdateToolsCategory 测试修改类工具的分类
func TestUpdateToolsCategory(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterUpdateTools(registry)
	require.NoError(t, err)

	updateTools := registry.GetToolsByCategory(tools.CategoryUpdate)
	assert.GreaterOrEqual(t, len(updateTools), 6, "应该至少有 6 个修改类工具")

	for _, tool := range updateTools {
		assert.Equal(t, tools.CategoryUpdate, tool.Category)
		assert.True(t, tool.RequiresConfirmation, "修改工具应该需要确认")
	}
}

// TestUpdateToolsRiskLevel 测试修改类工具的风险等级
func TestUpdateToolsRiskLevel(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterUpdateTools(registry)
	require.NoError(t, err)

	// 中等风险工具
	mediumRiskTools := []string{
		"scale_deployment",
	}

	for _, toolName := range mediumRiskTools {
		tool, _ := registry.GetTool(toolName)
		assert.Equal(t, "medium", tool.RiskLevel, "工具 '%s' 应该是中等风险", toolName)
	}

	// 高风险工具
	highRiskTools := []string{
		"scale_statefulset",
		"update_deployment_image",
		"restart_deployment",
		"apply_yaml",
		"patch_resource",
	}

	for _, toolName := range highRiskTools {
		tool, _ := registry.GetTool(toolName)
		assert.Equal(t, "high", tool.RiskLevel, "工具 '%s' 应该是高风险", toolName)
	}
}

// TestScaleDeploymentSuccess 测试成功扩缩容 Deployment
func TestScaleDeploymentSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	// 对于单元测试，我们主要测试工具的注册、Schema 定义和参数验证。
}

// TestScaleDeploymentNotFound 测试扩缩容不存在的 Deployment
func TestScaleDeploymentNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestScaleDeploymentMissingName 测试缺少 name 参数
func TestScaleDeploymentMissingName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")
}

// TestScaleDeploymentMissingReplicas 测试缺少 replicas 参数
func TestScaleDeploymentMissingReplicas(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")
}

// TestScaleDeploymentToZero 测试扩缩容到 0
func TestScaleDeploymentToZero(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestScaleStatefulSetSuccess 测试成功扩缩容 StatefulSet
func TestScaleStatefulSetSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestScaleStatefulSetNotFound 测试扩缩容不存在的 StatefulSet
func TestScaleStatefulSetNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestUpdateDeploymentImageSuccess 测试成功更新 Deployment 镜像
func TestUpdateDeploymentImageSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestUpdateDeploymentImageWithContainerName 测试更新指定容器的镜像
func TestUpdateDeploymentImageWithContainerName(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestUpdateDeploymentImageNotFound 测试更新不存在的 Deployment
func TestUpdateDeploymentImageNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestUpdateDeploymentImageContainerNotFound 测试更新不存在的容器
func TestUpdateDeploymentImageContainerNotFound(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestUpdateDeploymentImageMissingName 测试缺少 name 参数
func TestUpdateDeploymentImageMissingName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")
}

// TestUpdateDeploymentImageMissingImage 测试缺少 image 参数
func TestUpdateDeploymentImageMissingImage(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")
}

// TestDuplicateUpdateToolRegistration 测试重复注册修改工具
func TestDuplicateUpdateToolRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 第一次注册应该成功
	err := tools.RegisterUpdateTools(registry)
	assert.NoError(t, err)

	// 第二次注册应该失败
	err = tools.RegisterUpdateTools(registry)
	assert.Error(t, err, "重复注册工具应该失败")
}

// TestUpdateToolsRequireConfirmation 测试所有修改工具都需要确认
func TestUpdateToolsRequireConfirmation(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterUpdateTools(registry)
	require.NoError(t, err)

	updateTools := registry.GetToolsByCategory(tools.CategoryUpdate)
	for _, tool := range updateTools {
		assert.True(t, tool.RequiresConfirmation, "修改工具 '%s' 应该需要确认", tool.Name)
	}
}
