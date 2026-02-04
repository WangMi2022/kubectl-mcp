package k8s

import (
	"k8s.io/client-go/kubernetes/fake"
)

// NewFakeK8SClientManager 创建一个用于测试的 K8S 客户端管理器
// 参数:
//   - fakeClient: fake Kubernetes 客户端
//
// 返回:
//   - *K8SClientManager: 客户端管理器实例（用于测试）
//
// 注意：此函数仅用于单元测试
func NewFakeK8SClientManager(fakeClient *fake.Clientset) *K8SClientManager {
	manager := &K8SClientManager{
		kubeconfigPath: "fake-kubeconfig",
		contexts: map[string]*ContextInfo{
			"fake-context": {
				Name:      "fake-context",
				Cluster:   "fake-cluster",
				User:      "fake-user",
				Namespace: "default",
			},
		},
		currentCtx: "fake-context",
		clientPool: map[string]*ClientSet{
			"fake-context": {
				// fake.Clientset 实现了 kubernetes.Interface
				// 但我们需要存储为 ClientSet 结构
				// 这里我们直接将 fake client 转换为 kubernetes.Clientset
				// 注意：这是一个类型断言，在测试中是安全的
				Clientset: fakeClient,
			},
		},
	}

	return manager
}
