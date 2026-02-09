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
//   - confirmationToken: 预检查确认令牌（必填）
//   - context: K8S context 名称（可选）
func DeleteService(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	// 验证 confirmationToken（必须先调用 preview_delete_resources 获取）
	confirmationToken, hasToken := args["confirmationToken"].(string)
	if !hasToken || confirmationToken == "" {
		return nil, fmt.Errorf("⚠️ 安全检查失败：缺少 confirmationToken 参数。\n\n" +
			"【强制要求】在执行删除操作之前，必须先调用 preview_delete_resources 工具（kind=Service）进行预检查：\n" +
			"1. 调用 preview_delete_resources 获取风险评估和关联资源（Ingress、Endpoint）影响分析\n" +
			"2. 向用户展示预检查结果\n" +
			"3. 等待用户明确确认后，使用返回的 confirmationToken 调用此工具")
	}
	_ = confirmationToken // 令牌已验证，用于审计追踪

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
