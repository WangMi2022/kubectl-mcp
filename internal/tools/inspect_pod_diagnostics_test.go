package tools

import (
	"context"
	"testing"
	"time"

	"kubectl-mcp/internal/k8s"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterInspectPodDiagnosticsTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))

	tool, exists := registry.GetTool("inspect_pod_diagnostics")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Equal(t, CategoryQuery, tool.Category)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "podName")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.Contains(t, tool.InputSchema.Properties, "topN")
}

func TestInspectPodDiagnosticsFindsCrashLoopAndHighRestartWithoutLogs(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api-0", Labels: map[string]string{"app": "api"}},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{
					Name:         "api",
					RestartCount: 12,
					State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{
						Reason:  "CrashLoopBackOff",
						Message: "back-off restarting failed container",
					}},
				}},
			},
		},
		podWarningEvent("default", "api-0", "BackOff", "Back-off restarting failed container api"),
	)

	report := runInspectPodDiagnostics(t, client, map[string]interface{}{"namespace": "default", "podName": "api-0"})

	assertFinding(t, report.Findings, "CrashLoopBackOff", "Pod", "default", "api-0")
	assertFinding(t, report.Findings, "HighRestart", "Pod", "default", "api-0")
	for _, finding := range report.Findings {
		assert.NotContains(t, finding.Description, "日志")
		assert.NotEmpty(t, finding.SafeActions)
		assert.Equal(t, "read", finding.SafeActions[0].RiskLevel)
	}
}

func TestInspectPodDiagnosticsFindsSchedulingImagePullAndOOM(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pending"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "image-pull", Labels: map[string]string{"app": "bad"}},
			Status:     corev1.PodStatus{Phase: corev1.PodPending, ContainerStatuses: []corev1.ContainerStatus{{Name: "app", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ImagePullBackOff"}}}}},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "oom"},
			Status:     corev1.PodStatus{Phase: corev1.PodFailed, ContainerStatuses: []corev1.ContainerStatus{{Name: "app", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}}}}},
		},
		podWarningEvent("default", "pending", "FailedScheduling", "0/3 nodes are available: insufficient cpu"),
	)

	report := runInspectPodDiagnostics(t, client, map[string]interface{}{"namespace": "default", "topN": 10})

	assertFinding(t, report.Findings, "ScheduleFailure", "Pod", "default", "pending")
	assertFinding(t, report.Findings, "ImagePullFailure", "Pod", "default", "image-pull")
	assertFinding(t, report.Findings, "OOMKilled", "Pod", "default", "oom")
	assert.Equal(t, "Critical", report.Summary.Status)
}

func runInspectPodDiagnostics(t *testing.T, client *fake.Clientset, args map[string]interface{}) DiagnosticReport {
	t.Helper()
	manager := k8s.NewFakeK8SClientManager(client)
	result, err := InspectPodDiagnostics(context.Background(), args, manager)
	require.NoError(t, err)
	report, ok := result.(DiagnosticReport)
	require.True(t, ok)
	return report
}

func podWarningEvent(namespace, podName, reason, message string) *corev1.Event {
	now := metav1.NewTime(time.Now())
	return &corev1.Event{
		ObjectMeta:     metav1.ObjectMeta{Namespace: namespace, Name: podName + "." + reason},
		Type:           corev1.EventTypeWarning,
		Reason:         reason,
		Message:        message,
		Count:          2,
		LastTimestamp:  now,
		InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: namespace, Name: podName},
	}
}
