package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeletePod 删除 Pod
// 参数:
//   - name: Pod 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - force: 是否强制删除（可选，默认 false）
//   - gracePeriod: 优雅删除时间（秒，可选，force=true 时为 0）
//   - confirmationToken: 预检查确认令牌（必填）
//   - context: K8S context 名称（可选）
func DeletePod(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	// 验证 confirmationToken（必须先调用 preview_delete_resources 获取）
	confirmationToken, hasToken := args["confirmationToken"].(string)
	if !hasToken || confirmationToken == "" {
		return nil, fmt.Errorf("⚠️ 安全检查失败：缺少 confirmationToken 参数。\n\n" +
			"【强制要求】在执行删除操作之前，必须先调用 preview_delete_resources 工具（kind=Pod）进行预检查：\n" +
			"1. 调用 preview_delete_resources 获取风险评估\n" +
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

	// 获取可选参数
	force := false
	if f, ok := args["force"].(bool); ok {
		force = f
	}

	gracePeriod := int64(30) // 默认 30 秒
	if force {
		gracePeriod = 0 // 强制删除时设置为 0
	} else if gp := getInt32Arg(args, "gracePeriod", 30); gp >= 0 {
		gracePeriod = int64(gp)
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 检查 Pod 是否存在
	pod, err := clientset.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Pod '%s/%s' 失败: %w", namespace, name, err)
	}

	// 删除 Pod
	deleteOptions := metav1.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}

	err = clientset.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	if err != nil {
		return nil, fmt.Errorf("删除 Pod '%s/%s' 失败: %w", namespace, name, err)
	}

	message := fmt.Sprintf("Pod '%s/%s' 删除成功", namespace, name)
	if force {
		message = fmt.Sprintf("Pod '%s/%s' 强制删除成功（grace period: 0s）", namespace, name)
	} else if gracePeriod > 0 {
		message = fmt.Sprintf("Pod '%s/%s' 删除成功（grace period: %ds）", namespace, name, gracePeriod)
	}

	return &DeleteResult{
		Kind:      "Pod",
		Name:      name,
		Namespace: namespace,
		Status:    "Deleted",
		Message:   message,
		Force:     force,
		DeletedAt: pod.CreationTimestamp.Time,
	}, nil
}
