package tools

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultEventRootCauseLimit = 100
const defaultEventRootCauseSinceMinutes = 1440

type eventRootCauseAggregate struct {
	Reason     string
	ObjectKind string
	ObjectName string
	Namespace  string
	Message    string
	Count      int32
	LastSeen   time.Time
	Type       string
}

// InspectEventRootCauses analyzes Warning Events and groups them into stable root-cause findings.
func InspectEventRootCauses(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	namespace := ""
	if ns, ok := args["namespace"].(string); ok && strings.TrimSpace(ns) != "" {
		namespace = strings.TrimSpace(ns)
	}

	limit := intArg(args, "limit", defaultEventRootCauseLimit)
	if limit <= 0 {
		limit = defaultEventRootCauseLimit
	}
	sinceMinutes := intArg(args, "sinceMinutes", defaultEventRootCauseSinceMinutes)
	if sinceMinutes <= 0 {
		sinceMinutes = defaultEventRootCauseSinceMinutes
	}
	since := time.Now().Add(-time.Duration(sinceMinutes) * time.Minute)

	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	events, err := clientSet.Clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{FieldSelector: "type=Warning"})
	if err != nil {
		return nil, fmt.Errorf("查询 Warning 事件失败: %w", err)
	}

	aggregated := aggregateRootCauseEvents(events.Items, since)
	findings := buildRootCauseFindings(aggregated, limit)
	return DiagnosticReport{
		CheckTime: time.Now(),
		Scope: DiagnosticScope{
			Cluster:   contextName,
			Namespace: namespace,
		},
		Summary:  buildDiagnosticSummary(findings),
		Findings: findings,
	}, nil
}

func aggregateRootCauseEvents(events []corev1.Event, since time.Time) []eventRootCauseAggregate {
	type eventKey struct{ reason, kind, namespace, name string }
	aggregated := map[eventKey]*eventRootCauseAggregate{}
	for _, event := range events {
		lastSeen := eventLastSeen(event)
		if !since.IsZero() && lastSeen.Before(since) {
			continue
		}
		kind := event.InvolvedObject.Kind
		name := event.InvolvedObject.Name
		ns := event.Namespace
		if ns == "" {
			ns = event.InvolvedObject.Namespace
		}
		key := eventKey{reason: event.Reason, kind: kind, namespace: ns, name: name}
		if existing, ok := aggregated[key]; ok {
			existing.Count += normalizedEventCount(event.Count)
			if lastSeen.After(existing.LastSeen) {
				existing.LastSeen = lastSeen
				existing.Message = truncateMessage(event.Message, 220)
			}
			continue
		}
		aggregated[key] = &eventRootCauseAggregate{
			Reason:     event.Reason,
			ObjectKind: kind,
			ObjectName: name,
			Namespace:  ns,
			Message:    truncateMessage(event.Message, 220),
			Count:      normalizedEventCount(event.Count),
			LastSeen:   lastSeen,
			Type:       classifyEventFindingType(event.Reason, event.Message),
		}
	}
	items := make([]eventRootCauseAggregate, 0, len(aggregated))
	for _, item := range aggregated {
		items = append(items, *item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].LastSeen.After(items[j].LastSeen)
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func buildRootCauseFindings(items []eventRootCauseAggregate, limit int) []DiagnosticFinding {
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	findings := make([]DiagnosticFinding, 0, len(items))
	for _, item := range items {
		severity := "warning"
		if item.Type == "ImagePullFailure" || item.Type == "ScheduleFailure" || item.Type == "VolumeMountFailure" {
			severity = "critical"
		}
		obj := DiagnosticObjectRef{Kind: item.ObjectKind, Namespace: item.Namespace, Name: item.ObjectName}
		findings = append(findings, DiagnosticFinding{
			ID:             stableFindingID("event-root-cause", item.Type, item.Namespace, item.ObjectKind, item.ObjectName, item.Reason),
			Severity:       severity,
			FindingType:    item.Type,
			Title:          fmt.Sprintf("%s: %s/%s", item.Type, item.ObjectKind, item.ObjectName),
			Description:    fmt.Sprintf("Warning Event reason=%s 出现 %d 次。%s", item.Reason, item.Count, item.Message),
			AffectedObject: obj,
			Evidence: []DiagnosticEvidence{{
				Source:   "event",
				Message:  item.Message,
				Count:    item.Count,
				LastSeen: item.LastSeen.Format(time.RFC3339),
			}},
			Recommendation: recommendationForEventFinding(item.Type),
			RelatedObjects: []DiagnosticObjectRef{obj},
			SafeActions: []DiagnosticSafeAction{
				{Action: "describe", RiskLevel: "read", Reason: "查看对象详情与 Events 证据"},
				{Action: "get_logs", RiskLevel: "read", Reason: "如对象是 Pod/工作负载，进一步查看相关容器日志"},
			},
		})
	}
	return findings
}

func classifyEventFindingType(reason, message string) string {
	r := strings.ToLower(strings.TrimSpace(reason))
	m := strings.ToLower(message)
	switch r {
	case "unhealthy":
		return "ProbeFailure"
	case "failedscheduling":
		return "ScheduleFailure"
	case "failedmount", "failedattachvolume":
		return "VolumeMountFailure"
	case "backoff", "crashloopbackoff":
		return "ContainerRestart"
	case "failedgetscale":
		return "HPATargetMissing"
	case "errimagepull", "imagepullbackoff", "failed":
		if strings.Contains(m, "image") || strings.Contains(m, "pull") {
			return "ImagePullFailure"
		}
	}
	if strings.Contains(m, "failedgetscale") || strings.Contains(m, "failed to get scale") || strings.Contains(m, "the hpa was unable") {
		return "HPATargetMissing"
	}
	if strings.Contains(m, "imagepullbackoff") || strings.Contains(m, "errimagepull") || strings.Contains(m, "pull image") {
		return "ImagePullFailure"
	}
	if strings.Contains(m, "failedscheduling") || (strings.Contains(m, "0/") && strings.Contains(m, "nodes are available")) {
		return "ScheduleFailure"
	}
	if strings.Contains(m, "failedmount") || strings.Contains(m, "failedattachvolume") {
		return "VolumeMountFailure"
	}
	return "EventWarning"
}

func recommendationForEventFinding(findingType string) string {
	switch findingType {
	case "ProbeFailure":
		return "检查 Pod 探针配置、容器监听端口、应用健康接口和近期日志。"
	case "ScheduleFailure":
		return "检查节点资源、污点/容忍、亲和性、PVC 绑定状态与调度事件。"
	case "VolumeMountFailure":
		return "检查 PVC/PV/StorageClass、挂载路径、CSI 插件与节点存储事件。"
	case "ContainerRestart":
		return "查看当前与 previous 日志、退出码、OOMKilled 状态和依赖服务可用性。"
	case "HPATargetMissing":
		return "检查 HPA scaleTargetRef 指向的 Deployment/StatefulSet 是否存在且 apiVersion/kind/name 正确。"
	case "ImagePullFailure":
		return "检查镜像名称/tag、镜像仓库连通性、imagePullSecrets 与节点拉取权限。"
	default:
		return "根据事件对象继续 describe，并结合相关 Pod/工作负载状态定位根因。"
	}
}

func buildDiagnosticSummary(findings []DiagnosticFinding) DiagnosticSummary {
	critical := 0
	warning := 0
	for _, finding := range findings {
		switch strings.ToLower(finding.Severity) {
		case "critical":
			critical++
		case "warning":
			warning++
		}
	}
	status := "Healthy"
	score := 100
	if warning > 0 {
		status = "Warning"
		score = 80
	}
	if critical > 0 {
		status = "Critical"
		score = 50
	}
	return DiagnosticSummary{Status: status, Score: score, FindingsCount: len(findings), CriticalCount: critical, WarningCount: warning}
}

func eventLastSeen(event corev1.Event) time.Time {
	if !event.LastTimestamp.Time.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.EventTime.Time.IsZero() {
		return event.EventTime.Time
	}
	return event.CreationTimestamp.Time
}

func normalizedEventCount(count int32) int32 {
	if count <= 0 {
		return 1
	}
	return count
}

func intArg(args map[string]interface{}, key string, defaultValue int) int {
	if args == nil {
		return defaultValue
	}
	switch v := args[key].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return defaultValue
}

func stableFindingID(parts ...string) string {
	h := sha1.Sum([]byte(strings.Join(parts, ":")))
	return hex.EncodeToString(h[:])[:16]
}
