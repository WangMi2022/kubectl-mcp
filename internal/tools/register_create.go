package tools

// RegisterCreateTools 注册所有创建类工具
// 参数:
//   - registry: 工具注册表
//
// 返回:
//   - error: 错误信息
func RegisterCreateTools(registry *ToolRegistry) error {
	// 注册 create_namespace 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_namespace",
		Description:          "创建 Kubernetes Namespace，用于隔离资源",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		Example:              `{"tool": "create_namespace", "arguments": {"name": "my-namespace", "labels": {"env": "dev"}}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Namespace 名称",
					Required:    true,
				},
				"labels": {
					Type:        "object",
					Description: "标签，格式为 key-value 对象",
				},
				"annotations": {
					Type:        "object",
					Description: "注解，格式为 key-value 对象",
				},
			},
			Required: []string{"name"},
		},
		Handler: CreateNamespace,
	}); err != nil {
		return err
	}

	// 注册 create_pod 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_pod",
		Description:          "创建 Kubernetes Pod，用于运行单个容器实例",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		Example:              `{"tool": "create_pod", "arguments": {"name": "nginx-pod", "image": "nginx:latest", "namespace": "default"}}`,
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
				"image": {
					Type:        "string",
					Description: "容器镜像",
					Required:    true,
				},
				"containerName": {
					Type:        "string",
					Description: "容器名称，默认与 Pod 名称相同",
				},
				"command": {
					Type:        "array",
					Description: "容器启动命令",
					Items:       &ParameterSchema{Type: "string"},
				},
				"args": {
					Type:        "array",
					Description: "容器启动参数",
					Items:       &ParameterSchema{Type: "string"},
				},
				"env": {
					Type:        "object",
					Description: "环境变量，格式为 key-value 对象",
				},
				"labels": {
					Type:        "object",
					Description: "标签，格式为 key-value 对象",
				},
				"restartPolicy": {
					Type:        "string",
					Description: "重启策略: Always, OnFailure, Never",
					Default:     "Always",
					Enum:        []interface{}{"Always", "OnFailure", "Never"},
				},
			},
			Required: []string{"name", "image"},
		},
		Handler: CreatePod,
	}); err != nil {
		return err
	}

	// 注册 create_deployment 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_deployment",
		Description:          "创建 Kubernetes Deployment，用于管理无状态应用的多副本部署",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		Example:              `{"tool": "create_deployment", "arguments": {"name": "nginx", "image": "nginx:latest", "replicas": 3, "namespace": "default"}}`,
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
					Description: "容器镜像",
					Required:    true,
				},
				"replicas": {
					Type:        "integer",
					Description: "副本数，默认为 1",
					Default:     1,
				},
				"containerName": {
					Type:        "string",
					Description: "容器名称，默认与 Deployment 名称相同",
				},
				"containerPort": {
					Type:        "integer",
					Description: "容器端口",
				},
				"env": {
					Type:        "object",
					Description: "环境变量，格式为 key-value 对象",
				},
				"labels": {
					Type:        "object",
					Description: "标签，格式为 key-value 对象",
				},
				"limitsCPU": {
					Type:        "string",
					Description: "CPU 限制，如 '500m'",
				},
				"limitsMemory": {
					Type:        "string",
					Description: "内存限制，如 '256Mi'",
				},
				"requestsCPU": {
					Type:        "string",
					Description: "CPU 请求，如 '100m'",
				},
				"requestsMemory": {
					Type:        "string",
					Description: "内存请求，如 '128Mi'",
				},
			},
			Required: []string{"name", "image"},
		},
		Handler: CreateDeployment,
	}); err != nil {
		return err
	}

	// 注册 create_service 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_service",
		Description:          "创建 Kubernetes Service，用于暴露 Pod 服务",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "medium",
		Example:              `{"tool": "create_service", "arguments": {"name": "nginx-svc", "port": 80, "targetPort": 80, "type": "ClusterIP", "namespace": "default"}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Service 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"port": {
					Type:        "integer",
					Description: "服务端口",
					Required:    true,
				},
				"targetPort": {
					Type:        "integer",
					Description: "目标端口，默认与 port 相同",
				},
				"type": {
					Type:        "string",
					Description: "Service 类型: ClusterIP, NodePort, LoadBalancer",
					Default:     "ClusterIP",
					Enum:        []interface{}{"ClusterIP", "NodePort", "LoadBalancer", "ExternalName"},
				},
				"protocol": {
					Type:        "string",
					Description: "协议: TCP, UDP",
					Default:     "TCP",
					Enum:        []interface{}{"TCP", "UDP"},
				},
				"nodePort": {
					Type:        "integer",
					Description: "NodePort 端口（仅 NodePort/LoadBalancer 类型有效）",
				},
				"selector": {
					Type:        "object",
					Description: "Pod 选择器，默认为 {app: name}",
				},
				"labels": {
					Type:        "object",
					Description: "标签，格式为 key-value 对象",
				},
			},
			Required: []string{"name", "port"},
		},
		Handler: CreateService,
	}); err != nil {
		return err
	}

	// 注册 create_configmap 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_configmap",
		Description:          "创建 Kubernetes ConfigMap，用于存储非敏感配置数据",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "low",
		Example:              `{"tool": "create_configmap", "arguments": {"name": "app-config", "namespace": "default", "data": {"key1": "value1", "key2": "value2"}}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "ConfigMap 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"data": {
					Type:        "object",
					Description: "配置数据，格式为 key-value 对象",
				},
				"labels": {
					Type:        "object",
					Description: "标签，格式为 key-value 对象",
				},
				"annotations": {
					Type:        "object",
					Description: "注解，格式为 key-value 对象",
				},
			},
			Required: []string{"name"},
		},
		Handler: CreateConfigMap,
	}); err != nil {
		return err
	}

	// 注册 create_secret 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_secret",
		Description:          "创建 Kubernetes Secret，用于存储敏感数据（如密码、密钥等）",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "create_secret", "arguments": {"name": "db-secret", "namespace": "default", "stringData": {"username": "admin", "password": "secret123"}}}`,
		InputSchema: &InputSchema{
			Type: "object",
			Properties: map[string]*ParameterSchema{
				"context": {
					Type:        "string",
					Description: "Kubernetes context 名称，不指定则使用当前 context",
				},
				"name": {
					Type:        "string",
					Description: "Secret 名称",
					Required:    true,
				},
				"namespace": {
					Type:        "string",
					Description: "命名空间，默认为 default",
				},
				"type": {
					Type:        "string",
					Description: "Secret 类型，默认为 Opaque",
					Default:     "Opaque",
				},
				"data": {
					Type:        "object",
					Description: "数据（值会自动 base64 编码）",
				},
				"stringData": {
					Type:        "object",
					Description: "字符串数据（不需要 base64 编码）",
				},
				"labels": {
					Type:        "object",
					Description: "标签，格式为 key-value 对象",
				},
				"annotations": {
					Type:        "object",
					Description: "注解，格式为 key-value 对象",
				},
			},
			Required: []string{"name"},
		},
		Handler: CreateSecret,
	}); err != nil {
		return err
	}

	// 注册 create_from_yaml 工具
	if err := registry.RegisterTool(&Tool{
		Name:                 "create_from_yaml",
		Description:          "通过 YAML 内容创建 Kubernetes 资源，支持任意资源类型",
		Category:             CategoryCreate,
		RequiresConfirmation: true,
		RiskLevel:            "high",
		Example:              `{"tool": "create_from_yaml", "arguments": {"yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: test-config\ndata:\n  key: value"}}`,
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
		Handler: CreateFromYAML,
	}); err != nil {
		return err
	}

	return nil
}
