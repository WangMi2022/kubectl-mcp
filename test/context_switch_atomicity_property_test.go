package test

import (
	"testing"
	"time"

	"kubectl-mcp/internal/k8s"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_ContextSwitchAtomicity 测试 Context 切换原子性属性
// Property 9: Context 切换原子性
// Validates: Requirements 3.7
// Feature: kubectl-mcp-server, Property 9: 对于任何 context 切换操作，要么完全成功切换到新 context，要么保持原 context 不变，不得出现中间状态
func TestProperty_ContextSwitchAtomicity(t *testing.T) {
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

	// 属性 1: Context 切换要么成功要么保持原状态
	properties.Property("Context 切换要么成功要么保持原状态", prop.ForAll(
		func(contextName string) bool {
			// 记录原始 context
			originalContext := manager.GetCurrentContext()

			// 尝试切换 context
			err := manager.SwitchContext(contextName)

			// 获取切换后的 context
			currentContext := manager.GetCurrentContext()

			if err != nil {
				// 切换失败，验证保持原状态
				return currentContext == originalContext
			}

			// 切换成功，验证已切换到新 context
			return currentContext == contextName
		},
		genValidContextName(manager),
	))

	// 属性 2: 连续切换 context 保持一致性
	properties.Property("连续切换 context 保持一致性", prop.ForAll(
		func(contextNames []string) bool {
			if len(contextNames) == 0 {
				return true
			}

			// 记录原始 context
			originalContext := manager.GetCurrentContext()

			// 连续切换 context
			var lastSuccessfulContext string
			for _, ctxName := range contextNames {
				err := manager.SwitchContext(ctxName)
				if err == nil {
					lastSuccessfulContext = ctxName
				}
			}

			// 验证最终状态
			currentContext := manager.GetCurrentContext()

			// 如果有成功的切换，应该是最后一个成功的 context
			if lastSuccessfulContext != "" {
				return currentContext == lastSuccessfulContext
			}

			// 如果所有切换都失败，应该保持原状态
			return currentContext == originalContext
		},
		gen.SliceOfN(5, genValidContextName(manager)),
	))

	// 属性 3: 切换到当前 context 是幂等的
	properties.Property("切换到当前 context 是幂等的", prop.ForAll(
		func(iterations int) bool {
			if iterations <= 0 {
				return true
			}

			// 获取当前 context
			currentContext := manager.GetCurrentContext()

			// 多次切换到当前 context
			for i := 0; i < iterations; i++ {
				err := manager.SwitchContext(currentContext)
				if err != nil {
					return false
				}

				// 验证仍然是当前 context
				if manager.GetCurrentContext() != currentContext {
					return false
				}
			}

			return true
		},
		gen.IntRange(1, 10),
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// TestProperty_ContextSwitchAtomicity_EdgeCases 测试 Context 切换原子性的边界情况
func TestProperty_ContextSwitchAtomicity_EdgeCases(t *testing.T) {
	kubeconfigPath := createMultiContextKubeconfig(t)

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}
	defer manager.Close()

	t.Run("切换到不存在的 context 保持原状态", func(t *testing.T) {
		originalContext := manager.GetCurrentContext()

		err := manager.SwitchContext("nonexistent-context-12345")
		if err == nil {
			t.Error("期望切换失败，但成功了")
		}

		currentContext := manager.GetCurrentContext()
		if currentContext != originalContext {
			t.Errorf("切换失败后 context 被改变: 期望 %s, 实际 %s", originalContext, currentContext)
		}
	})

	t.Run("快速连续切换 context", func(t *testing.T) {
		contexts := manager.GetContextNames()
		if len(contexts) < 2 {
			t.Skip("需要至少 2 个 context")
		}

		// 快速连续切换
		for i := 0; i < 100; i++ {
			ctxName := contexts[i%len(contexts)]
			err := manager.SwitchContext(ctxName)
			if err != nil {
				t.Errorf("切换到 %s 失败: %v", ctxName, err)
			}

			// 验证切换成功
			if manager.GetCurrentContext() != ctxName {
				t.Errorf("切换后 context 不匹配: 期望 %s, 实际 %s", ctxName, manager.GetCurrentContext())
			}

			// 短暂延迟
			time.Sleep(1 * time.Millisecond)
		}
	})

	t.Run("切换到空字符串 context", func(t *testing.T) {
		originalContext := manager.GetCurrentContext()

		err := manager.SwitchContext("")
		if err == nil {
			t.Error("期望切换到空 context 失败，但成功了")
		}

		currentContext := manager.GetCurrentContext()
		if currentContext != originalContext {
			t.Errorf("切换失败后 context 被改变: 期望 %s, 实际 %s", originalContext, currentContext)
		}
	})
}
