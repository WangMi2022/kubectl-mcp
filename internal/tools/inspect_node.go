package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InspectNodeHealth 节点健康巡检
// 返回节点状态、资源分配率、不健康节点详情
func InspectNodeHealth(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	cs := clientSet.Clientset

	// 查询节点
	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}

	// 查询所有 Pod 用于统计每个节点的 Pod 数量
	pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}

	// 按节点统计 Pod 数量
	nodePodCount := make(map[string]int)
	for i := range pods.Items {
		if pods.Items[i].Spec.NodeName != "" {
			nodePodCount[pods.Items[i].Spec.NodeName]++
		}
	}

	report := NodeHealthReport{
		CheckTime: time.Now(),
		Total:     len(nodes.Items),
	}

	for _, node := range nodes.Items {
		status := getNodeStatus(node.Status.Conditions)

		if status == "Ready" {
			report.Ready++
		} else {
			report.NotReady++
			report.UnhealthyNodes = append(report.UnhealthyNodes, buildUnhealthyNodeInfo(node))
		}

		// Pod 容量
		podCapacity := 0
		if cap, ok := node.Status.Allocatable[corev1.ResourcePods]; ok {
			podCapacity, _ = strconv.Atoi(cap.String())
		}

		report.NodeResources = append(report.NodeResources, NodeResourceInfo{
			Name:              node.Name,
			Status:            status,
			AllocatableCPU:    node.Status.Allocatable.Cpu().String(),
			AllocatableMemory: node.Status.Allocatable.Memory().String(),
			PodCount:          nodePodCount[node.Name],
			PodCapacity:       podCapacity,
		})
	}

	return report, nil
}

// buildUnhealthyNodeInfo 构建不健康节点信息
func buildUnhealthyNodeInfo(node corev1.Node) UnhealthyNodeInfo {
	info := UnhealthyNodeInfo{
		Name:   node.Name,
		Status: getNodeStatus(node.Status.Conditions),
	}

	// 收集异常 Condition 的原因
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
			info.Reasons = append(info.Reasons, fmt.Sprintf("NodeReady=%s: %s", condition.Status, condition.Message))
		}
		if condition.Type == corev1.NodeMemoryPressure && condition.Status == corev1.ConditionTrue {
			info.Reasons = append(info.Reasons, "MemoryPressure")
		}
		if condition.Type == corev1.NodeDiskPressure && condition.Status == corev1.ConditionTrue {
			info.Reasons = append(info.Reasons, "DiskPressure")
		}
		if condition.Type == corev1.NodePIDPressure && condition.Status == corev1.ConditionTrue {
			info.Reasons = append(info.Reasons, "PIDPressure")
		}
		if condition.Type == corev1.NodeNetworkUnavailable && condition.Status == corev1.ConditionTrue {
			info.Reasons = append(info.Reasons, "NetworkUnavailable")
		}
	}

	return info
}
