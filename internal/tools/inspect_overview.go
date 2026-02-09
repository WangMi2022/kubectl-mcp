package tools

import (
	"context"
	"fmt"
	"time"

	"kubectl-mcp/internal/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InspectClusterOverview 集群概览巡检
// 返回集群整体健康状态的轻量摘要，用于快速判断集群是否存在问题
func InspectClusterOverview(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	cs := clientSet.Clientset
	overview := ClusterOverview{
		CheckTime:      time.Now(),
		ClusterServer:  k8sClient.GetClusterServer(),
		CurrentContext: k8sClient.GetCurrentContext(),
	}

	var issues []string

	// 1. 节点摘要
	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}
	overview.NodeSummary = buildNodeSummary(nodes)
	if overview.NodeSummary.NotReady > 0 {
		issues = append(issues, fmt.Sprintf("%d 个节点 NotReady", overview.NodeSummary.NotReady))
	}

	// 2. Pod 摘要（全命名空间）
	pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}
	overview.PodSummary = buildPodSummary(pods)
	if overview.PodSummary.CrashLoopBackOff > 0 {
		issues = append(issues, fmt.Sprintf("%d 个 Pod CrashLoopBackOff", overview.PodSummary.CrashLoopBackOff))
	}
	if overview.PodSummary.Pending > 0 {
		issues = append(issues, fmt.Sprintf("%d 个 Pod Pending", overview.PodSummary.Pending))
	}
	if overview.PodSummary.Failed > 0 {
		issues = append(issues, fmt.Sprintf("%d 个 Pod Failed", overview.PodSummary.Failed))
	}

	// 3. 工作负载摘要
	deployments, err := cs.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Deployment 失败: %w", err)
	}
	statefulSets, err := cs.AppsV1().StatefulSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 StatefulSet 失败: %w", err)
	}
	daemonSets, err := cs.AppsV1().DaemonSets("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 DaemonSet 失败: %w", err)
	}
	overview.WorkloadSummary = buildWorkloadSummary(deployments, statefulSets, daemonSets)
	unhealthyWorkloads := overview.WorkloadSummary.DeploymentUnhealthy + overview.WorkloadSummary.StatefulSetUnhealthy + overview.WorkloadSummary.DaemonSetUnhealthy
	if unhealthyWorkloads > 0 {
		issues = append(issues, fmt.Sprintf("%d 个工作负载副本不一致", unhealthyWorkloads))
	}

	// 4. Warning 事件摘要
	events, err := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}
	overview.EventSummary = EventSummaryBrief{
		WarningCount: len(events.Items),
	}
	if overview.EventSummary.WarningCount > 0 {
		issues = append(issues, fmt.Sprintf("%d 条 Warning 事件", overview.EventSummary.WarningCount))
	}

	// 5. 计算健康评分（100 分制）
	overview.HealthScore = calculateHealthScore(overview)
	overview.Issues = issues

	return overview, nil
}

// buildNodeSummary 构建节点摘要
func buildNodeSummary(nodes *corev1.NodeList) NodeSummary {
	summary := NodeSummary{Total: len(nodes.Items)}
	for _, node := range nodes.Items {
		if getNodeStatus(node.Status.Conditions) == "Ready" {
			summary.Ready++
		} else {
			summary.NotReady++
		}
	}
	return summary
}

// buildPodSummary 构建 Pod 摘要
func buildPodSummary(pods *corev1.PodList) PodSummary {
	summary := PodSummary{Total: len(pods.Items)}
	for i := range pods.Items {
		pod := &pods.Items[i]
		status := getPodStatus(pod)
		switch {
		case status == "Running":
			summary.Running++
		case status == "Pending":
			summary.Pending++
		case status == "Failed":
			summary.Failed++
		case status == "CrashLoopBackOff":
			summary.CrashLoopBackOff++
		case pod.Status.Phase == corev1.PodSucceeded:
			// Completed Pod 不计入异常
		default:
			summary.Unknown++
		}
	}
	return summary
}

// buildWorkloadSummary 构建工作负载摘要
func buildWorkloadSummary(deployments *appsv1.DeploymentList, statefulSets *appsv1.StatefulSetList, daemonSets *appsv1.DaemonSetList) WorkloadSummary {
	summary := WorkloadSummary{
		DeploymentTotal:  len(deployments.Items),
		StatefulSetTotal: len(statefulSets.Items),
		DaemonSetTotal:   len(daemonSets.Items),
	}
	for _, d := range deployments.Items {
		desired := int32(1)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if d.Status.ReadyReplicas < desired {
			summary.DeploymentUnhealthy++
		}
	}
	for _, s := range statefulSets.Items {
		desired := int32(1)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		if s.Status.ReadyReplicas < desired {
			summary.StatefulSetUnhealthy++
		}
	}
	for _, ds := range daemonSets.Items {
		if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
			summary.DaemonSetUnhealthy++
		}
	}
	return summary
}

// calculateHealthScore 计算集群健康评分（100 分制）
func calculateHealthScore(overview ClusterOverview) int {
	score := 100

	// 节点维度（权重最高）
	if overview.NodeSummary.Total > 0 {
		notReadyRatio := float64(overview.NodeSummary.NotReady) / float64(overview.NodeSummary.Total)
		score -= int(notReadyRatio * 40)
	}

	// Pod 异常维度
	if overview.PodSummary.Total > 0 {
		abnormalCount := overview.PodSummary.CrashLoopBackOff + overview.PodSummary.Failed + overview.PodSummary.Pending
		abnormalRatio := float64(abnormalCount) / float64(overview.PodSummary.Total)
		score -= int(abnormalRatio * 30)
	}

	// 工作负载维度
	totalWorkloads := overview.WorkloadSummary.DeploymentTotal + overview.WorkloadSummary.StatefulSetTotal + overview.WorkloadSummary.DaemonSetTotal
	if totalWorkloads > 0 {
		unhealthy := overview.WorkloadSummary.DeploymentUnhealthy + overview.WorkloadSummary.StatefulSetUnhealthy + overview.WorkloadSummary.DaemonSetUnhealthy
		unhealthyRatio := float64(unhealthy) / float64(totalWorkloads)
		score -= int(unhealthyRatio * 20)
	}

	// Warning 事件维度
	if overview.EventSummary.WarningCount > 50 {
		score -= 10
	} else if overview.EventSummary.WarningCount > 20 {
		score -= 5
	} else if overview.EventSummary.WarningCount > 0 {
		score -= 2
	}

	if score < 0 {
		score = 0
	}
	return score
}
