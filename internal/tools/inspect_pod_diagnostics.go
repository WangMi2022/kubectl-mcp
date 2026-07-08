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

const defaultPodDiagnosticsTopN = 20

// InspectPodDiagnostics diagnoses abnormal pods without fetching logs by default.
func InspectPodDiagnostics(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, defaultNamespace, _ := getContextAndNamespace(args, k8sClient)
	namespace := strings.TrimSpace(stringArg(args, "namespace", defaultNamespace))
	podName := strings.TrimSpace(stringArg(args, "podName", ""))
	labelSelector := strings.TrimSpace(stringArg(args, "labelSelector", ""))
	topN := intArg(args, "topN", defaultPodDiagnosticsTopN)
	if topN <= 0 {
		topN = defaultPodDiagnosticsTopN
	}

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	cs := clientSet.Clientset

	pods := []corev1.Pod{}
	if podName != "" {
		pod, err := cs.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Pod %s/%s 失败: %w", namespace, podName, err)
		}
		pods = append(pods, *pod)
	} else {
		podList, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		if err != nil {
			return nil, fmt.Errorf("查询 Pod 失败: %w", err)
		}
		pods = append(pods, podList.Items...)
	}

	eventsByPod := map[string][]corev1.Event{}
	if namespace != "" {
		events, err := cs.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
		if err == nil {
			eventsByPod = groupPodEvents(events.Items)
		}
	}

	findings := make([]DiagnosticFinding, 0)
	for i := range pods {
		pod := pods[i]
		findings = append(findings, buildPodDiagnosticFindings(pod, eventsByPod[namespacedKey(pod.Namespace, pod.Name)])...)
	}
	sort.Slice(findings, func(i, j int) bool {
		severityOrder := map[string]int{"critical": 0, "warning": 1, "info": 2}
		if severityOrder[findings[i].Severity] != severityOrder[findings[j].Severity] {
			return severityOrder[findings[i].Severity] < severityOrder[findings[j].Severity]
		}
		return findings[i].ID < findings[j].ID
	})
	if len(findings) > topN {
		findings = findings[:topN]
	}

	return DiagnosticReport{
		CheckTime: time.Now(),
		Scope: DiagnosticScope{
			Cluster:    contextName,
			Namespace:  namespace,
			ObjectKind: "Pod",
			ObjectName: podName,
		},
		Summary:  buildDiagnosticSummary(findings),
		Findings: findings,
	}, nil
}

func groupPodEvents(events []corev1.Event) map[string][]corev1.Event {
	out := map[string][]corev1.Event{}
	for _, event := range events {
		if event.InvolvedObject.Kind != "Pod" || event.InvolvedObject.Name == "" {
			continue
		}
		ns := event.InvolvedObject.Namespace
		if ns == "" {
			ns = event.Namespace
		}
		out[namespacedKey(ns, event.InvolvedObject.Name)] = append(out[namespacedKey(ns, event.InvolvedObject.Name)], event)
	}
	return out
}

func buildPodDiagnosticFindings(pod corev1.Pod, events []corev1.Event) []DiagnosticFinding {
	findings := make([]DiagnosticFinding, 0)
	if pod.Status.Phase == corev1.PodPending {
		findingType := "PodPending"
		title := "Pod 处于 Pending"
		if hasEventReason(events, "FailedScheduling") {
			findingType = "ScheduleFailure"
			title = "Pod 调度失败"
		}
		desc := fmt.Sprintf("Pod %s/%s 处于 Pending，当前 node=%s。", pod.Namespace, pod.Name, emptyDash(pod.Spec.NodeName))
		findings = append(findings, newPodDiagnosticFinding(findingType, "warning", title, desc, pod, eventEvidence(events, "event", 2), "查看 Pod describe 与调度事件，确认资源、亲和性、污点或 PVC 约束。"))
	}
	if pod.Status.Phase == corev1.PodFailed {
		desc := fmt.Sprintf("Pod %s/%s 处于 Failed，reason=%s。", pod.Namespace, pod.Name, emptyDash(pod.Status.Reason))
		findings = append(findings, newPodDiagnosticFinding("PodFailed", "critical", "Pod 处于 Failed", desc, pod, eventEvidence(events, "event", 2), "查看终止原因和最近事件，必要时重新创建上层工作负载的 Pod。"))
	}

	for _, status := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		if status.State.Waiting != nil && status.State.Waiting.Reason != "" {
			reason := status.State.Waiting.Reason
			findingType := classifyContainerWaitingFinding(reason)
			desc := fmt.Sprintf("Pod %s/%s 容器 %s Waiting: %s，message=%s。", pod.Namespace, pod.Name, status.Name, reason, emptyDash(status.State.Waiting.Message))
			findings = append(findings, newPodDiagnosticFinding(findingType, severityForPodFinding(findingType), reason, desc, pod, mergeEvidence(containerEvidence(reason, status.RestartCount), eventEvidence(events, "event", 2)), recommendationForPodFinding(findingType)))
		}
		if status.State.Terminated != nil && status.State.Terminated.Reason != "" {
			reason := status.State.Terminated.Reason
			findingType := "ContainerTerminated"
			if strings.EqualFold(reason, "OOMKilled") {
				findingType = "OOMKilled"
			}
			desc := fmt.Sprintf("Pod %s/%s 容器 %s Terminated: %s，exitCode=%d。", pod.Namespace, pod.Name, status.Name, reason, status.State.Terminated.ExitCode)
			findings = append(findings, newPodDiagnosticFinding(findingType, severityForPodFinding(findingType), reason, desc, pod, mergeEvidence(containerEvidence(reason, status.RestartCount), eventEvidence(events, "event", 2)), recommendationForPodFinding(findingType)))
		}
		if status.RestartCount >= highRestartThreshold {
			desc := fmt.Sprintf("Pod %s/%s 容器 %s 重启次数 %d，超过阈值 %d。", pod.Namespace, pod.Name, status.Name, status.RestartCount, highRestartThreshold)
			findings = append(findings, newPodDiagnosticFinding("HighRestart", "warning", "容器高重启", desc, pod, mergeEvidence(containerEvidence("RestartCount", status.RestartCount), eventEvidence(events, "event", 1)), "查看 previous 日志、探针配置、资源限制和依赖服务状态。"))
		}
	}
	return findings
}

func classifyContainerWaitingFinding(reason string) string {
	switch strings.ToLower(reason) {
	case "crashloopbackoff":
		return "CrashLoopBackOff"
	case "imagepullbackoff", "errimagepull":
		return "ImagePullFailure"
	case "createcontainerconfigerror", "createcontainererror":
		return "ContainerConfigError"
	case "containercreating":
		return "ContainerCreating"
	default:
		return "ContainerWaiting"
	}
}

func severityForPodFinding(findingType string) string {
	switch findingType {
	case "PodFailed", "CrashLoopBackOff", "ImagePullFailure", "OOMKilled":
		return "critical"
	default:
		return "warning"
	}
}

func recommendationForPodFinding(findingType string) string {
	switch findingType {
	case "CrashLoopBackOff":
		return "查看 previous 日志、启动命令、环境变量、依赖服务和探针配置。"
	case "ImagePullFailure":
		return "检查镜像名称、tag、镜像仓库连通性与 imagePullSecret。"
	case "OOMKilled":
		return "检查内存 limit/request、应用内存峰值和节点压力，必要时调整资源配置。"
	case "ContainerConfigError":
		return "检查 ConfigMap/Secret/volume/env 引用是否存在且键名正确。"
	default:
		return "继续 describe Pod、查看事件；如容器已重启，优先查看 previous 日志。"
	}
}

func newPodDiagnosticFinding(findingType, severity, title, description string, pod corev1.Pod, evidence []DiagnosticEvidence, recommendation string) DiagnosticFinding {
	return DiagnosticFinding{
		ID:             stableFindingID("pod-diagnostics", findingType, pod.Namespace, pod.Name, description),
		Severity:       severity,
		FindingType:    findingType,
		Title:          title,
		Description:    description,
		AffectedObject: DiagnosticObjectRef{Kind: "Pod", Namespace: pod.Namespace, Name: pod.Name},
		Evidence:       evidence,
		Recommendation: recommendation,
		SafeActions: []DiagnosticSafeAction{
			{Action: "describe", RiskLevel: "read", Reason: "查看 Pod 规格、状态和事件"},
			{Action: "get_pod_logs", RiskLevel: "read", Reason: "按需查看当前或 previous 日志，工具不会默认拉全量日志"},
		},
	}
}

func containerEvidence(reason string, restarts int32) []DiagnosticEvidence {
	msg := fmt.Sprintf("reason=%s", reason)
	if restarts > 0 {
		msg = fmt.Sprintf("%s, restartCount=%d", msg, restarts)
	}
	return []DiagnosticEvidence{{Source: "podStatus", Message: msg, Count: normalizedEventCount(restarts)}}
}

func eventEvidence(events []corev1.Event, source string, limit int) []DiagnosticEvidence {
	out := make([]DiagnosticEvidence, 0)
	for _, event := range events {
		if event.Type != corev1.EventTypeWarning && event.Reason == "" {
			continue
		}
		lastSeen := event.LastTimestamp.Time
		if lastSeen.IsZero() {
			lastSeen = event.EventTime.Time
		}
		ev := DiagnosticEvidence{Source: source, Message: strings.TrimSpace(event.Reason + ": " + event.Message), Count: normalizedEventCount(event.Count)}
		if !lastSeen.IsZero() {
			ev.LastSeen = lastSeen.Format(time.RFC3339)
		}
		out = append(out, ev)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func mergeEvidence(groups ...[]DiagnosticEvidence) []DiagnosticEvidence {
	out := make([]DiagnosticEvidence, 0)
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

func hasEventReason(events []corev1.Event, reason string) bool {
	for _, event := range events {
		if event.Reason == reason {
			return true
		}
	}
	return false
}

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
