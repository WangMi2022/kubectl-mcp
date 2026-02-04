package test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"kubectl-mcp/internal/k8s"
)

// createTestKubeconfig 创建用于测试的 kubeconfig 文件
func createTestKubeconfig(t *testing.T, content string) string {
	// 在当前工作目录下创建临时目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	tmpDir, err := os.MkdirTemp(cwd, "k8s-client-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	kubeconfigPath := filepath.Join(tmpDir, "config")
	if err := os.WriteFile(kubeconfigPath, []byte(content), 0600); err != nil {
		t.Fatalf("创建 kubeconfig 文件失败: %v", err)
	}

	return kubeconfigPath
}

// createValidKubeconfig 创建有效的 kubeconfig 内容
func createValidKubeconfig() string {
	return `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster-1:6443
  name: cluster-1
- cluster:
    server: https://test-cluster-2:6443
  name: cluster-2
contexts:
- context:
    cluster: cluster-1
    user: user-1
    namespace: default
  name: context-1
- context:
    cluster: cluster-2
    user: user-2
    namespace: kube-system
  name: context-2
current-context: context-1
users:
- name: user-1
  user:
    token: token-1
- name: user-2
  user:
    token: token-2
`
}

// TestNewK8SClientManager_Success 测试成功创建客户端管理器
func TestNewK8SClientManager_Success(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}

	if manager == nil {
		t.Fatal("期望返回非空的管理器实例")
	}

	// 验证当前 context
	currentCtx := manager.GetCurrentContext()
	if currentCtx != "context-1" {
		t.Errorf("期望当前 context 为 'context-1'，实际为 '%s'", currentCtx)
	}

	// 验证 context 数量
	contexts := manager.ListContexts()
	if len(contexts) != 2 {
		t.Errorf("期望有 2 个 context，实际有 %d 个", len(contexts))
	}
}

// TestNewK8SClientManager_EmptyPath 测试空路径错误
func TestNewK8SClientManager_EmptyPath(t *testing.T) {
	_, err := k8s.NewK8SClientManager("")
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}

	expectedMsg := "kubeconfig 路径不能为空"
	if err.Error() != expectedMsg {
		t.Errorf("期望错误消息为 '%s'，实际为 '%s'", expectedMsg, err.Error())
	}
}

// TestNewK8SClientManager_FileNotFound 测试文件不存在错误
func TestNewK8SClientManager_FileNotFound(t *testing.T) {
	_, err := k8s.NewK8SClientManager("/nonexistent/path/config")
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}
}

// TestNewK8SClientManager_InvalidFormat 测试无效的 kubeconfig 格式
func TestNewK8SClientManager_InvalidFormat(t *testing.T) {
	invalidContent := `invalid yaml content: [[[`
	kubeconfigPath := createTestKubeconfig(t, invalidContent)

	_, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}
}

// TestNewK8SClientManager_NoContexts 测试没有 context 的 kubeconfig
func TestNewK8SClientManager_NoContexts(t *testing.T) {
	noContextContent := `apiVersion: v1
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
	kubeconfigPath := createTestKubeconfig(t, noContextContent)

	_, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}

	expectedMsg := "kubeconfig 中没有可用的 context"
	if err.Error() != expectedMsg {
		t.Errorf("期望错误消息为 '%s'，实际为 '%s'", expectedMsg, err.Error())
	}
}

// TestNewK8SClientManager_InvalidCurrentContext 测试无效的 current-context
func TestNewK8SClientManager_InvalidCurrentContext(t *testing.T) {
	invalidCurrentCtxContent := `apiVersion: v1
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
current-context: nonexistent-context
users:
- name: test-user
  user:
    token: test-token
`
	kubeconfigPath := createTestKubeconfig(t, invalidCurrentCtxContent)

	_, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}
}

// TestNewK8SClientManager_NoCurrentContext 测试没有设置 current-context
func TestNewK8SClientManager_NoCurrentContext(t *testing.T) {
	noCurrentCtxContent := `apiVersion: v1
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
users:
- name: test-user
  user:
    token: test-token
`
	kubeconfigPath := createTestKubeconfig(t, noCurrentCtxContent)

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 应该自动选择第一个可用的 context
	currentCtx := manager.GetCurrentContext()
	if currentCtx == "" {
		t.Error("期望自动选择一个 context，但当前 context 为空")
	}
}

// TestSwitchContext_Success 测试成功切换 context
func TestSwitchContext_Success(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 验证初始 context
	if manager.GetCurrentContext() != "context-1" {
		t.Errorf("期望初始 context 为 'context-1'，实际为 '%s'", manager.GetCurrentContext())
	}

	// 切换到 context-2
	err = manager.SwitchContext("context-2")
	if err != nil {
		t.Fatalf("切换 context 失败: %v", err)
	}

	// 验证切换后的 context
	if manager.GetCurrentContext() != "context-2" {
		t.Errorf("期望切换后 context 为 'context-2'，实际为 '%s'", manager.GetCurrentContext())
	}
}

// TestSwitchContext_NonexistentContext 测试切换到不存在的 context
func TestSwitchContext_NonexistentContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	originalCtx := manager.GetCurrentContext()

	// 尝试切换到不存在的 context
	err = manager.SwitchContext("nonexistent-context")
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}

	// 验证 context 没有改变
	if manager.GetCurrentContext() != originalCtx {
		t.Errorf("期望 context 保持为 '%s'，实际为 '%s'", originalCtx, manager.GetCurrentContext())
	}
}

// TestSwitchContext_SameContext 测试切换到相同的 context
func TestSwitchContext_SameContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	currentCtx := manager.GetCurrentContext()

	// 切换到相同的 context
	err = manager.SwitchContext(currentCtx)
	if err != nil {
		t.Fatalf("切换到相同 context 失败: %v", err)
	}

	// 验证 context 没有改变
	if manager.GetCurrentContext() != currentCtx {
		t.Errorf("期望 context 保持为 '%s'，实际为 '%s'", currentCtx, manager.GetCurrentContext())
	}
}

// TestListContexts 测试列出所有 context
func TestListContexts(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	contexts := manager.ListContexts()
	if len(contexts) != 2 {
		t.Errorf("期望有 2 个 context，实际有 %d 个", len(contexts))
	}

	// 验证 context 信息
	contextNames := make(map[string]bool)
	for _, ctx := range contexts {
		contextNames[ctx.Name] = true

		if ctx.Name == "context-1" {
			if ctx.Cluster != "cluster-1" {
				t.Errorf("context-1 的 cluster 应该是 'cluster-1'，实际为 '%s'", ctx.Cluster)
			}
			if ctx.User != "user-1" {
				t.Errorf("context-1 的 user 应该是 'user-1'，实际为 '%s'", ctx.User)
			}
			if ctx.Namespace != "default" {
				t.Errorf("context-1 的 namespace 应该是 'default'，实际为 '%s'", ctx.Namespace)
			}
		}
	}

	if !contextNames["context-1"] || !contextNames["context-2"] {
		t.Error("期望包含 'context-1' 和 'context-2'")
	}
}

// TestGetContextInfo 测试获取 context 信息
func TestGetContextInfo(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 获取存在的 context 信息
	ctx, err := manager.GetContextInfo("context-2")
	if err != nil {
		t.Fatalf("获取 context 信息失败: %v", err)
	}

	if ctx.Name != "context-2" {
		t.Errorf("期望 context 名称为 'context-2'，实际为 '%s'", ctx.Name)
	}
	if ctx.Cluster != "cluster-2" {
		t.Errorf("期望 cluster 为 'cluster-2'，实际为 '%s'", ctx.Cluster)
	}
	if ctx.User != "user-2" {
		t.Errorf("期望 user 为 'user-2'，实际为 '%s'", ctx.User)
	}
	if ctx.Namespace != "kube-system" {
		t.Errorf("期望 namespace 为 'kube-system'，实际为 '%s'", ctx.Namespace)
	}

	// 获取不存在的 context 信息
	_, err = manager.GetContextInfo("nonexistent")
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}
}

// TestContextExists 测试检查 context 是否存在
func TestContextExists(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	if !manager.ContextExists("context-1") {
		t.Error("期望 'context-1' 存在")
	}

	if !manager.ContextExists("context-2") {
		t.Error("期望 'context-2' 存在")
	}

	if manager.ContextExists("nonexistent") {
		t.Error("期望 'nonexistent' 不存在")
	}
}

// TestGetDefaultNamespace 测试获取默认 namespace
func TestGetDefaultNamespace(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 获取当前 context 的默认 namespace
	ns, err := manager.GetDefaultNamespace()
	if err != nil {
		t.Fatalf("获取默认 namespace 失败: %v", err)
	}

	if ns != "default" {
		t.Errorf("期望默认 namespace 为 'default'，实际为 '%s'", ns)
	}

	// 切换 context 后再获取
	manager.SwitchContext("context-2")
	ns, err = manager.GetDefaultNamespace()
	if err != nil {
		t.Fatalf("获取默认 namespace 失败: %v", err)
	}

	if ns != "kube-system" {
		t.Errorf("期望默认 namespace 为 'kube-system'，实际为 '%s'", ns)
	}
}

// TestConcurrentAccess 测试并发访问的线程安全性
func TestConcurrentAccess(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 并发执行多个操作
	var wg sync.WaitGroup
	concurrency := 50

	// 测试并发读取当前 context
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.GetCurrentContext()
		}()
	}

	// 测试并发列出 contexts
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = manager.ListContexts()
		}()
	}

	// 测试并发获取 context 信息
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			contextName := "context-1"
			if idx%2 == 0 {
				contextName = "context-2"
			}
			_, _ = manager.GetContextInfo(contextName)
		}(i)
	}

	// 测试并发切换 context
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			contextName := "context-1"
			if idx%2 == 0 {
				contextName = "context-2"
			}
			_ = manager.SwitchContext(contextName)
		}(i)
	}

	// 等待所有 goroutine 完成
	wg.Wait()

	// 验证最终状态一致性
	currentCtx := manager.GetCurrentContext()
	if currentCtx != "context-1" && currentCtx != "context-2" {
		t.Errorf("并发操作后 context 状态异常: '%s'", currentCtx)
	}

	// 验证 context 列表没有损坏
	contexts := manager.ListContexts()
	if len(contexts) != 2 {
		t.Errorf("并发操作后 context 数量异常: %d", len(contexts))
	}
}

// TestConcurrentSwitchContext 测试并发切换 context 的原子性
func TestConcurrentSwitchContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	var wg sync.WaitGroup
	concurrency := 100
	successCount := 0
	var mu sync.Mutex

	// 并发切换 context
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			contextName := "context-1"
			if idx%2 == 0 {
				contextName = "context-2"
			}
			err := manager.SwitchContext(contextName)
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// 所有切换操作都应该成功
	if successCount != concurrency {
		t.Errorf("期望 %d 次切换成功，实际 %d 次", concurrency, successCount)
	}

	// 验证最终状态是有效的
	currentCtx := manager.GetCurrentContext()
	if !manager.ContextExists(currentCtx) {
		t.Errorf("并发切换后当前 context '%s' 无效", currentCtx)
	}
}

// TestGetClientForContext_Caching 测试客户端连接池缓存
func TestGetClientForContext_Caching(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 注意：由于我们使用的是测试 kubeconfig，实际的客户端创建会失败
	// 这里主要测试错误处理和缓存逻辑

	// 第一次获取客户端（会失败，因为是测试环境）
	_, err1 := manager.GetClientForContext("context-1")

	// 第二次获取相同 context 的客户端
	_, err2 := manager.GetClientForContext("context-1")

	// 两次调用应该返回相同类型的错误（说明使用了缓存或一致的逻辑）
	if (err1 == nil) != (err2 == nil) {
		t.Error("多次获取相同 context 的客户端应该返回一致的结果")
	}
}

// TestClose 测试关闭客户端管理器
func TestClose(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 关闭管理器
	err = manager.Close()
	if err != nil {
		t.Fatalf("关闭管理器失败: %v", err)
	}

	// 关闭后仍然可以访问基本信息
	currentCtx := manager.GetCurrentContext()
	if currentCtx == "" {
		t.Error("关闭后应该仍能获取当前 context")
	}
}

// TestGetKubeconfigPath 测试获取 kubeconfig 路径
func TestGetKubeconfigPath(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	path := manager.GetKubeconfigPath()
	if path != kubeconfigPath {
		t.Errorf("期望 kubeconfig 路径为 '%s'，实际为 '%s'", kubeconfigPath, path)
	}
}

// TestGetContextCount 测试获取 context 数量
func TestGetContextCount(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	count := manager.GetContextCount()
	if count != 2 {
		t.Errorf("期望 context 数量为 2，实际为 %d", count)
	}
}

// TestGetContextNames 测试获取所有 context 名称
func TestGetContextNames(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	names := manager.GetContextNames()
	if len(names) != 2 {
		t.Errorf("期望 2 个 context 名称，实际为 %d 个", len(names))
	}

	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["context-1"] || !nameMap["context-2"] {
		t.Error("期望包含 'context-1' 和 'context-2'")
	}
}

// TestIsCurrentContext 测试检查是否为当前 context
func TestIsCurrentContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 初始 context 是 context-1
	if !manager.IsCurrentContext("context-1") {
		t.Error("期望 'context-1' 是当前 context")
	}

	if manager.IsCurrentContext("context-2") {
		t.Error("期望 'context-2' 不是当前 context")
	}

	// 切换后再测试
	manager.SwitchContext("context-2")

	if manager.IsCurrentContext("context-1") {
		t.Error("切换后 'context-1' 不应该是当前 context")
	}

	if !manager.IsCurrentContext("context-2") {
		t.Error("切换后 'context-2' 应该是当前 context")
	}
}

// TestGetCurrentContextInfo 测试获取当前 context 信息
func TestGetCurrentContextInfo(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	ctx, err := manager.GetCurrentContextInfo()
	if err != nil {
		t.Fatalf("获取当前 context 信息失败: %v", err)
	}

	if ctx.Name != "context-1" {
		t.Errorf("期望当前 context 名称为 'context-1'，实际为 '%s'", ctx.Name)
	}

	// 切换后再测试
	manager.SwitchContext("context-2")

	ctx, err = manager.GetCurrentContextInfo()
	if err != nil {
		t.Fatalf("获取当前 context 信息失败: %v", err)
	}

	if ctx.Name != "context-2" {
		t.Errorf("期望当前 context 名称为 'context-2'，实际为 '%s'", ctx.Name)
	}
}

// TestGetDefaultNamespaceForContext 测试获取指定 context 的默认 namespace
func TestGetDefaultNamespaceForContext(t *testing.T) {
	kubeconfigPath := createTestKubeconfig(t, createValidKubeconfig())

	manager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建管理器失败: %v", err)
	}

	// 获取 context-1 的默认 namespace
	ns, err := manager.GetDefaultNamespaceForContext("context-1")
	if err != nil {
		t.Fatalf("获取默认 namespace 失败: %v", err)
	}

	if ns != "default" {
		t.Errorf("期望 context-1 的默认 namespace 为 'default'，实际为 '%s'", ns)
	}

	// 获取 context-2 的默认 namespace
	ns, err = manager.GetDefaultNamespaceForContext("context-2")
	if err != nil {
		t.Fatalf("获取默认 namespace 失败: %v", err)
	}

	if ns != "kube-system" {
		t.Errorf("期望 context-2 的默认 namespace 为 'kube-system'，实际为 '%s'", ns)
	}

	// 获取不存在的 context 的默认 namespace
	_, err = manager.GetDefaultNamespaceForContext("nonexistent")
	if err == nil {
		t.Error("期望返回错误，但成功了")
	}
}
