package tools

// RegisterInspectTools 注册所有巡检类工具
// 参数:
//   - registry: 工具注册表
//
// 返回:
//   - error: 错误信息
func RegisterInspectTools(registry *ToolRegistry) error {
	// 注册 inspect_cluster_overview 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_cluster_overview",
		Description:          "集群概览巡检，返回节点/Pod/工作负载/事件的数量统计和健康评分，用于快速判断集群整体健康状态。数据量极小，建议作为巡检的第一步调用",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_cluster_overview", "arguments": {}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{},
		},
		Handler: InspectClusterOverview,
	}); err != nil {
		return err
	}

	// 注册 inspect_node_health 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_node_health",
		Description:          "节点健康巡检，返回每个节点的状态、资源分配量、Pod 数量，以及不健康节点的详细原因。适用于 overview 发现节点异常后的深入排查",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_node_health", "arguments": {}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
			},
			Required: []string{},
		},
		Handler: InspectNodeHealth,
	}); err != nil {
		return err
	}

	// 注册 inspect_workload_health 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_workload_health",
		Description:          "工作负载健康巡检，返回副本不一致的 Deployment/StatefulSet/DaemonSet、异常状态的 Pod（Pending/Failed/CrashLoopBackOff 等）、高重启 Pod（重启次数>=10）。支持按 namespace 过滤",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_workload_health", "arguments": {"namespace": "production"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，不指定则检查所有命名空间",
				},
			},
			Required: []string{},
		},
		Handler: InspectWorkloadHealth,
	}); err != nil {
		return err
	}

	// 注册 inspect_event_root_causes 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_event_root_causes",
		Description:          "事件根因巡检，分析 Warning Events 并按 reason+involvedObject 聚合为结构化 findings，识别探针失败、调度失败、挂载失败、镜像拉取失败、HPA 目标缺失等根因。只读诊断工具，适合作为 K8S 深度巡检事件根因分析步骤",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_event_root_causes", "arguments": {"namespace": "default", "limit": 20, "sinceMinutes": 1440}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，不指定则查询所有命名空间",
				},
				"limit": {
					Type:        "integer",
					Description: "返回根因数量上限，默认 100",
					Default:     100,
				},
				"sinceMinutes": {
					Type:        "integer",
					Description: "只分析最近多少分钟内的 Warning Events，默认 1440",
					Default:     1440,
				},
			},
			Required: []string{},
		},
		Handler: InspectEventRootCauses,
	}); err != nil {
		return err
	}

	// 注册 inspect_events 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_events",
		Description:          "事件巡检，返回最近的 Warning 事件，按 Reason+资源 聚合去重，按时间倒序排列。消息自动截断避免过长。支持按 namespace 过滤和数量限制",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_events", "arguments": {"namespace": "default", "limit": 30}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，不指定则查询所有命名空间",
				},
				"limit": {
					Type:        "integer",
					Description: "返回事件数量上限，默认 50",
					Default:     50,
				},
			},
			Required: []string{},
		},
		Handler: InspectEvents,
	}); err != nil {
		return err
	}

	// 注册 inspect_workload_references 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_workload_references",
		Description:          "工作负载引用一致性巡检，检查 StatefulSet serviceName、Service selector/Endpoints、Ingress backend、HPA scaleTargetRef、PDB selector 等跨资源引用是否缺失或不一致。只读诊断工具",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_workload_references", "arguments": {"namespace": "middleware", "includeIngress": true, "includeHPA": true, "includePDB": true}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context":        {Type: "string", Description: "Kubernetes context 名称，不指定则使用当前 context"},
				"namespace":      {Type: "string", Description: "命名空间，不指定则检查所有命名空间"},
				"includeIngress": {Type: "boolean", Description: "是否检查 Ingress backend 引用，默认 true", Default: true},
				"includeHPA":     {Type: "boolean", Description: "是否检查 HPA scaleTargetRef 引用，默认 true", Default: true},
				"includePDB":     {Type: "boolean", Description: "是否检查 PDB selector 引用，默认 true", Default: true},
			},
			Required: []string{},
		},
		Handler: InspectWorkloadReferences,
	}); err != nil {
		return err
	}

	// 注册 inspect_pod_diagnostics 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_pod_diagnostics",
		Description:          "Pod 深度诊断，只读检查 Pending/Failed/CrashLoopBackOff/ImagePullBackOff/OOMKilled、高重启与相关事件；不默认拉全量日志，必要时建议 get_pod_logs。支持 namespace、podName、labelSelector、topN",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_pod_diagnostics", "arguments": {"namespace": "default", "podName": "nginx-abc", "topN": 20}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context":       {Type: "string", Description: "Kubernetes context 名称，不指定则使用当前 context"},
				"namespace":     {Type: "string", Description: "命名空间，建议显式指定"},
				"podName":       {Type: "string", Description: "Pod 名称；不指定则按 labelSelector 或 namespace 扫描"},
				"labelSelector": {Type: "string", Description: "Pod 标签选择器，如 app=api"},
				"topN":          {Type: "integer", Description: "返回 findings 上限，默认 20", Default: 20},
			},
			Required: []string{},
		},
		Handler: InspectPodDiagnostics,
	}); err != nil {
		return err
	}

	// 注册 inspect_service_connectivity 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_service_connectivity",
		Description:          "Service 连通性诊断，只读检查 selector 是否匹配 Pod、Endpoints 是否为空、targetPort 是否匹配容器端口、后端 Pod 是否 Ready。支持 namespace、serviceName、topN",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_service_connectivity", "arguments": {"namespace": "default", "serviceName": "api"}}`,
		InputSchema: &InputSchema{Type: "object", Properties: map[string]*ParameterSchema{
			"context":     {Type: "string", Description: "Kubernetes context 名称，不指定则使用当前 context"},
			"namespace":   {Type: "string", Description: "命名空间，建议显式指定"},
			"serviceName": {Type: "string", Description: "Service 名称；不指定则扫描 namespace 内所有 Service"},
			"topN":        {Type: "integer", Description: "返回 findings 上限，默认 20", Default: 20},
		}, Required: []string{}},
		Handler: InspectServiceConnectivity,
	}); err != nil {
		return err
	}

	// 注册 inspect_storage_diagnostics 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "inspect_storage_diagnostics",
		Description:          "存储诊断，只读检查 PVC Pending、PV Released/Failed、StorageClass 缺失，以及 FailedMount/FailedAttachVolume 事件并关联 Pod/PVC。支持 namespace、topN",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "inspect_storage_diagnostics", "arguments": {"namespace": "default", "topN": 20}}`,
		InputSchema: &InputSchema{Type: "object", Properties: map[string]*ParameterSchema{
			"context":   {Type: "string", Description: "Kubernetes context 名称，不指定则使用当前 context"},
			"namespace": {Type: "string", Description: "命名空间，建议显式指定"},
			"topN":      {Type: "integer", Description: "返回 findings 上限，默认 20", Default: 20},
		}, Required: []string{}},
		Handler: InspectStorageDiagnostics,
	}); err != nil {
		return err
	}
	return nil
}
