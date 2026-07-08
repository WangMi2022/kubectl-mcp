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

// InspectNamespaceHealth aggregates namespace-level health findings.
func InspectNamespaceHealth(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, defaultNamespace, _ := getContextAndNamespace(args, k8sClient)
	namespace := strings.TrimSpace(stringArg(args, "namespace", defaultNamespace))
	if namespace == "" {
		return nil, fmt.Errorf("namespace 不能为空")
	}
	topN := intArg(args, "topN", 20)
	if topN <= 0 {
		topN = 20
	}
	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	cs := clientSet.Clientset
	pods, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}
	services, err := cs.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Service 失败: %w", err)
	}
	endpoints, _ := cs.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	pvcs, _ := cs.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	events, _ := cs.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	endpointMap := map[string]corev1.Endpoints{}
	if endpoints != nil {
		endpointMap = buildEndpointMap(endpoints.Items)
	}
	findings := make([]DiagnosticFinding, 0)
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning && pod.Status.Phase != corev1.PodSucceeded {
			desc := fmt.Sprintf("Namespace %s 中 Pod %s 状态=%s。", namespace, pod.Name, getPodStatus(&pod))
			findings = append(findings, newNamespaceHealthFinding("NamespaceAbnormalPod", "warning", "命名空间存在异常 Pod", desc, DiagnosticObjectRef{Kind: "Pod", Namespace: namespace, Name: pod.Name}, []DiagnosticEvidence{{Source: "podStatus", Message: desc, Count: 1}}, "对异常 Pod 执行 inspect_pod_diagnostics。"))
		}
	}
	for _, svc := range services.Items {
		if svc.Spec.Type != corev1.ServiceTypeExternalName && len(svc.Spec.Selector) > 0 {
			if ep, ok := endpointMap[namespacedKey(svc.Namespace, svc.Name)]; !ok || !endpointHasAddress(ep) {
				desc := fmt.Sprintf("Namespace %s 中 Service %s Endpoints 为空。", namespace, svc.Name)
				findings = append(findings, newNamespaceHealthFinding("NamespaceServiceNoEndpoints", "warning", "命名空间存在无后端 Service", desc, DiagnosticObjectRef{Kind: "Service", Namespace: namespace, Name: svc.Name}, []DiagnosticEvidence{{Source: "endpoint", Message: desc, Count: 1}}, "对该 Service 执行 inspect_service_connectivity。"))
			}
		}
	}
	if pvcs != nil {
		for _, pvc := range pvcs.Items {
			if pvc.Status.Phase == corev1.ClaimPending {
				desc := fmt.Sprintf("Namespace %s 中 PVC %s Pending。", namespace, pvc.Name)
				findings = append(findings, newNamespaceHealthFinding("NamespacePVCPending", "warning", "命名空间存在 Pending PVC", desc, DiagnosticObjectRef{Kind: "PersistentVolumeClaim", Namespace: namespace, Name: pvc.Name}, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "执行 inspect_storage_diagnostics 检查存储供给。"))
			}
		}
	}
	if events != nil {
		count := 0
		for _, e := range events.Items {
			if e.Type == corev1.EventTypeWarning {
				ns := e.InvolvedObject.Namespace
				if ns == "" {
					ns = namespace
				}
				desc := fmt.Sprintf("Warning Event %s %s/%s: %s", e.Reason, ns, e.InvolvedObject.Name, e.Message)
				findings = append(findings, newNamespaceHealthFinding("NamespaceWarningEvent", "info", "命名空间存在 Warning 事件", desc, DiagnosticObjectRef{Kind: e.InvolvedObject.Kind, Namespace: ns, Name: e.InvolvedObject.Name}, []DiagnosticEvidence{{Source: "event", Message: desc, Count: normalizedEventCount(e.Count)}}, "执行 inspect_event_root_causes 聚合事件根因。"))
				count++
				if count >= 5 {
					break
				}
			}
		}
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
	return DiagnosticReport{CheckTime: time.Now(), Scope: DiagnosticScope{Cluster: contextName, Namespace: namespace}, Summary: buildDiagnosticSummary(findings), Findings: findings}, nil
}

func newNamespaceHealthFinding(findingType, severity, title, description string, affected DiagnosticObjectRef, evidence []DiagnosticEvidence, recommendation string) DiagnosticFinding {
	return DiagnosticFinding{ID: stableFindingID("namespace-health", findingType, affected.Namespace, affected.Kind, affected.Name, description), Severity: severity, FindingType: findingType, Title: title, Description: description, AffectedObject: affected, Evidence: evidence, Recommendation: recommendation, SafeActions: []DiagnosticSafeAction{{Action: "inspect_pod_diagnostics", RiskLevel: "read", Reason: "检查 Pod 级别异常"}, {Action: "inspect_service_connectivity", RiskLevel: "read", Reason: "检查 Service 连通性"}, {Action: "inspect_storage_diagnostics", RiskLevel: "read", Reason: "检查存储异常"}}}
}
