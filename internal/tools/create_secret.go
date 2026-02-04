package tools

import (
	"context"
	"encoding/base64"
	"fmt"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateSecret 创建 Secret
// 参数:
//   - name: Secret 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - type: Secret 类型（可选，默认 Opaque）
//   - data: 配置数据，值会自动进行 base64 编码（可选）
//   - stringData: 字符串数据，不需要 base64 编码（可选）
//   - labels: 标签（可选）
//   - annotations: 注解（可选）
//   - context: K8S context 名称（可选）
func CreateSecret(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
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
	secretType := getStringArg(args, "type", "Opaque")
	labels := getMapStringString(args, "labels")
	annotations := getMapStringString(args, "annotations")

	// 处理 data（需要 base64 编码）
	data := make(map[string][]byte)
	if dataMap := getMapStringString(args, "data"); len(dataMap) > 0 {
		for k, v := range dataMap {
			data[k] = []byte(base64.StdEncoding.EncodeToString([]byte(v)))
		}
	}

	// 处理 stringData（不需要 base64 编码，K8S 会自动处理）
	stringData := getMapStringString(args, "stringData")

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 构建 Secret 对象
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Type:       corev1.SecretType(secretType),
		StringData: stringData,
	}

	// 如果有 data，设置 data
	if len(data) > 0 {
		secret.Data = data
	}

	// 创建 Secret
	created, err := clientset.Clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Secret '%s/%s' 已存在", namespace, name)
		}
		return nil, fmt.Errorf("创建 Secret 失败: %w", err)
	}

	dataKeys := make([]string, 0, len(created.Data))
	for k := range created.Data {
		dataKeys = append(dataKeys, k)
	}

	return &CreateResult{
		Kind:      "Secret",
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    "Created",
		Message:   fmt.Sprintf("Secret '%s/%s' 创建成功，类型: %s，包含 %d 个密钥", namespace, name, secretType, len(dataKeys)),
		CreatedAt: created.CreationTimestamp.Time,
	}, nil
}
