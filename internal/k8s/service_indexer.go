package k8s

import (
	"context"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

// ServiceIndexer Service 索引器，提供 O(1) 复杂度的 NodePort/Host 查询
type ServiceIndexer struct {
	// 客户端
	clientset   kubernetes.Interface
	contextName string

	// NodePort 索引: nodePort -> []ServiceKey
	nodePortIndex map[int32][]ServiceKey

	// 所有 Service 缓存: ServiceKey -> *corev1.Service
	serviceCache map[ServiceKey]*corev1.Service

	// 并发控制
	mu sync.RWMutex

	// 生命周期控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 状态
	ready     bool
	lastSync  time.Time
	syncError error
}

// ServiceKey Service 的唯一标识
type ServiceKey struct {
	Namespace string
	Name      string
}

// String 返回 ServiceKey 的字符串表示
func (k ServiceKey) String() string {
	return fmt.Sprintf("%s/%s", k.Namespace, k.Name)
}

// NewServiceIndexer 创建新的 Service 索引器
func NewServiceIndexer(clientset kubernetes.Interface, contextName string) *ServiceIndexer {
	ctx, cancel := context.WithCancel(context.Background())
	return &ServiceIndexer{
		clientset:     clientset,
		contextName:   contextName,
		nodePortIndex: make(map[int32][]ServiceKey),
		serviceCache:  make(map[ServiceKey]*corev1.Service),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start 启动索引器，执行初始同步并开始监听变化
func (si *ServiceIndexer) Start() error {
	// 初始同步
	if err := si.fullSync(); err != nil {
		return fmt.Errorf("初始同步失败: %w", err)
	}

	// 启动 Watch 协程
	si.wg.Add(1)
	go si.watchLoop()

	return nil
}

// Stop 停止索引器
func (si *ServiceIndexer) Stop() {
	si.cancel()
	si.wg.Wait()
}

// IsReady 检查索引器是否就绪
func (si *ServiceIndexer) IsReady() bool {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return si.ready
}

// GetLastSyncTime 获取最后同步时间
func (si *ServiceIndexer) GetLastSyncTime() time.Time {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return si.lastSync
}

// GetSyncError 获取同步错误
func (si *ServiceIndexer) GetSyncError() error {
	si.mu.RLock()
	defer si.mu.RUnlock()
	return si.syncError
}

// fullSync 执行全量同步
func (si *ServiceIndexer) fullSync() error {
	services, err := si.clientset.CoreV1().Services(metav1.NamespaceAll).List(si.ctx, metav1.ListOptions{})
	if err != nil {
		si.mu.Lock()
		si.syncError = err
		si.mu.Unlock()
		fmt.Printf("[ServiceIndexer] 全量同步失败: %v\n", err)
		return err
	}

	si.mu.Lock()
	defer si.mu.Unlock()

	// 清空现有索引
	si.nodePortIndex = make(map[int32][]ServiceKey)
	si.serviceCache = make(map[ServiceKey]*corev1.Service)

	// 统计 NodePort 类型的 Service 数量
	nodePortCount := 0

	// 重建索引
	for i := range services.Items {
		svc := &services.Items[i]
		si.addServiceToIndex(svc)
		if svc.Spec.Type == corev1.ServiceTypeNodePort {
			nodePortCount++
		}
	}

	si.ready = true
	si.lastSync = time.Now()
	si.syncError = nil

	// 打印同步结果
	fmt.Printf("[ServiceIndexer] 全量同步完成: 共 %d 个 Service, 其中 %d 个 NodePort 类型, %d 个唯一 NodePort\n",
		len(services.Items), nodePortCount, len(si.nodePortIndex))

	return nil
}

// watchLoop Watch 循环，监听 Service 变化
func (si *ServiceIndexer) watchLoop() {
	defer si.wg.Done()

	for {
		select {
		case <-si.ctx.Done():
			return
		default:
		}

		// 创建 Watch
		watcher, err := si.clientset.CoreV1().Services(metav1.NamespaceAll).Watch(si.ctx, metav1.ListOptions{})
		if err != nil {
			si.mu.Lock()
			si.syncError = err
			si.mu.Unlock()

			// 等待后重试
			select {
			case <-si.ctx.Done():
				return
			case <-time.After(5 * time.Second):
				// 重新全量同步
				_ = si.fullSync()
				continue
			}
		}

		si.handleWatchEvents(watcher)
	}
}

// handleWatchEvents 处理 Watch 事件
func (si *ServiceIndexer) handleWatchEvents(watcher watch.Interface) {
	defer watcher.Stop()

	for {
		select {
		case <-si.ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				// Watch 通道关闭，需要重新建立
				return
			}

			svc, ok := event.Object.(*corev1.Service)
			if !ok {
				continue
			}

			si.mu.Lock()
			switch event.Type {
			case watch.Added, watch.Modified:
				// 先删除旧索引（如果存在）
				si.removeServiceFromIndex(svc.Namespace, svc.Name)
				// 添加新索引
				si.addServiceToIndex(svc)
			case watch.Deleted:
				si.removeServiceFromIndex(svc.Namespace, svc.Name)
			}
			si.lastSync = time.Now()
			si.mu.Unlock()
		}
	}
}

// addServiceToIndex 将 Service 添加到索引（调用者需持有写锁）
func (si *ServiceIndexer) addServiceToIndex(svc *corev1.Service) {
	key := ServiceKey{Namespace: svc.Namespace, Name: svc.Name}

	// 添加到缓存
	si.serviceCache[key] = svc.DeepCopy()

	// 添加 NodePort 索引
	for _, port := range svc.Spec.Ports {
		if port.NodePort > 0 {
			si.nodePortIndex[port.NodePort] = append(si.nodePortIndex[port.NodePort], key)
		}
	}
}

// removeServiceFromIndex 从索引中移除 Service（调用者需持有写锁）
func (si *ServiceIndexer) removeServiceFromIndex(namespace, name string) {
	key := ServiceKey{Namespace: namespace, Name: name}

	// 获取旧的 Service 以清理 NodePort 索引
	if oldSvc, exists := si.serviceCache[key]; exists {
		for _, port := range oldSvc.Spec.Ports {
			if port.NodePort > 0 {
				si.removeFromNodePortIndex(port.NodePort, key)
			}
		}
	}

	// 从缓存中删除
	delete(si.serviceCache, key)
}

// removeFromNodePortIndex 从 NodePort 索引中移除指定的 ServiceKey
func (si *ServiceIndexer) removeFromNodePortIndex(nodePort int32, key ServiceKey) {
	keys := si.nodePortIndex[nodePort]
	for i, k := range keys {
		if k == key {
			// 删除元素
			si.nodePortIndex[nodePort] = append(keys[:i], keys[i+1:]...)
			break
		}
	}
	// 如果索引为空，删除整个条目
	if len(si.nodePortIndex[nodePort]) == 0 {
		delete(si.nodePortIndex, nodePort)
	}
}

// FindByNodePort 通过 NodePort 查找 Service（O(1) 复杂度）
func (si *ServiceIndexer) FindByNodePort(nodePort int32) []*corev1.Service {
	si.mu.RLock()
	defer si.mu.RUnlock()

	keys, exists := si.nodePortIndex[nodePort]
	if !exists {
		return nil
	}

	result := make([]*corev1.Service, 0, len(keys))
	for _, key := range keys {
		if svc, exists := si.serviceCache[key]; exists {
			result = append(result, svc.DeepCopy())
		}
	}
	return result
}

// FindByNodePortInNamespace 在指定命名空间中通过 NodePort 查找 Service
func (si *ServiceIndexer) FindByNodePortInNamespace(nodePort int32, namespace string) []*corev1.Service {
	si.mu.RLock()
	defer si.mu.RUnlock()

	keys, exists := si.nodePortIndex[nodePort]
	if !exists {
		return nil
	}

	result := make([]*corev1.Service, 0)
	for _, key := range keys {
		if namespace != "" && key.Namespace != namespace {
			continue
		}
		if svc, exists := si.serviceCache[key]; exists {
			result = append(result, svc.DeepCopy())
		}
	}
	return result
}

// GetService 获取指定的 Service
func (si *ServiceIndexer) GetService(namespace, name string) *corev1.Service {
	si.mu.RLock()
	defer si.mu.RUnlock()

	key := ServiceKey{Namespace: namespace, Name: name}
	if svc, exists := si.serviceCache[key]; exists {
		return svc.DeepCopy()
	}
	return nil
}

// ListAllServices 列出所有缓存的 Service
func (si *ServiceIndexer) ListAllServices() []*corev1.Service {
	si.mu.RLock()
	defer si.mu.RUnlock()

	result := make([]*corev1.Service, 0, len(si.serviceCache))
	for _, svc := range si.serviceCache {
		result = append(result, svc.DeepCopy())
	}
	return result
}

// GetStats 获取索引器统计信息
func (si *ServiceIndexer) GetStats() IndexerStats {
	si.mu.RLock()
	defer si.mu.RUnlock()

	return IndexerStats{
		ServiceCount:  len(si.serviceCache),
		NodePortCount: len(si.nodePortIndex),
		Ready:         si.ready,
		LastSync:      si.lastSync,
		HasError:      si.syncError != nil,
	}
}

// IndexerStats 索引器统计信息
type IndexerStats struct {
	ServiceCount  int       `json:"serviceCount"`
	NodePortCount int       `json:"nodePortCount"`
	Ready         bool      `json:"ready"`
	LastSync      time.Time `json:"lastSync"`
	HasError      bool      `json:"hasError"`
}
