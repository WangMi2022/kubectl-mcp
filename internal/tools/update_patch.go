package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// PatchResource 使用 JSON Patch 或 Strategic Merge Patch 修改资源
// 参数:
//   - kind: 资源类型（必填），如 Deployment, Service, Pod 等
//   - name: 资源名称（必填）
//   - namespace: 命名空间（可选，对于 namespace-scoped 资源）
//   - patch: Patch 内容（必填），JSON 格式字符串
//   - patchType: Patch 类型（可选），支持 json, merge, strategic，默认 strategic
//   - context: K8S context 名称（可选）
func PatchResource(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)

	// 获取必填参数
	kind, ok := args["kind"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("参数 'kind' 是必填的")
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	patchContent, ok := args["patch"].(string)
	if !ok || patchContent == "" {
		return nil, fmt.Errorf("参数 'patch' 是必填的")
	}

	// 获取 patch 类型
	patchTypeStr := getStringArg(args, "patchType", "strategic")
	var patchType types.PatchType
	switch patchTypeStr {
	case "json":
		patchType = types.JSONPatchType
	case "merge":
		patchType = types.MergePatchType
	case "strategic":
		patchType = types.StrategicMergePatchType
	default:
		return nil, fmt.Errorf("不支持的 patch 类型: %s，支持的类型: json, merge, strategic", patchTypeStr)
	}

	// 验证 patch 内容是否为有效的 JSON
	var patchData interface{}
	if err := json.Unmarshal([]byte(patchContent), &patchData); err != nil {
		return nil, fmt.Errorf("patch 内容不是有效的 JSON: %w", err)
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
	gvr, err := getGVRForKind(kind)
	if err != nil {
		return nil, err
	}

	// 执行 patch
	var resourceClient dynamic.ResourceInterface
	if namespace != "" {
		resourceClient = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		resourceClient = dynamicClient.Resource(gvr)
	}

	result, err := resourceClient.Patch(ctx, name, patchType, []byte(patchContent), metav1.PatchOptions{})
	if err != nil {
		return nil, fmt.Errorf("patch 资源失败: %w", err)
	}

	message := fmt.Sprintf("%s '%s'", kind, name)
	if namespace != "" {
		message = fmt.Sprintf("%s '%s/%s'", kind, namespace, name)
	}

	return &UpdateResult{
		Kind:      kind,
		Name:      name,
		Namespace: namespace,
		Action:    "Patch",
		Status:    "Success",
		Message:   fmt.Sprintf("%s 已成功 patch (类型: %s)", message, patchTypeStr),
		OldValue:  "",
		NewValue:  patchContent,
		Details:   result.Object,
	}, nil
}

// getGVRForKind 根据 Kind 获取 GroupVersionResource
func getGVRForKind(kind string) (schema.GroupVersionResource, error) {
	// 常见资源的 GVR 映射
	gvrMap := map[string]schema.GroupVersionResource{
		// Core API (v1)
		"Pod": {
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		},
		"Service": {
			Group:    "",
			Version:  "v1",
			Resource: "services",
		},
		"ConfigMap": {
			Group:    "",
			Version:  "v1",
			Resource: "configmaps",
		},
		"Secret": {
			Group:    "",
			Version:  "v1",
			Resource: "secrets",
		},
		"Namespace": {
			Group:    "",
			Version:  "v1",
			Resource: "namespaces",
		},
		"Node": {
			Group:    "",
			Version:  "v1",
			Resource: "nodes",
		},
		"PersistentVolume": {
			Group:    "",
			Version:  "v1",
			Resource: "persistentvolumes",
		},
		"PersistentVolumeClaim": {
			Group:    "",
			Version:  "v1",
			Resource: "persistentvolumeclaims",
		},
		"ServiceAccount": {
			Group:    "",
			Version:  "v1",
			Resource: "serviceaccounts",
		},
		// Apps API (apps/v1)
		"Deployment": {
			Group:    "apps",
			Version:  "v1",
			Resource: "deployments",
		},
		"StatefulSet": {
			Group:    "apps",
			Version:  "v1",
			Resource: "statefulsets",
		},
		"DaemonSet": {
			Group:    "apps",
			Version:  "v1",
			Resource: "daemonsets",
		},
		"ReplicaSet": {
			Group:    "apps",
			Version:  "v1",
			Resource: "replicasets",
		},
		// Batch API (batch/v1)
		"Job": {
			Group:    "batch",
			Version:  "v1",
			Resource: "jobs",
		},
		"CronJob": {
			Group:    "batch",
			Version:  "v1",
			Resource: "cronjobs",
		},
		// Networking API (networking.k8s.io/v1)
		"Ingress": {
			Group:    "networking.k8s.io",
			Version:  "v1",
			Resource: "ingresses",
		},
		// RBAC API (rbac.authorization.k8s.io/v1)
		"Role": {
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "roles",
		},
		"RoleBinding": {
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "rolebindings",
		},
		"ClusterRole": {
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterroles",
		},
		"ClusterRoleBinding": {
			Group:    "rbac.authorization.k8s.io",
			Version:  "v1",
			Resource: "clusterrolebindings",
		},
	}

	gvr, ok := gvrMap[kind]
	if !ok {
		return schema.GroupVersionResource{}, fmt.Errorf("不支持的资源类型: %s", kind)
	}

	return gvr, nil
}
