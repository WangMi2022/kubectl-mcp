package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// CreateService 创建 Service
// 参数:
//   - name: Service 名称（必填）
//   - port: 服务端口（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - targetPort: 目标端口（可选，默认与 port 相同）
//   - type: Service 类型（可选，默认 ClusterIP）
//   - selector: Pod 选择器（可选，默认 app=name）
//   - labels: 标签（可选）
//   - protocol: 协议（可选，默认 TCP）
//   - nodePort: NodePort 端口（可选，仅 NodePort/LoadBalancer 类型有效）
//   - context: K8S context 名称（可选）
func CreateService(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	port := getInt32Arg(args, "port", 0)
	if port == 0 {
		return nil, fmt.Errorf("参数 'port' 是必填的")
	}

	// 获取可选参数
	targetPort := getInt32Arg(args, "targetPort", port)
	serviceType := getStringArg(args, "type", "ClusterIP")
	protocol := getStringArg(args, "protocol", "TCP")
	nodePort := getInt32Arg(args, "nodePort", 0)
	labels := getMapStringString(args, "labels")
	selector := getMapStringString(args, "selector")

	// 设置默认选择器
	if len(selector) == 0 {
		selector = map[string]string{"app": name}
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 构建 ServicePort
	servicePort := corev1.ServicePort{
		Port:       port,
		TargetPort: intstr.FromInt(int(targetPort)),
		Protocol:   corev1.Protocol(protocol),
	}

	// 如果是 NodePort 或 LoadBalancer 类型，设置 NodePort
	if (serviceType == "NodePort" || serviceType == "LoadBalancer") && nodePort > 0 {
		servicePort.NodePort = nodePort
	}

	// 构建 Service 对象
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(serviceType),
			Selector: selector,
			Ports:    []corev1.ServicePort{servicePort},
		},
	}

	// 创建 Service
	created, err := clientset.Clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Service '%s/%s' 已存在", namespace, name)
		}
		return nil, fmt.Errorf("创建 Service 失败: %w", err)
	}

	return &CreateResult{
		Kind:      "Service",
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    "Created",
		Message:   fmt.Sprintf("Service '%s/%s' 创建成功，类型: %s，端口: %d", namespace, name, serviceType, port),
		CreatedAt: created.CreationTimestamp.Time,
	}, nil
}
