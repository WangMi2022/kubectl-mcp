package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteService 删除 Service
// 参数:
//   - name: Service 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - context: K8S context 名称（可选）
func DeleteService(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 检查 Service 是否存在
	service, err := clientset.Clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Service '%s/%s' 失败: %w", namespace, name, err)
	}

	// 删除 Service
	err = clientset.Clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("删除 Service '%s/%s' 失败: %w", namespace, name, err)
	}

	return &DeleteResult{
		Kind:      "Service",
		Name:      name,
		Namespace: namespace,
		Status:    "Deleted",
		Message:   fmt.Sprintf("Service '%s/%s' (类型: %s) 删除成功", namespace, name, service.Spec.Type),
		DeletedAt: service.CreationTimestamp.Time,
	}, nil
}
