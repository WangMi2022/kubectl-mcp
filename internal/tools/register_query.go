package tools

// RegisterQueryTools 注册所有查询类工具
// 参数:
//   - registry: 工具注册表
//
// 返回:
//   - error: 错误信息
func RegisterQueryTools(registry *ToolRegistry) error {
	// 注册 get_nodes 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_nodes",
		Description:          "查询 Kubernetes 集群的节点列表，返回节点名称、状态、角色、版本、IP 地址等信息",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_nodes", "arguments": {"labelSelector": "node-role.kubernetes.io/master="}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器，格式如 'key=value' 或 'key1=value1,key2=value2'",
				},
			},
			Required: []string{},
		},
		Handler: GetNodes,
	}); err != nil {
		return err
	}

	// 注册 get_namespaces 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_namespaces",
		Description:          "查询 Kubernetes 集群的命名空间列表，返回命名空间名称、状态、标签等信息",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_namespaces", "arguments": {}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器，格式如 'key=value'",
				},
			},
			Required: []string{},
		},
		Handler: GetNamespaces,
	}); err != nil {
		return err
	}

	// 注册 get_pods 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_pods",
		Description:          "查询 Pod 列表，支持按命名空间、名称、标签过滤，返回 Pod 名称、状态、IP、节点、容器信息等。默认搜索所有命名空间，只有在用户明确指定命名空间时才传递namespace参数",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_pods", "arguments": {"labelSelector": "app=nginx"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间名称。留空（不传递此参数）将自动搜索所有命名空间，这是推荐的默认行为。只有当用户明确指定命名空间时才传递此参数",
				},
				"name": {
					Type:        "string",
					Description: "Pod 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器，格式如 'app=nginx' 或 'app=nginx,version=v1'",
				},
			},
			Required: []string{},
		},
		Handler: GetPods,
	}); err != nil {
		return err
	}

	// 注册 get_pod_filter 工具 - 专门用于快速定位单个 Pod
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_pod_filter",
		Description:          "在整个集群的所有命名空间中快速查找 Pod，返回 Pod 的详细信息和所在命名空间。支持精确匹配和模糊匹配两种模式。适用于需要快速定位特定 Pod 位置的场景，如删除操作前的查询",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_pod_filter", "arguments": {"name": "nginx-deployment-7d64c8d-abc123", "matchMode": "exact"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Pod 名称（必填）。根据 matchMode 参数决定匹配方式",
					Required:    true,
				},
				"matchMode": {
					Type:        "string",
					Description: "匹配模式：exact(精确匹配，默认) 或 fuzzy(模糊匹配)。精确匹配用于已知完整Pod名称的场景（如删除操作），模糊匹配用于搜索相关Pod",
					Default:     "exact",
					Enum:        []interface{}{"exact", "fuzzy"},
				},
			},
			Required: []string{"name"},
		},
		Handler: GetPodFilter,
	}); err != nil {
		return err
	}

	// 注册 get_deployments 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_deployments",
		Description:          "查询 Deployment 列表，支持按命名空间、名称、标签过滤，返回副本数、镜像、策略等信息。默认搜索所有命名空间",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_deployments", "arguments": {"labelSelector": "app=nginx"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间名称。留空将搜索所有命名空间（推荐），只有当用户明确指定命名空间时才传递此参数",
				},
				"name": {
					Type:        "string",
					Description: "Deployment 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetDeployments,
	}); err != nil {
		return err
	}

	// 注册 get_statefulsets 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_statefulsets",
		Description:          "查询 StatefulSet 列表，支持按命名空间、名称、标签过滤，返回副本数、镜像、服务名等信息",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_statefulsets", "arguments": {"namespace": "default"}}`,
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
				"name": {
					Type:        "string",
					Description: "StatefulSet 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetStatefulSets,
	}); err != nil {
		return err
	}

	// 注册 get_daemonsets 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_daemonsets",
		Description:          "查询 DaemonSet 列表，支持按命名空间、名称、标签过滤，返回调度数量、镜像、节点选择器等信息",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_daemonsets", "arguments": {"namespace": "kube-system"}}`,
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
				"name": {
					Type:        "string",
					Description: "DaemonSet 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetDaemonSets,
	}); err != nil {
		return err
	}

	// 注册 get_services 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_services",
		Description:          "查询 Service 列表，支持按命名空间、名称、标签过滤，返回类型、ClusterIP、端口等信息。默认搜索所有命名空间",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_services", "arguments": {"labelSelector": "app=nginx"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间名称。留空将搜索所有命名空间（推荐），只有当用户明确指定命名空间时才传递此参数",
				},
				"name": {
					Type:        "string",
					Description: "Service 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetServices,
	}); err != nil {
		return err
	}

	// 注册 get_configmaps 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_configmaps",
		Description:          "查询 ConfigMap 列表，支持按命名空间、名称、标签过滤，返回名称、数据键列表等信息",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_configmaps", "arguments": {"namespace": "default"}}`,
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
				"name": {
					Type:        "string",
					Description: "ConfigMap 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetConfigMaps,
	}); err != nil {
		return err
	}

	// 注册 get_secrets 工具（脱敏处理）
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_secrets",
		Description:          "查询 Secret 列表（脱敏处理，只返回键名不返回值），支持按命名空间、名称、标签过滤",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_secrets", "arguments": {"namespace": "default"}}`,
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
				"name": {
					Type:        "string",
					Description: "Secret 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetSecrets,
	}); err != nil {
		return err
	}

	// 注册 describe_resource 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "describe_resource",
		Description:          "获取 Kubernetes 资源的详细信息，支持 Pod、Deployment、Service、ConfigMap、Secret、Node、Namespace、StatefulSet、DaemonSet",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "describe_resource", "arguments": {"kind": "pod", "name": "nginx-xxx", "namespace": "default"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"kind": {
					Type:        "string",
					Description: "资源类型，支持: pod, deployment, service, configmap, secret, node, namespace, statefulset, daemonset",
					Required:    true,
				},
				"name": {
					Type:        "string",
					Description: "资源名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间（Node 和 Namespace 类型不需要）",
				},
			},
			Required: []string{"kind", "name"},
		},
		Handler: DescribeResource,
	}); err != nil {
		return err
	}

	// 注册 get_pod_logs 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_pod_logs",
		Description:          "获取 Pod 的日志，支持指定容器、行数限制、查看前一个容器的日志",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_pod_logs", "arguments": {"name": "nginx-xxx", "namespace": "default", "tailLines": 100}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Pod 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"container": {
					Type:        "string",
					Description: "容器名称，多容器 Pod 时需要指定",
				},
				"tailLines": {
					Type:        "integer",
					Description: "返回的日志行数，默认 100",
					Default:     100,
				},
				"previous": {
					Type:        "boolean",
					Description: "是否查看前一个容器的日志（容器重启后）",
					Default:     false,
				},
			},
			Required: []string{"name"},
		},
		Handler: GetPodLogs,
	}); err != nil {
		return err
	}

	// 注册 get_events 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_events",
		Description:          "查询 Kubernetes 事件列表，支持按命名空间、资源类型和名称过滤",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_events", "arguments": {"namespace": "default", "involvedObjectKind": "Pod", "involvedObjectName": "nginx-xxx"}}`,
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
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
				"involvedObjectKind": {
					Type:        "string",
					Description: "关联对象类型，如 Pod、Deployment、Node 等",
				},
				"involvedObjectName": {
					Type:        "string",
					Description: "关联对象名称，需要与 involvedObjectKind 一起使用",
				},
			},
			Required: []string{},
		},
		Handler: GetEvents,
	}); err != nil {
		return err
	}

	// ========== 反向查找工具 ==========

	// 注册 get_ingresses 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "get_ingresses",
		Description:          "查询 Ingress 列表，返回域名、路径、后端 Service 等信息",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "get_ingresses", "arguments": {"namespace": "default"}}`,
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
				"name": {
					Type:        "string",
					Description: "Ingress 名称，用于精确匹配",
				},
				"labelSelector": {
					Type:        "string",
					Description: "标签选择器",
				},
			},
			Required: []string{},
		},
		Handler: GetIngresses,
	}); err != nil {
		return err
	}

	// 注册 find_service_by_nodeport 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "find_service_by_nodeport",
		Description:          "通过 NodePort 端口号反向查找对应的 Service，用于快速定位使用特定端口的服务",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "find_service_by_nodeport", "arguments": {"nodePort": 30080, "includeEndpoints": true}}`,
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
				"nodePort": {
					Type:        "integer",
					Description: "NodePort 端口号（默认范围 30000-32767，支持自定义范围）",
					Required:    true,
				},
				"includeEndpoints": {
					Type:        "boolean",
					Description: "是否包含 Endpoints 信息（Pod IP 和端口），默认 false",
					Default:     false,
				},
			},
			Required: []string{"nodePort"},
		},
		Handler: FindServiceByNodePort,
	}); err != nil {
		return err
	}

	// 注册 find_service_by_ingress 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "find_service_by_ingress",
		Description:          "通过 Ingress 域名反向查找对应的 Service，支持多种域名匹配模式和路径过滤",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "find_service_by_ingress", "arguments": {"host": "api.example.com", "path": "/v1", "matchMode": "smart"}}`,
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
				"host": {
					Type:        "string",
					Description: "Ingress 域名",
					Required:    true,
				},
				"path": {
					Type:        "string",
					Description: "路径前缀过滤（可选），如 '/api' 或 '/v1'",
				},
				"matchMode": {
					Type:        "string",
					Description: "域名匹配模式: exact(精确), prefix(前缀), suffix(后缀), contains(包含), wildcard(通配符), smart(智能，默认)",
					Default:     "smart",
					Enum:        []interface{}{"exact", "prefix", "suffix", "contains", "wildcard", "smart"},
				},
				"includeEndpoints": {
					Type:        "boolean",
					Description: "是否包含 Endpoints 信息（Pod IP 和端口），默认 false",
					Default:     false,
				},
			},
			Required: []string{"host"},
		},
		Handler: FindServiceByIngress,
	}); err != nil {
		return err
	}

	// 注册 find_workload_by_service 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "find_workload_by_service",
		Description:          "通过 Service 名称反向查找对应的工作负载（Deployment、StatefulSet、DaemonSet），基于 Service selector 匹配",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "find_workload_by_service", "arguments": {"serviceName": "nginx-service", "namespace": "default", "includeEndpoints": true}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"serviceName": {
					Type:        "string",
					Description: "Service 名称",
					Required:    true,
				},
				"includeEndpoints": {
					Type:        "boolean",
					Description: "是否包含 Endpoints 信息（Pod IP 和端口），默认 false",
					Default:     false,
				},
			},
			Required: []string{"serviceName"},
		},
		Handler: FindWorkloadByService,
	}); err != nil {
		return err
	}

	// 注册 trace_by_nodeport 工具 - 完整链路追踪
	if err := registry.RegisterTool(&Tool{
		Name:                 "trace_by_nodeport",
		Description:          "通过 NodePort 端口号追踪完整链路，一次调用返回 NodePort → Service → Workload 的完整关系链",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "trace_by_nodeport", "arguments": {"nodePort": 30080}}`,
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
				"nodePort": {
					Type:        "integer",
					Description: "NodePort 端口号",
					Required:    true,
				},
			},
			Required: []string{"nodePort"},
		},
		Handler: TraceByNodePort,
	}); err != nil {
		return err
	}

	// 注册 trace_by_host 工具 - 完整链路追踪
	if err := registry.RegisterTool(&Tool{
		Name:                 "trace_by_host",
		Description:          "通过域名追踪完整链路，一次调用返回 Host → Ingress → Service → Workload 的完整关系链",
		Category:             CategoryQuery,
		RequiresConfirmation: false,
		RiskLevel:            "low",
		Example:              `{"tool": "trace_by_host", "arguments": {"host": "api.example.com", "path": "/v1"}}`,
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
				"host": {
					Type:        "string",
					Description: "域名",
					Required:    true,
				},
				"path": {
					Type:        "string",
					Description: "路径前缀过滤（可选）",
				},
				"matchMode": {
					Type:        "string",
					Description: "域名匹配模式: exact, prefix, suffix, contains, wildcard, smart(默认)",
					Default:     "smart",
					Enum:        []interface{}{"exact", "prefix", "suffix", "contains", "wildcard", "smart"},
				},
			},
			Required: []string{"host"},
		},
		Handler: TraceByHost,
	}); err != nil {
		return err
	}

	return nil
}
