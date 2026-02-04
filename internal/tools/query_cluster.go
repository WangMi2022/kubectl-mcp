package tools

import (
	"context"
	"fmt"
	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetNodes 查询节点列表
func GetNodes(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	labelSelector := buildLabelSelector(args)

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	nodes, err := clientset.Clientset.CoreV1().Nodes().List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询节点列表失败: %w", err)
	}

	result := make([]NodeInfo, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		// 获取 IP 地址
		var internalIP, externalIP string
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				internalIP = addr.Address
			case corev1.NodeExternalIP:
				externalIP = addr.Address
			}
		}

		// 获取 taints
		taints := make([]string, 0, len(node.Spec.Taints))
		for _, taint := range node.Spec.Taints {
			taints = append(taints, fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect))
		}

		nodeInfo := NodeInfo{
			Name:              node.Name,
			Status:            getNodeStatus(node.Status.Conditions),
			Roles:             getNodeRoles(node.Labels),
			Version:           node.Status.NodeInfo.KubeletVersion,
			InternalIP:        internalIP,
			ExternalIP:        externalIP,
			OS:                node.Status.NodeInfo.OperatingSystem,
			Architecture:      node.Status.NodeInfo.Architecture,
			ContainerRuntime:  node.Status.NodeInfo.ContainerRuntimeVersion,
			Labels:            node.Labels,
			Taints:            taints,
			CreatedAt:         node.CreationTimestamp.Time,
			AllocatableCPU:    node.Status.Allocatable.Cpu().String(),
			AllocatableMemory: node.Status.Allocatable.Memory().String(),
		}
		result = append(result, nodeInfo)
	}

	return result, nil
}

// GetNamespaces 查询命名空间列表
func GetNamespaces(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	labelSelector := buildLabelSelector(args)

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	namespaces, err := clientset.Clientset.CoreV1().Namespaces().List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询命名空间列表失败: %w", err)
	}

	result := make([]NamespaceInfo, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		nsInfo := NamespaceInfo{
			Name:      ns.Name,
			Status:    string(ns.Status.Phase),
			Labels:    ns.Labels,
			CreatedAt: ns.CreationTimestamp.Time,
		}
		result = append(result, nsInfo)
	}

	return result, nil
}
