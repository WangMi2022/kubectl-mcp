package tools

import (
	"fmt"
	"kubectl-mcp/internal/k8s"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// getContextAndNamespace 从参数中获取 context 和 namespace
func getContextAndNamespace(args map[string]interface{}, k8sClient *k8s.K8SClientManager) (string, string, error) {
	// 获取 context
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	// 获取 namespace
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// 如果没有指定 namespace，使用 context 的默认 namespace
	if namespace == "" {
		if contextName != "" {
			ns, err := k8sClient.GetDefaultNamespaceForContext(contextName)
			if err == nil {
				namespace = ns
			}
		} else {
			ns, err := k8sClient.GetDefaultNamespace()
			if err == nil {
				namespace = ns
			}
		}
	}

	return contextName, namespace, nil
}

// buildLabelSelector 构建 label selector
func buildLabelSelector(args map[string]interface{}) string {
	if labels, ok := args["labelSelector"].(string); ok && labels != "" {
		return labels
	}
	return ""
}

// getNodeRoles 获取节点角色
func getNodeRoles(labels map[string]string) []string {
	roles := []string{}
	for key := range labels {
		if strings.HasPrefix(key, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(key, "node-role.kubernetes.io/")
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if len(roles) == 0 {
		roles = append(roles, "<none>")
	}
	return roles
}

// getNodeStatus 获取节点状态
func getNodeStatus(conditions []corev1.NodeCondition) string {
	for _, condition := range conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

// getPodStatus 获取 Pod 状态
func getPodStatus(pod *corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return "Running"
		}
	}

	if pod.Status.Phase == corev1.PodPending {
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				return cs.State.Waiting.Reason
			}
		}
	}

	return string(pod.Status.Phase)
}

// getContainerState 获取容器状态
func getContainerState(status corev1.ContainerStatus) string {
	if status.State.Running != nil {
		return "Running"
	}
	if status.State.Waiting != nil {
		return fmt.Sprintf("Waiting: %s", status.State.Waiting.Reason)
	}
	if status.State.Terminated != nil {
		return fmt.Sprintf("Terminated: %s", status.State.Terminated.Reason)
	}
	return "Unknown"
}

// getTotalRestarts 获取 Pod 总重启次数
func getTotalRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return restarts
}

// getClientSet 获取 Kubernetes 客户端
func getClientSet(contextName string, k8sClient *k8s.K8SClientManager) (*k8s.ClientSet, error) {
	if contextName != "" {
		cs, err := k8sClient.GetClientForContext(contextName)
		if err != nil {
			return nil, fmt.Errorf("获取 context '%s' 的客户端失败: %w", contextName, err)
		}
		return &k8s.ClientSet{Clientset: cs}, nil
	}

	cs, err := k8sClient.GetClient()
	if err != nil {
		return nil, fmt.Errorf("获取客户端失败: %w", err)
	}
	return &k8s.ClientSet{Clientset: cs}, nil
}
