package tools

import (
	"context"
	"fmt"
	"sort"
	"time"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// 默认事件返回上限，避免数据量过大
const defaultEventLimit = 50

// InspectEvents 事件巡检
// 返回最近的 Warning 事件，按 Reason 和资源聚合，精简输出
func InspectEvents(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)

	// 获取 namespace 参数
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && ns != "" {
		namespace = ns
	}

	// 获取 limit 参数
	limit := defaultEventLimit
	if l, ok := args["limit"].(float64); ok && int(l) > 0 {
		limit = int(l)
	}

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	cs := clientSet.Clientset

	// 只查询 Warning 类型事件
	events, err := cs.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, fmt.Errorf("查询事件失败: %w", err)
	}

	report := EventInspectReport{
		CheckTime:     time.Now(),
		TotalWarnings: len(events.Items),
	}

	// 聚合事件：按 Reason + InvolvedObject 去重，保留最新的
	type eventKey struct {
		Reason     string
		ObjectKind string
		ObjectName string
		Namespace  string
	}
	aggregated := make(map[eventKey]*AggregatedEventInfo)

	for _, event := range events.Items {
		key := eventKey{
			Reason:     event.Reason,
			ObjectKind: event.InvolvedObject.Kind,
			ObjectName: event.InvolvedObject.Name,
			Namespace:  event.Namespace,
		}

		lastTime := event.LastTimestamp.Time
		if lastTime.IsZero() {
			lastTime = event.CreationTimestamp.Time
		}

		if existing, ok := aggregated[key]; ok {
			existing.Count += event.Count
			if lastTime.After(existing.LastOccurrence) {
				existing.LastOccurrence = lastTime
				existing.Message = truncateMessage(event.Message, 120)
			}
		} else {
			aggregated[key] = &AggregatedEventInfo{
				Reason:         event.Reason,
				ObjectKind:     event.InvolvedObject.Kind,
				ObjectName:     event.InvolvedObject.Name,
				Namespace:      event.Namespace,
				Message:        truncateMessage(event.Message, 120),
				Count:          event.Count,
				LastOccurrence: lastTime,
			}
		}
	}

	// 转为切片并按最近发生时间排序
	result := make([]AggregatedEventInfo, 0, len(aggregated))
	for _, v := range aggregated {
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastOccurrence.After(result[j].LastOccurrence)
	})

	// 限制返回数量
	if len(result) > limit {
		result = result[:limit]
	}

	report.AggregatedEvents = result
	return report, nil
}

// truncateMessage 截断消息，避免过长
func truncateMessage(msg string, maxLen int) string {
	runes := []rune(msg)
	if len(runes) <= maxLen {
		return msg
	}
	return string(runes[:maxLen]) + "..."
}
