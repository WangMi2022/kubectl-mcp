package k8s

import (
	"fmt"
)

// ListContexts 列出所有可用的 context
// 返回:
//   - []*ContextInfo: context 信息列表
func (m *K8SClientManager) ListContexts() []*ContextInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contexts := make([]*ContextInfo, 0, len(m.contexts))
	for _, ctx := range m.contexts {
		contexts = append(contexts, ctx)
	}

	return contexts
}

// SwitchContext 切换到指定的 context
// 参数:
//   - contextName: 要切换到的 context 名称
//
// 返回:
//   - error: 错误信息，如果切换成功则返回 nil
func (m *K8SClientManager) SwitchContext(contextName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 验证 context 是否存在
	if _, exists := m.contexts[contextName]; !exists {
		return fmt.Errorf("context '%s' 不存在", contextName)
	}

	// 验证 context 是否可用（尝试创建客户端）
	if _, exists := m.clientPool[contextName]; !exists {
		// 如果连接池中没有该 context 的客户端，尝试创建
		_, err := m.createClientForContext(contextName)
		if err != nil {
			return fmt.Errorf("无法切换到 context '%s': %w", contextName, err)
		}
	}

	// 切换当前 context
	oldContext := m.currentCtx
	m.currentCtx = contextName

	// 如果切换失败，回滚到原来的 context
	// 注意：这里的切换是原子性的，要么成功切换，要么保持原状态
	if m.currentCtx != contextName {
		m.currentCtx = oldContext
		return fmt.Errorf("切换 context 失败，保持原 context '%s'", oldContext)
	}

	return nil
}

// GetContextInfo 获取指定 context 的详细信息
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - *ContextInfo: context 信息
//   - error: 错误信息
func (m *K8SClientManager) GetContextInfo(contextName string) (*ContextInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, exists := m.contexts[contextName]
	if !exists {
		return nil, fmt.Errorf("context '%s' 不存在", contextName)
	}

	// 返回副本，避免外部修改
	return &ContextInfo{
		Name:      ctx.Name,
		Cluster:   ctx.Cluster,
		User:      ctx.User,
		Namespace: ctx.Namespace,
	}, nil
}

// GetCurrentContextInfo 获取当前 context 的详细信息
// 返回:
//   - *ContextInfo: 当前 context 信息
//   - error: 错误信息
func (m *K8SClientManager) GetCurrentContextInfo() (*ContextInfo, error) {
	m.mu.RLock()
	currentCtx := m.currentCtx
	m.mu.RUnlock()

	return m.GetContextInfo(currentCtx)
}

// ContextExists 检查指定的 context 是否存在
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - bool: context 是否存在
func (m *K8SClientManager) ContextExists(contextName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.contexts[contextName]
	return exists
}

// GetContextCount 获取可用 context 的数量
// 返回:
//   - int: context 数量
func (m *K8SClientManager) GetContextCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.contexts)
}

// GetContextNames 获取所有 context 的名称列表
// 返回:
//   - []string: context 名称列表
func (m *K8SClientManager) GetContextNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.contexts))
	for name := range m.contexts {
		names = append(names, name)
	}

	return names
}

// IsCurrentContext 检查指定的 context 是否为当前 context
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - bool: 是否为当前 context
func (m *K8SClientManager) IsCurrentContext(contextName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.currentCtx == contextName
}

// GetDefaultNamespace 获取当前 context 的默认 namespace
// 返回:
//   - string: 默认 namespace
//   - error: 错误信息
func (m *K8SClientManager) GetDefaultNamespace() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, exists := m.contexts[m.currentCtx]
	if !exists {
		return "", fmt.Errorf("当前 context '%s' 不存在", m.currentCtx)
	}

	return ctx.Namespace, nil
}

// GetDefaultNamespaceForContext 获取指定 context 的默认 namespace
// 参数:
//   - contextName: context 名称
//
// 返回:
//   - string: 默认 namespace
//   - error: 错误信息
func (m *K8SClientManager) GetDefaultNamespaceForContext(contextName string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ctx, exists := m.contexts[contextName]
	if !exists {
		return "", fmt.Errorf("context '%s' 不存在", contextName)
	}

	return ctx.Namespace, nil
}
