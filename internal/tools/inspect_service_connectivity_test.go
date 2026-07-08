package tools

import (
	"context"
	"testing"

	"kubectl-mcp/internal/k8s"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterInspectServiceConnectivityTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))
	tool, exists := registry.GetTool("inspect_service_connectivity")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Contains(t, tool.InputSchema.Properties, "serviceName")
}

func TestInspectServiceConnectivityFindsSelectorAndEndpointProblems(t *testing.T) {
	client := fake.NewSimpleClientset(serviceForConnectivity("default", "api", map[string]string{"app": "api"}, intstr.FromInt(8080)))
	report := runInspectServiceConnectivity(t, client, map[string]interface{}{"namespace": "default", "serviceName": "api"})
	assertFinding(t, report.Findings, "ServiceNoMatchedPods", "Service", "default", "api")
}

func TestInspectServiceConnectivityFindsTargetPortMismatchAndNotReady(t *testing.T) {
	client := fake.NewSimpleClientset(
		serviceForConnectivity("default", "web", map[string]string{"app": "web"}, intstr.FromInt(8080)),
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-0", Labels: map[string]string{"app": "web"}}, Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 9090}}}}}, Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}}}},
		&corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"}},
	)
	report := runInspectServiceConnectivity(t, client, map[string]interface{}{"namespace": "default", "serviceName": "web"})
	assertFinding(t, report.Findings, "ServiceTargetsNotReady", "Service", "default", "web")
	assertFinding(t, report.Findings, "ServiceNoEndpoints", "Service", "default", "web")
	assertFinding(t, report.Findings, "TargetPortMismatch", "Service", "default", "web")
}

func runInspectServiceConnectivity(t *testing.T, client *fake.Clientset, args map[string]interface{}) DiagnosticReport {
	t.Helper()
	manager := k8s.NewFakeK8SClientManager(client)
	result, err := InspectServiceConnectivity(context.Background(), args, manager)
	require.NoError(t, err)
	report, ok := result.(DiagnosticReport)
	require.True(t, ok)
	return report
}

func serviceForConnectivity(namespace, name string, selector map[string]string, targetPort intstr.IntOrString) *corev1.Service {
	return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}, Spec: corev1.ServiceSpec{Selector: selector, Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: targetPort}}}}
}
