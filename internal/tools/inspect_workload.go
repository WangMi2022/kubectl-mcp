package tools

import (
	"context"
	"fmt"
	"time"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 高重启阈值
const highRestartThreshold int32 = 10

// InspectWorkloadHealth 工作负载健康巡检
// 返回副本不一致的工作负载、异常 Pod、高重启 Pod
func InspectWorkloadHealth(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)

	// 获取 namespace 参数，为空则查全部
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	cs := clientSet.Clientset
	report := WorkloadHealthReport{
		CheckTime: time.Now(),
	}

	// 1. 检查 Deployment 副本一致性
	deployments, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Deployment 失败: %w", err)
	}
	for _, d := range deployments.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if d.Status.ReadyReplicas < desired {
			report.UnhealthyDeployments = append(report.UnhealthyDeployments, UnhealthyWorkload{
				Name:          d.Name,
				Namespace:     d.Namespace,
				Replicas:      desired,
				ReadyReplicas: d.Status.ReadyReplicas,
			})
		}
	}

	// 2. 检查 StatefulSet 副本一致性
	statefulSets, err := cs.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 StatefulSet 失败: %w", err)
	}
	for _, s := range statefulSets.Items {
		desired := int32(1)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		if s.Status.ReadyReplicas < desired {
			report.UnhealthyStatefulSets = append(report.UnhealthyStatefulSets, UnhealthyWorkload{
				Name:          s.Name,
				Namespace:     s.Namespace,
				Replicas:      desired,
				ReadyReplicas: s.Status.ReadyReplicas,
			})
		}
	}

	// 3. 检查 DaemonSet 调度一致性
	daemonSets, err := cs.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 DaemonSet 失败: %w", err)
	}
	for _, ds := range daemonSets.Items {
		if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			report.UnhealthyDaemonSets = append(report.UnhealthyDaemonSets, UnhealthyDaemonSet{
				Name:             ds.Name,
				Namespace:        ds.Namespace,
				DesiredScheduled: ds.Status.DesiredNumberScheduled,
				NumberReady:      ds.Status.NumberReady,
			})
		}
	}

	// 4. 检查异常 Pod 和高重启 Pod
	pods, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}
	for i := range pods.Items {
		pod := &pods.Items[i]

		// 跳过已完成的 Pod
		if pod.Status.Phase == corev1.PodSucceeded {
			continue
		}

		status := getPodStatus(pod)

		// 异常 Pod：非 Running 状态
		if status != "Running" && status != "Completed" {
			reason := getAbnormalPodReason(pod)
			report.AbnormalPods = append(report.AbnormalPods, AbnormalPodInfo{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Status:    status,
				Node:      pod.Spec.NodeName,
				Reason:    reason,
			})
		}

		// 高重启 Pod
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.RestartCount >= highRestartThreshold {
				report.HighRestartPods = append(report.HighRestartPods, HighRestartPodInfo{
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Restarts:  cs.RestartCount,
					Container: cs.Name,
				})
			}
		}
	}

	return report, nil
}

// getAbnormalPodReason 获取异常 Pod 的原因
func getAbnormalPodReason(pod *corev1.Pod) string {
	// 检查容器状态中的等待原因
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			return cs.State.Terminated.Reason
		}
	}

	// 检查 Pod Conditions
	for _, condition := range pod.Status.Conditions {
		if condition.Status == corev1.ConditionFalse && condition.Message != "" {
			return condition.Message
		}
	}

	return ""
}
