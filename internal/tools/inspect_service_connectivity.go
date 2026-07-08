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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// InspectServiceConnectivity validates Service backend connectivity wiring.
func InspectServiceConnectivity(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, defaultNamespace, _ := getContextAndNamespace(args, k8sClient)
	namespace := strings.TrimSpace(stringArg(args, "namespace", defaultNamespace))
	serviceName := strings.TrimSpace(stringArg(args, "serviceName", ""))
	topN := intArg(args, "topN", 20)
	if topN <= 0 {
		topN = 20
	}

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	cs := clientSet.Clientset

	services := []corev1.Service{}
	if serviceName != "" {
		svc, err := cs.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Service %s/%s 失败: %w", namespace, serviceName, err)
		}
		services = append(services, *svc)
	} else {
		list, err := cs.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Service 失败: %w", err)
		}
		services = append(services, list.Items...)
	}
	pods, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}
	endpoints, err := cs.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Endpoints 失败: %w", err)
	}
	endpointMap := buildEndpointMap(endpoints.Items)

	findings := make([]DiagnosticFinding, 0)
	for _, svc := range services {
		findings = append(findings, inspectServiceConnectivityOne(svc, pods.Items, endpointMap)...)
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].FindingType == findings[j].FindingType {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].FindingType < findings[j].FindingType
	})
	if len(findings) > topN {
		findings = findings[:topN]
	}
	return DiagnosticReport{CheckTime: time.Now(), Scope: DiagnosticScope{Cluster: contextName, Namespace: namespace, ObjectKind: "Service", ObjectName: serviceName}, Summary: buildDiagnosticSummary(findings), Findings: findings}, nil
}

func inspectServiceConnectivityOne(svc corev1.Service, pods []corev1.Pod, endpoints map[string]corev1.Endpoints) []DiagnosticFinding {
	if svc.Spec.Type == corev1.ServiceTypeExternalName || len(svc.Spec.Selector) == 0 {
		return nil
	}
	selector := labels.SelectorFromSet(labels.Set(svc.Spec.Selector))
	matchedPods := podsMatchingSelector(svc.Namespace, selector, pods)
	findings := make([]DiagnosticFinding, 0)
	if len(matchedPods) == 0 {
		desc := fmt.Sprintf("Service %s/%s selector=%s 未匹配到任何 Pod。", svc.Namespace, svc.Name, selector.String())
		findings = append(findings, newServiceConnectivityFinding("ServiceNoMatchedPods", "warning", "Service selector 未匹配 Pod", desc, svc, nil, []DiagnosticEvidence{{Source: "selector", Message: desc, Count: 1}}, "修正 Service selector 或目标 Pod labels。"))
		return findings
	}

	readyCount := 0
	for _, pod := range matchedPods {
		if isPodReady(pod) {
			readyCount++
		}
	}
	if readyCount == 0 {
		desc := fmt.Sprintf("Service %s/%s 匹配 %d 个 Pod，但没有 Ready Pod。", svc.Namespace, svc.Name, len(matchedPods))
		findings = append(findings, newServiceConnectivityFinding("ServiceTargetsNotReady", "warning", "Service 后端 Pod 均 NotReady", desc, svc, podRefs(matchedPods, 5), []DiagnosticEvidence{{Source: "podStatus", Message: desc, Count: int32(len(matchedPods))}}, "检查 Pod readinessProbe、容器状态和事件。"))
	}
	if ep, ok := endpoints[namespacedKey(svc.Namespace, svc.Name)]; !ok || !endpointHasAddress(ep) {
		desc := fmt.Sprintf("Service %s/%s Endpoints 为空。", svc.Namespace, svc.Name)
		findings = append(findings, newServiceConnectivityFinding("ServiceNoEndpoints", "warning", "Service Endpoints 为空", desc, svc, podRefs(matchedPods, 5), []DiagnosticEvidence{{Source: "endpoint", Message: desc, Count: 1}}, "检查 EndpointSlice/Endpoints 控制器、Pod Ready 状态和 selector。"))
	}
	for _, port := range svc.Spec.Ports {
		if !serviceTargetPortExists(port, matchedPods) {
			target := targetPortLabel(port)
			desc := fmt.Sprintf("Service %s/%s 端口 %s targetPort=%s 未在匹配 Pod 容器端口中找到。", svc.Namespace, svc.Name, port.Name, target)
			findings = append(findings, newServiceConnectivityFinding("TargetPortMismatch", "warning", "Service targetPort 与容器端口不匹配", desc, svc, podRefs(matchedPods, 5), []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "修正 Service targetPort 或目标容器 ports 定义。"))
		}
	}
	return findings
}

func newServiceConnectivityFinding(findingType, severity, title, description string, svc corev1.Service, related []DiagnosticObjectRef, evidence []DiagnosticEvidence, recommendation string) DiagnosticFinding {
	return DiagnosticFinding{ID: stableFindingID("service-connectivity", findingType, svc.Namespace, svc.Name, description), Severity: severity, FindingType: findingType, Title: title, Description: description, AffectedObject: DiagnosticObjectRef{Kind: "Service", Namespace: svc.Namespace, Name: svc.Name}, RelatedObjects: related, Evidence: evidence, Recommendation: recommendation, SafeActions: []DiagnosticSafeAction{{Action: "describe", RiskLevel: "read", Reason: "查看 Service、Endpoints 与相关 Pod"}, {Action: "inspect_pod_diagnostics", RiskLevel: "read", Reason: "进一步检查后端 Pod 健康状态"}}}
}

func serviceTargetPortExists(port corev1.ServicePort, pods []corev1.Pod) bool {
	target := port.TargetPort
	if target.Type == intstr.Int && target.IntVal == 0 {
		target = intstr.FromInt32(port.Port)
	}
	for _, pod := range pods {
		for _, container := range pod.Spec.Containers {
			for _, cp := range container.Ports {
				if target.Type == intstr.String && cp.Name == target.StrVal {
					return true
				}
				if target.Type == intstr.Int && cp.ContainerPort == target.IntVal {
					return true
				}
			}
		}
	}
	return false
}

func targetPortLabel(port corev1.ServicePort) string {
	if port.TargetPort.Type == intstr.String {
		return port.TargetPort.StrVal
	}
	if port.TargetPort.IntVal != 0 {
		return fmt.Sprintf("%d", port.TargetPort.IntVal)
	}
	return fmt.Sprintf("%d", port.Port)
}

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func podRefs(pods []corev1.Pod, limit int) []DiagnosticObjectRef {
	out := make([]DiagnosticObjectRef, 0)
	for _, pod := range pods {
		out = append(out, DiagnosticObjectRef{Kind: "Pod", Namespace: pod.Namespace, Name: pod.Name})
		if len(out) >= limit {
			break
		}
	}
	return out
}
