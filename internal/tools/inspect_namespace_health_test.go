package tools

import (
	"context"
	"testing"

	"kubectl-mcp/internal/k8s"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterInspectNamespaceHealthTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))
	tool, exists := registry.GetTool("inspect_namespace_health")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
}

func TestInspectNamespaceHealthAggregatesNamespaceFindings(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bad"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api"}, Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "api"}, Ports: []corev1.ServicePort{{Port: 80}}}},
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data"}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}},
		&corev1.Event{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "bad.event"}, Type: corev1.EventTypeWarning, Reason: "FailedScheduling", Message: "insufficient cpu", Count: 2, InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "bad"}},
	)
	report := runInspectNamespaceHealth(t, client, map[string]interface{}{"namespace": "default"})
	assertFinding(t, report.Findings, "NamespaceAbnormalPod", "Pod", "default", "bad")
	assertFinding(t, report.Findings, "NamespaceServiceNoEndpoints", "Service", "default", "api")
	assertFinding(t, report.Findings, "NamespacePVCPending", "PersistentVolumeClaim", "default", "data")
	assertFinding(t, report.Findings, "NamespaceWarningEvent", "Pod", "default", "bad")
}

func runInspectNamespaceHealth(t *testing.T, client *fake.Clientset, args map[string]interface{}) DiagnosticReport {
	t.Helper()
	manager := k8s.NewFakeK8SClientManager(client)
	result, err := InspectNamespaceHealth(context.Background(), args, manager)
	require.NoError(t, err)
	report, ok := result.(DiagnosticReport)
	require.True(t, ok)
	return report
}
