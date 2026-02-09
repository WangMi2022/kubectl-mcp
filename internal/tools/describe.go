package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"kubectl-mcp/internal/k8s"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DescribeResource 获取资源详情
func DescribeResource(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	verbose := isVerbose(args)

	kind, ok := args["kind"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("缺少必填参数: kind")
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("缺少必填参数: name")
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	if namespace == "" {
		namespace = "default"
	}

	switch strings.ToLower(kind) {
	case "pod":
		return describePod(ctx, clientset, namespace, name, verbose)
	case "deployment":
		return describeDeployment(ctx, clientset, namespace, name, verbose)
	case "service":
		return describeService(ctx, clientset, namespace, name, verbose)
	case "configmap":
		return describeConfigMap(ctx, clientset, namespace, name, verbose)
	case "secret":
		return describeSecret(ctx, clientset, namespace, name, verbose)
	case "node":
		return describeNode(ctx, clientset, name, verbose)
	case "namespace":
		return describeNamespace(ctx, clientset, name, verbose)
	case "statefulset":
		return describeStatefulSet(ctx, clientset, namespace, name, verbose)
	case "daemonset":
		return describeDaemonSet(ctx, clientset, namespace, name, verbose)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", kind)
	}
}

func describePod(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	pod, err := clientset.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Pod '%s/%s' 详情失败: %w", namespace, name, err)
	}

	detail := &ResourceDetail{
		Kind:      "Pod",
		Name:      pod.Name,
		Namespace: pod.Namespace,
		CreatedAt: pod.CreationTimestamp.Time,
	}

	// 精简模式：只返回关键状态信息
	status := map[string]interface{}{
		"phase":  string(pod.Status.Phase),
		"podIP":  pod.Status.PodIP,
		"hostIP": pod.Status.HostIP,
		"node":   pod.Spec.NodeName,
	}
	containers := make([]map[string]interface{}, 0)
	for _, cs := range pod.Status.ContainerStatuses {
		containers = append(containers, map[string]interface{}{
			"name":         cs.Name,
			"image":        cs.Image,
			"ready":        cs.Ready,
			"restartCount": cs.RestartCount,
			"state":        getContainerState(cs),
		})
	}
	status["containers"] = containers
	detail.Status = status

	if verbose {
		detail.Labels = pod.Labels
		detail.Annotations = pod.Annotations
		spec, _ := toMap(pod.Spec)
		detail.Spec = spec
	}

	return detail, nil
}

func describeDeployment(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	deploy, err := clientset.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Deployment '%s/%s' 详情失败: %w", namespace, name, err)
	}

	replicas := int32(0)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	images := make([]string, 0)
	for _, c := range deploy.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	detail := &ResourceDetail{
		Kind:      "Deployment",
		Name:      deploy.Name,
		Namespace: deploy.Namespace,
		CreatedAt: deploy.CreationTimestamp.Time,
		Status: map[string]interface{}{
			"replicas":          replicas,
			"readyReplicas":     deploy.Status.ReadyReplicas,
			"availableReplicas": deploy.Status.AvailableReplicas,
			"updatedReplicas":   deploy.Status.UpdatedReplicas,
			"images":            images,
			"strategy":          string(deploy.Spec.Strategy.Type),
			"selector":          deploy.Spec.Selector.MatchLabels,
		},
	}

	if verbose {
		detail.Labels = deploy.Labels
		detail.Annotations = deploy.Annotations
		spec, _ := toMap(deploy.Spec)
		detail.Spec = spec
	}

	return detail, nil
}

func describeService(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	svc, err := clientset.Clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Service '%s/%s' 详情失败: %w", namespace, name, err)
	}

	ports := make([]map[string]interface{}, 0)
	for _, p := range svc.Spec.Ports {
		portInfo := map[string]interface{}{
			"port":       p.Port,
			"targetPort": p.TargetPort.String(),
			"protocol":   string(p.Protocol),
		}
		if p.NodePort != 0 {
			portInfo["nodePort"] = p.NodePort
		}
		if p.Name != "" {
			portInfo["name"] = p.Name
		}
		ports = append(ports, portInfo)
	}

	detail := &ResourceDetail{
		Kind:      "Service",
		Name:      svc.Name,
		Namespace: svc.Namespace,
		CreatedAt: svc.CreationTimestamp.Time,
		Status: map[string]interface{}{
			"type":      string(svc.Spec.Type),
			"clusterIP": svc.Spec.ClusterIP,
			"ports":     ports,
			"selector":  svc.Spec.Selector,
		},
	}

	if verbose {
		detail.Labels = svc.Labels
		detail.Annotations = svc.Annotations
		spec, _ := toMap(svc.Spec)
		detail.Spec = spec
	}

	return detail, nil
}

func describeConfigMap(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	cm, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 ConfigMap '%s/%s' 详情失败: %w", namespace, name, err)
	}

	detail := &ResourceDetail{
		Kind:      "ConfigMap",
		Name:      cm.Name,
		Namespace: cm.Namespace,
		CreatedAt: cm.CreationTimestamp.Time,
	}

	// ConfigMap 的 Data 作为 Spec 返回
	spec := make(map[string]interface{})
	for k, v := range cm.Data {
		spec[k] = v
	}
	detail.Spec = spec

	if verbose {
		detail.Labels = cm.Labels
		detail.Annotations = cm.Annotations
	}

	return detail, nil
}

func describeSecret(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	secret, err := clientset.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Secret '%s/%s' 详情失败: %w", namespace, name, err)
	}

	dataKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		dataKeys = append(dataKeys, k)
	}

	detail := &ResourceDetail{
		Kind:      "Secret",
		Name:      secret.Name,
		Namespace: secret.Namespace,
		CreatedAt: secret.CreationTimestamp.Time,
		Spec: map[string]interface{}{
			"type":     string(secret.Type),
			"dataKeys": dataKeys,
		},
	}

	if verbose {
		detail.Labels = secret.Labels
		detail.Annotations = secret.Annotations
	}

	return detail, nil
}

func describeNode(ctx context.Context, clientset *k8s.ClientSet, name string, verbose bool) (interface{}, error) {
	node, err := clientset.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node '%s' 详情失败: %w", name, err)
	}

	detail := &ResourceDetail{
		Kind:      "Node",
		Name:      node.Name,
		CreatedAt: node.CreationTimestamp.Time,
		Status: map[string]interface{}{
			"status":            getNodeStatus(node.Status.Conditions),
			"roles":             getNodeRoles(node.Labels),
			"version":           node.Status.NodeInfo.KubeletVersion,
			"os":                node.Status.NodeInfo.OperatingSystem,
			"architecture":      node.Status.NodeInfo.Architecture,
			"containerRuntime":  node.Status.NodeInfo.ContainerRuntimeVersion,
			"allocatableCPU":    node.Status.Allocatable.Cpu().String(),
			"allocatableMemory": node.Status.Allocatable.Memory().String(),
		},
	}

	if verbose {
		detail.Labels = node.Labels
		detail.Annotations = node.Annotations
		spec, _ := toMap(node.Spec)
		detail.Spec = spec
		status, _ := toMap(node.Status)
		detail.Status = status
	}

	return detail, nil
}

func describeNamespace(ctx context.Context, clientset *k8s.ClientSet, name string, verbose bool) (interface{}, error) {
	ns, err := clientset.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Namespace '%s' 详情失败: %w", name, err)
	}

	detail := &ResourceDetail{
		Kind:      "Namespace",
		Name:      ns.Name,
		CreatedAt: ns.CreationTimestamp.Time,
		Status: map[string]interface{}{
			"phase": string(ns.Status.Phase),
		},
	}

	if verbose {
		detail.Labels = ns.Labels
		detail.Annotations = ns.Annotations
		spec, _ := toMap(ns.Spec)
		detail.Spec = spec
	}

	return detail, nil
}

func describeStatefulSet(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	sts, err := clientset.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 StatefulSet '%s/%s' 详情失败: %w", namespace, name, err)
	}

	replicas := int32(0)
	if sts.Spec.Replicas != nil {
		replicas = *sts.Spec.Replicas
	}

	images := make([]string, 0)
	for _, c := range sts.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	detail := &ResourceDetail{
		Kind:      "StatefulSet",
		Name:      sts.Name,
		Namespace: sts.Namespace,
		CreatedAt: sts.CreationTimestamp.Time,
		Status: map[string]interface{}{
			"replicas":      replicas,
			"readyReplicas": sts.Status.ReadyReplicas,
			"images":        images,
			"serviceName":   sts.Spec.ServiceName,
		},
	}

	if verbose {
		detail.Labels = sts.Labels
		detail.Annotations = sts.Annotations
		spec, _ := toMap(sts.Spec)
		detail.Spec = spec
	}

	return detail, nil
}

func describeDaemonSet(ctx context.Context, clientset *k8s.ClientSet, namespace, name string, verbose bool) (interface{}, error) {
	ds, err := clientset.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 DaemonSet '%s/%s' 详情失败: %w", namespace, name, err)
	}

	images := make([]string, 0)
	for _, c := range ds.Spec.Template.Spec.Containers {
		images = append(images, c.Image)
	}

	detail := &ResourceDetail{
		Kind:      "DaemonSet",
		Name:      ds.Name,
		Namespace: ds.Namespace,
		CreatedAt: ds.CreationTimestamp.Time,
		Status: map[string]interface{}{
			"desiredNumberScheduled": ds.Status.DesiredNumberScheduled,
			"numberReady":            ds.Status.NumberReady,
			"numberAvailable":        ds.Status.NumberAvailable,
			"images":                 images,
		},
	}

	if verbose {
		detail.Labels = ds.Labels
		detail.Annotations = ds.Annotations
		spec, _ := toMap(ds.Spec)
		detail.Spec = spec
	}

	return detail, nil
}

// toMap 将结构体转换为 map
func toMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}
