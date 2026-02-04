package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPodFilter 在整个集群的所有命名空间中查找 Pod
// 支持精确匹配和模糊匹配两种模式
// 参数:
//   - name: Pod 名称
//   - matchMode: 匹配模式 (exact=精确匹配, fuzzy=模糊匹配)，默认 exact
//   - context: K8S context 名称（可选）
//
// 返回:
//   - 如果找到匹配的 Pod，返回 Pod 的详细信息
//   - 如果找到多个匹配的 Pod，返回所有匹配的 Pod 列表
//   - 如果未找到，返回错误
func GetPodFilter(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	// 获取必填参数
	podName, ok := args["name"].(string)
	if !ok || podName == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	// 获取匹配模式，默认为精确匹配
	matchMode := "exact"
	if mode, ok := args["matchMode"].(string); ok && mode != "" {
		matchMode = mode
	}

	// 获取可选的 context
	contextName := ""
	if ctxName, ok := args["context"].(string); ok && ctxName != "" {
		contextName = ctxName
	}

	// 获取客户端
	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	var pods *corev1.PodList

	// 根据匹配模式选择查询方式
	if matchMode == "exact" {
		// 精确匹配：使用 FieldSelector 直接在 API 层面过滤
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", podName),
		}
		pods, err = clientset.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("查询 Pod 列表失败: %w", err)
		}
	} else {
		// 模糊匹配：获取所有 Pod，然后在代码中过滤
		listOptions := metav1.ListOptions{}
		pods, err = clientset.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, listOptions)
		if err != nil {
			return nil, fmt.Errorf("查询 Pod 列表失败: %w", err)
		}
	}

	// 构建匹配的 Pod 列表
	var matchedPods []PodInfo
	for _, pod := range pods.Items {
		// 根据匹配模式判断是否匹配
		matched := false
		if matchMode == "exact" {
			matched = pod.Name == podName
		} else {
			matched = strings.Contains(pod.Name, podName)
		}

		if matched {
			containers := make([]ContainerInfo, 0, len(pod.Status.ContainerStatuses))
			for _, cs := range pod.Status.ContainerStatuses {
				containers = append(containers, ContainerInfo{
					Name:         cs.Name,
					Image:        cs.Image,
					Ready:        cs.Ready,
					RestartCount: cs.RestartCount,
					State:        getContainerState(cs),
				})
			}

			podInfo := PodInfo{
				Name:       pod.Name,
				Namespace:  pod.Namespace,
				Status:     getPodStatus(&pod),
				Phase:      string(pod.Status.Phase),
				IP:         pod.Status.PodIP,
				Node:       pod.Spec.NodeName,
				Labels:     pod.Labels,
				Containers: containers,
				CreatedAt:  pod.CreationTimestamp.Time,
				Restarts:   getTotalRestarts(&pod),
			}
			matchedPods = append(matchedPods, podInfo)
		}
	}

	// 检查是否找到 Pod
	if len(matchedPods) == 0 {
		if matchMode == "exact" {
			return nil, fmt.Errorf("未找到名称为 '%s' 的 Pod", podName)
		}
		return nil, fmt.Errorf("未找到名称包含 '%s' 的 Pod", podName)
	}

	// 根据匹配结果返回
	if len(matchedPods) == 1 {
		// 找到唯一匹配的 Pod，返回详细信息
		return map[string]interface{}{
			"found":     true,
			"count":     1,
			"pod":       matchedPods[0],
			"message":   fmt.Sprintf("找到 Pod '%s' 在命名空间 '%s'", matchedPods[0].Name, matchedPods[0].Namespace),
			"namespace": matchedPods[0].Namespace,
			"matchMode": matchMode,
		}, nil
	}

	// 找到多个匹配的 Pod，返回列表
	namespaces := make([]string, 0, len(matchedPods))
	for _, pod := range matchedPods {
		namespaces = append(namespaces, pod.Namespace)
	}

	var message string
	if matchMode == "exact" {
		message = fmt.Sprintf("找到 %d 个同名 Pod '%s'，分布在命名空间: %s", len(matchedPods), podName, strings.Join(namespaces, ", "))
	} else {
		message = fmt.Sprintf("找到 %d 个匹配的 Pod（名称包含 '%s'），分布在命名空间: %s", len(matchedPods), podName, strings.Join(namespaces, ", "))
	}

	return map[string]interface{}{
		"found":      true,
		"count":      len(matchedPods),
		"pods":       matchedPods,
		"message":    message,
		"namespaces": namespaces,
		"matchMode":  matchMode,
	}, nil
}
