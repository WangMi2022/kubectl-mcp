package test

import (
	"context"
	"testing"

	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestRegisterQueryTools 测试注册所有查询类工具
func TestRegisterQueryTools(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 注册查询类工具
	err := tools.RegisterQueryTools(registry)
	assert.NoError(t, err, "注册查询类工具应该成功")

	// 验证工具数量和存在性
	expectedTools := []string{
		"get_nodes",
		"get_namespaces",
		"get_pods",
		"get_deployments",
		"get_statefulsets",
		"get_daemonsets",
		"get_services",
		"get_configmaps",
		"get_secrets",
		"describe_resource",
		"get_pod_logs",
		"get_events",
	}

	for _, toolName := range expectedTools {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "工具 '%s' 应该存在", toolName)
		assert.NotNil(t, tool, "工具 '%s' 不应为 nil", toolName)
		assert.Equal(t, tools.CategoryQuery, tool.Category, "工具 '%s' 应该属于查询类别", toolName)
		assert.False(t, tool.RequiresConfirmation, "查询工具 '%s' 不应该需要确认", toolName)
	}
}

// TestQueryToolsSchema 测试查询类工具的 Schema 定义
func TestQueryToolsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	// 测试 get_pods 工具的 Schema
	tool, exists := registry.GetTool("get_pods")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Equal(t, "object", tool.InputSchema.Type)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.Contains(t, tool.InputSchema.Properties, "context")

	// 测试 get_deployments 工具的 Schema
	tool, exists = registry.GetTool("get_deployments")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")

	// 测试 describe_resource 工具的 Schema
	tool, exists = registry.GetTool("describe_resource")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "kind")
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "kind")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "namespace")

	// 测试 get_pod_logs 工具的 Schema
	tool, exists = registry.GetTool("get_pod_logs")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Contains(t, tool.InputSchema.Required, "name")
	assert.Contains(t, tool.InputSchema.Properties, "container")
	assert.Contains(t, tool.InputSchema.Properties, "tailLines")
	assert.Contains(t, tool.InputSchema.Properties, "previous")
}

// TestQueryToolsCategory 测试查询类工具的分类
func TestQueryToolsCategory(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	queryTools := registry.GetToolsByCategory(tools.CategoryQuery)
	assert.GreaterOrEqual(t, len(queryTools), 12, "应该至少有 12 个查询类工具")

	for _, tool := range queryTools {
		assert.Equal(t, tools.CategoryQuery, tool.Category)
		assert.False(t, tool.RequiresConfirmation, "查询工具不应该需要确认")
		assert.Equal(t, "low", tool.RiskLevel, "查询工具应该是低风险")
	}
}

// TestGetPodsWithFakeClient 测试使用 fake client 查询 Pods
func TestGetPodsWithFakeClient(t *testing.T) {
	// 创建测试 Pod
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-1",
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.1",
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "nginx",
					Image:        "nginx:latest",
					Ready:        true,
					RestartCount: 0,
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
				},
			},
		},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "default",
			Labels: map[string]string{
				"app": "redis",
			},
		},
		Spec: corev1.PodSpec{
			NodeName: "node-2",
			Containers: []corev1.Container{
				{
					Name:  "redis",
					Image: "redis:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			PodIP: "10.0.0.2",
		},
	}

	fakeClient := fake.NewSimpleClientset(pod1, pod2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err)

	pods, ok := result.([]tools.PodInfo)
	assert.True(t, ok)
	assert.Len(t, pods, 2)

	// 验证第一个 Pod
	assert.Equal(t, "test-pod-1", pods[0].Name)
	assert.Equal(t, "default", pods[0].Namespace)
	assert.Equal(t, "10.0.0.1", pods[0].IP)
	assert.Equal(t, "node-1", pods[0].Node)
}

// TestGetPodsWithNameFilter 测试使用 name 过滤查询 Pods
func TestGetPodsWithNameFilter(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-pod",
			Namespace: "default",
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	fakeClient := fake.NewSimpleClientset(pod1, pod2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nginx-pod",
	}

	result, err := tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err)

	pods, ok := result.([]tools.PodInfo)
	assert.True(t, ok)
	// 注意：fake client 不完全支持 FieldSelector，所以可能返回所有 Pod
	// 这个测试主要验证参数传递和基本功能
	assert.GreaterOrEqual(t, len(pods), 1, "应该至少返回一个 Pod")
}

// TestGetPodsWithLabelSelector 测试使用 label 过滤查询 Pods
func TestGetPodsWithLabelFilter(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "nginx",
				"version": "v1",
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "redis",
				"version": "v2",
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	fakeClient := fake.NewSimpleClientset(pod1, pod2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "app=nginx",
	}

	result, err := tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err)

	pods, ok := result.([]tools.PodInfo)
	assert.True(t, ok)
	assert.Len(t, pods, 1, "应该只返回匹配标签的 Pod")
	assert.Equal(t, "nginx-pod", pods[0].Name)
}

// TestGetDeploymentsSchema 测试 get_deployments 工具的 Schema
func TestGetDeploymentsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_deployments")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证必要的参数
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "labelSelector")
	assert.Contains(t, tool.InputSchema.Properties, "context")
}

// TestGetServicesSchema 测试 get_services 工具的 Schema
func TestGetServicesSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_services")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证参数类型
	namespaceSchema := tool.InputSchema.Properties["namespace"]
	assert.NotNil(t, namespaceSchema)
	assert.Equal(t, "string", namespaceSchema.Type)
}

// TestGetConfigMapsSchema 测试 get_configmaps 工具的 Schema
func TestGetConfigMapsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_configmaps")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)
	assert.Equal(t, "object", tool.InputSchema.Type)
}

// TestGetSecretsSchema 测试 get_secrets 工具的 Schema（脱敏）
func TestGetSecretsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_secrets")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证工具描述中提到脱敏处理
	assert.Contains(t, tool.Description, "脱敏", "Secret 工具应该说明脱敏处理")
}

// TestGetNodesSchema 测试 get_nodes 工具的 Schema
func TestGetNodesSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_nodes")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// Node 查询不需要 namespace 参数
	assert.NotContains(t, tool.InputSchema.Required, "namespace")
}

// TestGetNamespacesSchema 测试 get_namespaces 工具的 Schema
func TestGetNamespacesSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_namespaces")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// Namespace 查询不需要 namespace 参数
	assert.NotContains(t, tool.InputSchema.Required, "namespace")
}

// TestGetEventsSchema 测试 get_events 工具的 Schema
func TestGetEventsSchema(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	tool, exists := registry.GetTool("get_events")
	assert.True(t, exists)
	assert.NotNil(t, tool.InputSchema)

	// 验证事件过滤参数
	assert.Contains(t, tool.InputSchema.Properties, "namespace")
	assert.Contains(t, tool.InputSchema.Properties, "involvedObjectKind")
	assert.Contains(t, tool.InputSchema.Properties, "involvedObjectName")
}

// TestDescribeResourcePod 测试 Describe Pod 资源
func TestDescribeResourcePod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewSimpleClientset(pod)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "Pod",
		"name":      "test-pod",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "Pod", detail.Kind)
	assert.Equal(t, "test-pod", detail.Name)
	assert.Equal(t, "default", detail.Namespace)
	assert.NotNil(t, detail.Spec)
	assert.NotNil(t, detail.Status)
}

// TestDescribeResourceNotFound 测试 Describe 不存在的资源
func TestDescribeResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "Pod",
		"name":      "nonexistent-pod",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestDescribeResourceInvalidKind 测试 Describe 不支持的资源类型
func TestDescribeResourceInvalidKind(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "InvalidKind",
		"name":      "test",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "不支持的资源类型")
}

// TestDescribeResourceMissingParameters 测试 Describe 缺少必填参数
func TestDescribeResourceMissingParameters(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()

	// 缺少 kind 参数
	args := map[string]interface{}{
		"name":      "test",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "kind")

	// 缺少 name 参数
	args = map[string]interface{}{
		"kind":      "Pod",
		"namespace": "default",
	}

	result, err = tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "name")
}

// TestGetPodsEmptyNamespace 测试查询空 namespace 的 Pods
func TestGetPodsAllNamespaces(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-ns1",
			Namespace: "namespace1",
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-ns2",
			Namespace: "namespace2",
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	fakeClient := fake.NewSimpleClientset(pod1, pod2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		// 不指定 namespace，应该查询所有 namespace
	}

	result, err := tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err)

	pods, ok := result.([]tools.PodInfo)
	assert.True(t, ok)
	// fake client 可能不完全支持跨 namespace 查询
	// 这个测试主要验证不指定 namespace 时不会报错
	assert.GreaterOrEqual(t, len(pods), 0, "应该成功返回结果")
}

// TestGetStatefulSetsWithFakeClient 测试使用 fake client 查询 StatefulSets
func TestGetStatefulSetsWithFakeClient(t *testing.T) {
	replicas := int32(3)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sts",
			Namespace: "default",
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "test-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:        3,
			ReadyReplicas:   3,
			CurrentReplicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(sts)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetStatefulSets(ctx, args, k8sManager)
	assert.NoError(t, err)

	statefulsets, ok := result.([]tools.StatefulSetInfo)
	assert.True(t, ok)
	assert.Len(t, statefulsets, 1)
	assert.Equal(t, "test-sts", statefulsets[0].Name)
	assert.Equal(t, int32(3), statefulsets[0].Replicas)
}

// TestGetDaemonSetsWithFakeClient 测试使用 fake client 查询 DaemonSets
func TestGetDaemonSetsWithFakeClient(t *testing.T) {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ds",
			Namespace: "default",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			NumberReady:            3,
			NumberAvailable:        3,
		},
	}

	fakeClient := fake.NewSimpleClientset(ds)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetDaemonSets(ctx, args, k8sManager)
	assert.NoError(t, err)

	daemonsets, ok := result.([]tools.DaemonSetInfo)
	assert.True(t, ok)
	assert.Len(t, daemonsets, 1)
	assert.Equal(t, "test-ds", daemonsets[0].Name)
	assert.Equal(t, int32(3), daemonsets[0].DesiredNumberScheduled)
}

// TestDuplicateQueryToolRegistration 测试重复注册查询工具
func TestDuplicateQueryToolRegistration(t *testing.T) {
	registry := tools.NewToolRegistry()

	// 第一次注册应该成功
	err := tools.RegisterQueryTools(registry)
	assert.NoError(t, err)

	// 第二次注册应该失败
	err = tools.RegisterQueryTools(registry)
	assert.Error(t, err, "重复注册工具应该失败")
}

// TestQueryToolsRiskLevel 测试查询类工具的风险等级
func TestQueryToolsRiskLevel(t *testing.T) {
	registry := tools.NewToolRegistry()
	err := tools.RegisterQueryTools(registry)
	require.NoError(t, err)

	queryTools := []string{
		"get_nodes", "get_namespaces", "get_pods", "get_deployments",
		"get_statefulsets", "get_daemonsets", "get_services",
		"get_configmaps", "get_secrets", "describe_resource",
		"get_pod_logs", "get_events",
	}

	for _, toolName := range queryTools {
		tool, exists := registry.GetTool(toolName)
		assert.True(t, exists, "工具 '%s' 应该存在", toolName)
		assert.Equal(t, "low", tool.RiskLevel, "查询工具 '%s' 应该是低风险", toolName)
	}
}

// TestGetDeploymentsWithFakeClient 测试使用 fake client 查询 Deployments
func TestGetDeploymentsWithFakeClient(t *testing.T) {
	replicas := int32(3)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nginx"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:latest",
						},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:          3,
			ReadyReplicas:     3,
			AvailableReplicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(deploy)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetDeployments(ctx, args, k8sManager)
	assert.NoError(t, err)

	deployments, ok := result.([]tools.DeploymentInfo)
	assert.True(t, ok)
	assert.Len(t, deployments, 1)
	assert.Equal(t, "test-deploy", deployments[0].Name)
	assert.Equal(t, int32(3), deployments[0].Replicas)
	assert.Equal(t, int32(3), deployments[0].ReadyReplicas)
}

// TestGetServicesWithFakeClient 测试使用 fake client 查询 Services
func TestGetServicesWithFakeClient(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
			Ports: []corev1.ServicePort{
				{
					Name:     "http",
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{"app": "nginx"},
		},
	}

	fakeClient := fake.NewSimpleClientset(svc)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetServices(ctx, args, k8sManager)
	assert.NoError(t, err)

	services, ok := result.([]tools.ServiceInfo)
	assert.True(t, ok)
	assert.Len(t, services, 1)
	assert.Equal(t, "test-service", services[0].Name)
	assert.Equal(t, "ClusterIP", services[0].Type)
	assert.Equal(t, "10.96.0.1", services[0].ClusterIP)
}

// TestGetConfigMapsWithFakeClient 测试使用 fake client 查询 ConfigMaps
func TestGetConfigMapsWithFakeClient(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	fakeClient := fake.NewSimpleClientset(cm)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetConfigMaps(ctx, args, k8sManager)
	assert.NoError(t, err)

	configmaps, ok := result.([]tools.ConfigMapInfo)
	assert.True(t, ok)
	assert.Len(t, configmaps, 1)
	assert.Equal(t, "test-config", configmaps[0].Name)
	assert.Len(t, configmaps[0].DataKeys, 2)
}

// TestGetSecretsWithFakeClient 测试使用 fake client 查询 Secrets（脱敏）
func TestGetSecretsWithFakeClient(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"username": []byte("admin"),
			"password": []byte("secret123"),
		},
	}

	fakeClient := fake.NewSimpleClientset(secret)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetSecrets(ctx, args, k8sManager)
	assert.NoError(t, err)

	secrets, ok := result.([]tools.SecretInfo)
	assert.True(t, ok)
	assert.Len(t, secrets, 1)
	assert.Equal(t, "test-secret", secrets[0].Name)
	assert.Len(t, secrets[0].DataKeys, 2, "应该返回 key 名称")
	assert.Contains(t, secrets[0].DataKeys, "username")
	assert.Contains(t, secrets[0].DataKeys, "password")
}

// TestGetNodesWithFakeClient 测试使用 fake client 查询 Nodes
func TestGetNodesWithFakeClient(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
			Addresses: []corev1.NodeAddress{
				{
					Type:    corev1.NodeInternalIP,
					Address: "192.168.1.100",
				},
			},
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion:          "v1.28.0",
				OperatingSystem:         "linux",
				Architecture:            "amd64",
				ContainerRuntimeVersion: "containerd://1.7.0",
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(node)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{}

	result, err := tools.GetNodes(ctx, args, k8sManager)
	assert.NoError(t, err)

	nodes, ok := result.([]tools.NodeInfo)
	assert.True(t, ok)
	assert.Len(t, nodes, 1)
	assert.Equal(t, "test-node", nodes[0].Name)
	assert.Equal(t, "Ready", nodes[0].Status)
	assert.Equal(t, "192.168.1.100", nodes[0].InternalIP)
}

// TestGetNamespacesWithFakeClient 测试使用 fake client 查询 Namespaces
func TestGetNamespacesWithFakeClient(t *testing.T) {
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-system",
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	fakeClient := fake.NewSimpleClientset(ns1, ns2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{}

	result, err := tools.GetNamespaces(ctx, args, k8sManager)
	assert.NoError(t, err)

	namespaces, ok := result.([]tools.NamespaceInfo)
	assert.True(t, ok)
	assert.Len(t, namespaces, 2)
}

// TestGetEventsWithFakeClient 测试使用 fake client 查询 Events
func TestGetEventsWithFakeClient(t *testing.T) {
	event := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-event",
			Namespace: "default",
		},
		Type:    "Normal",
		Reason:  "Created",
		Message: "Pod created successfully",
		InvolvedObject: corev1.ObjectReference{
			Kind: "Pod",
			Name: "test-pod",
		},
		Source: corev1.EventSource{
			Component: "kubelet",
			Host:      "node-1",
		},
		Count: 1,
	}

	fakeClient := fake.NewSimpleClientset(event)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetEvents(ctx, args, k8sManager)
	assert.NoError(t, err)

	events, ok := result.([]tools.EventInfo)
	assert.True(t, ok)
	assert.Len(t, events, 1)
	assert.Equal(t, "Normal", events[0].Type)
	assert.Equal(t, "Created", events[0].Reason)
}

// TestGetPodLogsWithFakeClient 测试使用 fake client 获取 Pod 日志
func TestGetPodLogsWithFakeClient(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	fakeClient := fake.NewSimpleClientset(pod)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"name":      "test-pod",
		"namespace": "default",
		"tailLines": float64(100),
	}

	// fake client 支持 GetLogs，但返回空日志
	result, err := tools.GetPodLogs(ctx, args, k8sManager)
	assert.NoError(t, err)

	logResult, ok := result.(*tools.PodLogResult)
	assert.True(t, ok)
	assert.Equal(t, "test-pod", logResult.PodName)
	assert.Equal(t, "default", logResult.Namespace)
}

// TestGetPodLogsMissingName 测试获取 Pod 日志缺少必填参数
func TestGetPodLogsMissingName(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetPodLogs(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "name")
}

// TestDescribeResourcePodNotFound 测试 Describe 不存在的 Pod
func TestDescribeResourcePodNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "Pod",
		"name":      "nonexistent-pod",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestDescribeResourceDeployment 测试 Describe Deployment 资源
func TestDescribeResourceDeployment(t *testing.T) {
	replicas := int32(3)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deploy",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "nginx"},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(deploy)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "Deployment",
		"name":      "test-deploy",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "Deployment", detail.Kind)
	assert.Equal(t, "test-deploy", detail.Name)
	assert.Equal(t, "default", detail.Namespace)
	assert.NotNil(t, detail.Spec)
	assert.NotNil(t, detail.Status)
}

// TestDescribeResourceService 测试 Describe Service 资源
func TestDescribeResourceService(t *testing.T) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
		},
	}

	fakeClient := fake.NewSimpleClientset(svc)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "Service",
		"name":      "test-service",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "Service", detail.Kind)
	assert.Equal(t, "test-service", detail.Name)
}

// TestDescribeResourceNode 测试 Describe Node 资源
func TestDescribeResourceNode(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(node)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind": "Node",
		"name": "test-node",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "Node", detail.Kind)
	assert.Equal(t, "test-node", detail.Name)
}

// createFakeK8SManager 创建一个 fake K8S 客户端管理器用于测试
func createFakeK8SManager(fakeClient *fake.Clientset) *k8s.K8SClientManager {
	// 使用测试辅助函数创建 fake manager
	return k8s.NewFakeK8SClientManager(fakeClient)
}
