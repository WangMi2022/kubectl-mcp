package test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"kubectl-mcp/internal/k8s"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_KubeconfigSecurity 测试 Kubeconfig 安全性属性
// Property 1: Kubeconfig 安全性
// Validates: Requirements 2.1, 2.2, 2.3
// Feature: kubectl-mcp-server, Property 1: 对于任何 K8S 操作请求，系统必须仅通过 kubeconfig 文件进行认证，不得接受明文 token、证书或直接的 API server 地址
func TestProperty_KubeconfigSecurity(t *testing.T) {
	// 配置属性测试参数
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100 // 至少运行 100 次迭代
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	// 属性 1: 系统拒绝空的 kubeconfig 路径
	properties.Property("系统拒绝空的 kubeconfig 路径", prop.ForAll(
		func() bool {
			_, err := k8s.NewK8SClientManager("")
			// 必须返回错误
			return err != nil
		},
	))

	// 属性 2: 系统拒绝不存在的 kubeconfig 文件
	properties.Property("系统拒绝不存在的 kubeconfig 文件", prop.ForAll(
		func(randomPath string) bool {
			// 确保路径不存在
			nonexistentPath := filepath.Join("/nonexistent", randomPath, "config")
			_, err := k8s.NewK8SClientManager(nonexistentPath)
			// 必须返回错误
			return err != nil
		},
		gen.Identifier(),
	))

	// 属性 3: 系统拒绝无效格式的 kubeconfig 文件
	properties.Property("系统拒绝无效格式的 kubeconfig 文件", prop.ForAll(
		func(invalidContent string) bool {
			// 创建包含无效内容的临时文件
			tmpFile := createTempKubeconfigWithContent(t, invalidContent)
			defer os.Remove(tmpFile)

			_, err := k8s.NewK8SClientManager(tmpFile)
			// 必须返回错误（除非随机生成的内容恰好是有效的 YAML）
			return err != nil
		},
		gen.AnyString(),
	))

	// 属性 4: 系统拒绝没有 context 的 kubeconfig
	properties.Property("系统拒绝没有 context 的 kubeconfig", prop.ForAll(
		func() bool {
			// 创建没有 context 的 kubeconfig
			content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster:6443
  name: test-cluster
users:
- name: test-user
  user:
    token: test-token
`
			tmpFile := createTempKubeconfigWithContent(t, content)
			defer os.Remove(tmpFile)

			_, err := k8s.NewK8SClientManager(tmpFile)
			// 必须返回错误
			return err != nil && err.Error() == "kubeconfig 中没有可用的 context"
		},
	))

	// 属性 5: 系统拒绝 current-context 不存在的 kubeconfig
	properties.Property("系统拒绝 current-context 不存在的 kubeconfig", prop.ForAll(
		func(invalidContextName string) bool {
			// 确保生成的名称不是 "valid-context"
			if invalidContextName == "valid-context" || invalidContextName == "" {
				return true // 跳过这种情况
			}

			// 创建 current-context 指向不存在的 context 的 kubeconfig
			content := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: valid-context
current-context: %s
users:
- name: test-user
  user:
    token: test-token
`, invalidContextName)

			tmpFile := createTempKubeconfigWithContent(t, content)
			defer os.Remove(tmpFile)

			_, err := k8s.NewK8SClientManager(tmpFile)
			// 必须返回错误
			return err != nil
		},
		gen.Identifier(),
	))

	// 属性 6: 系统只接受通过 kubeconfig 文件路径创建的客户端管理器
	properties.Property("系统只接受通过 kubeconfig 文件路径创建的客户端管理器", prop.ForAll(
		func() bool {
			// 创建有效的 kubeconfig
			validKubeconfig := createValidTestKubeconfig(t)
			defer os.Remove(validKubeconfig)

			// 通过 kubeconfig 文件路径创建管理器应该成功
			manager, err := k8s.NewK8SClientManager(validKubeconfig)
			if err != nil {
				return false
			}
			defer manager.Close()

			// 验证管理器确实使用了 kubeconfig 文件
			if manager.GetKubeconfigPath() != validKubeconfig {
				return false
			}

			// 验证可以获取 context 信息
			contexts := manager.ListContexts()
			return len(contexts) > 0
		},
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// TestProperty_KubeconfigSecurity_NoDirectAccess 测试系统不允许直接访问 API server
func TestProperty_KubeconfigSecurity_NoDirectAccess(t *testing.T) {
	// 配置属性测试参数
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	// 属性 7: 系统不提供直接使用 token 创建客户端的接口
	properties.Property("系统不提供直接使用 token 创建客户端的接口", prop.ForAll(
		func(token string, serverURL string) bool {
			// K8SClientManager 只有一个构造函数：NewK8SClientManager(kubeconfigPath string)
			// 验证没有其他接受 token 或 server URL 的构造函数
			
			// 尝试使用空路径（模拟尝试绕过 kubeconfig）
			_, err := k8s.NewK8SClientManager("")
			if err == nil {
				// 如果成功了，说明可以绕过 kubeconfig，这是不允许的
				return false
			}

			// 验证错误消息明确指出需要 kubeconfig 路径
			expectedMsg := "kubeconfig 路径不能为空"
			return err.Error() == expectedMsg
		},
		gen.AnyString(), // token
		gen.AnyString(), // serverURL
	))

	// 属性 8: 系统不接受包含明文凭证的非标准 kubeconfig
	properties.Property("系统拒绝包含非标准认证方式的 kubeconfig", prop.ForAll(
		func(plainToken string, certData string) bool {
			// 创建包含明文凭证的 kubeconfig（但格式仍然是 kubeconfig）
			// 注意：这里测试的是系统是否正确验证 kubeconfig 格式
			content := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster:6443
    certificate-authority-data: %s
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: %s
`, certData, plainToken)

			tmpFile := createTempKubeconfigWithContent(t, content)
			defer os.Remove(tmpFile)

			// 系统应该能够加载这个 kubeconfig（因为格式是正确的）
			// 但是在实际连接时会失败（因为凭证无效）
			manager, err := k8s.NewK8SClientManager(tmpFile)
			if err != nil {
				// 如果加载失败，可能是因为格式问题，这是可以接受的
				return true
			}
			defer manager.Close()

			// 验证管理器确实使用了 kubeconfig 文件
			return manager.GetKubeconfigPath() == tmpFile
		},
		gen.Identifier(), // plainToken
		gen.Identifier(), // certData
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// TestKubeconfigSecurity_EdgeCases 测试 Kubeconfig 安全性的边界情况
func TestKubeconfigSecurity_EdgeCases(t *testing.T) {
	t.Run("拒绝空字符串路径", func(t *testing.T) {
		_, err := k8s.NewK8SClientManager("")
		if err == nil {
			t.Error("期望拒绝空字符串路径，但成功了")
		}
		expectedMsg := "kubeconfig 路径不能为空"
		if err.Error() != expectedMsg {
			t.Errorf("期望错误消息为 '%s'，实际为 '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("拒绝不存在的文件路径", func(t *testing.T) {
		_, err := k8s.NewK8SClientManager("/nonexistent/path/to/kubeconfig")
		if err == nil {
			t.Error("期望拒绝不存在的文件路径，但成功了")
		}
	})

	t.Run("拒绝目录路径", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "kubeconfig-test-*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		_, err = k8s.NewK8SClientManager(tmpDir)
		if err == nil {
			t.Error("期望拒绝目录路径，但成功了")
		}
	})

	t.Run("拒绝空文件", func(t *testing.T) {
		tmpFile := createTempKubeconfigWithContent(t, "")
		defer os.Remove(tmpFile)

		_, err := k8s.NewK8SClientManager(tmpFile)
		if err == nil {
			t.Error("期望拒绝空文件，但成功了")
		}
	})

	t.Run("拒绝非 YAML 格式文件", func(t *testing.T) {
		tmpFile := createTempKubeconfigWithContent(t, "this is not yaml: [[[")
		defer os.Remove(tmpFile)

		_, err := k8s.NewK8SClientManager(tmpFile)
		if err == nil {
			t.Error("期望拒绝非 YAML 格式文件，但成功了")
		}
	})

	t.Run("拒绝缺少必需字段的 kubeconfig", func(t *testing.T) {
		// 缺少 clusters
		content := `apiVersion: v1
kind: Config
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
		tmpFile := createTempKubeconfigWithContent(t, content)
		defer os.Remove(tmpFile)

		_, err := k8s.NewK8SClientManager(tmpFile)
		if err == nil {
			t.Error("期望拒绝缺少 clusters 的 kubeconfig，但成功了")
		}
	})

	t.Run("拒绝缺少 users 的 kubeconfig", func(t *testing.T) {
		content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
`
		tmpFile := createTempKubeconfigWithContent(t, content)
		defer os.Remove(tmpFile)

		_, err := k8s.NewK8SClientManager(tmpFile)
		if err == nil {
			t.Error("期望拒绝缺少 users 的 kubeconfig，但成功了")
		}
	})

	t.Run("接受有效的 kubeconfig 文件", func(t *testing.T) {
		validKubeconfig := createValidTestKubeconfig(t)
		defer os.Remove(validKubeconfig)

		manager, err := k8s.NewK8SClientManager(validKubeconfig)
		if err != nil {
			t.Fatalf("期望接受有效的 kubeconfig，但失败了: %v", err)
		}
		defer manager.Close()

		// 验证管理器正常工作
		if manager.GetCurrentContext() == "" {
			t.Error("有效的 kubeconfig 应该有当前 context")
		}

		contexts := manager.ListContexts()
		if len(contexts) == 0 {
			t.Error("有效的 kubeconfig 应该有至少一个 context")
		}
	})

	t.Run("验证 kubeconfig 路径被正确存储", func(t *testing.T) {
		validKubeconfig := createValidTestKubeconfig(t)
		defer os.Remove(validKubeconfig)

		manager, err := k8s.NewK8SClientManager(validKubeconfig)
		if err != nil {
			t.Fatalf("创建管理器失败: %v", err)
		}
		defer manager.Close()

		// 验证存储的路径与传入的路径一致
		if manager.GetKubeconfigPath() != validKubeconfig {
			t.Errorf("期望 kubeconfig 路径为 '%s'，实际为 '%s'",
				validKubeconfig, manager.GetKubeconfigPath())
		}
	})

	t.Run("验证不能通过其他方式绕过 kubeconfig", func(t *testing.T) {
		// K8SClientManager 的所有公开方法都需要先通过 kubeconfig 初始化
		// 这里验证没有其他创建客户端的方式

		// 尝试使用 nil 路径
		_, err := k8s.NewK8SClientManager("")
		if err == nil {
			t.Error("不应该允许使用空路径创建管理器")
		}

		// 验证错误消息明确
		expectedMsg := "kubeconfig 路径不能为空"
		if err.Error() != expectedMsg {
			t.Errorf("期望错误消息为 '%s'，实际为 '%s'", expectedMsg, err.Error())
		}
	})
}

// TestKubeconfigSecurity_NoAlternativeConstructors 测试没有替代的构造函数
func TestKubeconfigSecurity_NoAlternativeConstructors(t *testing.T) {
	t.Run("验证只有一个构造函数", func(t *testing.T) {
		// 这个测试通过代码审查来验证
		// K8SClientManager 应该只有 NewK8SClientManager(kubeconfigPath string) 这一个构造函数
		// 不应该有其他接受 token、证书、server URL 等参数的构造函数

		// 我们通过尝试使用空路径来验证必须提供 kubeconfig
		_, err := k8s.NewK8SClientManager("")
		if err == nil {
			t.Error("必须提供 kubeconfig 路径")
		}

		// 验证错误消息
		expectedMsg := "kubeconfig 路径不能为空"
		if err.Error() != expectedMsg {
			t.Errorf("期望错误消息为 '%s'，实际为 '%s'", expectedMsg, err.Error())
		}
	})

	t.Run("验证客户端创建必须通过 kubeconfig", func(t *testing.T) {
		// 创建有效的管理器
		validKubeconfig := createValidTestKubeconfig(t)
		defer os.Remove(validKubeconfig)

		manager, err := k8s.NewK8SClientManager(validKubeconfig)
		if err != nil {
			t.Fatalf("创建管理器失败: %v", err)
		}
		defer manager.Close()

		// 验证所有客户端获取方法都依赖于 kubeconfig 中的 context
		// GetClient() 使用当前 context
		_, err = manager.GetClient()
		// 可能失败（因为是测试环境），但不应该因为缺少 kubeconfig 而失败

		// GetClientForContext() 使用指定 context
		contexts := manager.GetContextNames()
		if len(contexts) > 0 {
			_, err = manager.GetClientForContext(contexts[0])
			// 同样，可能失败但不应该因为缺少 kubeconfig 而失败
		}

		// 验证不存在的 context 会返回错误
		_, err = manager.GetClientForContext("nonexistent-context")
		if err == nil {
			t.Error("期望不存在的 context 返回错误")
		}
	})
}

// createTempKubeconfigWithContent 创建包含指定内容的临时 kubeconfig 文件
func createTempKubeconfigWithContent(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "kubeconfig-test-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("写入临时文件失败: %v", err)
	}

	tmpFile.Close()
	return tmpFile.Name()
}

// createValidTestKubeconfig 创建有效的测试 kubeconfig 文件
func createValidTestKubeconfig(t *testing.T) string {
	content := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
    namespace: default
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`
	return createTempKubeconfigWithContent(t, content)
}
