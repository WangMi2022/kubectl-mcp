package test

import (
	"testing"

	"kubectl-mcp/internal/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegisterCreateTools 测试注册所有创建类工具
func TestRegisterCreateTools(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 注册创建类工具
	err := tools.RegisterCreateTools(registry)
	assert.NoError(t, err, "注册创建类工具应该成功")

	// 验证工具数量和存在性
	expectedTools := []string{
		"create_namespace",
		"create_pod",
		"create_deployment",
		"create_service",
		"create_configmap",
		"create_secret",
		"create_from_yaml",
	}

	for _, toolName := range expectedTools {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "工具 '%s' 应该存在", toolName)
		assert.NotNil(t, tool, "工具 '%s' 不应为 nil", toolName)
		assert.Equal(t, tools.CategoryCreate, tool.Category, "工具 '%s' 应该属于创建类别", toolName)
		assert.True(t, tool.RequiresConfirmation, "创建工具 '%s' 应该需要确认", toolName)
	}
}

// TestCreateToolsSchema 测试创建类工具的 Schema 定义
func TestCreateToolsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	// 测试 create_namespace 工具的 Schema
	tool, exists := registry.GetTool("create_namespace")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Equal(t, "object", tool.InputSchema.Type)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "labels")
	assert.Contains(t, tool.InputSchema.Properties, "annotations")

	// 测试 create_pod 工具的 Schema
	tool, exists = registry.GetTool("create_pod")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "image")
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "containerName")
	assert.Contains(t, tool.InputSchema.Properties, "command")
	assert.Contains(t, tool.InputSchema.Properties, "args")
	assert.Contains(t, tool.InputSchema.Properties, "env")
	assert.Contains(t, tool.InputSchema.Properties, "restartPolicy")

	// 测试 create_deployment 工具的 Schema
	tool, exists = registry.GetTool("create_deployment")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "image")
	assert.Contains(t, tool.InputSchema.Properties, "replicas")
	assert.Contains(t, tool.InputSchema.Properties, "containerPort")

	// 测试 create_service 工具的 Schema
	tool, exists = registry.GetTool("create_service")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "port")
	assert.Contains(t, tool.InputSchema.Properties, "type")
	assert.Contains(t, tool.InputSchema.Properties, "targetPort")

	// 测试 create_configmap 工具的 Schema
	tool, exists = registry.GetTool("create_configmap")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "data")

	// 测试 create_secret 工具的 Schema
	tool, exists = registry.GetTool("create_secret")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "data")
	assert.Contains(t, tool.InputSchema.Properties, "stringData")

	// 测试 create_from_yaml 工具的 Schema
	tool, exists = registry.GetTool("create_from_yaml")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "yaml")
	assert.Contains(t, tool.InputSchema.Properties, "yaml")
}

// TestCreateToolsCategory 测试创建类工具的分类
func TestCreateToolsCategory(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	createTools := registry.GetToolsByCategory(tools.CategoryCreate)
	assert.GreaterOrEqual(t, len(createTools), 7, "应该至少有 7 个创建类工具")

	for _, tool := range createTools {
		assert.Equal(t, tools.CategoryCreate, tool.Category)
		assert.True(t, tool.RequiresConfirmation, "创建工具应该需要确认")
	}
}

// TestCreateToolsRiskLevel 测试创建类工具的风险等级
func TestCreateToolsRiskLevel(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	// 高风险工具
	highRiskTools := []string{
		"create_secret",
		"create_from_yaml",
	}

	for _, toolName := range highRiskTools {
		tool, _ := registry.GetTool(toolName)
		assert.Equal(t, "high", tool.RiskLevel, "工具 '%s' 应该是高风险", toolName)
	}

	// 中等风险工具
	mediumRiskTools := []string{
		"create_namespace",
		"create_pod",
		"create_deployment",
		"create_service",
	}

	for _, toolName := range mediumRiskTools {
		tool, _ := registry.GetTool(toolName)
		assert.Equal(t, "medium", tool.RiskLevel, "工具 '%s' 应该是中等风险", toolName)
	}

	// 低风险工具
	lowRiskTools := []string{
		"create_configmap",
	}

	for _, toolName := range lowRiskTools {
		tool, _ := registry.GetTool(toolName)
		assert.Equal(t, "low", tool.RiskLevel, "工具 '%s' 应该是低风险", toolName)
	}
}

// TestCreateNamespaceSuccess 测试成功创建 Namespace
func TestCreateNamespaceSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：由于 K8SClientManager 需要真实的 kubeconfig 文件，
	// 这个测试需要在集成测试环境中运行。
	// 对于单元测试，我们主要测试工具的注册、Schema 定义和参数验证。
}

// TestCreateNamespaceAlreadyExists 测试创建已存在的 Namespace（参数验证）
func TestCreateNamespaceAlreadyExists(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")

	// 注意：资源冲突测试需要在集成测试环境中运行
}

// TestCreateNamespaceMissingName 测试缺少必填参数 name（参数验证）
func TestCreateNamespaceMissingName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
}

// TestCreatePodSuccess 测试成功创建 Pod
func TestCreatePodSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestCreatePodAlreadyExists 测试创建已存在的 Pod
func TestCreatePodAlreadyExists(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestCreatePodMissingRequiredParams 测试缺少必填参数（参数验证）
func TestCreatePodMissingRequiredParams(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
}

// TestCreatePodWithOptionalParams 测试使用可选参数创建 Pod
func TestCreatePodWithOptionalParams(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestCreateDeploymentSuccess 测试成功创建 Deployment
func TestCreateDeploymentSuccess(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestCreateDeploymentAlreadyExists 测试创建已存在的 Deployment
func TestCreateDeploymentAlreadyExists(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestCreateDeploymentMissingRequiredParams 测试缺少必填参数（参数验证）
func TestCreateDeploymentMissingRequiredParams(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
}

// TestCreateDeploymentWithDefaultReplicas 测试使用默认副本数创建 Deployment
func TestCreateDeploymentWithDefaultReplicas(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestCreateDeploymentWithContainerPort 测试创建带容器端口的 Deployment
func TestCreateDeploymentWithContainerPort(t *testing.T) {
	t.Skip("跳过需要真实 K8S 集群的测试 - 此测试需要集成测试环境")
}

// TestDuplicateCreateToolRegistration 测试重复注册创建工具
func TestDuplicateCreateToolRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 第一次注册应该成功
	err := tools.RegisterCreateTools(registry)
	assert.NoError(t, err)

	// 第二次注册应该失败
	err = tools.RegisterCreateTools(registry)
	assert.Error(t, err, "重复注册工具应该失败")
}

// TestCreateToolsRequireConfirmation 测试所有创建工具都需要确认
func TestCreateToolsRequireConfirmation(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	createTools := registry.GetToolsByCategory(tools.CategoryCreate)
	for _, tool := range createTools {
		assert.True(t, tool.RequiresConfirmation, "创建工具 '%s' 应该需要确认", tool.Name)
	}
}

// TestCreateNamespaceInvalidName 测试使用无效的 name 参数（参数验证）
func TestCreateNamespaceInvalidName(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
}

// TestCreatePodInvalidImage 测试使用无效的 image 参数（参数验证）
func TestCreatePodInvalidImage(t *testing.T) {
	t.Skip("跳过需要 K8S 客户端的测试 - 参数验证在 getContextAndNamespace 之后")

	// 注意：虽然这是参数验证测试，但由于 getContextAndNamespace 在参数验证之前被调用，
	// 我们仍然需要一个有效的 K8SClientManager。这个测试应该在集成测试中进行。
}

// TestCreateServiceSchema 测试 create_service 工具的 Schema
func TestCreateServiceSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("create_service")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证必填参数
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Required, "port")

	// 验证可选参数
	assert.Contains(t, tool.InputSchema.Properties, "type")
	assert.Contains(t, tool.InputSchema.Properties, "targetPort")
	assert.Contains(t, tool.InputSchema.Properties, "protocol")
	assert.Contains(t, tool.InputSchema.Properties, "nodePort")
	assert.Contains(t, tool.InputSchema.Properties, "selector")

	// 验证默认值
	typeSchema := tool.InputSchema.Properties["type"]
	assert.Equal(t, "ClusterIP", typeSchema.Default)

	protocolSchema := tool.InputSchema.Properties["protocol"]
	assert.Equal(t, "TCP", protocolSchema.Default)
}

// TestCreateConfigMapSchema 测试 create_configmap 工具的 Schema
func TestCreateConfigMapSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("create_configmap")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证必填参数
	assert.Contains(t, tool.InputSchema.Required, "name")

	// 验证可选参数
	assert.Contains(t, tool.InputSchema.Properties, "data")
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "labels")
	assert.Contains(t, tool.InputSchema.Properties, "annotations")
}

// TestCreateSecretSchema 测试 create_secret 工具的 Schema
func TestCreateSecretSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("create_secret")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证必填参数
	assert.Contains(t, tool.InputSchema.Required, "name")

	// 验证可选参数
	assert.Contains(t, tool.InputSchema.Properties, "type")
	assert.Contains(t, tool.InputSchema.Properties, "data")
	assert.Contains(t, tool.InputSchema.Properties, "stringData")

	// 验证默认值
	typeSchema := tool.InputSchema.Properties["type"]
	assert.Equal(t, "Opaque", typeSchema.Default)

	// 验证风险等级
	assert.Equal(t, "high", tool.RiskLevel, "create_secret 应该是高风险")
}

// TestCreateFromYAMLSchema 测试 create_from_yaml 工具的 Schema
func TestCreateFromYAMLSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterCreateTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("create_from_yaml")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证必填参数
	assert.Contains(t, tool.InputSchema.Required, "yaml")

	// 验证可选参数
	assert.Contains(t, tool.InputSchema.Properties, "yaml")
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "context")

	// 验证风险等级
	assert.Equal(t, "high", tool.RiskLevel, "create_from_yaml 应该是高风险")
}
