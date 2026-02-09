package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetResourceFilter 在整个集群的所有命名空间中查找指定类型的资源
// 支持精确匹配和模糊匹配两种模式
// 参数:
//   - kind: 资源类型 (Pod, Deployment, Service, StatefulSet, DaemonSet, ConfigMap, Secret)
//   - name: 资源名称
//   - matchMode: 匹配模式 (exact=精确匹配, fuzzy=模糊匹配)，默认 exact
//   - context: K8S context 名称（可选）
//
// 返回:
//   - 如果找到匹配的资源，返回资源的详细信息
//   - 如果找到多个匹配的资源，返回所有匹配的资源列表
//   - 如果未找到，返回错误
func GetResourceFilter(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	// 获取必填参数
	kind, ok := args["kind"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("参数 'kind' 是必填的")
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	// 获取匹配模式，默认为精确匹配
	matchMode := "exact"
	if mode, ok := args["matchMode"].(string); ok && mode != "" {
		matchMode = mode
	}

	// 获取可选的 context
	contextName := ""
	if ctxName, ok := args["context"].(string); ok && ctxName != "" {
		contextName = ctxName
	}

	// 获取客户端
	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 获取 verbose 参数
	verbose := isVerbose(args)

	// 根据资源类型调用不同的查询函数
	switch strings.ToLower(kind) {
	case "pod", "pods":
		return filterPods(ctx, clientset, name, matchMode, verbose)
	case "deployment", "deployments":
		return filterDeployments(ctx, clientset, name, matchMode, verbose)
	case "service", "services", "svc":
		return filterServices(ctx, clientset, name, matchMode, verbose)
	case "statefulset", "statefulsets", "sts":
		return filterStatefulSets(ctx, clientset, name, matchMode, verbose)
	case "daemonset", "daemonsets", "ds":
		return filterDaemonSets(ctx, clientset, name, matchMode, verbose)
	case "configmap", "configmaps", "cm":
		return filterConfigMaps(ctx, clientset, name, matchMode, verbose)
	case "secret", "secrets":
		return filterSecrets(ctx, clientset, name, matchMode, verbose)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s。支持的类型: Pod, Deployment, Service, StatefulSet, DaemonSet, ConfigMap, Secret", kind)
	}
}

// filterPods 过滤 Pod 资源
func filterPods(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var pods *corev1.PodList
	var err error

	// 根据匹配模式选择查询方式
	if matchMode == "exact" {
		// 精确匹配：使用 FieldSelector 直接在 API 层面过滤
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		pods, err = clientset.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		// 模糊匹配：获取所有 Pod，然后在代码中过滤
		listOptions := metav1.ListOptions{}
		pods, err = clientset.Clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 Pod 列表失败: %w", err)
	}

	// 构建匹配的 Pod 列表
	var matchedPods []PodInfo
	for _, pod := range pods.Items {
		// 根据匹配模式判断是否匹配
		matched := false
		if matchMode == "exact" {
			matched = pod.Name == name
		} else {
			matched = strings.Contains(pod.Name, name)
		}

		if matched {
			containers := make([]ContainerInfo, 0, len(pod.Status.ContainerStatuses))
			for _, cs := range pod.Status.ContainerStatuses {
				containers = append(containers, ContainerInfo{
					Name:         cs.Name,
					Image:        cs.Image,
					Ready:        cs.Ready,
					RestartCount: cs.RestartCount,
					State:        getContainerState(cs),
				})
			}

			podInfo := PodInfo{
				Name:       pod.Name,
				Namespace:  pod.Namespace,
				Status:     getPodStatus(&pod),
				IP:         pod.Status.PodIP,
				Node:       pod.Spec.NodeName,
				Containers: containers,
				Restarts:   getTotalRestarts(&pod),
			}

			if verbose {
				podInfo.Phase = string(pod.Status.Phase)
				podInfo.Labels = pod.Labels
				podInfo.CreatedAt = pod.CreationTimestamp.Time
			}

			matchedPods = append(matchedPods, podInfo)
		}
	}

	return buildResourceResponse("Pod", name, matchMode, len(matchedPods), matchedPods)
}

// filterDeployments 过滤 Deployment 资源
func filterDeployments(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var deployments *appsv1.DeploymentList
	var err error

	if matchMode == "exact" {
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		deployments, err = clientset.Clientset.AppsV1().Deployments(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		listOptions := metav1.ListOptions{}
		deployments, err = clientset.Clientset.AppsV1().Deployments(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 Deployment 列表失败: %w", err)
	}

	var matchedDeployments []DeploymentInfo
	for _, deploy := range deployments.Items {
		matched := false
		if matchMode == "exact" {
			matched = deploy.Name == name
		} else {
			matched = strings.Contains(deploy.Name, name)
		}

		if matched {
			deployInfo := DeploymentInfo{
				Name:          deploy.Name,
				Namespace:     deploy.Namespace,
				Replicas:      *deploy.Spec.Replicas,
				ReadyReplicas: deploy.Status.ReadyReplicas,
			}

			// 获取容器镜像
			if len(deploy.Spec.Template.Spec.Containers) > 0 {
				images := make([]string, 0, len(deploy.Spec.Template.Spec.Containers))
				for _, container := range deploy.Spec.Template.Spec.Containers {
					images = append(images, container.Image)
				}
				deployInfo.Images = images
			}

			if verbose {
				deployInfo.AvailableReplicas = deploy.Status.AvailableReplicas
				deployInfo.UpdatedReplicas = deploy.Status.UpdatedReplicas
				deployInfo.Labels = deploy.Labels
				deployInfo.Selector = deploy.Spec.Selector.MatchLabels
				deployInfo.CreatedAt = deploy.CreationTimestamp.Time
			}

			matchedDeployments = append(matchedDeployments, deployInfo)
		}
	}

	return buildResourceResponse("Deployment", name, matchMode, len(matchedDeployments), matchedDeployments)
}

// filterServices 过滤 Service 资源
func filterServices(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var services *corev1.ServiceList
	var err error

	if matchMode == "exact" {
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		services, err = clientset.Clientset.CoreV1().Services(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		listOptions := metav1.ListOptions{}
		services, err = clientset.Clientset.CoreV1().Services(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 Service 列表失败: %w", err)
	}

	var matchedServices []ServiceInfo
	for _, svc := range services.Items {
		matched := false
		if matchMode == "exact" {
			matched = svc.Name == name
		} else {
			matched = strings.Contains(svc.Name, name)
		}

		if matched {
			ports := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
			for _, port := range svc.Spec.Ports {
				ports = append(ports, ServicePortInfo{
					Name:       port.Name,
					Protocol:   string(port.Protocol),
					Port:       port.Port,
					TargetPort: port.TargetPort.String(),
					NodePort:   port.NodePort,
				})
			}

			svcInfo := ServiceInfo{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Type:      string(svc.Spec.Type),
				ClusterIP: svc.Spec.ClusterIP,
				Ports:     ports,
			}

			if verbose {
				// 获取 ExternalIP（取第一个）
				if len(svc.Spec.ExternalIPs) > 0 {
					svcInfo.ExternalIP = svc.Spec.ExternalIPs[0]
				}
				svcInfo.Selector = svc.Spec.Selector
				svcInfo.Labels = svc.Labels
				svcInfo.CreatedAt = svc.CreationTimestamp.Time
			}

			matchedServices = append(matchedServices, svcInfo)
		}
	}

	return buildResourceResponse("Service", name, matchMode, len(matchedServices), matchedServices)
}

// filterStatefulSets 过滤 StatefulSet 资源
func filterStatefulSets(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var statefulsets *appsv1.StatefulSetList
	var err error

	if matchMode == "exact" {
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		statefulsets, err = clientset.Clientset.AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		listOptions := metav1.ListOptions{}
		statefulsets, err = clientset.Clientset.AppsV1().StatefulSets(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 StatefulSet 列表失败: %w", err)
	}

	var matchedStatefulSets []StatefulSetInfo
	for _, sts := range statefulsets.Items {
		matched := false
		if matchMode == "exact" {
			matched = sts.Name == name
		} else {
			matched = strings.Contains(sts.Name, name)
		}

		if matched {
			stsInfo := StatefulSetInfo{
				Name:          sts.Name,
				Namespace:     sts.Namespace,
				Replicas:      *sts.Spec.Replicas,
				ReadyReplicas: sts.Status.ReadyReplicas,
				ServiceName:   sts.Spec.ServiceName,
			}

			// 获取容器镜像
			if len(sts.Spec.Template.Spec.Containers) > 0 {
				images := make([]string, 0, len(sts.Spec.Template.Spec.Containers))
				for _, container := range sts.Spec.Template.Spec.Containers {
					images = append(images, container.Image)
				}
				stsInfo.Images = images
			}

			if verbose {
				stsInfo.Labels = sts.Labels
				stsInfo.CreatedAt = sts.CreationTimestamp.Time
			}

			matchedStatefulSets = append(matchedStatefulSets, stsInfo)
		}
	}

	return buildResourceResponse("StatefulSet", name, matchMode, len(matchedStatefulSets), matchedStatefulSets)
}

// filterDaemonSets 过滤 DaemonSet 资源
func filterDaemonSets(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var daemonsets *appsv1.DaemonSetList
	var err error

	if matchMode == "exact" {
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		daemonsets, err = clientset.Clientset.AppsV1().DaemonSets(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		listOptions := metav1.ListOptions{}
		daemonsets, err = clientset.Clientset.AppsV1().DaemonSets(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 DaemonSet 列表失败: %w", err)
	}

	var matchedDaemonSets []DaemonSetInfo
	for _, ds := range daemonsets.Items {
		matched := false
		if matchMode == "exact" {
			matched = ds.Name == name
		} else {
			matched = strings.Contains(ds.Name, name)
		}

		if matched {
			dsInfo := DaemonSetInfo{
				Name:                   ds.Name,
				Namespace:              ds.Namespace,
				DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
				NumberReady:            ds.Status.NumberReady,
			}

			// 获取容器镜像
			if len(ds.Spec.Template.Spec.Containers) > 0 {
				images := make([]string, 0, len(ds.Spec.Template.Spec.Containers))
				for _, container := range ds.Spec.Template.Spec.Containers {
					images = append(images, container.Image)
				}
				dsInfo.Images = images
			}

			if verbose {
				dsInfo.CurrentNumberScheduled = ds.Status.CurrentNumberScheduled
				dsInfo.Labels = ds.Labels
				dsInfo.CreatedAt = ds.CreationTimestamp.Time
			}

			matchedDaemonSets = append(matchedDaemonSets, dsInfo)
		}
	}

	return buildResourceResponse("DaemonSet", name, matchMode, len(matchedDaemonSets), matchedDaemonSets)
}

// filterConfigMaps 过滤 ConfigMap 资源
func filterConfigMaps(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var configmaps *corev1.ConfigMapList
	var err error

	if matchMode == "exact" {
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		configmaps, err = clientset.Clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		listOptions := metav1.ListOptions{}
		configmaps, err = clientset.Clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 ConfigMap 列表失败: %w", err)
	}

	var matchedConfigMaps []ConfigMapInfo
	for _, cm := range configmaps.Items {
		matched := false
		if matchMode == "exact" {
			matched = cm.Name == name
		} else {
			matched = strings.Contains(cm.Name, name)
		}

		if matched {
			dataKeys := make([]string, 0, len(cm.Data))
			for key := range cm.Data {
				dataKeys = append(dataKeys, key)
			}

			cmInfo := ConfigMapInfo{
				Name:      cm.Name,
				Namespace: cm.Namespace,
				DataKeys:  dataKeys,
			}

			if verbose {
				cmInfo.Labels = cm.Labels
				cmInfo.CreatedAt = cm.CreationTimestamp.Time
			}

			matchedConfigMaps = append(matchedConfigMaps, cmInfo)
		}
	}

	return buildResourceResponse("ConfigMap", name, matchMode, len(matchedConfigMaps), matchedConfigMaps)
}

// filterSecrets 过滤 Secret 资源（脱敏处理）
func filterSecrets(ctx context.Context, clientset *k8s.ClientSet, name, matchMode string, verbose bool) (interface{}, error) {
	var secrets *corev1.SecretList
	var err error

	if matchMode == "exact" {
		listOptions := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", name),
		}
		secrets, err = clientset.Clientset.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, listOptions)
	} else {
		listOptions := metav1.ListOptions{}
		secrets, err = clientset.Clientset.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("查询 Secret 列表失败: %w", err)
	}

	var matchedSecrets []SecretInfo
	for _, secret := range secrets.Items {
		matched := false
		if matchMode == "exact" {
			matched = secret.Name == name
		} else {
			matched = strings.Contains(secret.Name, name)
		}

		if matched {
			dataKeys := make([]string, 0, len(secret.Data))
			for key := range secret.Data {
				dataKeys = append(dataKeys, key)
			}

			secretInfo := SecretInfo{
				Name:      secret.Name,
				Namespace: secret.Namespace,
				Type:      string(secret.Type),
				DataKeys:  dataKeys,
			}

			if verbose {
				secretInfo.Labels = secret.Labels
				secretInfo.CreatedAt = secret.CreationTimestamp.Time
			}

			matchedSecrets = append(matchedSecrets, secretInfo)
		}
	}

	return buildResourceResponse("Secret", name, matchMode, len(matchedSecrets), matchedSecrets)
}

// buildResourceResponse 构建统一的响应格式
func buildResourceResponse(kind, name, matchMode string, count int, resources interface{}) (interface{}, error) {
	// 检查是否找到资源
	if count == 0 {
		if matchMode == "exact" {
			return nil, fmt.Errorf("未找到名称为 '%s' 的 %s", name, kind)
		}
		return nil, fmt.Errorf("未找到名称包含 '%s' 的 %s", name, kind)
	}

	// 根据匹配结果返回
	if count == 1 {
		// 找到唯一匹配的资源，返回详细信息
		var namespace string
		switch v := resources.(type) {
		case []PodInfo:
			namespace = v[0].Namespace
		case []DeploymentInfo:
			namespace = v[0].Namespace
		case []ServiceInfo:
			namespace = v[0].Namespace
		case []StatefulSetInfo:
			namespace = v[0].Namespace
		case []DaemonSetInfo:
			namespace = v[0].Namespace
		case []ConfigMapInfo:
			namespace = v[0].Namespace
		case []SecretInfo:
			namespace = v[0].Namespace
		}

		return map[string]interface{}{
			"found":     true,
			"count":     1,
			"kind":      kind,
			"resource":  getFirstResource(resources),
			"message":   fmt.Sprintf("找到 %s '%s' 在命名空间 '%s'", kind, name, namespace),
			"namespace": namespace,
			"matchMode": matchMode,
		}, nil
	}

	// 找到多个匹配的资源，返回列表
	namespaces := extractNamespaces(resources)

	var message string
	if matchMode == "exact" {
		message = fmt.Sprintf("找到 %d 个同名 %s '%s'，分布在命名空间: %s", count, kind, name, strings.Join(namespaces, ", "))
	} else {
		message = fmt.Sprintf("找到 %d 个匹配的 %s（名称包含 '%s'），分布在命名空间: %s", count, kind, name, strings.Join(namespaces, ", "))
	}

	return map[string]interface{}{
		"found":      true,
		"count":      count,
		"kind":       kind,
		"resources":  resources,
		"message":    message,
		"namespaces": namespaces,
		"matchMode":  matchMode,
	}, nil
}

// getFirstResource 获取资源列表的第一个元素
func getFirstResource(resources interface{}) interface{} {
	switch v := resources.(type) {
	case []PodInfo:
		return v[0]
	case []DeploymentInfo:
		return v[0]
	case []ServiceInfo:
		return v[0]
	case []StatefulSetInfo:
		return v[0]
	case []DaemonSetInfo:
		return v[0]
	case []ConfigMapInfo:
		return v[0]
	case []SecretInfo:
		return v[0]
	}
	return nil
}

// extractNamespaces 提取资源列表中的所有命名空间
func extractNamespaces(resources interface{}) []string {
	var namespaces []string
	switch v := resources.(type) {
	case []PodInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	case []DeploymentInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	case []ServiceInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	case []StatefulSetInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	case []DaemonSetInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	case []ConfigMapInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	case []SecretInfo:
		for _, r := range v {
			namespaces = append(namespaces, r.Namespace)
		}
	}
	return namespaces
}
