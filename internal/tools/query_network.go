package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetServices 查询 Service 列表
func GetServices(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	namespace := metav1.NamespaceAll
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

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

	services, err := clientset.Clientset.CoreV1().Services(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询 Service 列表失败: %w", err)
	}

	result := make([]ServiceInfo, 0, len(services.Items))
	for _, svc := range services.Items {
		ports := make([]ServicePortInfo, 0, len(svc.Spec.Ports))
		for _, port := range svc.Spec.Ports {
			ports = append(ports, ServicePortInfo{
				Name:       port.Name,
				Port:       port.Port,
				TargetPort: port.TargetPort.String(),
				NodePort:   port.NodePort,
				Protocol:   string(port.Protocol),
			})
		}

		svcInfo := ServiceInfo{
			Name:      svc.Name,
			Namespace: svc.Namespace,
			Type:      string(svc.Spec.Type),
			ClusterIP: svc.Spec.ClusterIP,
			Ports:     ports,
			Selector:  svc.Spec.Selector,
		}

		if verbose {
			externalIP := ""
			if len(svc.Spec.ExternalIPs) > 0 {
				externalIP = strings.Join(svc.Spec.ExternalIPs, ",")
			} else if len(svc.Status.LoadBalancer.Ingress) > 0 {
				ips := make([]string, 0)
				for _, ingress := range svc.Status.LoadBalancer.Ingress {
					if ingress.IP != "" {
						ips = append(ips, ingress.IP)
					} else if ingress.Hostname != "" {
						ips = append(ips, ingress.Hostname)
					}
				}
				externalIP = strings.Join(ips, ",")
			}
			svcInfo.ExternalIP = externalIP
			svcInfo.Labels = svc.Labels
			svcInfo.CreatedAt = svc.CreationTimestamp.Time
		}

		result = append(result, svcInfo)
	}

	return result, nil
}
