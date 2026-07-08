package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InspectNodePressure diagnoses node pressure, high allocation and abnormal pods per node.
func InspectNodePressure(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	nodeName := strings.TrimSpace(stringArg(args, "nodeName", ""))
	topN := intArg(args, "topN", 20)
	if topN <= 0 {
		topN = 20
	}
	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	cs := clientSet.Clientset
	nodes := []corev1.Node{}
	if nodeName != "" {
		n, err := cs.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Node %s 失败: %w", nodeName, err)
		}
		nodes = append(nodes, *n)
	} else {
		list, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Node 失败: %w", err)
		}
		nodes = append(nodes, list.Items...)
	}
	pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}
	events, _ := cs.CoreV1().Events("").List(ctx, metav1.ListOptions{})
	podsByNode := groupPodsByNode(pods.Items)
	eventsByNode := map[string][]corev1.Event{}
	if events != nil {
		for _, e := range events.Items {
			if e.InvolvedObject.Kind == "Node" {
				eventsByNode[e.InvolvedObject.Name] = append(eventsByNode[e.InvolvedObject.Name], e)
			}
		}
	}
	findings := make([]DiagnosticFinding, 0)
	for _, node := range nodes {
		findings = append(findings, inspectNodePressureOne(node, podsByNode[node.Name], eventsByNode[node.Name])...)
	}
	sort.Slice(findings, func(i, j int) bool {
		order := map[string]int{"critical": 0, "warning": 1, "info": 2}
		if order[findings[i].Severity] != order[findings[j].Severity] {
			return order[findings[i].Severity] < order[findings[j].Severity]
		}
		return findings[i].ID < findings[j].ID
	})
	if len(findings) > topN {
		findings = findings[:topN]
	}
	return DiagnosticReport{CheckTime: time.Now(), Scope: DiagnosticScope{Cluster: contextName, ObjectKind: "Node", ObjectName: nodeName}, Summary: buildDiagnosticSummary(findings), Findings: findings}, nil
}

func inspectNodePressureOne(node corev1.Node, pods []corev1.Pod, events []corev1.Event) []DiagnosticFinding {
	findings := make([]DiagnosticFinding, 0)
	for _, c := range node.Status.Conditions {
		if (c.Type == corev1.NodeMemoryPressure || c.Type == corev1.NodeDiskPressure || c.Type == corev1.NodePIDPressure) && c.Status == corev1.ConditionTrue {
			desc := fmt.Sprintf("Node %s 存在 %s：%s %s。", node.Name, c.Type, c.Reason, c.Message)
			findings = append(findings, newNodePressureFinding(string(c.Type), "critical", "节点压力条件为 True", desc, node.Name, eventEvidence(events, "event", 2), "优先检查节点资源、kubelet、磁盘/PID/内存压力来源，并迁移或重启异常 Pod。"))
		}
	}
	cpuReq, memReq := sumPodRequests(pods)
	cpuAlloc := node.Status.Allocatable.Cpu().MilliValue()
	memAlloc := node.Status.Allocatable.Memory().Value()
	if cpuAlloc > 0 && cpuReq*100/cpuAlloc >= 90 {
		desc := fmt.Sprintf("Node %s CPU requests %.1f%% (%dm/%dm)。", node.Name, float64(cpuReq)*100/float64(cpuAlloc), cpuReq, cpuAlloc)
		findings = append(findings, newNodePressureFinding("NodeCPURequestsHigh", "warning", "节点 CPU requests 占比过高", desc, node.Name, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "检查该节点 Pod requests 分布，必要时扩容或迁移负载。"))
	}
	if memAlloc > 0 && memReq*100/memAlloc >= 90 {
		desc := fmt.Sprintf("Node %s Memory requests %.1f%%。", node.Name, float64(memReq)*100/float64(memAlloc))
		findings = append(findings, newNodePressureFinding("NodeMemoryRequestsHigh", "warning", "节点内存 requests 占比过高", desc, node.Name, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "检查该节点 Pod 内存 requests/limits，必要时扩容或迁移负载。"))
	}
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodFailed || getTotalRestarts(&pod) >= highRestartThreshold {
			desc := fmt.Sprintf("Node %s 上 Pod %s/%s 状态=%s，总重启=%d。", node.Name, pod.Namespace, pod.Name, pod.Status.Phase, getTotalRestarts(&pod))
			findings = append(findings, newNodePressureFinding("NodeAbnormalPod", "warning", "节点上存在异常/高重启 Pod", desc, node.Name, []DiagnosticEvidence{{Source: "podStatus", Message: desc, Count: normalizedEventCount(getTotalRestarts(&pod))}}, "对该 Pod 执行 inspect_pod_diagnostics，确认是否造成节点压力或受节点压力影响。"))
		}
	}
	return findings
}

func newNodePressureFinding(findingType, severity, title, description, nodeName string, evidence []DiagnosticEvidence, recommendation string) DiagnosticFinding {
	return DiagnosticFinding{ID: stableFindingID("node-pressure", findingType, nodeName, description), Severity: severity, FindingType: findingType, Title: title, Description: description, AffectedObject: DiagnosticObjectRef{Kind: "Node", Name: nodeName}, Evidence: evidence, Recommendation: recommendation, SafeActions: []DiagnosticSafeAction{{Action: "describe", RiskLevel: "read", Reason: "查看 Node 详情和事件"}, {Action: "inspect_pod_diagnostics", RiskLevel: "read", Reason: "检查节点上的异常 Pod"}}}
}
func groupPodsByNode(pods []corev1.Pod) map[string][]corev1.Pod {
	out := map[string][]corev1.Pod{}
	for _, p := range pods {
		if p.Spec.NodeName != "" {
			out[p.Spec.NodeName] = append(out[p.Spec.NodeName], p)
		}
	}
	return out
}
func sumPodRequests(pods []corev1.Pod) (int64, int64) {
	var cpu int64
	var mem int64
	for _, p := range pods {
		for _, c := range p.Spec.Containers {
			cpu += c.Resources.Requests.Cpu().MilliValue()
			mem += c.Resources.Requests.Memory().Value()
		}
	}
	return cpu, mem
}
