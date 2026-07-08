package tools

import (
	"context"
	"testing"

	"kubectl-mcp/internal/k8s"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterInspectNodePressureTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))
	tool, exists := registry.GetTool("inspect_node_pressure")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Contains(t, tool.InputSchema.Properties, "nodeName")
}

func TestInspectNodePressureFindsPressureAndHighRequestsAndPods(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "node-1"}, Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1000m"), corev1.ResourceMemory: resource.MustParse("1Gi")}, Conditions: []corev1.NodeCondition{{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue, Reason: "KubeletHasInsufficientMemory", Message: "memory pressure"}}}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api"}, Spec: corev1.PodSpec{NodeName: "node-1", Containers: []corev1.Container{{Name: "api", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("950m"), corev1.ResourceMemory: resource.MustParse("950Mi")}}}}}, Status: corev1.PodStatus{Phase: corev1.PodRunning, ContainerStatuses: []corev1.ContainerStatus{{Name: "api", RestartCount: 12}}}},
	)
	report := runInspectNodePressure(t, client, map[string]interface{}{"nodeName": "node-1"})
	assertFinding(t, report.Findings, "MemoryPressure", "Node", "", "node-1")
	assertFinding(t, report.Findings, "NodeCPURequestsHigh", "Node", "", "node-1")
	assertFinding(t, report.Findings, "NodeAbnormalPod", "Node", "", "node-1")
}

func runInspectNodePressure(t *testing.T, client *fake.Clientset, args map[string]interface{}) DiagnosticReport {
	t.Helper()
	manager := k8s.NewFakeK8SClientManager(client)
	result, err := InspectNodePressure(context.Background(), args, manager)
	require.NoError(t, err)
	report, ok := result.(DiagnosticReport)
	require.True(t, ok)
	return report
}
