package k8s

import (
	"fmt"
	"sync"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// K8SClientManager 管理 Kubernetes 客户端和多 context
type K8SClientManager struct {
	// kubeconfig 配置
	kubeconfigPath string
	rawConfig      *api.Config

	// context 管理
	contexts   map[string]*ContextInfo
	currentCtx string

	// 客户端连接池
	clientPool map[string]*ClientSet

	// Service 索引器（每个 context 一个）
	serviceIndexers map[string]*ServiceIndexer

	// 并发控制
	mu sync.RWMutex
}

// ClientSet 包含 Kubernetes 客户端集合
type ClientSet struct {
	Clientset     kubernetes.Interface // 使用接口以支持 fake client
	DynamicClient dynamic.Interface
	RestConfig    *rest.Config
}

// ContextInfo 存储 context 的详细信息
type ContextInfo struct {
	Name      string
	Cluster   string
	User      string
	Namespace string
}

// NewK8SClientManager 创建新的 K8S 客户端管理器
// 参数:
//   - kubeconfigPath: kubeconfig 文件路径
//
// 返回:
//   - *K8SClientManager: 客户端管理器实例
//   - error: 错误信息
func NewK8SClientManager(kubeconfigPath string) (*K8SClientManager, error) {
	if kubeconfigPath == "" {
		return nil, fmt.Errorf("kubeconfig 路径不能为空")
	}

	// 加载 kubeconfig
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("加载 kubeconfig 文件失败: %w", err)
	}

	// 验证 kubeconfig 格式
	if err := clientcmd.Validate(*config); err != nil {
		return nil, fmt.Errorf("kubeconfig 文件格式无效: %w", err)
	}

	// 解析所有 context
	contexts := make(map[string]*ContextInfo)
	for name, ctx := range config.Contexts {
		namespace := ctx.Namespace
		if namespace == "" {
			namespace = "default"
		}

		contexts[name] = &ContextInfo{
			Name:      name,
			Cluster:   ctx.Cluster,
			User:      ctx.AuthInfo,
			Namespace: namespace,
		}
	}

	// 检查是否有可用的 context
	if len(contexts) == 0 {
		return nil, fmt.Errorf("kubeconfig 中没有可用的 context")
	}

	// 确定当前 context
	currentCtx := config.CurrentContext
	if currentCtx == "" {
		// 如果没有设置 current-context，使用第一个可用的 context
		for name := range contexts {
			currentCtx = name
			break
		}
	}

	// 验证当前 context 是否存在
	if _, exists := contexts[currentCtx]; !exists {
		return nil, fmt.Errorf("当前 context '%s' 在 kubeconfig 中不存在", currentCtx)
	}

	manager := &K8SClientManager{
		kubeconfigPath:  kubeconfigPath,
		rawConfig:       config,
		contexts:        contexts,
		currentCtx:      currentCtx,
		clientPool:      make(map[string]*ClientSet),
		serviceIndexers: make(map[string]*ServiceIndexer),
	}

	return manager, nil
}

// GetClient 获取当前 context 的客户端
// 返回:
//   - kubernetes.Interface: Kubernetes 客户端接口
//   - error: 错误信息
func (m *K8SClientManager) GetClient() (kubernetes.Interface, error) {
	m.mu.RLock()
	currentCtx := m.currentCtx
	m.mu.RUnlock()

	return m.GetClientForContext(currentCtx)
}

// GetClientForContext 获取指定 context 的客户端
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - kubernetes.Interface: Kubernetes 客户端接口
//   - error: 错误信息
func (m *K8SClientManager) GetClientForContext(contextName string) (kubernetes.Interface, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证 context 是否存在
	if _, exists := m.contexts[contextName]; !exists {
		return nil, fmt.Errorf("context '%s' 不存在", contextName)
	}

	// 检查连接池中是否已有客户端
	if clientSet, exists := m.clientPool[contextName]; exists {
		return clientSet.Clientset, nil
	}

	// 创建新的客户端
	clientSet, err := m.createClientForContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("创建 context '%s' 的客户端失败: %w", contextName, err)
	}

	// 添加到连接池
	m.clientPool[contextName] = clientSet

	return clientSet.Clientset, nil
}

// GetDynamicClient 获取当前 context 的动态客户端
// 返回:
//   - dynamic.Interface: 动态客户端
//   - error: 错误信息
func (m *K8SClientManager) GetDynamicClient() (dynamic.Interface, error) {
	m.mu.RLock()
	currentCtx := m.currentCtx
	m.mu.RUnlock()

	return m.GetDynamicClientForContext(currentCtx)
}

// GetDynamicClientForContext 获取指定 context 的动态客户端
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - dynamic.Interface: 动态客户端
//   - error: 错误信息
func (m *K8SClientManager) GetDynamicClientForContext(contextName string) (dynamic.Interface, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证 context 是否存在
	if _, exists := m.contexts[contextName]; !exists {
		return nil, fmt.Errorf("context '%s' 不存在", contextName)
	}

	// 检查连接池中是否已有客户端
	if clientSet, exists := m.clientPool[contextName]; exists {
		return clientSet.DynamicClient, nil
	}

	// 创建新的客户端
	clientSet, err := m.createClientForContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("创建 context '%s' 的客户端失败: %w", contextName, err)
	}

	// 添加到连接池
	m.clientPool[contextName] = clientSet

	return clientSet.DynamicClient, nil
}

// GetRestConfig 获取当前 context 的 REST 配置
// 返回:
//   - *rest.Config: REST 配置
//   - error: 错误信息
func (m *K8SClientManager) GetRestConfig() (*rest.Config, error) {
	m.mu.RLock()
	currentCtx := m.currentCtx
	m.mu.RUnlock()

	return m.GetRestConfigForContext(currentCtx)
}

// GetRestConfigForContext 获取指定 context 的 REST 配置
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - *rest.Config: REST 配置
//   - error: 错误信息
func (m *K8SClientManager) GetRestConfigForContext(contextName string) (*rest.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证 context 是否存在
	if _, exists := m.contexts[contextName]; !exists {
		return nil, fmt.Errorf("context '%s' 不存在", contextName)
	}

	// 检查连接池中是否已有客户端
	if clientSet, exists := m.clientPool[contextName]; exists {
		return clientSet.RestConfig, nil
	}

	// 创建新的客户端
	clientSet, err := m.createClientForContext(contextName)
	if err != nil {
		return nil, fmt.Errorf("创建 context '%s' 的客户端失败: %w", contextName, err)
	}

	// 添加到连接池
	m.clientPool[contextName] = clientSet

	return clientSet.RestConfig, nil
}

// createClientForContext 为指定 context 创建客户端
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - *ClientSet: 客户端集合
//   - error: 错误信息
func (m *K8SClientManager) createClientForContext(contextName string) (*ClientSet, error) {
	// 构建 REST 配置
	configLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: m.kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: contextName,
		},
	)

	restConfig, err := configLoader.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("构建 REST 配置失败: %w", err)
	}

	// 创建标准客户端
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建 Kubernetes 客户端失败: %w", err)
	}

	// 创建动态客户端
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("创建动态客户端失败: %w", err)
	}

	return &ClientSet{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		RestConfig:    restConfig,
	}, nil
}

// GetCurrentContext 获取当前使用的 context 名称
// 返回:
//   - string: 当前 context 名称
func (m *K8SClientManager) GetCurrentContext() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentCtx
}

// GetKubeconfigPath 获取 kubeconfig 文件路径
// 返回:
//   - string: kubeconfig 文件路径
func (m *K8SClientManager) GetKubeconfigPath() string {
	return m.kubeconfigPath
}

// Close 关闭所有客户端连接
func (m *K8SClientManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止所有 Service 索引器
	for _, indexer := range m.serviceIndexers {
		indexer.Stop()
	}
	m.serviceIndexers = make(map[string]*ServiceIndexer)

	// 清空连接池
	m.clientPool = make(map[string]*ClientSet)

	return nil
}

// GetServiceIndexer 获取当前 context 的 Service 索引器
func (m *K8SClientManager) GetServiceIndexer() (*ServiceIndexer, error) {
	m.mu.RLock()
	currentCtx := m.currentCtx
	m.mu.RUnlock()

	return m.GetServiceIndexerForContext(currentCtx)
}

// GetServiceIndexerForContext 获取指定 context 的 Service 索引器
func (m *K8SClientManager) GetServiceIndexerForContext(contextName string) (*ServiceIndexer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在
	if indexer, exists := m.serviceIndexers[contextName]; exists {
		return indexer, nil
	}

	// 确保客户端已创建
	clientSet, exists := m.clientPool[contextName]
	if !exists {
		var err error
		clientSet, err = m.createClientForContext(contextName)
		if err != nil {
			return nil, fmt.Errorf("创建客户端失败: %w", err)
		}
		m.clientPool[contextName] = clientSet
	}

	// 创建并启动索引器
	indexer := NewServiceIndexer(clientSet.Clientset, contextName)
	if err := indexer.Start(); err != nil {
		return nil, fmt.Errorf("启动 Service 索引器失败: %w", err)
	}

	m.serviceIndexers[contextName] = indexer
	return indexer, nil
}

// GetClusterServer 获取当前 context 对应的集群服务器地址
func (m *K8SClientManager) GetClusterServer() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 获取当前 context 信息
	ctxInfo, exists := m.contexts[m.currentCtx]
	if !exists {
		return "unknown"
	}

	// 获取集群信息
	if cluster, exists := m.rawConfig.Clusters[ctxInfo.Cluster]; exists {
		return cluster.Server
	}

	return "unknown"
}
