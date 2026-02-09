package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// ========== 反向查找结果类型 ==========

// IngressInfo Ingress 信息
type IngressInfo struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Hosts       []string          `json:"hosts"`
	Paths       []IngressPathInfo `json:"paths"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	CreatedAt   string            `json:"createdAt,omitempty"`
}

// IngressPathInfo Ingress 路径信息
type IngressPathInfo struct {
	Host        string `json:"host"`
	Path        string `json:"path"`
	PathType    string `json:"pathType"`
	ServiceName string `json:"serviceName"`
	ServicePort string `json:"servicePort"`
}

// EndpointInfo Endpoint 信息
type EndpointInfo struct {
	PodName   string `json:"podName,omitempty"`
	IP        string `json:"ip"`
	Port      int32  `json:"port"`
	Protocol  string `json:"protocol"`
	Ready     bool   `json:"ready"`
	NodeName  string `json:"nodeName,omitempty"`
	TargetRef string `json:"targetRef,omitempty"`
}

// ServiceInfoWithEndpoints 带 Endpoints 的 Service 信息
type ServiceInfoWithEndpoints struct {
	ServiceInfo
	Endpoints []EndpointInfo `json:"endpoints,omitempty"`
}

// ReverseServiceResult 反向查找 Service 结果
type ReverseServiceResult struct {
	Query    string        `json:"query"`
	Type     string        `json:"type"` // nodeport 或 ingress
	Services []ServiceInfo `json:"services"`
	Message  string        `json:"message"`
}

// ReverseServiceResultWithEndpoints 带 Endpoints 的反向查找结果
type ReverseServiceResultWithEndpoints struct {
	Query    string                     `json:"query"`
	Type     string                     `json:"type"`
	Services []ServiceInfoWithEndpoints `json:"services"`
	Message  string                     `json:"message"`
	Warning  string                     `json:"warning,omitempty"`
}

// WorkloadInfo 工作负载信息（通用）
type WorkloadInfo struct {
	Kind          string            `json:"kind"` // Deployment, StatefulSet, DaemonSet, ReplicaSet
	Name          string            `json:"name"`
	Namespace     string            `json:"namespace"`
	Replicas      int32             `json:"replicas,omitempty"`
	ReadyReplicas int32             `json:"readyReplicas,omitempty"`
	Images        []string          `json:"images"`
	Labels        map[string]string `json:"labels,omitempty"`
	Selector      map[string]string `json:"selector,omitempty"`
	CreatedAt     string            `json:"createdAt,omitempty"`
}

// ReverseWorkloadResult 反向查找工作负载结果
type ReverseWorkloadResult struct {
	ServiceName      string            `json:"serviceName"`
	ServiceNamespace string            `json:"serviceNamespace"`
	ServiceSelector  map[string]string `json:"serviceSelector"`
	Workloads        []WorkloadInfo    `json:"workloads"`
	Endpoints        []EndpointInfo    `json:"endpoints,omitempty"`
	Message          string            `json:"message"`
}

// FullTraceResult 完整链路追踪结果
type FullTraceResult struct {
	Query      string                     `json:"query"`
	QueryType  string                     `json:"queryType"` // nodeport 或 host
	Ingresses  []IngressInfo              `json:"ingresses,omitempty"`
	Services   []ServiceInfoWithEndpoints `json:"services"`
	Workloads  []WorkloadInfo             `json:"workloads"`
	Message    string                     `json:"message"`
	Warning    string                     `json:"warning,omitempty"`
	TraceChain []string                   `json:"traceChain"`
}

// ========== 反向查找工具实现 ==========

// FindServiceByNodePort 通过 NodePort 端口号查找 Service
func FindServiceByNodePort(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	verbose := isVerbose(args)

	// 显式获取 namespace 参数，如果没有传则查询所有 namespace
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// 获取 nodePort 参数
	var nodePort int32
	switch v := args["nodePort"].(type) {
	case float64:
		nodePort = int32(v)
	case int:
		nodePort = int32(v)
	case int32:
		nodePort = v
	default:
		return nil, fmt.Errorf("nodePort 参数必须是数字类型")
	}

	// 是否包含 Endpoints
	includeEndpoints := getBoolArg(args, "includeEndpoints", false)

	// 放宽端口范围校验，改为警告
	var warning string
	if nodePort < 1 || nodePort > 65535 {
		return nil, fmt.Errorf("端口号必须在 1-65535 范围内，当前值: %d", nodePort)
	}
	if nodePort < 30000 || nodePort > 32767 {
		warning = fmt.Sprintf("NodePort %d 不在默认范围 30000-32767 内，可能是自定义配置", nodePort)
	}

	// 尝试使用索引器（O(1) 查询）
	indexer, indexerErr := k8sClient.GetServiceIndexerForContext(contextName)

	var services []*corev1.Service
	if indexerErr == nil && indexer.IsReady() {
		// 使用索引器快速查询
		if namespace != "" {
			services = indexer.FindByNodePortInNamespace(nodePort, namespace)
		} else {
			services = indexer.FindByNodePort(nodePort)
		}
	} else {
		// 回退到传统 API 查询
		clientset, err := getClientSet(contextName, k8sClient)
		if err != nil {
			return nil, err
		}

		queryNs := namespace
		if queryNs == "" {
			queryNs = metav1.NamespaceAll
		}

		svcList, err := clientset.Clientset.CoreV1().Services(queryNs).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Service 列表失败: %w", err)
		}

		// 过滤匹配的 Service
		for i := range svcList.Items {
			svc := &svcList.Items[i]
			for _, port := range svc.Spec.Ports {
				if port.NodePort == nodePort {
					services = append(services, svc)
					break
				}
			}
		}
	}

	// 构建结果
	clientset, _ := getClientSet(contextName, k8sClient)
	matchedServices := make([]ServiceInfoWithEndpoints, 0, len(services))

	for _, svc := range services {
		ports := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, ServicePortInfo{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.String(),
				NodePort:   p.NodePort,
				Protocol:   string(p.Protocol),
			})
		}

		externalIP := getExternalIP(svc)

		svcInfo := ServiceInfoWithEndpoints{
			ServiceInfo: ServiceInfo{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Type:      string(svc.Spec.Type),
				ClusterIP: svc.Spec.ClusterIP,
				Ports:     ports,
			},
		}

		if verbose {
			svcInfo.ExternalIP = externalIP
			svcInfo.Labels = svc.Labels
			svcInfo.Selector = svc.Spec.Selector
			svcInfo.CreatedAt = svc.CreationTimestamp.Time
		}

		// 获取 Endpoints
		if includeEndpoints && clientset != nil {
			svcInfo.Endpoints = getServiceEndpoints(ctx, clientset, svc.Namespace, svc.Name)
		}

		matchedServices = append(matchedServices, svcInfo)
	}

	message := fmt.Sprintf("找到 %d 个使用 NodePort %d 的 Service", len(matchedServices), nodePort)
	if len(matchedServices) == 0 {
		message = fmt.Sprintf("未找到使用 NodePort %d 的 Service", nodePort)
	}

	// 添加索引器状态信息
	queryMethod := "indexed"
	if indexerErr != nil || (indexer != nil && !indexer.IsReady()) {
		queryMethod = "api"
	}

	return &ReverseServiceResultWithEndpoints{
		Query:    fmt.Sprintf("%d (via %s)", nodePort, queryMethod),
		Type:     "nodeport",
		Services: matchedServices,
		Message:  message,
		Warning:  warning,
	}, nil
}

// GetIngresses 查询 Ingress 列表
func GetIngresses(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	nameFilter := ""
	if name, ok := args["name"].(string); ok {
		nameFilter = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	if nameFilter != "" {
		listOptions.FieldSelector = fmt.Sprintf("metadata.name=%s", nameFilter)
	}

	queryNs := namespace
	if queryNs == "" {
		queryNs = metav1.NamespaceAll
	}

	ingresses, err := clientset.Clientset.NetworkingV1().Ingresses(queryNs).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 Ingress 列表失败: %w", err)
	}

	result := make([]IngressInfo, 0, len(ingresses.Items))
	for _, ing := range ingresses.Items {
		hosts := make([]string, 0)
		paths := make([]IngressPathInfo, 0)

		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}
			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					pathType := "Prefix"
					if path.PathType != nil {
						pathType = string(*path.PathType)
					}
					servicePort := ""
					if path.Backend.Service != nil {
						if path.Backend.Service.Port.Name != "" {
							servicePort = path.Backend.Service.Port.Name
						} else {
							servicePort = fmt.Sprintf("%d", path.Backend.Service.Port.Number)
						}
					}
					paths = append(paths, IngressPathInfo{
						Host:        rule.Host,
						Path:        path.Path,
						PathType:    pathType,
						ServiceName: path.Backend.Service.Name,
						ServicePort: servicePort,
					})
				}
			}
		}

		ingInfo := IngressInfo{
			Name:      ing.Name,
			Namespace: ing.Namespace,
			Hosts:     hosts,
			Paths:     paths,
		}

		if verbose {
			ingInfo.Labels = ing.Labels
			ingInfo.Annotations = ing.Annotations
			ingInfo.CreatedAt = ing.CreationTimestamp.Format("2006-01-02T15:04:05Z")
		}

		result = append(result, ingInfo)
	}

	return result, nil
}

// FindServiceByIngress 通过 Ingress 域名查找 Service
func FindServiceByIngress(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	verbose := isVerbose(args)

	// 获取 host 参数
	host, ok := args["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("host 参数是必需的")
	}

	// 可选的 path 参数
	pathFilter := ""
	if p, ok := args["path"].(string); ok {
		pathFilter = p
	}

	// 是否包含 Endpoints
	includeEndpoints := getBoolArg(args, "includeEndpoints", false)

	// 匹配模式：exact（精确）、prefix（前缀）、contains（包含）、wildcard（通配符）
	matchMode := getStringArg(args, "matchMode", "smart")

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	queryNs := namespace
	if queryNs == "" {
		queryNs = metav1.NamespaceAll
	}

	ingresses, err := clientset.Clientset.NetworkingV1().Ingresses(queryNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Ingress 列表失败: %w", err)
	}

	// 收集匹配的 Service 名称和命名空间
	type svcKey struct {
		name      string
		namespace string
	}
	matchedSvcKeys := make(map[svcKey]bool)

	for _, ing := range ingresses.Items {
		for _, rule := range ing.Spec.Rules {
			// 使用优化后的域名匹配
			if !matchHostWithMode(rule.Host, host, matchMode) {
				continue
			}

			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					// 如果指定了 path，则需要匹配
					if pathFilter != "" && !strings.HasPrefix(path.Path, pathFilter) {
						continue
					}
					if path.Backend.Service != nil {
						matchedSvcKeys[svcKey{
							name:      path.Backend.Service.Name,
							namespace: ing.Namespace,
						}] = true
					}
				}
			}
		}
	}

	// 查询匹配的 Service 详情
	matchedServices := make([]ServiceInfoWithEndpoints, 0)
	for key := range matchedSvcKeys {
		svc, err := clientset.Clientset.CoreV1().Services(key.namespace).Get(ctx, key.name, metav1.GetOptions{})
		if err != nil {
			continue // 跳过不存在的 Service
		}

		ports := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, ServicePortInfo{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.String(),
				NodePort:   p.NodePort,
				Protocol:   string(p.Protocol),
			})
		}

		svcInfo := ServiceInfoWithEndpoints{
			ServiceInfo: ServiceInfo{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Type:      string(svc.Spec.Type),
				ClusterIP: svc.Spec.ClusterIP,
				Ports:     ports,
			},
		}

		if verbose {
			svcInfo.ExternalIP = getExternalIP(svc)
			svcInfo.Labels = svc.Labels
			svcInfo.Selector = svc.Spec.Selector
			svcInfo.CreatedAt = svc.CreationTimestamp.Time
		}

		// 获取 Endpoints
		if includeEndpoints {
			svcInfo.Endpoints = getServiceEndpoints(ctx, clientset, svc.Namespace, svc.Name)
		}

		matchedServices = append(matchedServices, svcInfo)
	}

	message := fmt.Sprintf("找到 %d 个与域名 '%s' 关联的 Service", len(matchedServices), host)
	if pathFilter != "" {
		message = fmt.Sprintf("找到 %d 个与域名 '%s' 路径 '%s' 关联的 Service", len(matchedServices), host, pathFilter)
	}
	if len(matchedServices) == 0 {
		message = fmt.Sprintf("未找到与域名 '%s' 关联的 Service", host)
	}

	return &ReverseServiceResultWithEndpoints{
		Query:    host,
		Type:     "ingress",
		Services: matchedServices,
		Message:  message,
	}, nil
}

// FindWorkloadByService 通过 Service 查找对应的工作负载
func FindWorkloadByService(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	verbose := isVerbose(args)

	// 获取 Service 名称
	serviceName, ok := args["serviceName"].(string)
	if !ok || serviceName == "" {
		return nil, fmt.Errorf("serviceName 参数是必需的")
	}

	// 是否包含 Endpoints
	includeEndpoints := getBoolArg(args, "includeEndpoints", false)

	// 如果没有指定 namespace，默认使用 default
	if namespace == "" {
		namespace = "default"
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 获取 Service
	svc, err := clientset.Clientset.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Service '%s' 失败: %w", serviceName, err)
	}

	// 获取 Endpoints
	var endpoints []EndpointInfo
	if includeEndpoints {
		endpoints = getServiceEndpoints(ctx, clientset, namespace, serviceName)
	}

	if len(svc.Spec.Selector) == 0 {
		return &ReverseWorkloadResult{
			ServiceName:      serviceName,
			ServiceNamespace: namespace,
			ServiceSelector:  nil,
			Workloads:        []WorkloadInfo{},
			Endpoints:        endpoints,
			Message:          fmt.Sprintf("Service '%s' 没有定义 selector，无法关联工作负载", serviceName),
		}, nil
	}

	// 构建 label selector
	selectorSet := labels.Set(svc.Spec.Selector)
	labelSelector := selectorSet.AsSelector().String()

	workloads := make([]WorkloadInfo, 0)

	// 查找 Deployment
	deployments, err := clientset.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deploy := range deployments.Items {
			if matchSelector(deploy.Spec.Selector.MatchLabels, svc.Spec.Selector) {
				images := make([]string, 0)
				for _, c := range deploy.Spec.Template.Spec.Containers {
					images = append(images, c.Image)
				}
				w := WorkloadInfo{
					Kind:          "Deployment",
					Name:          deploy.Name,
					Namespace:     deploy.Namespace,
					Replicas:      *deploy.Spec.Replicas,
					ReadyReplicas: deploy.Status.ReadyReplicas,
					Images:        images,
				}
				if verbose {
					w.Labels = deploy.Labels
					w.Selector = deploy.Spec.Selector.MatchLabels
					w.CreatedAt = deploy.CreationTimestamp.Format("2006-01-02T15:04:05Z")
				}
				workloads = append(workloads, w)
			}
		}
	}

	// 查找 StatefulSet
	statefulsets, err := clientset.Clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sts := range statefulsets.Items {
			if matchSelector(sts.Spec.Selector.MatchLabels, svc.Spec.Selector) {
				images := make([]string, 0)
				for _, c := range sts.Spec.Template.Spec.Containers {
					images = append(images, c.Image)
				}
				w := WorkloadInfo{
					Kind:          "StatefulSet",
					Name:          sts.Name,
					Namespace:     sts.Namespace,
					Replicas:      *sts.Spec.Replicas,
					ReadyReplicas: sts.Status.ReadyReplicas,
					Images:        images,
				}
				if verbose {
					w.Labels = sts.Labels
					w.Selector = sts.Spec.Selector.MatchLabels
					w.CreatedAt = sts.CreationTimestamp.Format("2006-01-02T15:04:05Z")
				}
				workloads = append(workloads, w)
			}
		}
	}

	// 查找 DaemonSet
	daemonsets, err := clientset.Clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ds := range daemonsets.Items {
			if matchSelector(ds.Spec.Selector.MatchLabels, svc.Spec.Selector) {
				images := make([]string, 0)
				for _, c := range ds.Spec.Template.Spec.Containers {
					images = append(images, c.Image)
				}
				w := WorkloadInfo{
					Kind:          "DaemonSet",
					Name:          ds.Name,
					Namespace:     ds.Namespace,
					Replicas:      ds.Status.DesiredNumberScheduled,
					ReadyReplicas: ds.Status.NumberReady,
					Images:        images,
				}
				if verbose {
					w.Labels = ds.Labels
					w.Selector = ds.Spec.Selector.MatchLabels
					w.CreatedAt = ds.CreationTimestamp.Format("2006-01-02T15:04:05Z")
				}
				workloads = append(workloads, w)
			}
		}
	}

	message := fmt.Sprintf("找到 %d 个与 Service '%s' 关联的工作负载 (selector: %s)", len(workloads), serviceName, labelSelector)
	if len(workloads) == 0 {
		message = fmt.Sprintf("未找到与 Service '%s' 关联的工作负载 (selector: %s)", serviceName, labelSelector)
	}

	return &ReverseWorkloadResult{
		ServiceName:      serviceName,
		ServiceNamespace: namespace,
		ServiceSelector:  svc.Spec.Selector,
		Workloads:        workloads,
		Endpoints:        endpoints,
		Message:          message,
	}, nil
}

// ========== 辅助函数 ==========

// getServiceEndpoints 获取 Service 的 Endpoints 信息
func getServiceEndpoints(ctx context.Context, clientset *k8s.ClientSet, namespace, serviceName string) []EndpointInfo {
	endpoints := make([]EndpointInfo, 0)

	// 优先使用 EndpointSlice (K8s 1.21+)
	endpointSlices, err := clientset.Clientset.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("kubernetes.io/service-name=%s", serviceName),
	})
	if err == nil && len(endpointSlices.Items) > 0 {
		for _, slice := range endpointSlices.Items {
			for _, endpoint := range slice.Endpoints {
				ready := endpoint.Conditions.Ready != nil && *endpoint.Conditions.Ready
				nodeName := ""
				if endpoint.NodeName != nil {
					nodeName = *endpoint.NodeName
				}
				targetRef := ""
				if endpoint.TargetRef != nil {
					targetRef = fmt.Sprintf("%s/%s", endpoint.TargetRef.Kind, endpoint.TargetRef.Name)
				}
				podName := ""
				if endpoint.TargetRef != nil && endpoint.TargetRef.Kind == "Pod" {
					podName = endpoint.TargetRef.Name
				}

				for _, addr := range endpoint.Addresses {
					for _, port := range slice.Ports {
						portNum := int32(0)
						if port.Port != nil {
							portNum = *port.Port
						}
						protocol := "TCP"
						if port.Protocol != nil {
							protocol = string(*port.Protocol)
						}
						endpoints = append(endpoints, EndpointInfo{
							PodName:   podName,
							IP:        addr,
							Port:      portNum,
							Protocol:  protocol,
							Ready:     ready,
							NodeName:  nodeName,
							TargetRef: targetRef,
						})
					}
				}
			}
		}
		return endpoints
	}

	// 回退到传统 Endpoints API
	ep, err := clientset.Clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return endpoints
	}

	for _, subset := range ep.Subsets {
		for _, addr := range subset.Addresses {
			targetRef := ""
			podName := ""
			if addr.TargetRef != nil {
				targetRef = fmt.Sprintf("%s/%s", addr.TargetRef.Kind, addr.TargetRef.Name)
				if addr.TargetRef.Kind == "Pod" {
					podName = addr.TargetRef.Name
				}
			}
			nodeName := ""
			if addr.NodeName != nil {
				nodeName = *addr.NodeName
			}

			for _, port := range subset.Ports {
				endpoints = append(endpoints, EndpointInfo{
					PodName:   podName,
					IP:        addr.IP,
					Port:      port.Port,
					Protocol:  string(port.Protocol),
					Ready:     true,
					NodeName:  nodeName,
					TargetRef: targetRef,
				})
			}
		}

		// 不就绪的地址
		for _, addr := range subset.NotReadyAddresses {
			targetRef := ""
			podName := ""
			if addr.TargetRef != nil {
				targetRef = fmt.Sprintf("%s/%s", addr.TargetRef.Kind, addr.TargetRef.Name)
				if addr.TargetRef.Kind == "Pod" {
					podName = addr.TargetRef.Name
				}
			}
			nodeName := ""
			if addr.NodeName != nil {
				nodeName = *addr.NodeName
			}

			for _, port := range subset.Ports {
				endpoints = append(endpoints, EndpointInfo{
					PodName:   podName,
					IP:        addr.IP,
					Port:      port.Port,
					Protocol:  string(port.Protocol),
					Ready:     false,
					NodeName:  nodeName,
					TargetRef: targetRef,
				})
			}
		}
	}

	return endpoints
}

// getExternalIP 获取 Service 的外部 IP
func getExternalIP(svc *corev1.Service) string {
	if len(svc.Spec.ExternalIPs) > 0 {
		return strings.Join(svc.Spec.ExternalIPs, ",")
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ips := make([]string, 0)
		for _, ingress := range svc.Status.LoadBalancer.Ingress {
			if ingress.IP != "" {
				ips = append(ips, ingress.IP)
			} else if ingress.Hostname != "" {
				ips = append(ips, ingress.Hostname)
			}
		}
		return strings.Join(ips, ",")
	}
	return ""
}

// matchHost 匹配域名（支持通配符）- 保留向后兼容
func matchHost(pattern, host string) bool {
	return matchHostWithMode(pattern, host, "smart")
}

// matchHostWithMode 使用指定模式匹配域名
// 模式说明:
//   - exact: 精确匹配
//   - prefix: 前缀匹配（host 以 pattern 开头）
//   - suffix: 后缀匹配（host 以 pattern 结尾）
//   - contains: 包含匹配（host 包含 pattern）
//   - wildcard: 通配符匹配（支持 *.example.com 格式）
//   - smart: 智能匹配（默认，综合多种策略）
func matchHostWithMode(pattern, host string, mode string) bool {
	if pattern == "" {
		return true
	}
	if host == "" {
		return false
	}

	// 统一转小写进行比较
	pattern = strings.ToLower(pattern)
	host = strings.ToLower(host)

	switch mode {
	case "exact":
		return pattern == host

	case "prefix":
		return strings.HasPrefix(host, pattern)

	case "suffix":
		return strings.HasSuffix(host, pattern)

	case "contains":
		return strings.Contains(host, pattern)

	case "wildcard":
		return matchWildcard(pattern, host)

	case "smart":
		fallthrough
	default:
		// 智能匹配策略
		// 1. 精确匹配
		if pattern == host {
			return true
		}
		// 2. 通配符匹配 (*.example.com)
		if strings.HasPrefix(pattern, "*.") {
			return matchWildcard(pattern, host)
		}
		// 3. 用户输入可能是部分域名，检查是否为子域名关系
		// 例如: pattern="api.example.com", host="api.example.com" 或
		//       pattern="example.com", host="api.example.com"
		if strings.HasSuffix(host, "."+pattern) {
			return true
		}
		// 4. 反向检查：用户输入的是完整域名，Ingress 配置的是父域名
		if strings.HasSuffix(pattern, "."+host) {
			return true
		}
		return false
	}
}

// matchWildcard 通配符域名匹配
// 支持格式: *.example.com 匹配 api.example.com, www.example.com 等
func matchWildcard(pattern, host string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return pattern == host
	}

	// *.example.com -> .example.com
	suffix := pattern[1:]

	// api.example.com 应该匹配 *.example.com
	// 但 example.com 不应该匹配 *.example.com
	if !strings.HasSuffix(host, suffix) {
		return false
	}

	// 确保 host 在 suffix 之前还有内容（即有子域名部分）
	prefix := strings.TrimSuffix(host, suffix)
	// prefix 应该是非空的，且不包含点（单级子域名）
	// 例如: api.example.com -> prefix="api", 这是有效的
	// 但: sub.api.example.com -> prefix="sub.api", 对于 *.example.com 也应该匹配
	return len(prefix) > 0
}

// matchSelector 检查工作负载的 selector 是否与 Service 的 selector 匹配
func matchSelector(workloadSelector, serviceSelector map[string]string) bool {
	if len(serviceSelector) == 0 {
		return false
	}
	// Service selector 的所有标签都必须在 workload selector 中存在且值相同
	for k, v := range serviceSelector {
		if workloadSelector[k] != v {
			return false
		}
	}
	return true
}

// getBoolArg 获取布尔参数
func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// ========== 完整链路追踪工具 ==========

// TraceByNodePort 通过 NodePort 追踪完整链路 (NodePort → Service → Workload)
func TraceByNodePort(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	verbose := isVerbose(args)

	// 显式获取 namespace 参数，如果没有传则查询所有 namespace
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// 获取 nodePort 参数
	var nodePort int32
	switch v := args["nodePort"].(type) {
	case float64:
		nodePort = int32(v)
	case int:
		nodePort = int32(v)
	case int32:
		nodePort = v
	default:
		return nil, fmt.Errorf("nodePort 参数必须是数字类型")
	}

	// 端口范围校验
	var warning string
	if nodePort < 1 || nodePort > 65535 {
		return nil, fmt.Errorf("端口号必须在 1-65535 范围内，当前值: %d", nodePort)
	}
	if nodePort < 30000 || nodePort > 32767 {
		warning = fmt.Sprintf("NodePort %d 不在默认范围 30000-32767 内，可能是自定义配置", nodePort)
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 尝试使用索引器查询 Service
	var services []*corev1.Service
	indexer, indexerErr := k8sClient.GetServiceIndexerForContext(contextName)

	if indexerErr == nil && indexer.IsReady() {
		if namespace != "" {
			services = indexer.FindByNodePortInNamespace(nodePort, namespace)
		} else {
			services = indexer.FindByNodePort(nodePort)
		}
	} else {
		// 回退到 API 查询
		queryNs := namespace
		if queryNs == "" {
			queryNs = metav1.NamespaceAll
		}

		svcList, err := clientset.Clientset.CoreV1().Services(queryNs).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("查询 Service 列表失败: %w", err)
		}

		for i := range svcList.Items {
			svc := &svcList.Items[i]
			for _, port := range svc.Spec.Ports {
				if port.NodePort == nodePort {
					services = append(services, svc)
					break
				}
			}
		}
	}

	matchedServices := make([]ServiceInfoWithEndpoints, 0)
	allWorkloads := make([]WorkloadInfo, 0)
	traceChain := make([]string, 0)

	for _, svc := range services {
		ports := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, ServicePortInfo{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.String(),
				NodePort:   p.NodePort,
				Protocol:   string(p.Protocol),
			})
		}

		svcInfo := ServiceInfoWithEndpoints{
			ServiceInfo: ServiceInfo{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Type:      string(svc.Spec.Type),
				ClusterIP: svc.Spec.ClusterIP,
				Ports:     ports,
			},
			Endpoints: getServiceEndpoints(ctx, clientset, svc.Namespace, svc.Name),
		}

		if verbose {
			svcInfo.ExternalIP = getExternalIP(svc)
			svcInfo.Labels = svc.Labels
			svcInfo.Selector = svc.Spec.Selector
			svcInfo.CreatedAt = svc.CreationTimestamp.Time
		}

		matchedServices = append(matchedServices, svcInfo)

		traceChain = append(traceChain, fmt.Sprintf("NodePort:%d → Service:%s/%s", nodePort, svc.Namespace, svc.Name))

		// Step 2: 查找关联的 Workload
		if len(svc.Spec.Selector) > 0 {
			workloads := findWorkloadsForService(ctx, clientset, svc.Namespace, svc.Spec.Selector, verbose)
			for _, w := range workloads {
				allWorkloads = append(allWorkloads, w)
				traceChain = append(traceChain, fmt.Sprintf("Service:%s/%s → %s:%s/%s", svc.Namespace, svc.Name, w.Kind, w.Namespace, w.Name))
			}
		}
	}

	message := fmt.Sprintf("NodePort %d 完整链路追踪: 找到 %d 个 Service, %d 个工作负载", nodePort, len(matchedServices), len(allWorkloads))
	if len(matchedServices) == 0 {
		message = fmt.Sprintf("未找到使用 NodePort %d 的 Service", nodePort)
	}

	return &FullTraceResult{
		Query:      fmt.Sprintf("%d", nodePort),
		QueryType:  "nodeport",
		Services:   matchedServices,
		Workloads:  allWorkloads,
		Message:    message,
		Warning:    warning,
		TraceChain: traceChain,
	}, nil
}

// TraceByHost 通过域名追踪完整链路 (Host → Ingress → Service → Workload)
func TraceByHost(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	verbose := isVerbose(args)

	// 获取 host 参数
	host, ok := args["host"].(string)
	if !ok || host == "" {
		return nil, fmt.Errorf("host 参数是必需的")
	}

	// 可选的 path 参数
	pathFilter := ""
	if p, ok := args["path"].(string); ok {
		pathFilter = p
	}

	// 匹配模式
	matchMode := getStringArg(args, "matchMode", "smart")

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	queryNs := namespace
	if queryNs == "" {
		queryNs = metav1.NamespaceAll
	}

	// Step 1: 查找 Ingress
	ingresses, err := clientset.Clientset.NetworkingV1().Ingresses(queryNs).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 Ingress 列表失败: %w", err)
	}

	matchedIngresses := make([]IngressInfo, 0)
	type svcKey struct {
		name      string
		namespace string
	}
	matchedSvcKeys := make(map[svcKey]bool)
	traceChain := make([]string, 0)

	for _, ing := range ingresses.Items {
		ingMatched := false
		hosts := make([]string, 0)
		paths := make([]IngressPathInfo, 0)

		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				hosts = append(hosts, rule.Host)
			}

			if !matchHostWithMode(rule.Host, host, matchMode) {
				continue
			}

			if rule.HTTP != nil {
				for _, path := range rule.HTTP.Paths {
					if pathFilter != "" && !strings.HasPrefix(path.Path, pathFilter) {
						continue
					}

					pathType := "Prefix"
					if path.PathType != nil {
						pathType = string(*path.PathType)
					}
					servicePort := ""
					if path.Backend.Service != nil {
						if path.Backend.Service.Port.Name != "" {
							servicePort = path.Backend.Service.Port.Name
						} else {
							servicePort = fmt.Sprintf("%d", path.Backend.Service.Port.Number)
						}

						matchedSvcKeys[svcKey{
							name:      path.Backend.Service.Name,
							namespace: ing.Namespace,
						}] = true

						ingMatched = true
						traceChain = append(traceChain, fmt.Sprintf("Host:%s%s → Ingress:%s/%s → Service:%s/%s",
							rule.Host, path.Path, ing.Namespace, ing.Name, ing.Namespace, path.Backend.Service.Name))
					}

					paths = append(paths, IngressPathInfo{
						Host:        rule.Host,
						Path:        path.Path,
						PathType:    pathType,
						ServiceName: path.Backend.Service.Name,
						ServicePort: servicePort,
					})
				}
			}
		}

		if ingMatched {
			ingInfo := IngressInfo{
				Name:      ing.Name,
				Namespace: ing.Namespace,
				Hosts:     hosts,
				Paths:     paths,
			}
			if verbose {
				ingInfo.Labels = ing.Labels
				ingInfo.Annotations = ing.Annotations
				ingInfo.CreatedAt = ing.CreationTimestamp.Format("2006-01-02T15:04:05Z")
			}
			matchedIngresses = append(matchedIngresses, ingInfo)
		}
	}

	// Step 2: 查找 Service 详情
	matchedServices := make([]ServiceInfoWithEndpoints, 0)
	allWorkloads := make([]WorkloadInfo, 0)

	for key := range matchedSvcKeys {
		svc, err := clientset.Clientset.CoreV1().Services(key.namespace).Get(ctx, key.name, metav1.GetOptions{})
		if err != nil {
			continue
		}

		ports := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
		for _, p := range svc.Spec.Ports {
			ports = append(ports, ServicePortInfo{
				Name:       p.Name,
				Port:       p.Port,
				TargetPort: p.TargetPort.String(),
				NodePort:   p.NodePort,
				Protocol:   string(p.Protocol),
			})
		}

		svcInfo := ServiceInfoWithEndpoints{
			ServiceInfo: ServiceInfo{
				Name:      svc.Name,
				Namespace: svc.Namespace,
				Type:      string(svc.Spec.Type),
				ClusterIP: svc.Spec.ClusterIP,
				Ports:     ports,
			},
			Endpoints: getServiceEndpoints(ctx, clientset, svc.Namespace, svc.Name),
		}

		if verbose {
			svcInfo.ExternalIP = getExternalIP(svc)
			svcInfo.Labels = svc.Labels
			svcInfo.Selector = svc.Spec.Selector
			svcInfo.CreatedAt = svc.CreationTimestamp.Time
		}

		matchedServices = append(matchedServices, svcInfo)

		// Step 3: 查找关联的 Workload
		if len(svc.Spec.Selector) > 0 {
			workloads := findWorkloadsForService(ctx, clientset, svc.Namespace, svc.Spec.Selector, verbose)
			for _, w := range workloads {
				allWorkloads = append(allWorkloads, w)
				traceChain = append(traceChain, fmt.Sprintf("Service:%s/%s → %s:%s/%s", svc.Namespace, svc.Name, w.Kind, w.Namespace, w.Name))
			}
		}
	}

	message := fmt.Sprintf("域名 '%s' 完整链路追踪: 找到 %d 个 Ingress, %d 个 Service, %d 个工作负载",
		host, len(matchedIngresses), len(matchedServices), len(allWorkloads))
	if len(matchedIngresses) == 0 {
		message = fmt.Sprintf("未找到与域名 '%s' 关联的 Ingress", host)
	}

	return &FullTraceResult{
		Query:      host,
		QueryType:  "host",
		Ingresses:  matchedIngresses,
		Services:   matchedServices,
		Workloads:  allWorkloads,
		Message:    message,
		TraceChain: traceChain,
	}, nil
}

// findWorkloadsForService 查找与 Service selector 匹配的工作负载
func findWorkloadsForService(ctx context.Context, clientset *k8s.ClientSet, namespace string, selector map[string]string, verbose bool) []WorkloadInfo {
	workloads := make([]WorkloadInfo, 0)

	// 查找 Deployment
	deployments, err := clientset.Clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deploy := range deployments.Items {
			if deploy.Spec.Selector != nil && matchSelector(deploy.Spec.Selector.MatchLabels, selector) {
				images := make([]string, 0)
				for _, c := range deploy.Spec.Template.Spec.Containers {
					images = append(images, c.Image)
				}
				replicas := int32(0)
				if deploy.Spec.Replicas != nil {
					replicas = *deploy.Spec.Replicas
				}
				w := WorkloadInfo{
					Kind:          "Deployment",
					Name:          deploy.Name,
					Namespace:     deploy.Namespace,
					Replicas:      replicas,
					ReadyReplicas: deploy.Status.ReadyReplicas,
					Images:        images,
				}
				if verbose {
					w.Labels = deploy.Labels
					w.Selector = deploy.Spec.Selector.MatchLabels
					w.CreatedAt = deploy.CreationTimestamp.Format("2006-01-02T15:04:05Z")
				}
				workloads = append(workloads, w)
			}
		}
	}

	// 查找 StatefulSet
	statefulsets, err := clientset.Clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, sts := range statefulsets.Items {
			if sts.Spec.Selector != nil && matchSelector(sts.Spec.Selector.MatchLabels, selector) {
				images := make([]string, 0)
				for _, c := range sts.Spec.Template.Spec.Containers {
					images = append(images, c.Image)
				}
				replicas := int32(0)
				if sts.Spec.Replicas != nil {
					replicas = *sts.Spec.Replicas
				}
				w := WorkloadInfo{
					Kind:          "StatefulSet",
					Name:          sts.Name,
					Namespace:     sts.Namespace,
					Replicas:      replicas,
					ReadyReplicas: sts.Status.ReadyReplicas,
					Images:        images,
				}
				if verbose {
					w.Labels = sts.Labels
					w.Selector = sts.Spec.Selector.MatchLabels
					w.CreatedAt = sts.CreationTimestamp.Format("2006-01-02T15:04:05Z")
				}
				workloads = append(workloads, w)
			}
		}
	}

	// 查找 DaemonSet
	daemonsets, err := clientset.Clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, ds := range daemonsets.Items {
			if ds.Spec.Selector != nil && matchSelector(ds.Spec.Selector.MatchLabels, selector) {
				images := make([]string, 0)
				for _, c := range ds.Spec.Template.Spec.Containers {
					images = append(images, c.Image)
				}
				w := WorkloadInfo{
					Kind:          "DaemonSet",
					Name:          ds.Name,
					Namespace:     ds.Namespace,
					Replicas:      ds.Status.DesiredNumberScheduled,
					ReadyReplicas: ds.Status.NumberReady,
					Images:        images,
				}
				if verbose {
					w.Labels = ds.Labels
					w.Selector = ds.Spec.Selector.MatchLabels
					w.CreatedAt = ds.CreationTimestamp.Format("2006-01-02T15:04:05Z")
				}
				workloads = append(workloads, w)
			}
		}
	}

	return workloads
}
