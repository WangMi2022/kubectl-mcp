package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetPods 查询 Pod 列表
func GetPods(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	pods, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 Pod 列表失败: %w", err)
	}

	result := make([]PodInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
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

		// verbose 模式下返回额外字段
		if verbose {
			podInfo.Phase = string(pod.Status.Phase)
			podInfo.Labels = pod.Labels
			podInfo.CreatedAt = pod.CreationTimestamp.Time
		}

		result = append(result, podInfo)
	}

	return result, nil
}

// GetDeployments 查询 Deployment 列表
func GetDeployments(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	deployments, err := clientset.Clientset.AppsV1().Deployments(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 Deployment 列表失败: %w", err)
	}

	result := make([]DeploymentInfo, 0, len(deployments.Items))
	for _, deploy := range deployments.Items {
		images := make([]string, 0)
		for _, container := range deploy.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}

		replicas := int32(0)
		if deploy.Spec.Replicas != nil {
			replicas = *deploy.Spec.Replicas
		}

		deployInfo := DeploymentInfo{
			Name:          deploy.Name,
			Namespace:     deploy.Namespace,
			Replicas:      replicas,
			ReadyReplicas: deploy.Status.ReadyReplicas,
			Images:        images,
		}

		if verbose {
			deployInfo.AvailableReplicas = deploy.Status.AvailableReplicas
			deployInfo.UpdatedReplicas = deploy.Status.UpdatedReplicas
			deployInfo.Labels = deploy.Labels
			deployInfo.Selector = deploy.Spec.Selector.MatchLabels
			deployInfo.CreatedAt = deploy.CreationTimestamp.Time
			deployInfo.Strategy = string(deploy.Spec.Strategy.Type)
		}

		result = append(result, deployInfo)
	}

	return result, nil
}

// GetStatefulSets 查询 StatefulSet 列表
func GetStatefulSets(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	statefulsets, err := clientset.Clientset.AppsV1().StatefulSets(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 StatefulSet 列表失败: %w", err)
	}

	result := make([]StatefulSetInfo, 0, len(statefulsets.Items))
	for _, sts := range statefulsets.Items {
		images := make([]string, 0)
		for _, container := range sts.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}

		replicas := int32(0)
		if sts.Spec.Replicas != nil {
			replicas = *sts.Spec.Replicas
		}

		stsInfo := StatefulSetInfo{
			Name:          sts.Name,
			Namespace:     sts.Namespace,
			Replicas:      replicas,
			ReadyReplicas: sts.Status.ReadyReplicas,
			Images:        images,
			ServiceName:   sts.Spec.ServiceName,
		}

		if verbose {
			stsInfo.CurrentReplicas = sts.Status.CurrentReplicas
			stsInfo.Labels = sts.Labels
			stsInfo.CreatedAt = sts.CreationTimestamp.Time
		}

		result = append(result, stsInfo)
	}

	return result, nil
}

// GetDaemonSets 查询 DaemonSet 列表
func GetDaemonSets(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName := ""
	if ctx, ok := args["context"].(string); ok && ctx != "" {
		contextName = ctx
	}

	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	daemonsets, err := clientset.Clientset.AppsV1().DaemonSets(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 DaemonSet 列表失败: %w", err)
	}

	result := make([]DaemonSetInfo, 0, len(daemonsets.Items))
	for _, ds := range daemonsets.Items {
		images := make([]string, 0)
		for _, container := range ds.Spec.Template.Spec.Containers {
			images = append(images, container.Image)
		}

		dsInfo := DaemonSetInfo{
			Name:                   ds.Name,
			Namespace:              ds.Namespace,
			DesiredNumberScheduled: ds.Status.DesiredNumberScheduled,
			NumberReady:            ds.Status.NumberReady,
			Images:                 images,
		}

		if verbose {
			dsInfo.CurrentNumberScheduled = ds.Status.CurrentNumberScheduled
			dsInfo.NumberAvailable = ds.Status.NumberAvailable
			dsInfo.Labels = ds.Labels
			dsInfo.NodeSelector = ds.Spec.Template.Spec.NodeSelector
			dsInfo.CreatedAt = ds.CreationTimestamp.Time
		}

		result = append(result, dsInfo)
	}

	return result, nil
}
