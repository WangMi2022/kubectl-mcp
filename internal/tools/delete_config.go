package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteConfigMap 删除 ConfigMap
// 参数:
//   - name: ConfigMap 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - context: K8S context 名称（可选）
func DeleteConfigMap(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
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

	// 检查 ConfigMap 是否存在
	configMap, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 ConfigMap '%s/%s' 失败: %w", namespace, name, err)
	}

	// 删除 ConfigMap
	err = clientset.Clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("删除 ConfigMap '%s/%s' 失败: %w", namespace, name, err)
	}

	dataCount := len(configMap.Data)
	return &DeleteResult{
		Kind:      "ConfigMap",
		Name:      name,
		Namespace: namespace,
		Status:    "Deleted",
		Message:   fmt.Sprintf("ConfigMap '%s/%s' 删除成功（包含 %d 个数据项）", namespace, name, dataCount),
		DeletedAt: configMap.CreationTimestamp.Time,
	}, nil
}

// DeleteSecret 删除 Secret
// 参数:
//   - name: Secret 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - context: K8S context 名称（可选）
func DeleteSecret(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
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

	// 检查 Secret 是否存在
	secret, err := clientset.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Secret '%s/%s' 失败: %w", namespace, name, err)
	}

	// 删除 Secret
	err = clientset.Clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return nil, fmt.Errorf("删除 Secret '%s/%s' 失败: %w", namespace, name, err)
	}

	dataCount := len(secret.Data)
	return &DeleteResult{
		Kind:      "Secret",
		Name:      name,
		Namespace: namespace,
		Status:    "Deleted",
		Message:   fmt.Sprintf("Secret '%s/%s' (类型: %s) 删除成功（包含 %d 个数据项）", namespace, name, secret.Type, dataCount),
		DeletedAt: secret.CreationTimestamp.Time,
	}, nil
}
