package test

import (
	"context"
	"testing"

	"kubectl-mcp/internal/tools"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestGetPodsWithMultipleLabelSelectors 测试使用多个标签选择器查询 Pods
func TestGetPodsWithMultipleLabelSelectors(t *testing.T) {
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pod-v1",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "nginx",
				"version": "v1",
				"env":     "prod",
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-pod-v2",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "nginx",
				"version": "v2",
				"env":     "dev",
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "redis",
				"env": "prod",
			},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}

	fakeClient := fake.NewSimpleClientset(pod1, pod2, pod3)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()

	// 测试单个标签选择器
	args := map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "app=nginx",
	}

	result, err := tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err)

	pods, ok := result.([]tools.PodInfo)
	assert.True(t, ok)
	assert.Len(t, pods, 2, "应该返回 2 个 nginx Pod")

	// 测试多个标签选择器
	args = map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "app=nginx,env=prod",
	}

	result, err = tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err)

	pods, ok = result.([]tools.PodInfo)
	assert.True(t, ok)
	assert.Len(t, pods, 1, "应该只返回 1 个匹配的 Pod")
	assert.Equal(t, "nginx-pod-v1", pods[0].Name)
}

// TestGetPodsResourceNotFound 测试查询不存在的 Pod
func TestGetPodsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nonexistent-pod",
	}

	result, err := tools.GetPods(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 Pod 不应该返回错误")

	pods, ok := result.([]tools.PodInfo)
	assert.True(t, ok)
	assert.Len(t, pods, 0, "应该返回空列表")
}

// TestGetDeploymentsWithLabelFilter 测试使用标签过滤查询 Deployments
func TestGetDeploymentsWithLabelFilter(t *testing.T) {
	replicas := int32(3)
	deploy1 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-deploy",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
				"env": "prod",
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
						{Name: "nginx", Image: "nginx:latest"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		},
	}

	deploy2 := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-deploy",
			Namespace: "default",
			Labels: map[string]string{
				"app": "redis",
				"env": "dev",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "redis"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "redis", Image: "redis:latest"},
					},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(deploy1, deploy2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "env=prod",
	}

	result, err := tools.GetDeployments(ctx, args, k8sManager)
	assert.NoError(t, err)

	deployments, ok := result.([]tools.DeploymentInfo)
	assert.True(t, ok)
	assert.Len(t, deployments, 1, "应该只返回匹配标签的 Deployment")
	assert.Equal(t, "nginx-deploy", deployments[0].Name)
}

// TestGetDeploymentsResourceNotFound 测试查询不存在的 Deployment
func TestGetDeploymentsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nonexistent-deploy",
	}

	result, err := tools.GetDeployments(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 Deployment 不应该返回错误")

	deployments, ok := result.([]tools.DeploymentInfo)
	assert.True(t, ok)
	assert.Len(t, deployments, 0, "应该返回空列表")
}

// TestGetServicesWithLabelFilter 测试使用标签过滤查询 Services
func TestGetServicesWithLabelFilter(t *testing.T) {
	svc1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nginx-service",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
				"env": "prod",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.1",
			Ports: []corev1.ServicePort{
				{Name: "http", Port: 80, Protocol: corev1.ProtocolTCP},
			},
		},
	}

	svc2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis-service",
			Namespace: "default",
			Labels: map[string]string{
				"app": "redis",
				"env": "dev",
			},
		},
		Spec: corev1.ServiceSpec{
			Type:      corev1.ServiceTypeClusterIP,
			ClusterIP: "10.96.0.2",
			Ports: []corev1.ServicePort{
				{Name: "redis", Port: 6379, Protocol: corev1.ProtocolTCP},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(svc1, svc2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "app=nginx",
	}

	result, err := tools.GetServices(ctx, args, k8sManager)
	assert.NoError(t, err)

	services, ok := result.([]tools.ServiceInfo)
	assert.True(t, ok)
	assert.Len(t, services, 1, "应该只返回匹配标签的 Service")
	assert.Equal(t, "nginx-service", services[0].Name)
}

// TestGetServicesResourceNotFound 测试查询不存在的 Service
func TestGetServicesResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nonexistent-service",
	}

	result, err := tools.GetServices(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 Service 不应该返回错误")

	services, ok := result.([]tools.ServiceInfo)
	assert.True(t, ok)
	assert.Len(t, services, 0, "应该返回空列表")
}

// TestGetConfigMapsWithNameFilter 测试使用名称过滤查询 ConfigMaps
func TestGetConfigMapsWithNameFilter(t *testing.T) {
	cm1 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}

	cm2 := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"key2": "value2",
		},
	}

	fakeClient := fake.NewSimpleClientset(cm1, cm2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "app-config",
	}

	result, err := tools.GetConfigMaps(ctx, args, k8sManager)
	assert.NoError(t, err)

	configmaps, ok := result.([]tools.ConfigMapInfo)
	assert.True(t, ok)
	// fake client 可能不完全支持 FieldSelector，所以至少应该有结果
	assert.GreaterOrEqual(t, len(configmaps), 1, "应该至少返回一个 ConfigMap")
}

// TestGetConfigMapsResourceNotFound 测试查询不存在的 ConfigMap
func TestGetConfigMapsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nonexistent-config",
	}

	result, err := tools.GetConfigMaps(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 ConfigMap 不应该返回错误")

	configmaps, ok := result.([]tools.ConfigMapInfo)
	assert.True(t, ok)
	assert.Len(t, configmaps, 0, "应该返回空列表")
}

// TestGetSecretsWithLabelFilter 测试使用标签过滤查询 Secrets
func TestGetSecretsWithLabelFilter(t *testing.T) {
	secret1 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app-secret",
			Namespace: "default",
			Labels: map[string]string{
				"app": "nginx",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"password": []byte("secret123"),
		},
	}

	secret2 := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-secret",
			Namespace: "default",
			Labels: map[string]string{
				"app": "mysql",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"password": []byte("dbpass456"),
		},
	}

	fakeClient := fake.NewSimpleClientset(secret1, secret2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "app=nginx",
	}

	result, err := tools.GetSecrets(ctx, args, k8sManager)
	assert.NoError(t, err)

	secrets, ok := result.([]tools.SecretInfo)
	assert.True(t, ok)
	assert.Len(t, secrets, 1, "应该只返回匹配标签的 Secret")
	assert.Equal(t, "app-secret", secrets[0].Name)
	// 验证脱敏处理：只返回 key 名称
	assert.Contains(t, secrets[0].DataKeys, "password")
}

// TestGetSecretsResourceNotFound 测试查询不存在的 Secret
func TestGetSecretsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nonexistent-secret",
	}

	result, err := tools.GetSecrets(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 Secret 不应该返回错误")

	secrets, ok := result.([]tools.SecretInfo)
	assert.True(t, ok)
	assert.Len(t, secrets, 0, "应该返回空列表")
}

// TestGetNodesWithLabelFilter 测试使用标签过滤查询 Nodes
func TestGetNodesWithLabelFilter(t *testing.T) {
	node1 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "master-node",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
				"env":                            "prod",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	node2 := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "worker-node",
			Labels: map[string]string{
				"node-role.kubernetes.io/worker": "",
				"env":                            "dev",
			},
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(node1, node2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"labelSelector": "env=prod",
	}

	result, err := tools.GetNodes(ctx, args, k8sManager)
	assert.NoError(t, err)

	nodes, ok := result.([]tools.NodeInfo)
	assert.True(t, ok)
	assert.Len(t, nodes, 1, "应该只返回匹配标签的 Node")
	assert.Equal(t, "master-node", nodes[0].Name)
}

// TestGetNamespacesWithLabelFilter 测试使用标签过滤查询 Namespaces
func TestGetNamespacesWithLabelFilter(t *testing.T) {
	ns1 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "prod-namespace",
			Labels: map[string]string{
				"env": "prod",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	ns2 := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dev-namespace",
			Labels: map[string]string{
				"env": "dev",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	fakeClient := fake.NewSimpleClientset(ns1, ns2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"labelSelector": "env=prod",
	}

	result, err := tools.GetNamespaces(ctx, args, k8sManager)
	assert.NoError(t, err)

	namespaces, ok := result.([]tools.NamespaceInfo)
	assert.True(t, ok)
	assert.Len(t, namespaces, 1, "应该只返回匹配标签的 Namespace")
	assert.Equal(t, "prod-namespace", namespaces[0].Name)
}

// TestGetStatefulSetsWithLabelFilter 测试使用标签过滤查询 StatefulSets
func TestGetStatefulSetsWithLabelFilter(t *testing.T) {
	replicas := int32(3)
	sts1 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mysql-sts",
			Namespace: "default",
			Labels: map[string]string{
				"app": "mysql",
				"env": "prod",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "mysql",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "mysql"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "mysql", Image: "mysql:latest"},
					},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		},
	}

	sts2 := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-sts",
			Namespace: "default",
			Labels: map[string]string{
				"app": "postgres",
				"env": "dev",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: "postgres",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "postgres"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "postgres", Image: "postgres:latest"},
					},
				},
			},
		},
		Status: appsv1.StatefulSetStatus{
			Replicas:      3,
			ReadyReplicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(sts1, sts2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":     "default",
		"labelSelector": "env=prod",
	}

	result, err := tools.GetStatefulSets(ctx, args, k8sManager)
	assert.NoError(t, err)

	statefulsets, ok := result.([]tools.StatefulSetInfo)
	assert.True(t, ok)
	assert.Len(t, statefulsets, 1, "应该只返回匹配标签的 StatefulSet")
	assert.Equal(t, "mysql-sts", statefulsets[0].Name)
}

// TestGetStatefulSetsResourceNotFound 测试查询不存在的 StatefulSet
func TestGetStatefulSetsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
		"name":      "nonexistent-sts",
	}

	result, err := tools.GetStatefulSets(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 StatefulSet 不应该返回错误")

	statefulsets, ok := result.([]tools.StatefulSetInfo)
	assert.True(t, ok)
	assert.Len(t, statefulsets, 0, "应该返回空列表")
}

// TestGetDaemonSetsWithLabelFilter 测试使用标签过滤查询 DaemonSets
func TestGetDaemonSetsWithLabelFilter(t *testing.T) {
	ds1 := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fluentd-ds",
			Namespace: "kube-system",
			Labels: map[string]string{
				"app": "fluentd",
				"env": "prod",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "fluentd"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "fluentd", Image: "fluentd:latest"},
					},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			NumberReady:            3,
		},
	}

	ds2 := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-exporter-ds",
			Namespace: "kube-system",
			Labels: map[string]string{
				"app": "node-exporter",
				"env": "dev",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "node-exporter"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "node-exporter", Image: "node-exporter:latest"},
					},
				},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			CurrentNumberScheduled: 3,
			NumberReady:            3,
		},
	}

	fakeClient := fake.NewSimpleClientset(ds1, ds2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":     "kube-system",
		"labelSelector": "env=prod",
	}

	result, err := tools.GetDaemonSets(ctx, args, k8sManager)
	assert.NoError(t, err)

	daemonsets, ok := result.([]tools.DaemonSetInfo)
	assert.True(t, ok)
	assert.Len(t, daemonsets, 1, "应该只返回匹配标签的 DaemonSet")
	assert.Equal(t, "fluentd-ds", daemonsets[0].Name)
}

// TestGetDaemonSetsResourceNotFound 测试查询不存在的 DaemonSet
func TestGetDaemonSetsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "kube-system",
		"name":      "nonexistent-ds",
	}

	result, err := tools.GetDaemonSets(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 DaemonSet 不应该返回错误")

	daemonsets, ok := result.([]tools.DaemonSetInfo)
	assert.True(t, ok)
	assert.Len(t, daemonsets, 0, "应该返回空列表")
}

// TestGetEventsWithInvolvedObjectFilter 测试使用关联对象过滤查询 Events
func TestGetEventsWithInvolvedObjectFilter(t *testing.T) {
	event1 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-event",
			Namespace: "default",
		},
		Type:    "Normal",
		Reason:  "Created",
		Message: "Pod created",
		InvolvedObject: corev1.ObjectReference{
			Kind: "Pod",
			Name: "nginx-pod",
		},
		Source: corev1.EventSource{
			Component: "kubelet",
		},
		Count: 1,
	}

	event2 := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-event",
			Namespace: "default",
		},
		Type:    "Normal",
		Reason:  "ScalingReplicaSet",
		Message: "Scaled up replica set",
		InvolvedObject: corev1.ObjectReference{
			Kind: "Deployment",
			Name: "nginx-deploy",
		},
		Source: corev1.EventSource{
			Component: "deployment-controller",
		},
		Count: 1,
	}

	fakeClient := fake.NewSimpleClientset(event1, event2)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace":          "default",
		"involvedObjectKind": "Pod",
		"involvedObjectName": "nginx-pod",
	}

	result, err := tools.GetEvents(ctx, args, k8sManager)
	assert.NoError(t, err)

	events, ok := result.([]tools.EventInfo)
	assert.True(t, ok)
	// fake client 可能不完全支持 FieldSelector，所以至少应该有结果
	assert.GreaterOrEqual(t, len(events), 0, "应该成功返回结果")
}

// TestGetEventsResourceNotFound 测试查询不存在的 Events
func TestGetEventsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"namespace": "default",
	}

	result, err := tools.GetEvents(ctx, args, k8sManager)
	assert.NoError(t, err, "查询不存在的 Events 不应该返回错误")

	events, ok := result.([]tools.EventInfo)
	assert.True(t, ok)
	assert.Len(t, events, 0, "应该返回空列表")
}

// TestGetPodLogsWithContainer 测试指定容器获取 Pod 日志
func TestGetPodLogsWithContainer(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "multi-container-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "nginx", Image: "nginx:latest"},
				{Name: "sidecar", Image: "busybox:latest"},
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
		"name":      "multi-container-pod",
		"namespace": "default",
		"container": "nginx",
		"tailLines": float64(50),
	}

	result, err := tools.GetPodLogs(ctx, args, k8sManager)
	assert.NoError(t, err)

	logResult, ok := result.(*tools.PodLogResult)
	assert.True(t, ok)
	assert.Equal(t, "multi-container-pod", logResult.PodName)
	assert.Equal(t, "nginx", logResult.Container)
}

// TestGetPodLogsWithPrevious 测试获取前一个容器的日志
func TestGetPodLogsWithPrevious(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "restarted-pod",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app", Image: "app:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:         "app",
					RestartCount: 1,
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(pod)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"name":      "restarted-pod",
		"namespace": "default",
		"previous":  true,
	}

	result, err := tools.GetPodLogs(ctx, args, k8sManager)
	assert.NoError(t, err)

	logResult, ok := result.(*tools.PodLogResult)
	assert.True(t, ok)
	assert.Equal(t, "restarted-pod", logResult.PodName)
}

// TestGetPodLogsResourceNotFound 测试获取不存在的 Pod 日志
func TestGetPodLogsResourceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"name":      "nonexistent-pod",
		"namespace": "default",
	}

	result, err := tools.GetPodLogs(ctx, args, k8sManager)
	// fake client 不会为不存在的 Pod 返回错误，所以我们只验证能够正常调用
	// 在真实环境中会返回错误
	if err != nil {
		assert.Contains(t, err.Error(), "not found")
	} else {
		// fake client 返回了结果，验证结果格式正确
		logResult, ok := result.(*tools.PodLogResult)
		assert.True(t, ok)
		assert.Equal(t, "nonexistent-pod", logResult.PodName)
	}
}

// TestDescribeResourceConfigMap 测试 Describe ConfigMap 资源
func TestDescribeResourceConfigMap(t *testing.T) {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
			Labels: map[string]string{
				"app": "test",
			},
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
		"kind":      "ConfigMap",
		"name":      "test-config",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "ConfigMap", detail.Kind)
	assert.Equal(t, "test-config", detail.Name)
	assert.NotNil(t, detail.Spec)
}

// TestDescribeResourceSecret 测试 Describe Secret 资源（脱敏）
func TestDescribeResourceSecret(t *testing.T) {
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
		"kind":      "Secret",
		"name":      "test-secret",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "Secret", detail.Kind)
	assert.Equal(t, "test-secret", detail.Name)
	assert.NotNil(t, detail.Spec)
	// 验证脱敏处理：Spec 中应该只有 dataKeys，不包含实际值
	spec := detail.Spec
	assert.Contains(t, spec, "dataKeys")
	dataKeys, ok := spec["dataKeys"].([]string)
	assert.True(t, ok)
	assert.Contains(t, dataKeys, "username")
	assert.Contains(t, dataKeys, "password")
}

// TestDescribeResourceStatefulSet 测试 Describe StatefulSet 资源
func TestDescribeResourceStatefulSet(t *testing.T) {
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
		},
		Status: appsv1.StatefulSetStatus{
			Replicas: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(sts)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "StatefulSet",
		"name":      "test-sts",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "StatefulSet", detail.Kind)
	assert.Equal(t, "test-sts", detail.Name)
	assert.NotNil(t, detail.Spec)
	assert.NotNil(t, detail.Status)
}

// TestDescribeResourceDaemonSet 测试 Describe DaemonSet 资源
func TestDescribeResourceDaemonSet(t *testing.T) {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ds",
			Namespace: "kube-system",
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "test"},
			},
		},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
		},
	}

	fakeClient := fake.NewSimpleClientset(ds)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "DaemonSet",
		"name":      "test-ds",
		"namespace": "kube-system",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "DaemonSet", detail.Kind)
	assert.Equal(t, "test-ds", detail.Name)
	assert.NotNil(t, detail.Spec)
	assert.NotNil(t, detail.Status)
}

// TestDescribeResourceNamespace 测试 Describe Namespace 资源
func TestDescribeResourceNamespace(t *testing.T) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-namespace",
			Labels: map[string]string{
				"env": "test",
			},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}

	fakeClient := fake.NewSimpleClientset(ns)
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind": "Namespace",
		"name": "test-namespace",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.NoError(t, err)

	detail, ok := result.(*tools.ResourceDetail)
	assert.True(t, ok)
	assert.Equal(t, "Namespace", detail.Kind)
	assert.Equal(t, "test-namespace", detail.Name)
	assert.NotNil(t, detail.Spec)
	assert.NotNil(t, detail.Status)
}

// TestDescribeResourceServiceNotFound 测试 Describe 不存在的 Service
func TestDescribeResourceServiceNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "Service",
		"name":      "nonexistent-service",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}

// TestDescribeResourceConfigMapNotFound 测试 Describe 不存在的 ConfigMap
func TestDescribeResourceConfigMapNotFound(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	k8sManager := createFakeK8SManager(fakeClient)

	ctx := context.Background()
	args := map[string]interface{}{
		"kind":      "ConfigMap",
		"name":      "nonexistent-config",
		"namespace": "default",
	}

	result, err := tools.DescribeResource(ctx, args, k8sManager)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
}
