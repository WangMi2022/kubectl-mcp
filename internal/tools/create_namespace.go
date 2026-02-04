package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateNamespace 创建 Namespace
// 参数:
//   - name: Namespace 名称（必填）
//   - labels: 标签（可选）
//   - annotations: 注解（可选）
//   - context: K8S context 名称（可选）
func CreateNamespace(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	// 获取可选参数
	labels := getMapStringString(args, "labels")
	annotations := getMapStringString(args, "annotations")

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 构建 Namespace 对象
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	// 创建 Namespace
	created, err := clientset.Clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Namespace '%s' 已存在", name)
		}
		return nil, fmt.Errorf("创建 Namespace 失败: %w", err)
	}

	return &CreateResult{
		Kind:      "Namespace",
		Name:      created.Name,
		Status:    string(created.Status.Phase),
		Message:   fmt.Sprintf("Namespace '%s' 创建成功", name),
		CreatedAt: created.CreationTimestamp.Time,
	}, nil
}
