package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetConfigMaps 查询 ConfigMap 列表
func GetConfigMaps(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	// 只有用户明确指定 namespace 时才使用，否则搜索所有命名空间
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	labelSelector := buildLabelSelector(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	// 如果用户没有指定 namespace，搜索所有命名空间
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	configmaps, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 ConfigMap 列表失败: %w", err)
	}

	result := make([]ConfigMapInfo, 0, len(configmaps.Items))
	for _, cm := range configmaps.Items {
		dataKeys := make([]string, 0, len(cm.Data))
		for key := range cm.Data {
			dataKeys = append(dataKeys, key)
		}

		cmInfo := ConfigMapInfo{
			Name:      cm.Name,
			Namespace: cm.Namespace,
			Labels:    cm.Labels,
			DataKeys:  dataKeys,
			CreatedAt: cm.CreationTimestamp.Time,
		}
		result = append(result, cmInfo)
	}

	return result, nil
}

// GetSecrets 查询 Secret 列表（脱敏处理）
func GetSecrets(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	// 只有用户明确指定 namespace 时才使用，否则搜索所有命名空间
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	labelSelector := buildLabelSelector(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	// 如果用户没有指定 namespace，搜索所有命名空间
	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	secrets, err := clientset.Clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 Secret 列表失败: %w", err)
	}

	result := make([]SecretInfo, 0, len(secrets.Items))
	for _, secret := range secrets.Items {
		// 只返回 key 名称，不返回实际值（脱敏处理）
		dataKeys := make([]string, 0, len(secret.Data))
		for key := range secret.Data {
			dataKeys = append(dataKeys, key)
		}

		secretInfo := SecretInfo{
			Name:      secret.Name,
			Namespace: secret.Namespace,
			Type:      string(secret.Type),
			Labels:    secret.Labels,
			DataKeys:  dataKeys,
			CreatedAt: secret.CreationTimestamp.Time,
		}
		result = append(result, secretInfo)
	}

	return result, nil
}
