package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateConfigMap 创建 ConfigMap
// 参数:
//   - name: ConfigMap 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - data: 配置数据（可选）
//   - labels: 标签（可选）
//   - annotations: 注解（可选）
//   - context: K8S context 名称（可选）
func CreateConfigMap(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	// 获取可选参数
	data := getMapStringString(args, "data")
	labels := getMapStringString(args, "labels")
	annotations := getMapStringString(args, "annotations")

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 构建 ConfigMap 对象
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Data: data,
	}

	// 创建 ConfigMap
	created, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("ConfigMap '%s/%s' 已存在", namespace, name)
		}
		return nil, fmt.Errorf("创建 ConfigMap 失败: %w", err)
	}

	dataKeys := make([]string, 0, len(created.Data))
	for k := range created.Data {
		dataKeys = append(dataKeys, k)
	}

	return &CreateResult{
		Kind:      "ConfigMap",
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    "Created",
		Message:   fmt.Sprintf("ConfigMap '%s/%s' 创建成功，包含 %d 个配置项", namespace, name, len(dataKeys)),
		CreatedAt: created.CreationTimestamp.Time,
	}, nil
}
