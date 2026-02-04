package k8s

import (
	"fmt"
	"sync"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// MockK8SClientManager Mock K8S 客户端管理器（用于测试）
type MockK8SClientManager struct {
	fakeClient kubernetes.Interface
	contexts   map[string]*ContextInfo
	currentCtx string
	mu         sync.RWMutex
}

// NewMockK8SClientManager 创建 Mock K8S 客户端管理器
func NewMockK8SClientManager(fakeClient kubernetes.Interface) *MockK8SClientManager {
	// 创建默认 context
	contexts := map[string]*ContextInfo{
		"default": {
			Name:      "default",
			Cluster:   "test-cluster",
			User:      "test-user",
			Namespace: "default",
		},
		"context1": {
			Name:      "context1",
			Cluster:   "cluster1",
			User:      "user1",
			Namespace: "default",
		},
		"context2": {
			Name:      "context2",
			Cluster:   "cluster2",
			User:      "user2",
			Namespace: "default",
		},
		"context3": {
			Name:      "context3",
			Cluster:   "cluster3",
			User:      "user3",
			Namespace: "default",
		},
	}

	return &MockK8SClientManager{
		fakeClient: fakeClient,
		contexts:   contexts,
		currentCtx: "default",
	}
}

// GetClient 获取当前 context 的客户端
func (m *MockK8SClientManager) GetClient() (kubernetes.Interface, error) {
	return m.fakeClient, nil
}

// GetClientForContext 获取指定 context 的客户端
func (m *MockK8SClientManager) GetClientForContext(contextName string) (kubernetes.Interface, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.contexts[contextName]; !exists {
		return nil, fmt.Errorf("context '%s' 不存在", contextName)
	}

	return m.fakeClient, nil
}

// GetDynamicClient 获取当前 context 的动态客户端
func (m *MockK8SClientManager) GetDynamicClient() (dynamic.Interface, error) {
	return nil, fmt.Errorf("mock 不支持动态客户端")
}

// GetDynamicClientForContext 获取指定 context 的动态客户端
func (m *MockK8SClientManager) GetDynamicClientForContext(contextName string) (dynamic.Interface, error) {
	return nil, fmt.Errorf("mock 不支持动态客户端")
}

// GetRestConfig 获取当前 context 的 REST 配置
func (m *MockK8SClientManager) GetRestConfig() (*rest.Config, error) {
	return &rest.Config{}, nil
}

// GetRestConfigForContext 获取指定 context 的 REST 配置
func (m *MockK8SClientManager) GetRestConfigForContext(contextName string) (*rest.Config, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.contexts[contextName]; !exists {
		return nil, fmt.Errorf("context '%s' 不存在", contextName)
	}

	return &rest.Config{}, nil
}

// GetCurrentContext 获取当前使用的 context 名称
func (m *MockK8SClientManager) GetCurrentContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCtx
}

// SwitchContext 切换 context
func (m *MockK8SClientManager) SwitchContext(contextName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.contexts[contextName]; !exists {
		return fmt.Errorf("context '%s' 不存在", contextName)
	}

	m.currentCtx = contextName
	return nil
}

// ListContexts 列出所有 context
func (m *MockK8SClientManager) ListContexts() []*ContextInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contexts := make([]*ContextInfo, 0, len(m.contexts))
	for _, ctx := range m.contexts {
		contexts = append(contexts, ctx)
	}
	return contexts
}

// GetKubeconfigPath 获取 kubeconfig 文件路径
func (m *MockK8SClientManager) GetKubeconfigPath() string {
	return "/mock/kubeconfig"
}

// Close 关闭所有客户端连接
func (m *MockK8SClientManager) Close() error {
	return nil
}
