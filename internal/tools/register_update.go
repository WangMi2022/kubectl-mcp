package tools

// RegisterUpdateTools 注册所有修改类工具
// 参数:
//   - registry: 工具注册表
//
// 返回:
//   - error: 错误信息
func RegisterUpdateTools(registry *ToolRegistry) error {
	// 注册 scale_deployment 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "scale_deployment",
		Description:          "扩缩容 Kubernetes Deployment，调整副本数量",
		Category:             CategoryUpdate,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		Example:              `{"tool": "scale_deployment", "arguments": {"name": "nginx", "namespace": "default", "replicas": 5}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Deployment 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"replicas": {
					Type:        "integer",
					Description: "目标副本数（必须大于等于 0）",
					Required:    true,
				},
			},
			Required: []string{"name", "replicas"},
		},
		Handler: ScaleDeployment,
	}); err != nil {
		return err
	}

	// 注册 scale_statefulset 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "scale_statefulset",
		Description:          "扩缩容 Kubernetes StatefulSet，调整副本数量",
		Category:             CategoryUpdate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "scale_statefulset", "arguments": {"name": "mysql", "namespace": "default", "replicas": 3}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "StatefulSet 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"replicas": {
					Type:        "integer",
					Description: "目标副本数（必须大于等于 0）",
					Required:    true,
				},
			},
			Required: []string{"name", "replicas"},
		},
		Handler: ScaleStatefulSet,
	}); err != nil {
		return err
	}

	// 注册 update_deployment_image 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "update_deployment_image",
		Description:          "更新 Kubernetes Deployment 的容器镜像，触发滚动更新",
		Category:             CategoryUpdate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "update_deployment_image", "arguments": {"name": "nginx", "namespace": "default", "image": "nginx:1.21"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Deployment 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"image": {
					Type:        "string",
					Description: "新的容器镜像（格式: image:tag）",
					Required:    true,
				},
				"containerName": {
					Type:        "string",
					Description: "容器名称（可选，不指定则更新第一个容器）",
				},
			},
			Required: []string{"name", "image"},
		},
		Handler: UpdateDeploymentImage,
	}); err != nil {
		return err
	}

	// 注册 restart_deployment 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "restart_deployment",
		Description:          "重启 Kubernetes Deployment，通过滚动更新重启所有 Pod",
		Category:             CategoryUpdate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "restart_deployment", "arguments": {"name": "nginx", "namespace": "default"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Deployment 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
			},
			Required: []string{"name"},
		},
		Handler: RestartDeployment,
	}); err != nil {
		return err
	}

	// 注册 apply_yaml 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "apply_yaml",
		Description:          "通过 YAML 应用 Kubernetes 资源（类似 kubectl apply），如果资源存在则更新，不存在则创建",
		Category:             CategoryUpdate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "apply_yaml", "arguments": {"yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test-config\ndata:\n  key: value"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"yaml": {
					Type:        "string",
					Description: "YAML 格式的资源定义",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（如果 YAML 中未指定则使用此值）",
				},
			},
			Required: []string{"yaml"},
		},
		Handler: ApplyYAML,
	}); err != nil {
		return err
	}

	// 注册 patch_resource 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "patch_resource",
		Description:          "使用 JSON Patch 修改 Kubernetes 资源，支持精细化的资源修改",
		Category:             CategoryUpdate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "patch_resource", "arguments": {"kind": "Deployment", "name": "nginx", "namespace": "default", "patch": "{\"spec\":{\"replicas\":3}}", "patchType": "strategic"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"kind": {
					Type:        "string",
					Description: "资源类型（如 Deployment, Service, Pod 等）",
					Required:    true,
				},
				"name": {
					Type:        "string",
					Description: "资源名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（对于 namespace-scoped 资源）",
				},
				"patch": {
					Type:        "string",
					Description: "Patch 内容（JSON 格式字符串）",
					Required:    true,
				},
				"patchType": {
					Type:        "string",
					Description: "Patch 类型: json, merge, strategic（默认 strategic）",
					Default:     "strategic",
					Enum:        []interface{}{"json", "merge", "strategic"},
				},
			},
			Required: []string{"kind", "name", "patch"},
		},
		Handler: PatchResource,
	}); err != nil {
		return err
	}

	return nil
}
