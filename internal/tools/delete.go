package tools

// RegisterDeleteTools 注册所有删除类工具
// 参数:
//   - registry: 工具注册表
//
// 返回:
//   - error: 错误信息
func RegisterDeleteTools(registry *ToolRegistry) error {
	// 注册 delete_pod 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_pod",
		Description:          "删除指定的 Pod。支持强制删除和自定义优雅删除时间。重要：必须先使用 get_pods 查询确定 Pod 所在的命名空间，然后再执行删除操作。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Pod 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（必填）。必须先通过 get_pods 查询确定 Pod 所在的命名空间，不能猜测或使用默认值",
					Required:    true,
				},
				"force": {
					Type:        "boolean",
					Description: "是否强制删除（grace period 为 0）",
					Default:     false,
				},
				"gracePeriod": {
					Type:        "integer",
					Description: "优雅删除时间（秒），默认 30 秒",
					Default:     30,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Handler: DeletePod,
		Example: `{"name": "nginx-pod", "namespace": "production", "force": false}`,
	}); err != nil {
		return err
	}

	// 注册 delete_deployment 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_deployment",
		Description:          "删除指定的 Deployment。可选择是否级联删除关联的 ReplicaSet 和 Pod。重要：必须先使用 get_deployments 查询确定 Deployment 所在的命名空间，然后再执行删除操作。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Deployment 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（必填）。必须先通过 get_deployments 查询确定 Deployment 所在的命名空间，不能猜测或使用默认值",
					Required:    true,
				},
				"cascade": {
					Type:        "boolean",
					Description: "是否级联删除关联的 ReplicaSet 和 Pod，默认为 true",
					Default:     true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Handler: DeleteDeployment,
		Example: `{"name": "nginx-deployment", "namespace": "production", "cascade": true}`,
	}); err != nil {
		return err
	}

	// 注册 delete_service 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_service",
		Description:          "删除指定的 Service。重要：必须先使用 get_services 查询确定 Service 所在的命名空间，然后再执行删除操作。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Service 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（必填）。必须先通过 get_services 查询确定 Service 所在的命名空间，不能猜测或使用默认值",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Handler: DeleteService,
		Example: `{"name": "nginx-service", "namespace": "production"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_configmap 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_configmap",
		Description:          "删除指定的 ConfigMap。重要：必须先使用 get_configmaps 查询确定 ConfigMap 所在的命名空间，然后再执行删除操作。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"name": {
					Type:        "string",
					Description: "ConfigMap 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（必填）。必须先通过 get_configmaps 查询确定 ConfigMap 所在的命名空间，不能猜测或使用默认值",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Handler: DeleteConfigMap,
		Example: `{"name": "app-config", "namespace": "production"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_secret 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_secret",
		Description:          "删除指定的 Secret。重要：必须先使用 get_secrets 查询确定 Secret 所在的命名空间，然后再执行删除操作。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Secret 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（必填）。必须先通过 get_secrets 查询确定 Secret 所在的命名空间，不能猜测或使用默认值",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace"},
		},
		Handler: DeleteSecret,
		Example: `{"name": "db-secret", "namespace": "production"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_namespace 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_namespace",
		Description:          "删除指定的 Namespace。警告：这是一个高危操作，会删除 namespace 下的所有资源。系统 namespace（default、kube-system、kube-public、kube-node-lease）不允许删除。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "critical",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Namespace 名称",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name"},
		},
		Handler: DeleteNamespace,
		Example: `{"name": "test-namespace"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_resources 批量删除工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_resources",
		Description:          "批量删除指定类型的资源。可以通过资源名称列表或标签选择器来筛选要删除的资源。支持的资源类型：Pod、Deployment、StatefulSet、DaemonSet、Service、ConfigMap、Secret。重要：必须先查询确定资源所在的命名空间，然后再执行删除操作。",
		Category:             CategoryDelete,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"kind": {
					Type:        "string",
					Description: "资源类型，如 Pod、Deployment、Service 等",
					Required:    true,
					Enum:        []interface{}{"Pod", "Deployment", "StatefulSet", "DaemonSet", "Service", "ConfigMap", "Secret"},
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（必填）。必须先查询确定资源所在的命名空间，不能猜测或使用默认值",
					Required:    true,
				},
				"names": {
					Type:        "array",
					Description: "要删除的资源名称列表",
					Items: &ParameterSchema{
						Type: "string",
					},
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器，用于筛选要删除的资源，如 app=nginx",
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"kind", "namespace"},
		},
		Handler: DeleteResources,
		Example: `{"kind": "Pod", "namespace": "production", "labelSelector": "app=nginx"}`,
	}); err != nil {
		return err
	}

	return nil
}
