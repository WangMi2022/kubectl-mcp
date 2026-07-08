package tools

import "time"

// DiagnosticScope describes the inspected target scope for structured diagnostics.
type DiagnosticScope struct {
	Cluster    string `json:"cluster,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	ObjectKind string `json:"objectKind,omitempty"`
	ObjectName string `json:"objectName,omitempty"`
}

// DiagnosticSummary is the common summary contract returned by inspect_* tools.
type DiagnosticSummary struct {
	Status        string `json:"status"`
	Score         int    `json:"score"`
	FindingsCount int    `json:"findingsCount"`
	CriticalCount int    `json:"criticalCount"`
	WarningCount  int    `json:"warningCount"`
}

// DiagnosticObjectRef identifies an affected or related Kubernetes object.
type DiagnosticObjectRef struct {
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// DiagnosticEvidence captures a compact evidence item for a finding.
type DiagnosticEvidence struct {
	Source   string `json:"source"`
	Message  string `json:"message"`
	Count    int32  `json:"count,omitempty"`
	LastSeen string `json:"lastSeen,omitempty"`
}

// DiagnosticSafeAction suggests follow-up actions with the expected backend risk tier.
type DiagnosticSafeAction struct {
	Action    string `json:"action"`
	RiskLevel string `json:"riskLevel"`
	Reason    string `json:"reason"`
}

// DiagnosticFinding is the common finding contract returned by inspect_* tools.
type DiagnosticFinding struct {
	ID             string                 `json:"id"`
	Severity       string                 `json:"severity"`
	FindingType    string                 `json:"findingType"`
	Title          string                 `json:"title"`
	Description    string                 `json:"description"`
	AffectedObject DiagnosticObjectRef    `json:"affectedObject"`
	Evidence       []DiagnosticEvidence   `json:"evidence,omitempty"`
	Recommendation string                 `json:"recommendation,omitempty"`
	RelatedObjects []DiagnosticObjectRef  `json:"relatedObjects,omitempty"`
	SafeActions    []DiagnosticSafeAction `json:"safeActions,omitempty"`
}

// DiagnosticReport is the shared structured protocol for new inspect_* tools.
type DiagnosticReport struct {
	CheckTime time.Time           `json:"checkTime"`
	Scope     DiagnosticScope     `json:"scope"`
	Summary   DiagnosticSummary   `json:"summary"`
	Findings  []DiagnosticFinding `json:"findings"`
}
