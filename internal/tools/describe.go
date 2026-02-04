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

	// 获取必填参数
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

	// 根据资源类型获取详情
	switch strings.ToLower(kind) {
	case "pod":
		return describePod(ctx, clientset, namespace, name)
	case "deployment":
		return describeDeployment(ctx, clientset, namespace, name)
	case "service":
		return describeService(ctx, clientset, namespace, name)
	case "configmap":
		return describeConfigMap(ctx, clientset, namespace, name)
	case "secret":
		return describeSecret(ctx, clientset, namespace, name)
	case "node":
		return describeNode(ctx, clientset, name)
	case "namespace":
		return describeNamespace(ctx, clientset, name)
	case "statefulset":
		return describeStatefulSet(ctx, clientset, namespace, name)
	case "daemonset":
		return describeDaemonSet(ctx, clientset, namespace, name)
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", kind)
	}
}

// describePod 获取 Pod 详情
func describePod(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	pod, err := clientset.Clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Pod '%s/%s' 详情失败: %w", namespace, name, err)
	}

	spec, _ := toMap(pod.Spec)
	status, _ := toMap(pod.Status)

	return &ResourceDetail{
		Kind:        "Pod",
		Name:        pod.Name,
		Namespace:   pod.Namespace,
		Labels:      pod.Labels,
		Annotations: pod.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   pod.CreationTimestamp.Time,
	}, nil
}

// describeDeployment 获取 Deployment 详情
func describeDeployment(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	deploy, err := clientset.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Deployment '%s/%s' 详情失败: %w", namespace, name, err)
	}

	spec, _ := toMap(deploy.Spec)
	status, _ := toMap(deploy.Status)

	return &ResourceDetail{
		Kind:        "Deployment",
		Name:        deploy.Name,
		Namespace:   deploy.Namespace,
		Labels:      deploy.Labels,
		Annotations: deploy.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   deploy.CreationTimestamp.Time,
	}, nil
}

// describeService 获取 Service 详情
func describeService(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	svc, err := clientset.Clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Service '%s/%s' 详情失败: %w", namespace, name, err)
	}

	spec, _ := toMap(svc.Spec)
	status, _ := toMap(svc.Status)

	return &ResourceDetail{
		Kind:        "Service",
		Name:        svc.Name,
		Namespace:   svc.Namespace,
		Labels:      svc.Labels,
		Annotations: svc.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   svc.CreationTimestamp.Time,
	}, nil
}

// describeConfigMap 获取 ConfigMap 详情
func describeConfigMap(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	cm, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 ConfigMap '%s/%s' 详情失败: %w", namespace, name, err)
	}

	// ConfigMap 的 Data 作为 Spec 返回
	spec := make(map[string]interface{})
	for k, v := range cm.Data {
		spec[k] = v
	}

	return &ResourceDetail{
		Kind:        "ConfigMap",
		Name:        cm.Name,
		Namespace:   cm.Namespace,
		Labels:      cm.Labels,
		Annotations: cm.Annotations,
		Spec:        spec,
		CreatedAt:   cm.CreationTimestamp.Time,
	}, nil
}

// describeSecret 获取 Secret 详情（脱敏处理）
func describeSecret(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	secret, err := clientset.Clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Secret '%s/%s' 详情失败: %w", namespace, name, err)
	}

	// 脱敏处理：只返回 key 名称，不返回实际值
	spec := make(map[string]interface{})
	spec["type"] = string(secret.Type)
	dataKeys := make([]string, 0, len(secret.Data))
	for k := range secret.Data {
		dataKeys = append(dataKeys, k)
	}
	spec["dataKeys"] = dataKeys

	return &ResourceDetail{
		Kind:        "Secret",
		Name:        secret.Name,
		Namespace:   secret.Namespace,
		Labels:      secret.Labels,
		Annotations: secret.Annotations,
		Spec:        spec,
		CreatedAt:   secret.CreationTimestamp.Time,
	}, nil
}

// describeNode 获取 Node 详情
func describeNode(ctx context.Context, clientset *k8s.ClientSet, name string) (*ResourceDetail, error) {
	node, err := clientset.Clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Node '%s' 详情失败: %w", name, err)
	}

	spec, _ := toMap(node.Spec)
	status, _ := toMap(node.Status)

	return &ResourceDetail{
		Kind:        "Node",
		Name:        node.Name,
		Labels:      node.Labels,
		Annotations: node.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   node.CreationTimestamp.Time,
	}, nil
}

// describeNamespace 获取 Namespace 详情
func describeNamespace(ctx context.Context, clientset *k8s.ClientSet, name string) (*ResourceDetail, error) {
	ns, err := clientset.Clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Namespace '%s' 详情失败: %w", name, err)
	}

	spec, _ := toMap(ns.Spec)
	status, _ := toMap(ns.Status)

	return &ResourceDetail{
		Kind:        "Namespace",
		Name:        ns.Name,
		Labels:      ns.Labels,
		Annotations: ns.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   ns.CreationTimestamp.Time,
	}, nil
}

// describeStatefulSet 获取 StatefulSet 详情
func describeStatefulSet(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	sts, err := clientset.Clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 StatefulSet '%s/%s' 详情失败: %w", namespace, name, err)
	}

	spec, _ := toMap(sts.Spec)
	status, _ := toMap(sts.Status)

	return &ResourceDetail{
		Kind:        "StatefulSet",
		Name:        sts.Name,
		Namespace:   sts.Namespace,
		Labels:      sts.Labels,
		Annotations: sts.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   sts.CreationTimestamp.Time,
	}, nil
}

// describeDaemonSet 获取 DaemonSet 详情
func describeDaemonSet(ctx context.Context, clientset *k8s.ClientSet, namespace, name string) (*ResourceDetail, error) {
	ds, err := clientset.Clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 DaemonSet '%s/%s' 详情失败: %w", namespace, name, err)
	}

	spec, _ := toMap(ds.Spec)
	status, _ := toMap(ds.Status)

	return &ResourceDetail{
		Kind:        "DaemonSet",
		Name:        ds.Name,
		Namespace:   ds.Namespace,
		Labels:      ds.Labels,
		Annotations: ds.Annotations,
		Spec:        spec,
		Status:      status,
		CreatedAt:   ds.CreationTimestamp.Time,
	}, nil
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
