package tools

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PreviewDeleteResources 预检查删除资源（不执行真正的删除）
func PreviewDeleteResources(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	kind, ok := args["kind"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("参数 'kind' 是必填的")
	}

	labelSelector := ""
	if ls, ok := args["labelSelector"].(string); ok {
		labelSelector = ls
	}

	var names []string
	if namesArg, ok := args["names"].([]interface{}); ok {
		for _, n := range namesArg {
			if name, ok := n.(string); ok {
				names = append(names, name)
			}
		}
	}

	if len(names) == 0 && labelSelector == "" {
		return nil, fmt.Errorf("必须指定 'names' 或 'labelSelector' 参数")
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	result := &PreviewDeleteResult{
		Kind:          kind,
		Namespace:     namespace,
		LabelSelector: labelSelector,
		Resources:     []DeleteResourceDetail{},
		Timestamp:     time.Now(),
	}

	// 获取要删除的资源列表
	var resourcesToDelete []string
	if len(names) > 0 {
		resourcesToDelete = names
		result.TotalCount = len(names)
	} else {
		queryNames, err := listResourceNamesByKind(ctx, clientset, kind, namespace, labelSelector)
		if err != nil {
			return nil, fmt.Errorf("查询资源列表失败: %w", err)
		}
		resourcesToDelete = queryNames
		result.TotalCount = len(queryNames)
	}

	if result.TotalCount == 0 {
		result.Message = fmt.Sprintf("没有找到符合条件的 %s 资源", kind)
		return result, nil
	}

	// 分析每个资源的详细信息和影响
	var wg sync.WaitGroup
	var mu sync.Mutex
	maxRiskLevel := "low"

	for _, resourceName := range resourcesToDelete {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			detail, err := analyzeResourceForDeletion(ctx, clientset, kind, namespace, name)
			if err != nil {
				detail = &DeleteResourceDetail{
					Name:      name,
					Kind:      kind,
					Namespace: namespace,
					RiskLevel: "medium",
					Warnings:  []string{fmt.Sprintf("无法完全分析资源: %v", err)},
				}
			}

			mu.Lock()
			defer mu.Unlock()
			result.Resources = append(result.Resources, *detail)
			result.TotalImpactCount += len(detail.ImpactedResources)

			if compareRiskLevel(detail.RiskLevel, maxRiskLevel) > 0 {
				maxRiskLevel = detail.RiskLevel
			}
		}(resourceName)
	}

	wg.Wait()

	result.TotalRiskLevel = maxRiskLevel
	result.ConfirmationToken = generateConfirmationToken()
	result.Message = fmt.Sprintf("预检查完成：将删除 %d 个 %s 资源，总体风险等级为 %s，将影响 %d 个关联资源",
		result.TotalCount, kind, getRiskLevelChinese(maxRiskLevel), result.TotalImpactCount)

	return result, nil
}

// getRiskLevelChinese 获取风险等级的中文描述
func getRiskLevelChinese(level string) string {
	switch level {
	case "low":
		return "低"
	case "medium":
		return "中"
	case "high":
		return "高"
	case "critical":
		return "严重"
	default:
		return level
	}
}

// analyzeResourceForDeletion 分析单个资源的删除影响
func analyzeResourceForDeletion(ctx context.Context, clientset *k8s.ClientSet, kind, namespace, name string) (*DeleteResourceDetail, error) {
	detail := &DeleteResourceDetail{
		Name:              name,
		Kind:              kind,
		Namespace:         namespace,
		ImpactedResources: []ResourceImpact{},
		Warnings:          []string{},
	}

	switch kind {
	case "Pod":
		return analyzePodDeletion(ctx, clientset, namespace, name, detail)
	case "Deployment":
		return analyzeDeploymentDeletion(ctx, clientset, namespace, name, detail)
	case "StatefulSet":
		return analyzeStatefulSetDeletion(ctx, clientset, namespace, name, detail)
	case "DaemonSet":
		return analyzeDaemonSetDeletion(ctx, clientset, namespace, name, detail)
	case "Service":
		return analyzeServiceDeletion(ctx, clientset, namespace, name, detail)
	case "ConfigMap":
		return analyzeConfigMapDeletion(ctx, clientset, namespace, name, detail)
	case "Secret":
		return analyzeSecretDeletion(ctx, clientset, namespace, name, detail)
	default:
		detail.RiskLevel = "medium"
		return detail, nil
	}
}

// analyzePodDeletion 分析 Pod 删除的影响
func analyzePodDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	pod, err := clientset.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = pod.Labels
	detail.CreatedAt = pod.CreationTimestamp.Time
	detail.RiskLevel = "low"

	if len(pod.OwnerReferences) > 0 {
		detail.Warnings = append(detail.Warnings, fmt.Sprintf("该 Pod 由 %s 管理，删除后会自动重建", pod.OwnerReferences[0].Kind))
		detail.RiskLevel = "medium"
	}

	return detail, nil
}

// analyzeDeploymentDeletion 分析 Deployment 删除的影响
func analyzeDeploymentDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	deploy, err := clientset.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = deploy.Labels
	detail.CreatedAt = deploy.CreationTimestamp.Time
	detail.Replicas = *deploy.Spec.Replicas
	detail.ReadyReplicas = deploy.Status.ReadyReplicas
	detail.Selector = deploy.Spec.Selector.MatchLabels

	// 分析关联的 Pod
	pods, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deploy.Spec.Selector),
	})
	if err == nil && len(pods.Items) > 0 {
		podNames := make([]string, len(pods.Items))
		for i, pod := range pods.Items {
			podNames[i] = pod.Name
		}
		detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
			Type:        "Pod",
			Names:       podNames,
			Count:       len(podNames),
			Description: fmt.Sprintf("删除该 Deployment 会级联删除 %d 个 Pod", len(podNames)),
		})
	}

	// 分析关联的 ReplicaSet
	replicaSets, err := clientset.Clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deploy.Spec.Selector),
	})
	if err == nil && len(replicaSets.Items) > 0 {
		rsNames := make([]string, len(replicaSets.Items))
		for i, rs := range replicaSets.Items {
			rsNames[i] = rs.Name
		}
		detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
			Type:        "ReplicaSet",
			Names:       rsNames,
			Count:       len(rsNames),
			Description: fmt.Sprintf("删除该 Deployment 会级联删除 %d 个 ReplicaSet", len(rsNames)),
		})
	}

	if detail.Replicas > 1 {
		detail.RiskLevel = "high"
	} else {
		detail.RiskLevel = "medium"
	}

	if len(detail.ImpactedResources) > 0 {
		detail.Warnings = append(detail.Warnings, fmt.Sprintf("该 Deployment 管理 %d 个 Pod，删除会影响服务可用性", detail.ReadyReplicas))
	}

	return detail, nil
}

// analyzeStatefulSetDeletion 分析 StatefulSet 删除的影响
func analyzeStatefulSetDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	sts, err := clientset.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = sts.Labels
	detail.CreatedAt = sts.CreationTimestamp.Time
	detail.Replicas = *sts.Spec.Replicas
	detail.ReadyReplicas = sts.Status.ReadyReplicas
	detail.Selector = sts.Spec.Selector.MatchLabels

	pods, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(sts.Spec.Selector),
	})
	if err == nil && len(pods.Items) > 0 {
		podNames := make([]string, len(pods.Items))
		for i, pod := range pods.Items {
			podNames[i] = pod.Name
		}
		detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
			Type:        "Pod",
			Names:       podNames,
			Count:       len(podNames),
			Description: fmt.Sprintf("删除该 StatefulSet 会级联删除 %d 个 Pod", len(podNames)),
		})
	}

	detail.RiskLevel = "high"
	detail.Warnings = append(detail.Warnings, "StatefulSet 通常用于有状态应用，删除可能导致数据丢失")

	return detail, nil
}

// analyzeDaemonSetDeletion 分析 DaemonSet 删除的影响
func analyzeDaemonSetDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	ds, err := clientset.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = ds.Labels
	detail.CreatedAt = ds.CreationTimestamp.Time
	detail.DesiredScheduled = ds.Status.DesiredNumberScheduled
	detail.Selector = ds.Spec.Selector.MatchLabels

	pods, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(ds.Spec.Selector),
	})
	if err == nil && len(pods.Items) > 0 {
		podNames := make([]string, len(pods.Items))
		for i, pod := range pods.Items {
			podNames[i] = pod.Name
		}
		detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
			Type:        "Pod",
			Names:       podNames,
			Count:       len(podNames),
			Description: fmt.Sprintf("删除该 DaemonSet 会级联删除 %d 个 Pod", len(podNames)),
		})
	}

	detail.RiskLevel = "critical"
	detail.Warnings = append(detail.Warnings, "DaemonSet 在所有节点上运行，删除会影响整个集群的功能")

	return detail, nil
}

// analyzeServiceDeletion 分析 Service 删除的影响
func analyzeServiceDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	svc, err := clientset.Clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = svc.Labels
	detail.CreatedAt = svc.CreationTimestamp.Time
	detail.ServiceType = string(svc.Spec.Type)
	detail.Selector = svc.Spec.Selector

	// 分析关联的 Ingress
	ingresses, err := clientset.Clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		var relatedIngresses []string
		for _, ing := range ingresses.Items {
			for _, rule := range ing.Spec.Rules {
				if rule.HTTP != nil {
					for _, path := range rule.HTTP.Paths {
						if path.Backend.Service != nil && path.Backend.Service.Name == name {
							relatedIngresses = append(relatedIngresses, ing.Name)
							break
						}
					}
				}
			}
		}
		if len(relatedIngresses) > 0 {
			detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
				Type:        "Ingress",
				Names:       relatedIngresses,
				Count:       len(relatedIngresses),
				Description: fmt.Sprintf("该 Service 被 %d 个 Ingress 引用", len(relatedIngresses)),
			})
		}
	}

	// 分析关联的 Endpoint
	endpoints, err := clientset.Clientset.CoreV1().Endpoints(namespace).Get(ctx, name, metav1.GetOptions{})
	if err == nil && endpoints != nil {
		totalAddresses := 0
		for _, subset := range endpoints.Subsets {
			totalAddresses += len(subset.Addresses)
		}
		if totalAddresses > 0 {
			detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
				Type:        "Endpoint",
				Count:       totalAddresses,
				Description: fmt.Sprintf("该 Service 有 %d 个活跃的 Endpoint", totalAddresses),
			})
		}
	}

	if detail.ServiceType == "LoadBalancer" {
		detail.RiskLevel = "high"
		detail.Warnings = append(detail.Warnings, "该 Service 是 LoadBalancer 类型，删除会影响外部访问")
	} else if detail.ServiceType == "NodePort" {
		detail.RiskLevel = "medium"
		detail.Warnings = append(detail.Warnings, "该 Service 是 NodePort 类型，删除会影响节点端口访问")
	} else {
		detail.RiskLevel = "medium"
	}

	return detail, nil
}

// analyzeConfigMapDeletion 分析 ConfigMap 删除的影响
func analyzeConfigMapDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	cm, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = cm.Labels
	detail.CreatedAt = cm.CreationTimestamp.Time

	pods, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		var relatedPods []string
		for _, pod := range pods.Items {
			for _, volume := range pod.Spec.Volumes {
				if volume.ConfigMap != nil && volume.ConfigMap.Name == name {
					relatedPods = append(relatedPods, pod.Name)
					break
				}
			}
		}
		if len(relatedPods) > 0 {
			detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
				Type:        "Pod",
				Names:       relatedPods,
				Count:       len(relatedPods),
				Description: fmt.Sprintf("有 %d 个 Pod 使用了该 ConfigMap", len(relatedPods)),
			})
			detail.RiskLevel = "high"
			detail.Warnings = append(detail.Warnings, "删除该 ConfigMap 会导致使用它的 Pod 无法正常运行")
		} else {
			detail.RiskLevel = "low"
		}
	} else {
		detail.RiskLevel = "low"
	}

	return detail, nil
}

// analyzeSecretDeletion 分析 Secret 删除的影响
func analyzeSecretDeletion(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, detail *DeleteResourceDetail) (*DeleteResourceDetail, error) {
	secret, err := clientset.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return detail, err
	}

	detail.Labels = secret.Labels
	detail.CreatedAt = secret.CreationTimestamp.Time

	pods, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		var relatedPods []string
		for _, pod := range pods.Items {
			for _, volume := range pod.Spec.Volumes {
				if volume.Secret != nil && volume.Secret.SecretName == name {
					relatedPods = append(relatedPods, pod.Name)
					break
				}
			}
		}
		if len(relatedPods) > 0 {
			detail.ImpactedResources = append(detail.ImpactedResources, ResourceImpact{
				Type:        "Pod",
				Names:       relatedPods,
				Count:       len(relatedPods),
				Description: fmt.Sprintf("有 %d 个 Pod 使用了该 Secret", len(relatedPods)),
			})
			detail.RiskLevel = "critical"
			detail.Warnings = append(detail.Warnings, "删除该 Secret 会导致使用它的 Pod 无法访问敏感信息，可能导致应用故障")
		} else {
			detail.RiskLevel = "medium"
		}
	} else {
		detail.RiskLevel = "medium"
	}

	return detail, nil
}

// compareRiskLevel 比较风险等级
func compareRiskLevel(level1, level2 string) int {
	riskOrder := map[string]int{
		"low":      1,
		"medium":   2,
		"high":     3,
		"critical": 4,
	}

	order1 := riskOrder[level1]
	order2 := riskOrder[level2]

	if order1 > order2 {
		return 1
	} else if order1 < order2 {
		return -1
	}
	return 0
}

// generateConfirmationToken 生成确认令牌
func generateConfirmationToken() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
