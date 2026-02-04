package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteNamespace 删除 Namespace
// 警告：这是一个高危操作，会删除 namespace 下的所有资源
// 参数:
//   - name: Namespace 名称（必填）
//   - context: K8S context 名称（可选）
func DeleteNamespace(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if c, ok := args["context"].(string); ok && c != "" {
		contextName = c
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	// 防止删除系统 namespace
	systemNamespaces := []string{"default", "kube-system", "kube-public", "kube-node-lease"}
	for _, sysNs := range systemNamespaces {
		if name == sysNs {
			return nil, fmt.Errorf("禁止删除系统 namespace '%s'", name)
		}
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 检查 Namespace 是否存在
	namespace, err := clientset.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Namespace '%s' 失败: %w", name, err)
	}

	// 获取 namespace 中的资源统计（可选，用于提示）
	pods, _ := clientset.Clientset.CoreV1().Pods(name).List(ctx, metav1.ListOptions{})
	deployments, _ := clientset.Clientset.AppsV1().Deployments(name).List(ctx, metav1.ListOptions{})
	services, _ := clientset.Clientset.CoreV1().Services(name).List(ctx, metav1.ListOptions{})

	resourceCount := len(pods.Items) + len(deployments.Items) + len(services.Items)

	// 删除 Namespace
	err = clientset.Clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("删除 Namespace '%s' 失败: %w", name, err)
	}

	return &DeleteResult{
		Kind:      "Namespace",
		Name:      name,
		Namespace: "",
		Status:    "Deleted",
		Message:   fmt.Sprintf("Namespace '%s' 删除成功（包含约 %d 个资源，正在清理中）", name, resourceCount),
		DeletedAt: namespace.CreationTimestamp.Time,
		Details: map[string]interface{}{
			"pods":        len(pods.Items),
			"deployments": len(deployments.Items),
			"services":    len(services.Items),
		},
	}, nil
}
