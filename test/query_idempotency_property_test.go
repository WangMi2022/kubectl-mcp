package test

import (
	"os"
	"path/filepath"
	"testing"

	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/tools"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_QueryIdempotency 测试资源查询幂等性属性
// Property 8: 资源查询幂等性
// Validates: Requirements 4.1-4.15
// Feature: kubectl-mcp-server, Property 8: 对于任何查询类工具（GET 操作），多次执行相同的查询应该返回一致的结果（在资源未变化的情况下）
func TestProperty_QueryIdempotency(t *testing.T) {
	// 创建测试用的 kubeconfig
	kubeconfigPath := createTestKubeconfigForIdempotency(t)
	defer os.Remove(kubeconfigPath)

	// 创建 K8S 客户端管理器
	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}
	defer manager.Close()

	// 配置属性测试参数
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100 // 至少运行 100 次迭代
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	// 属性 1: 工具注册表查询幂等性
	// 多次获取工具列表应该返回相同的结果
	properties.Property("工具注册表查询多次返回一致结果", prop.ForAll(
		func(category string) bool {
			registry := tools.NewToolRegistry()
			tools.RegisterQueryTools(registry)

			// 第一次查询
			var result1 []*tools.Tool
			if category == "" || category == "all" {
				result1 = registry.GetAllTools()
			} else {
				result1 = registry.GetToolsByCategory(tools.ToolCategory(category))
			}

			// 第二次查询
			var result2 []*tools.Tool
			if category == "" || category == "all" {
				result2 = registry.GetAllTools()
			} else {
				result2 = registry.GetToolsByCategory(tools.ToolCategory(category))
			}

			// 第三次查询
			var result3 []*tools.Tool
			if category == "" || category == "all" {
				result3 = registry.GetAllTools()
			} else {
				result3 = registry.GetToolsByCategory(tools.ToolCategory(category))
			}

			// 验证三次查询结果一致
			if len(result1) != len(result2) || len(result2) != len(result3) {
				return false
			}

			// 验证工具名称集合一致（不考虑顺序）
			names1 := extractToolNamesSet(result1)
			names2 := extractToolNamesSet(result2)
			names3 := extractToolNamesSet(result3)

			return mapsEqual(names1, names2) && mapsEqual(names2, names3)
		},
		genToolCategory(),
	))

	// 属性 2: 工具定义查询幂等性
	// 多次获取同一个工具的定义应该返回相同的结果
	properties.Property("工具定义查询多次返回一致结果", prop.ForAll(
		func(toolName string) bool {
			registry := tools.NewToolRegistry()
			tools.RegisterQueryTools(registry)

			// 第一次查询
			tool1, exists1 := registry.GetTool(toolName)

			// 第二次查询
			tool2, exists2 := registry.GetTool(toolName)

			// 第三次查询
			tool3, exists3 := registry.GetTool(toolName)

			// 验证存在性一致
			if exists1 != exists2 || exists2 != exists3 {
				return false
			}

			// 如果工具不存在，这是预期的
			if !exists1 {
				return true
			}

			// 验证工具定义一致
			return tool1.Name == tool2.Name && tool2.Name == tool3.Name &&
				tool1.Category == tool2.Category && tool2.Category == tool3.Category &&
				tool1.RequiresConfirmation == tool2.RequiresConfirmation &&
				tool2.RequiresConfirmation == tool3.RequiresConfirmation
		},
		genToolName(),
	))

	// 属性 3: Context 列表查询幂等性
	// 多次查询 context 列表应该返回相同的结果
	properties.Property("Context 列表查询多次返回一致结果", prop.ForAll(
		func(dummy int) bool {
			// 第一次查询
			contexts1 := manager.ListContexts()

			// 第二次查询
			contexts2 := manager.ListContexts()

			// 第三次查询
			contexts3 := manager.ListContexts()

			// 验证数量一致
			if len(contexts1) != len(contexts2) || len(contexts2) != len(contexts3) {
				return false
			}

			// 验证 context 名称集合一致（不考虑顺序）
			names1 := extractContextNamesSet(contexts1)
			names2 := extractContextNamesSet(contexts2)
			names3 := extractContextNamesSet(contexts3)

			return mapsEqual(names1, names2) && mapsEqual(names2, names3)
		},
		gen.Int(),
	))

	// 属性 4: Context 信息查询幂等性
	// 多次查询同一个 context 的信息应该返回相同的结果
	properties.Property("Context 信息查询多次返回一致结果", prop.ForAll(
		func(contextName string) bool {
			// 第一次查询
			info1, err1 := manager.GetContextInfo(contextName)

			// 第二次查询
			info2, err2 := manager.GetContextInfo(contextName)

			// 第三次查询
			info3, err3 := manager.GetContextInfo(contextName)

			// 验证错误状态一致
			if (err1 != nil) != (err2 != nil) || (err2 != nil) != (err3 != nil) {
				return false
			}

			// 如果 context 不存在，这是预期的
			if err1 != nil {
				return true
			}

			// 验证 context 信息一致
			return info1.Name == info2.Name && info2.Name == info3.Name &&
				info1.Cluster == info2.Cluster && info2.Cluster == info3.Cluster &&
				info1.Namespace == info2.Namespace && info2.Namespace == info3.Namespace
		},
		genContextName(manager),
	))

	// 属性 5: 当前 Context 查询幂等性
	// 多次查询当前 context 应该返回相同的结果（在没有切换的情况下）
	properties.Property("当前 Context 查询多次返回一致结果", prop.ForAll(
		func(dummy int) bool {
			// 第一次查询
			current1 := manager.GetCurrentContext()

			// 第二次查询
			current2 := manager.GetCurrentContext()

			// 第三次查询
			current3 := manager.GetCurrentContext()

			// 验证结果一致
			return current1 == current2 && current2 == current3
		},
		gen.Int(),
	))

	// 属性 6: 默认 Namespace 查询幂等性
	// 多次查询同一个 context 的默认 namespace 应该返回相同的结果
	properties.Property("默认 Namespace 查询多次返回一致结果", prop.ForAll(
		func(contextName string) bool {
			// 第一次查询
			ns1, err1 := manager.GetDefaultNamespaceForContext(contextName)

			// 第二次查询
			ns2, err2 := manager.GetDefaultNamespaceForContext(contextName)

			// 第三次查询
			ns3, err3 := manager.GetDefaultNamespaceForContext(contextName)

			// 验证错误状态一致
			if (err1 != nil) != (err2 != nil) || (err2 != nil) != (err3 != nil) {
				return false
			}

			// 如果 context 不存在，这是预期的
			if err1 != nil {
				return true
			}

			// 验证 namespace 一致
			return ns1 == ns2 && ns2 == ns3
		},
		genContextName(manager),
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// 辅助函数：生成工具类别
func genToolCategory() gopter.Gen {
	return gen.OneGenOf(
		gen.Const("all"),
		gen.Const(string(tools.CategoryQuery)),
		gen.Const(string(tools.CategoryCreate)),
		gen.Const(string(tools.CategoryUpdate)),
		gen.Const(string(tools.CategoryDelete)),
		gen.Const("invalid-category"),
	).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：生成工具名称
func genToolName() gopter.Gen {
	validTools := []interface{}{
		"get_nodes",
		"get_namespaces",
		"get_pods",
		"get_deployments",
		"get_services",
		"get_configmaps",
		"get_secrets",
		"describe_resource",
		"get_pod_logs",
		"get_events",
	}

	return gen.OneGenOf(
		gen.OneConstOf(validTools...),
		gen.Identifier(),
	).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：生成 context 名称
func genContextName(manager *k8s.K8SClientManager) gopter.Gen {
	contextNames := manager.GetContextNames()
	if len(contextNames) == 0 {
		return gen.Identifier()
	}

	interfaces := make([]interface{}, len(contextNames))
	for i, name := range contextNames {
		interfaces[i] = name
	}

	return gen.OneGenOf(
		gen.OneConstOf(interfaces...),
		gen.Identifier(),
	).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：提取工具名称列表
func extractToolNames(tools []*tools.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

// 辅助函数：提取工具名称集合（用于忽略顺序的比较）
func extractToolNamesSet(toolsList []*tools.Tool) map[string]bool {
	names := make(map[string]bool)
	for _, tool := range toolsList {
		names[tool.Name] = true
	}
	return names
}

// 辅助函数：提取 context 名称列表
func extractContextNames(contexts []*k8s.ContextInfo) []string {
	names := make([]string, len(contexts))
	for i, ctx := range contexts {
		names[i] = ctx.Name
	}
	return names
}

// 辅助函数：提取 context 名称集合（用于忽略顺序的比较）
func extractContextNamesSet(contexts []*k8s.ContextInfo) map[string]bool {
	names := make(map[string]bool)
	for _, ctx := range contexts {
		names[ctx.Name] = true
	}
	return names
}

// 辅助函数：比较两个 map 是否相等
func mapsEqual(m1, m2 map[string]bool) bool {
	if len(m1) != len(m2) {
		return false
	}
	for k := range m1 {
		if !m2[k] {
			return false
		}
	}
	return true
}

// 辅助函数：创建测试用的 kubeconfig 文件
func createTestKubeconfigForIdempotency(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	tmpDir, err := os.MkdirTemp(cwd, "kubectl-mcp-idempotency-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	kubeconfigPath := filepath.Join(tmpDir, "config")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster-1:6443
  name: test-cluster-1
- cluster:
    server: https://test-cluster-2:6443
  name: test-cluster-2
contexts:
- context:
    cluster: test-cluster-1
    user: test-user-1
    namespace: default
  name: test-context-1
- context:
    cluster: test-cluster-2
    user: test-user-2
    namespace: production
  name: test-context-2
current-context: test-context-1
users:
- name: test-user-1
  user:
    token: test-token-1
- name: test-user-2
  user:
    token: test-token-2
`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("创建临时 kubeconfig 失败: %v", err)
	}

	return kubeconfigPath
}

// TestProperty_QueryIdempotency_EdgeCases 测试查询幂等性的边界情况
func TestProperty_QueryIdempotency_EdgeCases(t *testing.T) {
	kubeconfigPath := createTestKubeconfigForIdempotency(t)
	defer os.Remove(kubeconfigPath)

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}
	defer manager.Close()

	t.Run("空工具注册表查询返回空列表", func(t *testing.T) {
		registry := tools.NewToolRegistry()

		// 多次查询空注册表
		result1 := registry.GetAllTools()
		result2 := registry.GetAllTools()
		result3 := registry.GetAllTools()

		// 验证都返回空列表
		if len(result1) != 0 || len(result2) != 0 || len(result3) != 0 {
			t.Error("空注册表应该返回空列表")
		}
	})

	t.Run("查询不存在的工具返回一致的结果", func(t *testing.T) {
		registry := tools.NewToolRegistry()
		tools.RegisterQueryTools(registry)

		// 多次查询不存在的工具
		_, exists1 := registry.GetTool("nonexistent-tool")
		_, exists2 := registry.GetTool("nonexistent-tool")
		_, exists3 := registry.GetTool("nonexistent-tool")

		// 验证都返回不存在
		if exists1 || exists2 || exists3 {
			t.Error("不存在的工具应该返回 false")
		}
	})

	t.Run("查询不存在的 context 返回一致的错误", func(t *testing.T) {
		// 多次查询不存在的 context
		_, err1 := manager.GetContextInfo("nonexistent-context")
		_, err2 := manager.GetContextInfo("nonexistent-context")
		_, err3 := manager.GetContextInfo("nonexistent-context")

		// 验证都返回错误
		if err1 == nil || err2 == nil || err3 == nil {
			t.Error("不存在的 context 应该返回错误")
		}
	})

	t.Run("并发查询工具列表返回一致结果", func(t *testing.T) {
		registry := tools.NewToolRegistry()
		tools.RegisterQueryTools(registry)

		results := make(chan []*tools.Tool, 10)

		// 并发查询
		for i := 0; i < 10; i++ {
			go func() {
				results <- registry.GetAllTools()
			}()
		}

		// 收集结果
		var allResults [][]*tools.Tool
		for i := 0; i < 10; i++ {
			allResults = append(allResults, <-results)
		}

		// 验证所有结果的长度一致
		firstLen := len(allResults[0])
		for i := 1; i < len(allResults); i++ {
			if len(allResults[i]) != firstLen {
				t.Errorf("并发查询结果长度不一致: %d vs %d", len(allResults[i]), firstLen)
			}
		}
	})

	t.Run("并发查询 context 列表返回一致结果", func(t *testing.T) {
		results := make(chan []*k8s.ContextInfo, 10)

		// 并发查询
		for i := 0; i < 10; i++ {
			go func() {
				results <- manager.ListContexts()
			}()
		}

		// 收集结果
		var allResults [][]*k8s.ContextInfo
		for i := 0; i < 10; i++ {
			allResults = append(allResults, <-results)
		}

		// 验证所有结果的长度一致
		firstLen := len(allResults[0])
		for i := 1; i < len(allResults); i++ {
			if len(allResults[i]) != firstLen {
				t.Errorf("并发查询结果长度不一致: %d vs %d", len(allResults[i]), firstLen)
			}
		}
	})

	t.Run("查询工具类别返回一致结果", func(t *testing.T) {
		registry := tools.NewToolRegistry()
		tools.RegisterQueryTools(registry)

		// 多次查询查询类工具
		result1 := registry.GetToolsByCategory(tools.CategoryQuery)
		result2 := registry.GetToolsByCategory(tools.CategoryQuery)
		result3 := registry.GetToolsByCategory(tools.CategoryQuery)

		// 验证结果一致
		if len(result1) != len(result2) || len(result2) != len(result3) {
			t.Errorf("查询类工具数量不一致: %d, %d, %d", len(result1), len(result2), len(result3))
		}

		// 验证工具名称集合一致（不考虑顺序）
		names1 := extractToolNamesSet(result1)
		names2 := extractToolNamesSet(result2)
		names3 := extractToolNamesSet(result3)

		if !mapsEqual(names1, names2) || !mapsEqual(names2, names3) {
			t.Error("查询类工具名称集合不一致")
		}
	})
}
