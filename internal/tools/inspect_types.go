package tools

import "time"

// ========== 巡检数据结构定义 ==========

// ClusterOverview 集群概览
type ClusterOverview struct {
	CheckTime       time.Time         `json:"checkTime"`
	ClusterServer   string            `json:"clusterServer"`
	CurrentContext  string            `json:"currentContext"`
	NodeSummary     NodeSummary       `json:"nodeSummary"`
	PodSummary      PodSummary        `json:"podSummary"`
	WorkloadSummary WorkloadSummary   `json:"workloadSummary"`
	EventSummary    EventSummaryBrief `json:"eventSummary"`
	HealthScore     int               `json:"healthScore"`
	Issues          []string          `json:"issues,omitempty"`
}

// NodeSummary 节点摘要
type NodeSummary struct {
	Total    int `json:"total"`
	Ready    int `json:"ready"`
	NotReady int `json:"notReady"`
}

// PodSummary Pod 摘要
type PodSummary struct {
	Total            int `json:"total"`
	Running          int `json:"running"`
	Pending          int `json:"pending"`
	Failed           int `json:"failed"`
	CrashLoopBackOff int `json:"crashLoopBackOff"`
	Unknown          int `json:"unknown"`
}

// WorkloadSummary 工作负载摘要
type WorkloadSummary struct {
	DeploymentTotal      int `json:"deploymentTotal"`
	DeploymentUnhealthy  int `json:"deploymentUnhealthy"`
	StatefulSetTotal     int `json:"statefulSetTotal"`
	StatefulSetUnhealthy int `json:"statefulSetUnhealthy"`
	DaemonSetTotal       int `json:"daemonSetTotal"`
	DaemonSetUnhealthy   int `json:"daemonSetUnhealthy"`
}

// EventSummaryBrief 事件摘要（简要）
type EventSummaryBrief struct {
	WarningCount int `json:"warningCount"`
	ErrorCount   int `json:"errorCount"`
}

// ========== 节点巡检 ==========

// NodeHealthReport 节点健康报告
type NodeHealthReport struct {
	CheckTime      time.Time           `json:"checkTime"`
	Total          int                 `json:"total"`
	Ready          int                 `json:"ready"`
	NotReady       int                 `json:"notReady"`
	UnhealthyNodes []UnhealthyNodeInfo `json:"unhealthyNodes,omitempty"`
	NodeResources  []NodeResourceInfo  `json:"nodeResources"`
}

// UnhealthyNodeInfo 不健康节点信息
type UnhealthyNodeInfo struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Reasons []string `json:"reasons"`
}

// NodeResourceInfo 节点资源信息
type NodeResourceInfo struct {
	Name              string `json:"name"`
	Status            string `json:"status"`
	AllocatableCPU    string `json:"allocatableCPU"`
	AllocatableMemory string `json:"allocatableMemory"`
	PodCount          int    `json:"podCount"`
	PodCapacity       int    `json:"podCapacity"`
}

// ========== 工作负载巡检 ==========

// WorkloadHealthReport 工作负载健康报告
type WorkloadHealthReport struct {
	CheckTime             time.Time            `json:"checkTime"`
	UnhealthyDeployments  []UnhealthyWorkload  `json:"unhealthyDeployments,omitempty"`
	UnhealthyStatefulSets []UnhealthyWorkload  `json:"unhealthyStatefulSets,omitempty"`
	UnhealthyDaemonSets   []UnhealthyDaemonSet `json:"unhealthyDaemonSets,omitempty"`
	AbnormalPods          []AbnormalPodInfo    `json:"abnormalPods,omitempty"`
	HighRestartPods       []HighRestartPodInfo `json:"highRestartPods,omitempty"`
}

// UnhealthyWorkload 不健康的工作负载
type UnhealthyWorkload struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Replicas      int32  `json:"replicas"`
	ReadyReplicas int32  `json:"readyReplicas"`
}

// UnhealthyDaemonSet 不健康的 DaemonSet
type UnhealthyDaemonSet struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	DesiredScheduled int32  `json:"desiredScheduled"`
	NumberReady      int32  `json:"numberReady"`
}

// AbnormalPodInfo 异常 Pod 信息
type AbnormalPodInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status"`
	Node      string `json:"node"`
	Reason    string `json:"reason,omitempty"`
}

// HighRestartPodInfo 高重启 Pod 信息
type HighRestartPodInfo struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Restarts  int32  `json:"restarts"`
	Container string `json:"container"`
}

// ========== 事件巡检 ==========

// EventInspectReport 事件巡检报告
type EventInspectReport struct {
	CheckTime        time.Time             `json:"checkTime"`
	TotalWarnings    int                   `json:"totalWarnings"`
	AggregatedEvents []AggregatedEventInfo `json:"aggregatedEvents,omitempty"`
}

// AggregatedEventInfo 聚合事件信息
type AggregatedEventInfo struct {
	Reason         string    `json:"reason"`
	ObjectKind     string    `json:"objectKind"`
	ObjectName     string    `json:"objectName"`
	Namespace      string    `json:"namespace"`
	Message        string    `json:"message"`
	Count          int32     `json:"count"`
	LastOccurrence time.Time `json:"lastOccurrence"`
}
