package tools

import (
	"context"
	"fmt"
	"io"
	"kubectl-mcp/internal/k8s"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetEvents 查询事件列表
func GetEvents(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	labelSelector := buildLabelSelector(args)
	verbose := isVerbose(args)

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	listOptions := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	// 支持按资源类型和名称过滤
	if involvedObjectKind, ok := args["involvedObjectKind"].(string); ok && involvedObjectKind != "" {
		if involvedObjectName, ok := args["involvedObjectName"].(string); ok && involvedObjectName != "" {
			listOptions.FieldSelector = fmt.Sprintf("involvedObject.kind=%s,involvedObject.name=%s", involvedObjectKind, involvedObjectName)
		}
	}

	if namespace == "" {
		namespace = metav1.NamespaceAll
	}

	events, err := clientset.Clientset.CoreV1().Events(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("查询事件列表失败: %w", err)
	}

	result := make([]EventInfo, 0, len(events.Items))
	for _, event := range events.Items {
		involvedObject := fmt.Sprintf("%s/%s", event.InvolvedObject.Kind, event.InvolvedObject.Name)

		eventInfo := EventInfo{
			Namespace:      event.Namespace,
			Type:           event.Type,
			Reason:         event.Reason,
			Message:        truncateMessage(event.Message, 200),
			InvolvedObject: involvedObject,
			Count:          event.Count,
			LastTimestamp:  event.LastTimestamp.Time,
		}

		if verbose {
			eventInfo.Name = event.Name
			eventInfo.Source = fmt.Sprintf("%s/%s", event.Source.Component, event.Source.Host)
			eventInfo.FirstTimestamp = event.FirstTimestamp.Time
		}

		result = append(result, eventInfo)
	}

	return result, nil
}

func buildPodLogOptions(args map[string]interface{}, tailLines int64, previous bool, container string) (*corev1.PodLogOptions, error) {
	podLogOptions := &corev1.PodLogOptions{TailLines: &tailLines, Previous: previous}
	if container != "" {
		podLogOptions.Container = container
	}
	if sinceMinutes := intArg(args, "sinceMinutes", 0); sinceMinutes > 0 {
		seconds := int64(sinceMinutes) * 60
		podLogOptions.SinceSeconds = &seconds
	}
	if since := stringArg(args, "since", ""); since != "" {
		parsed, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return nil, fmt.Errorf("since 必须是 RFC3339 时间，例如 2026-07-08T08:00:00Z: %w", err)
		}
		metav1Since := metav1.NewTime(parsed)
		podLogOptions.SinceTime = &metav1Since
		podLogOptions.SinceSeconds = nil
	}
	return podLogOptions, nil
}

// GetPodLogs 获取 Pod 日志
func GetPodLogs(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)

	podName, ok := args["name"].(string)
	if !ok || podName == "" {
		return nil, fmt.Errorf("缺少必填参数: name")
	}

	if namespace == "" {
		namespace = "default"
	}

	container := ""
	if c, ok := args["container"].(string); ok {
		container = c
	}

	tailLines := int64(100)
	if t, ok := args["tailLines"].(float64); ok {
		tailLines = int64(t)
	}

	previous := false
	if p, ok := args["previous"].(bool); ok {
		previous = p
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	podLogOptions, err := buildPodLogOptions(args, tailLines, previous, container)
	if err != nil {
		return nil, err
	}

	req := clientset.Clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions)
	stream, err := req.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取 Pod '%s/%s' 日志失败: %w", namespace, podName, err)
	}
	defer stream.Close()

	logs, err := io.ReadAll(stream)
	if err != nil {
		return nil, fmt.Errorf("读取日志流失败: %w", err)
	}

	return &PodLogResult{
		PodName:   podName,
		Namespace: namespace,
		Container: container,
		Logs:      string(logs),
	}, nil
}
