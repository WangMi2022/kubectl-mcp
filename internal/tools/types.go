package tools

import "time"

// ========== 数据结构定义 ==========

// NodeInfo 节点信息
type NodeInfo struct {
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	Roles             []string          `json:"roles"`
	Version           string            `json:"version"`
	InternalIP        string            `json:"internalIP"`
	ExternalIP        string            `json:"externalIP,omitempty"`
	OS                string            `json:"os"`
	Architecture      string            `json:"architecture"`
	ContainerRuntime  string            `json:"containerRuntime"`
	Labels            map[string]string `json:"labels"`
	Taints            []string          `json:"taints,omitempty"`
	CreatedAt         time.Time         `json:"createdAt"`
	AllocatableCPU    string            `json:"allocatableCPU"`
	AllocatableMemory string            `json:"allocatableMemory"`
}

// NamespaceInfo 命名空间信息
type NamespaceInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"createdAt"`
}

// PodInfo Pod 信息
type PodInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Status     string            `json:"status"`
	Phase      string            `json:"phase"`
	IP         string            `json:"ip"`
	Node       string            `json:"node"`
	Labels     map[string]string `json:"labels"`
	Containers []ContainerInfo   `json:"containers"`
	CreatedAt  time.Time         `json:"createdAt"`
	Restarts   int32             `json:"restarts"`
}

// ContainerInfo 容器信息
type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state"`
}

// DeploymentInfo Deployment 信息
type DeploymentInfo struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Replicas          int32             `json:"replicas"`
	ReadyReplicas     int32             `json:"readyReplicas"`
	AvailableReplicas int32             `json:"availableReplicas"`
	UpdatedReplicas   int32             `json:"updatedReplicas"`
	Images            []string          `json:"images"`
	Labels            map[string]string `json:"labels"`
	Selector          map[string]string `json:"selector"`
	CreatedAt         time.Time         `json:"createdAt"`
	Strategy          string            `json:"strategy"`
}

// StatefulSetInfo StatefulSet 信息
type StatefulSetInfo struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Replicas        int32             `json:"replicas"`
	ReadyReplicas   int32             `json:"readyReplicas"`
	CurrentReplicas int32             `json:"currentReplicas"`
	Images          []string          `json:"images"`
	Labels          map[string]string `json:"labels"`
	ServiceName     string            `json:"serviceName"`
	CreatedAt       time.Time         `json:"createdAt"`
}

// DaemonSetInfo DaemonSet 信息
type DaemonSetInfo struct {
	Name                   string            `json:"name"`
	Namespace              string            `json:"namespace"`
	DesiredNumberScheduled int32             `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32             `json:"currentNumberScheduled"`
	NumberReady            int32             `json:"numberReady"`
	NumberAvailable        int32             `json:"numberAvailable"`
	Images                 []string          `json:"images"`
	Labels                 map[string]string `json:"labels"`
	NodeSelector           map[string]string `json:"nodeSelector,omitempty"`
	CreatedAt              time.Time         `json:"createdAt"`
}

// ServiceInfo Service 信息
type ServiceInfo struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Type       string            `json:"type"`
	ClusterIP  string            `json:"clusterIP"`
	ExternalIP string            `json:"externalIP,omitempty"`
	Ports      []ServicePortInfo `json:"ports"`
	Labels     map[string]string `json:"labels"`
	Selector   map[string]string `json:"selector"`
	CreatedAt  time.Time         `json:"createdAt"`
}

// ServicePortInfo Service 端口信息
type ServicePortInfo struct {
	Name       string `json:"name,omitempty"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort"`
	NodePort   int32  `json:"nodePort,omitempty"`
	Protocol   string `json:"protocol"`
}

// ConfigMapInfo ConfigMap 信息
type ConfigMapInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
	DataKeys  []string          `json:"dataKeys"`
	CreatedAt time.Time         `json:"createdAt"`
}

// SecretInfo Secret 信息（脱敏）
type SecretInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	Labels    map[string]string `json:"labels"`
	DataKeys  []string          `json:"dataKeys"`
	CreatedAt time.Time         `json:"createdAt"`
}

// EventInfo 事件信息
type EventInfo struct {
	Name           string    `json:"name"`
	Namespace      string    `json:"namespace"`
	Type           string    `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Source         string    `json:"source"`
	InvolvedObject string    `json:"involvedObject"`
	Count          int32     `json:"count"`
	FirstTimestamp time.Time `json:"firstTimestamp"`
	LastTimestamp  time.Time `json:"lastTimestamp"`
}

// ResourceDetail 资源详情
type ResourceDetail struct {
	Kind        string                 `json:"kind"`
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace,omitempty"`
	Labels      map[string]string      `json:"labels,omitempty"`
	Annotations map[string]string      `json:"annotations,omitempty"`
	Spec        map[string]interface{} `json:"spec,omitempty"`
	Status      map[string]interface{} `json:"status,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
}

// PodLogResult Pod 日志结果
type PodLogResult struct {
	PodName   string `json:"podName"`
	Namespace string `json:"namespace"`
	Container string `json:"container,omitempty"`
	Logs      string `json:"logs"`
}

// ========== 创建操作结果 ==========

// CreateResult 创建操作结果
type CreateResult struct {
	Kind      string    `json:"kind"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace,omitempty"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

// ========== 修改操作结果 ==========

// UpdateResult 修改操作结果
type UpdateResult struct {
	Kind      string      `json:"kind"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace,omitempty"`
	Action    string      `json:"action"` // Scale, UpdateImage, Restart, Patch, Apply
	Status    string      `json:"status"`
	Message   string      `json:"message"`
	OldValue  string      `json:"oldValue,omitempty"`
	NewValue  string      `json:"newValue,omitempty"`
	Details   interface{} `json:"details,omitempty"`
}

// ContainerSpec 容器规格定义
type ContainerSpec struct {
	Name            string            `json:"name"`
	Image           string            `json:"image"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	Ports           []ContainerPort   `json:"ports,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	Resources       *ResourceSpec     `json:"resources,omitempty"`
	ImagePullPolicy string            `json:"imagePullPolicy,omitempty"`
}

// ContainerPort 容器端口定义
type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

// ResourceSpec 资源规格定义
type ResourceSpec struct {
	LimitsCPU      string `json:"limitsCPU,omitempty"`
	LimitsMemory   string `json:"limitsMemory,omitempty"`
	RequestsCPU    string `json:"requestsCPU,omitempty"`
	RequestsMemory string `json:"requestsMemory,omitempty"`
}

// ========== 删除操作结果 ==========

// DeleteResult 删除操作结果
type DeleteResult struct {
	Kind      string                 `json:"kind"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace,omitempty"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Force     bool                   `json:"force,omitempty"`
	Cascade   bool                   `json:"cascade,omitempty"`
	DeletedAt time.Time              `json:"deletedAt"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// BatchDeleteResult 批量删除操作结果
type BatchDeleteResult struct {
	Kind          string        `json:"kind"`
	Namespace     string        `json:"namespace,omitempty"`
	TotalCount    int           `json:"totalCount"`
	SuccessCount  int           `json:"successCount"`
	FailureCount  int           `json:"failureCount"`
	SuccessList   []string      `json:"successList"`
	FailureList   []DeleteError `json:"failureList,omitempty"`
	LabelSelector string        `json:"labelSelector,omitempty"`
	Message       string        `json:"message"`
}

// DeleteError 删除错误信息
type DeleteError struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}
