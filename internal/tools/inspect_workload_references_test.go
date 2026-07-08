package tools

import (
	"context"
	"testing"

	"kubectl-mcp/internal/k8s"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRegisterInspectWorkloadReferencesTool(t *testing.T) {
	registry := NewToolRegistry()
	require.NoError(t, RegisterInspectTools(registry))

	tool, exists := registry.GetTool("inspect_workload_references")
	require.True(t, exists)
	assert.False(t, tool.RequiresConfirmation)
	assert.Equal(t, CategoryQuery, tool.Category)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "includeIngress")
	assert.Contains(t, tool.InputSchema.Properties, "includeHPA")
	assert.Contains(t, tool.InputSchema.Properties, "includePDB")
}

func TestInspectWorkloadReferencesFindsMissingStatefulSetServices(t *testing.T) {
	client := fake.NewSimpleClientset(
		statefulSet("middleware", "mqbroker", "mqbroker", map[string]string{"app": "mqbroker"}),
		statefulSet("middleware", "mqnamesrv", "mqnamesrv", map[string]string{"app": "mqnamesrv"}),
	)
	report := runInspectWorkloadReferences(t, client, map[string]interface{}{
		"namespace":      "middleware",
		"includeIngress": false,
		"includeHPA":     false,
		"includePDB":     false,
	})

	assert.Equal(t, 2, report.Summary.FindingsCount)
	assertFinding(t, report.Findings, "MissingService", "StatefulSet", "middleware", "mqbroker")
	assertFinding(t, report.Findings, "MissingService", "StatefulSet", "middleware", "mqnamesrv")
	for _, finding := range report.Findings {
		require.NotEmpty(t, finding.AffectedObject.Name)
		require.NotEmpty(t, finding.RelatedObjects)
		require.NotEmpty(t, finding.Evidence)
		require.NotEmpty(t, finding.Recommendation)
	}
}

func TestInspectWorkloadReferencesFindsServiceSelectorAndEndpointProblems(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api"},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": "api"},
				Ports:    []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt(8080)}},
			},
		},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "worker", Labels: map[string]string{"app": "worker"}}},
		&corev1.Endpoints{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "api"}},
	)
	report := runInspectWorkloadReferences(t, client, map[string]interface{}{
		"namespace":      "default",
		"includeIngress": false,
		"includeHPA":     false,
		"includePDB":     false,
	})

	assertFinding(t, report.Findings, "ServiceNoMatchedPods", "Service", "default", "api")
	assertFinding(t, report.Findings, "ServiceNoEndpoints", "Service", "default", "api")
}

func TestInspectWorkloadReferencesFindsIngressHPAPDBProblems(t *testing.T) {
	client := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web"},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{Name: "http", Port: 80}},
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-ing"},
			Spec: networkingv1.IngressSpec{Rules: []networkingv1.IngressRule{{
				Host: "web.example.test",
				IngressRuleValue: networkingv1.IngressRuleValue{HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
					Path: "/",
					Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{
						Name: "web",
						Port: networkingv1.ServiceBackendPort{Number: 81},
					}},
				}}}},
			}}},
		},
		&autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-hpa"},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{Kind: "Deployment", Name: "web-missing", APIVersion: "apps/v1"},
			},
		},
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "web-pdb"},
			Spec:       policyv1.PodDisruptionBudgetSpec{Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}}},
		},
	)
	report := runInspectWorkloadReferences(t, client, map[string]interface{}{
		"namespace":      "default",
		"includeIngress": true,
		"includeHPA":     true,
		"includePDB":     true,
	})

	assertFinding(t, report.Findings, "IngressBackendPortMissing", "Ingress", "default", "web-ing")
	assertFinding(t, report.Findings, "HPATargetMissing", "HorizontalPodAutoscaler", "default", "web-hpa")
	assertFinding(t, report.Findings, "PDBNoMatchedPods", "PodDisruptionBudget", "default", "web-pdb")
}

func runInspectWorkloadReferences(t *testing.T, client *fake.Clientset, args map[string]interface{}) DiagnosticReport {
	t.Helper()
	manager := k8s.NewFakeK8SClientManager(client)
	result, err := InspectWorkloadReferences(context.Background(), args, manager)
	require.NoError(t, err)
	report, ok := result.(DiagnosticReport)
	require.True(t, ok)
	return report
}

func assertFinding(t *testing.T, findings []DiagnosticFinding, findingType, kind, namespace, name string) {
	t.Helper()
	for _, finding := range findings {
		if finding.FindingType == findingType &&
			finding.AffectedObject.Kind == kind &&
			finding.AffectedObject.Namespace == namespace &&
			finding.AffectedObject.Name == name {
			return
		}
	}
	t.Fatalf("finding not found: type=%s affected=%s/%s/%s in %#v", findingType, kind, namespace, name, findings)
}

func statefulSet(namespace, name, serviceName string, labels map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: serviceName,
			Selector:    &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
			},
		},
	}
}
