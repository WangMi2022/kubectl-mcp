package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScaleDeployment 扩缩容 Deployment
// 参数:
//   - name: Deployment 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - replicas: 目标副本数（必填）
//   - context: K8S context 名称（可选）
func ScaleDeployment(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	replicas := getInt32Arg(args, "replicas", -1)
	if replicas < 0 {
		return nil, fmt.Errorf("参数 'replicas' 是必填的且必须大于等于 0")
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

	oldReplicas := int32(0)
	if deployment.Spec.Replicas != nil {
		oldReplicas = *deployment.Spec.Replicas
	}

	// 更新副本数
	deployment.Spec.Replicas = &replicas
	_, err = clientset.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("扩缩容 Deployment '%s/%s' 失败: %w", namespace, name, err)
	}

	return &UpdateResult{
		Kind:      "Deployment",
		Name:      name,
		Namespace: namespace,
		Action:    "Scale",
		Status:    "Success",
		Message:   fmt.Sprintf("Deployment '%s/%s' 副本数从 %d 调整为 %d", namespace, name, oldReplicas, replicas),
		OldValue:  fmt.Sprintf("%d", oldReplicas),
		NewValue:  fmt.Sprintf("%d", replicas),
	}, nil
}

// ScaleStatefulSet 扩缩容 StatefulSet
// 参数:
//   - name: StatefulSet 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - replicas: 目标副本数（必填）
//   - context: K8S context 名称（可选）
func ScaleStatefulSet(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	replicas := getInt32Arg(args, "replicas", -1)
	if replicas < 0 {
		return nil, fmt.Errorf("参数 'replicas' 是必填的且必须大于等于 0")
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 获取当前 StatefulSet
	statefulSet, err := clientset.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 StatefulSet '%s/%s' 失败: %w", namespace, name, err)
	}

	oldReplicas := int32(0)
	if statefulSet.Spec.Replicas != nil {
		oldReplicas = *statefulSet.Spec.Replicas
	}

	// 更新副本数
	statefulSet.Spec.Replicas = &replicas
	_, err = clientset.Clientset.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("扩缩容 StatefulSet '%s/%s' 失败: %w", namespace, name, err)
	}

	return &UpdateResult{
		Kind:      "StatefulSet",
		Name:      name,
		Namespace: namespace,
		Action:    "Scale",
		Status:    "Success",
		Message:   fmt.Sprintf("StatefulSet '%s/%s' 副本数从 %d 调整为 %d", namespace, name, oldReplicas, replicas),
		OldValue:  fmt.Sprintf("%d", oldReplicas),
		NewValue:  fmt.Sprintf("%d", replicas),
	}, nil
}
