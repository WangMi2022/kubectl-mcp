package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kubectl-mcp/internal/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// InspectWorkloadReferences validates cross-resource references used by workloads and traffic objects.
func InspectWorkloadReferences(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	namespace := strings.TrimSpace(stringArg(args, "namespace", ""))
	includeIngress := boolArg(args, "includeIngress", true)
	includeHPA := boolArg(args, "includeHPA", true)
	includePDB := boolArg(args, "includePDB", true)

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	cs := clientSet.Clientset

	services, err := cs.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Service 失败: %w", err)
	}
	pods, err := cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 失败: %w", err)
	}
	endpoints, err := cs.CoreV1().Endpoints(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Endpoints 失败: %w", err)
	}
	statefulSets, err := cs.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 StatefulSet 失败: %w", err)
	}
	deployments, err := cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Deployment 失败: %w", err)
	}
	replicaSets, err := cs.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 ReplicaSet 失败: %w", err)
	}
	daemonSets, err := cs.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 DaemonSet 失败: %w", err)
	}

	serviceMap := buildServiceMap(services.Items)
	endpointMap := buildEndpointMap(endpoints.Items)
	podItems := pods.Items
	findings := make([]DiagnosticFinding, 0)
	for _, sts := range statefulSets.Items {
		findings = append(findings, inspectStatefulSetServiceReference(sts, serviceMap)...)
	}
	for _, svc := range services.Items {
		findings = append(findings, inspectServiceReferences(svc, podItems, endpointMap)...)
	}
	if includeIngress {
		ingresses, err := cs.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Ingress 失败: %w", err)
		}
		for _, ing := range ingresses.Items {
			findings = append(findings, inspectIngressReferences(ing, serviceMap)...)
		}
	}
	if includeHPA {
		workloads := buildWorkloadReferenceSet(deployments.Items, statefulSets.Items, replicaSets.Items, daemonSets.Items)
		hpaFindings, err := inspectHPAReferences(ctx, cs, namespace, workloads)
		if err != nil {
			return nil, fmt.Errorf("查询 HPA 失败: %w", err)
		}
		findings = append(findings, hpaFindings...)
	}
	if includePDB {
		pdbs, err := cs.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 PDB 失败: %w", err)
		}
		for _, pdb := range pdbs.Items {
			findings = append(findings, inspectPDBReference(pdb, podItems)...)
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].FindingType == findings[j].FindingType {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].FindingType < findings[j].FindingType
	})
	return DiagnosticReport{
		CheckTime: time.Now(),
		Scope:     DiagnosticScope{Cluster: contextName, Namespace: namespace},
		Summary:   buildDiagnosticSummary(findings),
		Findings:  findings,
	}, nil
}

func inspectStatefulSetServiceReference(sts appsv1.StatefulSet, services map[string]corev1.Service) []DiagnosticFinding {
	if strings.TrimSpace(sts.Spec.ServiceName) == "" {
		return nil
	}
	key := namespacedKey(sts.Namespace, sts.Spec.ServiceName)
	if _, ok := services[key]; ok {
		return nil
	}
	desc := fmt.Sprintf("StatefulSet %s/%s spec.serviceName=%s 指向的 Service 不存在。", sts.Namespace, sts.Name, sts.Spec.ServiceName)
	return []DiagnosticFinding{newReferenceFinding("MissingService", "warning", "StatefulSet 引用的 Headless Service 不存在", desc, DiagnosticObjectRef{Kind: "StatefulSet", Namespace: sts.Namespace, Name: sts.Name}, []DiagnosticObjectRef{{Kind: "Service", Namespace: sts.Namespace, Name: sts.Spec.ServiceName}}, []DiagnosticEvidence{{Source: "ownerReference", Message: desc, Count: 1}}, "创建匹配的 Headless Service，或修正 StatefulSet spec.serviceName。")}
}

func inspectServiceReferences(svc corev1.Service, pods []corev1.Pod, endpoints map[string]corev1.Endpoints) []DiagnosticFinding {
	if svc.Spec.Type == corev1.ServiceTypeExternalName || len(svc.Spec.Selector) == 0 {
		return nil
	}
	selector := labels.SelectorFromSet(labels.Set(svc.Spec.Selector))
	matched := podsMatchingSelector(svc.Namespace, selector, pods)
	findings := make([]DiagnosticFinding, 0, 2)
	if len(matched) == 0 {
		desc := fmt.Sprintf("Service %s/%s selector=%s 未匹配到任何 Pod。", svc.Namespace, svc.Name, selector.String())
		findings = append(findings, newReferenceFinding("ServiceNoMatchedPods", "warning", "Service selector 未匹配 Pod", desc, DiagnosticObjectRef{Kind: "Service", Namespace: svc.Namespace, Name: svc.Name}, nil, []DiagnosticEvidence{{Source: "endpoint", Message: desc, Count: 1}}, "检查 Service selector 与 Pod labels 是否一致。"))
	}
	if ep, ok := endpoints[namespacedKey(svc.Namespace, svc.Name)]; !ok || !endpointHasAddress(ep) {
		desc := fmt.Sprintf("Service %s/%s 当前 Endpoints 为空，流量无法转发到后端 Pod。", svc.Namespace, svc.Name)
		findings = append(findings, newReferenceFinding("ServiceNoEndpoints", "warning", "Service Endpoints 为空", desc, DiagnosticObjectRef{Kind: "Service", Namespace: svc.Namespace, Name: svc.Name}, nil, []DiagnosticEvidence{{Source: "endpoint", Message: desc, Count: 1}}, "检查后端 Pod Ready 状态、selector、端口与 EndpointSlice/Endpoints 控制器。"))
	}
	return findings
}

func inspectIngressReferences(ing networkingv1.Ingress, services map[string]corev1.Service) []DiagnosticFinding {
	findings := make([]DiagnosticFinding, 0)
	checkBackend := func(path string, backend networkingv1.IngressBackend) {
		if backend.Service == nil {
			return
		}
		svcName := backend.Service.Name
		svc, ok := services[namespacedKey(ing.Namespace, svcName)]
		if !ok {
			desc := fmt.Sprintf("Ingress %s/%s backend %s 指向的 Service %s 不存在。", ing.Namespace, ing.Name, path, svcName)
			findings = append(findings, newReferenceFinding("MissingService", "warning", "Ingress backend Service 不存在", desc, DiagnosticObjectRef{Kind: "Ingress", Namespace: ing.Namespace, Name: ing.Name}, []DiagnosticObjectRef{{Kind: "Service", Namespace: ing.Namespace, Name: svcName}}, []DiagnosticEvidence{{Source: "endpoint", Message: desc, Count: 1}}, "创建目标 Service 或修正 Ingress backend serviceName。"))
			return
		}
		if !serviceHasPort(svc, backend.Service.Port) {
			desc := fmt.Sprintf("Ingress %s/%s backend %s 指向 Service %s 的端口不存在。", ing.Namespace, ing.Name, path, svcName)
			findings = append(findings, newReferenceFinding("IngressBackendPortMissing", "warning", "Ingress backend Service 端口不存在", desc, DiagnosticObjectRef{Kind: "Ingress", Namespace: ing.Namespace, Name: ing.Name}, []DiagnosticObjectRef{{Kind: "Service", Namespace: ing.Namespace, Name: svcName}}, []DiagnosticEvidence{{Source: "endpoint", Message: desc, Count: 1}}, "修正 Ingress backend service.port，或在目标 Service 上补齐端口。"))
		}
	}
	if ing.Spec.DefaultBackend != nil {
		checkBackend("defaultBackend", *ing.Spec.DefaultBackend)
	}
	for _, rule := range ing.Spec.Rules {
		if rule.HTTP == nil {
			continue
		}
		for _, path := range rule.HTTP.Paths {
			checkBackend(fmt.Sprintf("%s%s", rule.Host, path.Path), path.Backend)
		}
	}
	return findings
}

func inspectHPAReferences(ctx context.Context, cs kubernetes.Interface, namespace string, workloads map[string]struct{}) ([]DiagnosticFinding, error) {
	hpasV2, err := cs.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		findings := make([]DiagnosticFinding, 0)
		for _, hpa := range hpasV2.Items {
			findings = append(findings, inspectHPAReference(hpa.Namespace, hpa.Name, hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name, workloads)...)
		}
		return findings, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// 兼容较老集群：部分环境未开启 autoscaling/v2，仅支持 autoscaling/v1。
	hpasV1, fallbackErr := cs.AutoscalingV1().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if fallbackErr != nil {
		return nil, err
	}
	findings := make([]DiagnosticFinding, 0)
	for _, hpa := range hpasV1.Items {
		findings = append(findings, inspectHPAReference(hpa.Namespace, hpa.Name, hpa.Spec.ScaleTargetRef.Kind, hpa.Spec.ScaleTargetRef.Name, workloads)...)
	}
	return findings, nil
}

func inspectHPAReference(namespace, hpaName, targetKind, targetName string, workloads map[string]struct{}) []DiagnosticFinding {
	kind := normalizeWorkloadKind(targetKind)
	key := workloadReferenceKey(namespace, kind, targetName)
	if _, ok := workloads[key]; ok {
		return nil
	}
	desc := fmt.Sprintf("HPA %s/%s scaleTargetRef 指向的 %s/%s 不存在。", namespace, hpaName, kind, targetName)
	return []DiagnosticFinding{newReferenceFinding("HPATargetMissing", "warning", "HPA scaleTargetRef 目标不存在", desc, DiagnosticObjectRef{Kind: "HorizontalPodAutoscaler", Namespace: namespace, Name: hpaName}, []DiagnosticObjectRef{{Kind: kind, Namespace: namespace, Name: targetName}}, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "检查 HPA scaleTargetRef 的 apiVersion/kind/name 是否正确，或恢复目标工作负载。")}
}

func inspectPDBReference(pdb policyv1.PodDisruptionBudget, pods []corev1.Pod) []DiagnosticFinding {
	if pdb.Spec.Selector == nil {
		return nil
	}
	selector, err := metav1.LabelSelectorAsSelector(pdb.Spec.Selector)
	if err != nil {
		desc := fmt.Sprintf("PDB %s/%s selector 无法解析：%s。", pdb.Namespace, pdb.Name, err.Error())
		return []DiagnosticFinding{newReferenceFinding("PDBSelectorInvalid", "warning", "PDB selector 无法解析", desc, DiagnosticObjectRef{Kind: "PodDisruptionBudget", Namespace: pdb.Namespace, Name: pdb.Name}, nil, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "修正 PDB spec.selector。")}
	}
	matched := podsMatchingSelector(pdb.Namespace, selector, pods)
	if len(matched) > 0 {
		return nil
	}
	desc := fmt.Sprintf("PDB %s/%s selector=%s 未匹配到任何 Pod。", pdb.Namespace, pdb.Name, selector.String())
	return []DiagnosticFinding{newReferenceFinding("PDBNoMatchedPods", "warning", "PDB selector 未匹配 Pod", desc, DiagnosticObjectRef{Kind: "PodDisruptionBudget", Namespace: pdb.Namespace, Name: pdb.Name}, nil, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "检查 PDB selector 与目标工作负载 Pod labels 是否一致。")}
}

func newReferenceFinding(findingType, severity, title, description string, affected DiagnosticObjectRef, related []DiagnosticObjectRef, evidence []DiagnosticEvidence, recommendation string) DiagnosticFinding {
	return DiagnosticFinding{
		ID:             stableFindingID("workload-reference", findingType, affected.Namespace, affected.Kind, affected.Name, description),
		Severity:       severity,
		FindingType:    findingType,
		Title:          title,
		Description:    description,
		AffectedObject: affected,
		Evidence:       evidence,
		Recommendation: recommendation,
		RelatedObjects: related,
		SafeActions: []DiagnosticSafeAction{
			{Action: "describe", RiskLevel: "read", Reason: "查看引用双方资源详情"},
			{Action: "get_events", RiskLevel: "read", Reason: "查看相关事件证据"},
		},
	}
}

func buildServiceMap(services []corev1.Service) map[string]corev1.Service {
	out := make(map[string]corev1.Service, len(services))
	for _, svc := range services {
		out[namespacedKey(svc.Namespace, svc.Name)] = svc
	}
	return out
}

func buildEndpointMap(endpoints []corev1.Endpoints) map[string]corev1.Endpoints {
	out := make(map[string]corev1.Endpoints, len(endpoints))
	for _, ep := range endpoints {
		out[namespacedKey(ep.Namespace, ep.Name)] = ep
	}
	return out
}

func buildWorkloadReferenceSet(deployments []appsv1.Deployment, statefulSets []appsv1.StatefulSet, replicaSets []appsv1.ReplicaSet, daemonSets []appsv1.DaemonSet) map[string]struct{} {
	out := map[string]struct{}{}
	for _, item := range deployments {
		out[workloadReferenceKey(item.Namespace, "Deployment", item.Name)] = struct{}{}
	}
	for _, item := range statefulSets {
		out[workloadReferenceKey(item.Namespace, "StatefulSet", item.Name)] = struct{}{}
	}
	for _, item := range replicaSets {
		out[workloadReferenceKey(item.Namespace, "ReplicaSet", item.Name)] = struct{}{}
	}
	for _, item := range daemonSets {
		out[workloadReferenceKey(item.Namespace, "DaemonSet", item.Name)] = struct{}{}
	}
	return out
}

func podsMatchingSelector(namespace string, selector labels.Selector, pods []corev1.Pod) []corev1.Pod {
	matched := []corev1.Pod{}
	for _, pod := range pods {
		if pod.Namespace != namespace {
			continue
		}
		if selector.Matches(labels.Set(pod.Labels)) {
			matched = append(matched, pod)
		}
	}
	return matched
}

func endpointHasAddress(ep corev1.Endpoints) bool {
	for _, subset := range ep.Subsets {
		if len(subset.Addresses) > 0 {
			return true
		}
	}
	return false
}

func serviceHasPort(svc corev1.Service, port networkingv1.ServiceBackendPort) bool {
	for _, svcPort := range svc.Spec.Ports {
		if port.Name != "" && svcPort.Name == port.Name {
			return true
		}
		if port.Number != 0 && svcPort.Port == port.Number {
			return true
		}
	}
	return false
}

func namespacedKey(namespace, name string) string {
	return namespace + "/" + name
}

func workloadReferenceKey(namespace, kind, name string) string {
	return namespacedKey(namespace, normalizeWorkloadKind(kind)+"/"+name)
}

func normalizeWorkloadKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment", "deployments":
		return "Deployment"
	case "statefulset", "statefulsets":
		return "StatefulSet"
	case "replicaset", "replicasets":
		return "ReplicaSet"
	case "daemonset", "daemonsets":
		return "DaemonSet"
	default:
		if strings.TrimSpace(kind) == "" {
			return "Unknown"
		}
		return kind
	}
}

func stringArg(args map[string]interface{}, key, defaultValue string) string {
	if args == nil {
		return defaultValue
	}
	if value, ok := args[key].(string); ok {
		return value
	}
	return defaultValue
}

func boolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if args == nil {
		return defaultValue
	}
	if value, ok := args[key].(bool); ok {
		return value
	}
	return defaultValue
}
