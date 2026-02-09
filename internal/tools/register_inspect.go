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

	return nil
}
