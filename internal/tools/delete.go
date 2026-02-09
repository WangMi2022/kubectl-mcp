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
		Description:          "【危险操作】删除指定的 Pod。⚠️ 强制要求：在执行删除之前，必须先调用 preview_delete_resources 工具（kind=Pod）进行预检查，获取风险评估，并向用户展示预检查结果，等待用户明确确认后才能执行删除。支持强制删除和自定义优雅删除时间。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌。必须先调用 preview_delete_resources 获取此令牌",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace", "confirmationToken"},
		},
		Handler: DeletePod,
		Example: `{"name": "nginx-pod", "namespace": "production", "force": false, "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_deployment 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_deployment",
		Description:          "【危险操作】删除指定的 Deployment。⚠️ 强制要求：在执行删除之前，必须先调用 preview_delete_resources 工具（kind=Deployment）进行预检查，获取风险评估和关联资源（Pod、ReplicaSet）影响分析，并向用户展示预检查结果，等待用户明确确认后才能执行删除。可选择是否级联删除关联的 ReplicaSet 和 Pod。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌。必须先调用 preview_delete_resources 获取此令牌",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace", "confirmationToken"},
		},
		Handler: DeleteDeployment,
		Example: `{"name": "nginx-deployment", "namespace": "production", "cascade": true, "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_service 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_service",
		Description:          "【危险操作】删除指定的 Service。⚠️ 强制要求：在执行删除之前，必须先调用 preview_delete_resources 工具（kind=Service）进行预检查，获取风险评估和关联资源（Ingress、Endpoint）影响分析，并向用户展示预检查结果，等待用户明确确认后才能执行删除。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌。必须先调用 preview_delete_resources 获取此令牌",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace", "confirmationToken"},
		},
		Handler: DeleteService,
		Example: `{"name": "nginx-service", "namespace": "production", "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_configmap 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_configmap",
		Description:          "【危险操作】删除指定的 ConfigMap。⚠️ 强制要求：在执行删除之前，必须先调用 preview_delete_resources 工具（kind=ConfigMap）进行预检查，获取风险评估和关联资源（使用该 ConfigMap 的 Pod）影响分析，并向用户展示预检查结果，等待用户明确确认后才能执行删除。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌。必须先调用 preview_delete_resources 获取此令牌",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace", "confirmationToken"},
		},
		Handler: DeleteConfigMap,
		Example: `{"name": "app-config", "namespace": "production", "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_secret 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_secret",
		Description:          "【危险操作】删除指定的 Secret。⚠️ 强制要求：在执行删除之前，必须先调用 preview_delete_resources 工具（kind=Secret）进行预检查，获取风险评估和关联资源（使用该 Secret 的 Pod）影响分析，并向用户展示预检查结果，等待用户明确确认后才能执行删除。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌。必须先调用 preview_delete_resources 获取此令牌",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "namespace", "confirmationToken"},
		},
		Handler: DeleteSecret,
		Example: `{"name": "db-secret", "namespace": "production", "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_namespace 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_namespace",
		Description:          "【极度危险操作】删除指定的 Namespace。⚠️ 强制要求：这是最高风险的操作，会删除 namespace 下的所有资源！在执行删除之前，必须先调用 preview_delete_resources 工具进行预检查，获取该命名空间下所有资源的详细列表和风险评估，并向用户展示完整的预检查结果，等待用户明确确认后才能执行删除。系统 namespace（default、kube-system、kube-public、kube-node-lease）不允许删除。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌。必须先调用 preview_delete_resources 获取此令牌",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"name", "confirmationToken"},
		},
		Handler: DeleteNamespace,
		Example: `{"name": "test-namespace", "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 delete_resources 批量删除工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "delete_resources",
		Description:          "【危险操作】批量删除指定类型的资源。⚠️ 强制要求：在调用此工具之前，必须先调用 preview_delete_resources 工具进行预检查，获取风险评估和关联资源影响分析，并向用户展示预检查结果，等待用户明确确认后才能执行删除。禁止跳过预检查直接删除！支持的资源类型：Pod、Deployment、StatefulSet、DaemonSet、Service、ConfigMap、Secret。",
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
				"confirmationToken": {
					Type:        "string",
					Description: "预检查确认令牌（必填）。必须先调用 preview_delete_resources 获取此令牌，证明已完成预检查",
					Required:    true,
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"kind", "namespace", "confirmationToken"},
		},
		Handler: DeleteResources,
		Example: `{"kind": "Pod", "namespace": "production", "labelSelector": "app=nginx", "confirmationToken": "1234567890"}`,
	}); err != nil {
		return err
	}

	// 注册 preview_delete_resources 预检查工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "preview_delete_resources",
		Description:          "预检查删除资源（不执行真正的删除）。分析将要删除的资源详细信息、风险等级和关联资源影响。支持的资源类型：Pod、Deployment、StatefulSet、DaemonSet、Service、ConfigMap、Secret。",
		Category:             CategoryDelete,
		RequiresConfirmation: false,
		RiskLevel:            "low",
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
					Description: "命名空间（必填）",
					Required:    true,
				},
				"names": {
					Type:        "array",
					Description: "要检查的资源名称列表",
					Items: &ParameterSchema{
						Type: "string",
					},
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器，用于筛选要检查的资源，如 app=nginx",
				},
				"context": {
					Type:        "string",
					Description: "K8S context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{"kind", "namespace"},
		},
		Handler: PreviewDeleteResources,
		Example: `{"kind": "Deployment", "namespace": "production", "names": ["nginx-deployment"]}`,
	}); err != nil {
		return err
	}

	return nil
}
