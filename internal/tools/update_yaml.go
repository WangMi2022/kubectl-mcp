package tools

import (
	"bytes"
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
)

// ApplyYAML 通过 YAML 应用资源（类似 kubectl apply）
// 如果资源存在则更新，不存在则创建
// 参数:
//   - yaml: YAML 格式的资源定义（必填）
//   - namespace: 命名空间（可选，如果 YAML 中未指定则使用此值）
//   - context: K8S context 名称（可选）
func ApplyYAML(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)

	// 获取必填参数
	yamlContent, ok := args["yaml"].(string)
	if !ok || yamlContent == "" {
		return nil, fmt.Errorf("参数 'yaml' 是必填的")
	}

	// 解析 YAML
	obj := &unstructured.Unstructured{}
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader([]byte(yamlContent)), 4096)
	if err := decoder.Decode(obj); err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %w", err)
	}

	// 获取资源信息
	gvk := obj.GroupVersionKind()
	kind := gvk.Kind
	name := obj.GetName()
	objNamespace := obj.GetNamespace()

	// 如果 YAML 中没有指定 namespace，使用参数中的 namespace
	if objNamespace == "" && namespace != "" {
		obj.SetNamespace(namespace)
		objNamespace = namespace
	}

	if name == "" {
		return nil, fmt.Errorf("YAML 中缺少资源名称")
	}

	// 获取 dynamic client
	var dynamicClient dynamic.Interface
	if contextName != "" {
		dc, err := k8sClient.GetDynamicClientForContext(contextName)
		if err != nil {
			return nil, fmt.Errorf("获取 context '%s' 的动态客户端失败: %w", contextName, err)
		}
		dynamicClient = dc
	} else {
		dc, err := k8sClient.GetDynamicClient()
		if err != nil {
			return nil, fmt.Errorf("获取动态客户端失败: %w", err)
		}
		dynamicClient = dc
	}

	// 构建 GVR (GroupVersionResource)
	gvr := schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: getResourceName(kind),
	}

	// 尝试获取现有资源
	var resourceClient dynamic.ResourceInterface
	if objNamespace != "" {
		resourceClient = dynamicClient.Resource(gvr).Namespace(objNamespace)
	} else {
		resourceClient = dynamicClient.Resource(gvr)
	}

	existing, err := resourceClient.Get(ctx, name, metav1.GetOptions{})
	action := "Created"

	if err == nil {
		// 资源存在，执行更新
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = resourceClient.Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			return nil, fmt.Errorf("更新资源失败: %w", err)
		}
		action = "Updated"
	} else {
		// 资源不存在，执行创建
		_, err = resourceClient.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return nil, fmt.Errorf("创建资源失败: %w", err)
		}
		action = "Created"
	}

	message := fmt.Sprintf("%s '%s'", kind, name)
	if objNamespace != "" {
		message = fmt.Sprintf("%s '%s/%s'", kind, objNamespace, name)
	}

	return &UpdateResult{
		Kind:      kind,
		Name:      name,
		Namespace: objNamespace,
		Action:    action,
		Status:    "Success",
		Message:   fmt.Sprintf("%s %s", message, action),
		OldValue:  "",
		NewValue:  "",
	}, nil
}
