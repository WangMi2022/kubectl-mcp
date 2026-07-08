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

func TestRegisterInspectStorageDiagnosticsTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))
	tool, exists := registry.GetTool("inspect_storage_diagnostics")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
}

func TestInspectStorageDiagnosticsFindsPVCAndPVProblems(t *testing.T) {
	sc := "missing-sc"
	client := fake.NewSimpleClientset(
		&corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "data"}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: &sc}, Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending}},
		&corev1.PersistentVolume{ObjectMeta: metav1.ObjectMeta{Name: "pv-old"}, Status: corev1.PersistentVolumeStatus{Phase: corev1.VolumeReleased}},
	)
	report := runInspectStorageDiagnostics(t, client, map[string]interface{}{"namespace": "default"})
	assertFinding(t, report.Findings, "PVCPending", "PersistentVolumeClaim", "default", "data")
	assertFinding(t, report.Findings, "StorageClassMissing", "PersistentVolumeClaim", "default", "data")
	assertFinding(t, report.Findings, "PVReleased", "PersistentVolume", "", "pv-old")
}

func TestInspectStorageDiagnosticsFindsMountEvents(t *testing.T) {
	client := fake.NewSimpleClientset(&corev1.Event{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "pod.mount"}, Type: corev1.EventTypeWarning, Reason: "FailedMount", Message: "MountVolume.SetUp failed for volume config", Count: 4, LastTimestamp: metav1.NewTime(time.Now()), InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "default", Name: "api-0"}})
	report := runInspectStorageDiagnostics(t, client, map[string]interface{}{"namespace": "default"})
	assertFinding(t, report.Findings, "VolumeMountFailure", "Pod", "default", "api-0")
}

func runInspectStorageDiagnostics(t *testing.T, client *fake.Clientset, args map[string]interface{}) DiagnosticReport {
	t.Helper()
	manager := k8s.NewFakeK8SClientManager(client)
	result, err := InspectStorageDiagnostics(context.Background(), args, manager)
	require.NoError(t, err)
	report, ok := result.(DiagnosticReport)
	require.True(t, ok)
	return report
}
