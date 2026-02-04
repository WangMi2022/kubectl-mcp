package tools

import (
	"context"
	"fmt"
	"strings"

	"kubectl-mcp/internal/k8s"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// CreateFromYAML 通过 YAML 创建资源
// 参数:
//   - yaml: YAML 内容（必填）
//   - namespace: 命名空间（可选，如果 YAML 中未指定则使用此值）
//   - context: K8S context 名称（可选）
func CreateFromYAML(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	yamlContent, ok := args["yaml"].(string)
	if !ok || yamlContent == "" {
		return nil, fmt.Errorf("参数 'yaml' 是必填的")
	}

	// 解析 YAML
	obj := &unstructured.Unstructured{}
	decoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(yamlContent), 4096)
	if err := decoder.Decode(obj); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	// 获取资源信息
	gvk := obj.GroupVersionKind()
	if gvk.Kind == "" {
		return nil, fmt.Errorf("YAML 中缺少 kind 字段")
	}

	// 如果 YAML 中没有指定 namespace，使用参数中的 namespace
	if obj.GetNamespace() == "" && isNamespacedResource(gvk.Kind) {
		obj.SetNamespace(namespace)
	}

	// 获取动态客户端
	var dynamicClient k8s.ClientSet
	if contextName != "" {
		dc, err := k8sClient.GetDynamicClientForContext(contextName)
		if err != nil {
			return nil, fmt.Errorf("获取动态客户端失败: %w", err)
		}
		dynamicClient.DynamicClient = dc
	} else {
		dc, err := k8sClient.GetDynamicClient()
		if err != nil {
			return nil, fmt.Errorf("获取动态客户端失败: %w", err)
		}
		dynamicClient.DynamicClient = dc
	}

	// 构建 GVR
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: getResourceName(gvk.Kind),
	}

	// 创建资源
	var created *unstructured.Unstructured
	var err error

	if isNamespacedResource(gvk.Kind) {
		created, err = dynamicClient.DynamicClient.Resource(gvr).Namespace(obj.GetNamespace()).Create(ctx, obj, metav1.CreateOptions{})
	} else {
		created, err = dynamicClient.DynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	}

	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("%s '%s' 已存在", gvk.Kind, obj.GetName())
		}
		return nil, fmt.Errorf("创建资源失败: %w", err)
	}

	return &CreateResult{
		Kind:      created.GetKind(),
		Name:      created.GetName(),
		Namespace: created.GetNamespace(),
		Status:    "Created",
		Message:   fmt.Sprintf("%s '%s' 创建成功", gvk.Kind, obj.GetName()),
		CreatedAt: created.GetCreationTimestamp().Time,
	}, nil
}

// isNamespacedResource 判断资源是否是命名空间级别的
func isNamespacedResource(kind string) bool {
	clusterScopedResources := map[string]bool{
		"Namespace":                true,
		"Node":                     true,
		"PersistentVolume":         true,
		"ClusterRole":              true,
		"ClusterRoleBinding":       true,
		"StorageClass":             true,
		"PriorityClass":            true,
		"CustomResourceDefinition": true,
	}
	return !clusterScopedResources[kind]
}

// getResourceName 根据 Kind 获取资源名称（复数形式）
func getResourceName(kind string) string {
	resourceMap := map[string]string{
		"Pod":                   "pods",
		"Service":               "services",
		"Deployment":            "deployments",
		"StatefulSet":           "statefulsets",
		"DaemonSet":             "daemonsets",
		"ReplicaSet":            "replicasets",
		"ConfigMap":             "configmaps",
		"Secret":                "secrets",
		"Namespace":             "namespaces",
		"Node":                  "nodes",
		"PersistentVolume":      "persistentvolumes",
		"PersistentVolumeClaim": "persistentvolumeclaims",
		"ServiceAccount":        "serviceaccounts",
		"Role":                  "roles",
		"RoleBinding":           "rolebindings",
		"ClusterRole":           "clusterroles",
		"ClusterRoleBinding":    "clusterrolebindings",
		"Ingress":               "ingresses",
		"NetworkPolicy":         "networkpolicies",
		"Job":                   "jobs",
		"CronJob":               "cronjobs",
	}

	if resource, ok := resourceMap[kind]; ok {
		return resource
	}
	// 默认转换：小写 + s
	return strings.ToLower(kind) + "s"
}
