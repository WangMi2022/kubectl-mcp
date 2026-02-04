package test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"kubectl-mcp/internal/k8s"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_ContextIsolation 测试 Context 隔离性属性
// Property 2: Context 隔离性
// Validates: Requirements 3.4, 3.5
// Feature: kubectl-mcp-server, Property 2: 对于任何工具调用，如果指定了 context 参数，则该操作必须在指定的 context 中执行，不得影响其他 context
func TestProperty_ContextIsolation(t *testing.T) {
	// 创建包含多个 context 的 kubeconfig
	kubeconfigPath := createMultiContextKubeconfig(t)

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

	// 属性 1: 在指定 context 中获取客户端不影响当前 context
	properties.Property("获取指定 context 的客户端不改变当前 context", prop.ForAll(
		func(contextName string) bool {
			// 记录操作前的当前 context
			originalContext := manager.GetCurrentContext()

			// 获取指定 context 的客户端
			_, err := manager.GetClientForContext(contextName)
			if err != nil {
				// 如果 context 不存在，这是预期的错误，不算失败
				return true
			}

			// 验证当前 context 没有改变
			currentContext := manager.GetCurrentContext()
			return currentContext == originalContext
		},
		genValidContextName(manager),
	))

	// 属性 2: 并发访问不同 context 的客户端是线程安全的
	properties.Property("并发访问不同 context 的客户端是线程安全的", prop.ForAll(
		func(contextNames []string) bool {
			if len(contextNames) == 0 {
				return true
			}

			// 记录操作前的当前 context
			originalContext := manager.GetCurrentContext()

			var wg sync.WaitGroup
			errors := make(chan error, len(contextNames))

			// 并发获取不同 context 的客户端
			for _, ctxName := range contextNames {
				wg.Add(1)
				go func(name string) {
					defer wg.Done()
					_, err := manager.GetClientForContext(name)
					if err != nil && manager.ContextExists(name) {
						// 如果 context 存在但获取失败，记录错误
						errors <- err
					}
				}(ctxName)
			}

			wg.Wait()
			close(errors)

			// 检查是否有错误
			for err := range errors {
				if err != nil {
					t.Logf("并发访问错误: %v", err)
					return false
				}
			}

			// 验证当前 context 没有改变
			currentContext := manager.GetCurrentContext()
			return currentContext == originalContext
		},
		gen.SliceOf(genValidContextName(manager)),
	))

	// 属性 3: 切换 context 后，获取客户端应该使用新的 context
	properties.Property("切换 context 后获取客户端使用新 context", prop.ForAll(
		func(contextName string) bool {
			// 如果 context 不存在，跳过
			if !manager.ContextExists(contextName) {
				return true
			}

			// 记录原始 context
			originalContext := manager.GetCurrentContext()

			// 切换到新 context
			err := manager.SwitchContext(contextName)
			if err != nil {
				// 切换失败，验证当前 context 没有改变（原子性）
				return manager.GetCurrentContext() == originalContext
			}

			// 验证当前 context 已经改变
			if manager.GetCurrentContext() != contextName {
				return false
			}

			// 获取客户端（应该使用新 context）
			_, err = manager.GetClient()
			if err != nil {
				return false
			}

			// 恢复原始 context
			manager.SwitchContext(originalContext)

			return true
		},
		genValidContextName(manager),
	))

	// 属性 4: 获取不同 context 的默认 namespace 不互相影响
	properties.Property("获取不同 context 的默认 namespace 不互相影响", prop.ForAll(
		func(contextNames []string) bool {
			if len(contextNames) < 2 {
				return true
			}

			// 过滤出存在的 context
			validContexts := make([]string, 0)
			for _, name := range contextNames {
				if manager.ContextExists(name) {
					validContexts = append(validContexts, name)
				}
			}

			if len(validContexts) < 2 {
				return true
			}

			// 获取每个 context 的默认 namespace
			namespaces := make(map[string]string)
			for _, ctxName := range validContexts {
				ns, err := manager.GetDefaultNamespaceForContext(ctxName)
				if err != nil {
					return false
				}
				namespaces[ctxName] = ns
			}

			// 验证每个 context 的 namespace 是独立的
			// 再次获取应该返回相同的值
			for _, ctxName := range validContexts {
				ns, err := manager.GetDefaultNamespaceForContext(ctxName)
				if err != nil {
					return false
				}
				if ns != namespaces[ctxName] {
					return false
				}
			}

			return true
		},
		gen.SliceOfN(5, genValidContextName(manager)),
	))

	// 属性 5: Context 信息查询不改变当前 context
	properties.Property("查询 context 信息不改变当前 context", prop.ForAll(
		func(contextName string) bool {
			// 记录操作前的当前 context
			originalContext := manager.GetCurrentContext()

			// 查询 context 信息
			_, err := manager.GetContextInfo(contextName)
			if err != nil && !manager.ContextExists(contextName) {
				// context 不存在是预期的
				return true
			}

			// 验证当前 context 没有改变
			currentContext := manager.GetCurrentContext()
			return currentContext == originalContext
		},
		genValidContextName(manager),
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// genValidContextName 生成有效的 context 名称生成器
func genValidContextName(manager *k8s.K8SClientManager) gopter.Gen {
	contextNames := manager.GetContextNames()
	if len(contextNames) == 0 {
		// 如果没有可用的 context，生成随机字符串
		return gen.Identifier()
	}

	// 80% 的概率生成存在的 context，20% 的概率生成不存在的 context
	return gen.OneGenOf(
		gen.OneConstOf(contextNamesToInterfaces(contextNames)...).WithLabel("existing-context"),
		gen.Identifier().WithLabel("random-context"),
	).WithShrinker(gopter.NoShrinker)
}

// contextNamesToInterfaces 将 context 名称切片转换为 interface{} 切片
func contextNamesToInterfaces(names []string) []interface{} {
	result := make([]interface{}, len(names))
	for i, name := range names {
		result[i] = name
	}
	return result
}

// createMultiContextKubeconfig 创建包含多个 context 的临时 kubeconfig 文件
func createMultiContextKubeconfig(t *testing.T) string {
	// 在当前工作目录下创建临时目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	tmpDir, err := os.MkdirTemp(cwd, "kubectl-mcp-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	kubeconfigPath := filepath.Join(tmpDir, "config")

	// 创建包含多个 context 的 kubeconfig
	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://prod-cluster:6443
  name: prod-cluster
- cluster:
    server: https://dev-cluster:6443
  name: dev-cluster
- cluster:
    server: https://test-cluster:6443
  name: test-cluster
contexts:
- context:
    cluster: prod-cluster
    user: prod-user
    namespace: production
  name: prod-context
- context:
    cluster: dev-cluster
    user: dev-user
    namespace: development
  name: dev-context
- context:
    cluster: test-cluster
    user: test-user
    namespace: testing
  name: test-context
- context:
    cluster: prod-cluster
    user: admin-user
    namespace: default
  name: admin-context
current-context: prod-context
users:
- name: prod-user
  user:
    token: prod-token
- name: dev-user
  user:
    token: dev-token
- name: test-user
  user:
    token: test-token
- name: admin-user
  user:
    token: admin-token
`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("创建临时 kubeconfig 失败: %v", err)
	}

	return kubeconfigPath
}

// TestProperty_ContextIsolation_EdgeCases 测试 Context 隔离性的边界情况
func TestProperty_ContextIsolation_EdgeCases(t *testing.T) {
	kubeconfigPath := createMultiContextKubeconfig(t)

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}
	defer manager.Close()

	t.Run("获取不存在的 context 不影响当前 context", func(t *testing.T) {
		originalContext := manager.GetCurrentContext()

		// 尝试获取不存在的 context
		_, err := manager.GetClientForContext("nonexistent-context")
		if err == nil {
			t.Error("期望获取不存在的 context 失败，但成功了")
		}

		// 验证当前 context 没有改变
		if manager.GetCurrentContext() != originalContext {
			t.Errorf("当前 context 被改变了: 期望 %s, 实际 %s", originalContext, manager.GetCurrentContext())
		}
	})

	t.Run("切换到不存在的 context 保持原状态", func(t *testing.T) {
		originalContext := manager.GetCurrentContext()

		// 尝试切换到不存在的 context
		err := manager.SwitchContext("nonexistent-context")
		if err == nil {
			t.Error("期望切换到不存在的 context 失败，但成功了")
		}

		// 验证当前 context 没有改变
		if manager.GetCurrentContext() != originalContext {
			t.Errorf("当前 context 被改变了: 期望 %s, 实际 %s", originalContext, manager.GetCurrentContext())
		}
	})

	t.Run("并发切换 context 是线程安全的", func(t *testing.T) {
		contexts := manager.GetContextNames()
		if len(contexts) < 2 {
			t.Skip("需要至少 2 个 context")
		}

		var wg sync.WaitGroup
		iterations := 50

		// 并发切换 context
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				ctxName := contexts[idx%len(contexts)]
				manager.SwitchContext(ctxName)
			}(i)
		}

		wg.Wait()

		// 验证最终状态是有效的
		currentContext := manager.GetCurrentContext()
		if !manager.ContextExists(currentContext) {
			t.Errorf("并发切换后当前 context 无效: %s", currentContext)
		}
	})

	t.Run("获取 context 信息返回副本，不影响内部状态", func(t *testing.T) {
		contexts := manager.GetContextNames()
		if len(contexts) == 0 {
			t.Skip("没有可用的 context")
		}

		ctxName := contexts[0]
		info1, err := manager.GetContextInfo(ctxName)
		if err != nil {
			t.Fatalf("获取 context 信息失败: %v", err)
		}

		// 修改返回的信息
		info1.Name = "modified-name"
		info1.Namespace = "modified-namespace"

		// 再次获取，验证内部状态没有被修改
		info2, err := manager.GetContextInfo(ctxName)
		if err != nil {
			t.Fatalf("获取 context 信息失败: %v", err)
		}

		if info2.Name == "modified-name" || info2.Namespace == "modified-namespace" {
			t.Error("内部 context 信息被外部修改影响了")
		}
	})

	t.Run("多次获取同一 context 的客户端返回相同实例", func(t *testing.T) {
		contexts := manager.GetContextNames()
		if len(contexts) == 0 {
			t.Skip("没有可用的 context")
		}

		ctxName := contexts[0]

		// 第一次获取
		client1, err := manager.GetClientForContext(ctxName)
		if err != nil {
			t.Fatalf("获取客户端失败: %v", err)
		}

		// 第二次获取
		client2, err := manager.GetClientForContext(ctxName)
		if err != nil {
			t.Fatalf("获取客户端失败: %v", err)
		}

		// 验证返回的是同一个实例（连接池）
		if fmt.Sprintf("%p", client1) != fmt.Sprintf("%p", client2) {
			t.Error("多次获取同一 context 的客户端返回了不同的实例")
		}
	})
}
