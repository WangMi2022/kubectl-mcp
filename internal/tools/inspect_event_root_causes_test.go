package tools

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRegisterInspectEventRootCausesTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))

	tool, exists := registry.GetTool("inspect_event_root_causes")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Equal(t, CategoryQuery, tool.Category)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "limit")
	assert.Contains(t, tool.InputSchema.Properties, "sinceMinutes")
}

func TestInspectEventRootCausesClassifiesFailedGetScale(t *testing.T) {
	now := time.Now()
	events := []corev1.Event{
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "demo", CreationTimestamp: metav1.NewTime(now)},
			Reason:     "FailedGetScale",
			Message:    "the HPA was unable to get the target's current scale: deployments.apps \"demo\" not found",
			Count:      3,
			Type:       corev1.EventTypeWarning,
			InvolvedObject: corev1.ObjectReference{
				Kind:      "HorizontalPodAutoscaler",
				Namespace: "demo",
				Name:      "demo-hpa",
			},
			LastTimestamp: metav1.NewTime(now),
		},
	}

	agg := aggregateRootCauseEvents(events, now.Add(-time.Hour))
	findings := buildRootCauseFindings(agg, 10)

	require.Len(t, findings, 1)
	assert.Equal(t, "HPATargetMissing", findings[0].FindingType)
	assert.Equal(t, "warning", findings[0].Severity)
	assert.Equal(t, int32(3), findings[0].Evidence[0].Count)
	assert.Equal(t, "demo-hpa", findings[0].AffectedObject.Name)
	assert.Equal(t, "read", findings[0].SafeActions[0].RiskLevel)
	assert.Equal(t, 1, buildDiagnosticSummary(findings).FindingsCount)
}

func TestClassifyEventFindingTypeKnownReasons(t *testing.T) {
	cases := map[string]string{
		"Unhealthy":        "ProbeFailure",
		"FailedScheduling": "ScheduleFailure",
		"FailedMount":      "VolumeMountFailure",
		"BackOff":          "ContainerRestart",
		"ImagePullBackOff": "ImagePullFailure",
		"ErrImagePull":     "ImagePullFailure",
	}
	for reason, expected := range cases {
		assert.Equal(t, expected, classifyEventFindingType(reason, "image pull failed"), reason)
	}
}
