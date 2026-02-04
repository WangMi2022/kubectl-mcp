package tools

import (
	"context"
	"fmt"
	"time"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestartDeployment 重启 Deployment
// 通过更新 Pod 模板的注解来触发滚动重启
// 参数:
//   - name: Deployment 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - context: K8S context 名称（可选）
func RestartDeployment(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
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

	// 获取当前 Deployment
	deployment, err := clientset.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Deployment '%s/%s' 失败: %w", namespace, name, err)
	}

	// 添加或更新重启注解
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// 更新 Deployment
	_, err = clientset.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("重启 Deployment '%s/%s' 失败: %w", namespace, name, err)
	}

	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	return &UpdateResult{
		Kind:      "Deployment",
		Name:      name,
		Namespace: namespace,
		Action:    "Restart",
		Status:    "Success",
		Message:   fmt.Sprintf("Deployment '%s/%s' 已触发滚动重启，副本数: %d", namespace, name, replicas),
		OldValue:  "",
		NewValue:  time.Now().Format(time.RFC3339),
	}, nil
}
