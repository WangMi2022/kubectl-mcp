package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateDeployment 创建 Deployment
// 参数:
//   - name: Deployment 名称（必填）
//   - image: 容器镜像（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - replicas: 副本数（可选，默认 1）
//   - containerName: 容器名称（可选，默认与 Deployment 名称相同）
//   - containerPort: 容器端口（可选）
//   - labels: 标签（可选）
//   - env: 环境变量（可选）
//   - context: K8S context 名称（可选）
func CreateDeployment(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}

	// 获取必填参数
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("参数 'name' 是必填的")
	}

	image, ok := args["image"].(string)
	if !ok || image == "" {
		return nil, fmt.Errorf("参数 'image' 是必填的")
	}

	// 获取可选参数
	replicas := getInt32Arg(args, "replicas", 1)
	containerName := getStringArg(args, "containerName", name)
	containerPort := getInt32Arg(args, "containerPort", 0)
	labels := getMapStringString(args, "labels")
	envVars := buildEnvVars(args)
	resources := buildResourceRequirements(args)

	// 设置默认标签
	if labels == nil {
		labels = make(map[string]string)
	}
	if _, exists := labels["app"]; !exists {
		labels["app"] = name
	}

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 构建容器
	container := corev1.Container{
		Name:      containerName,
		Image:     image,
		Env:       envVars,
		Resources: resources,
	}

	// 添加容器端口
	if containerPort > 0 {
		container.Ports = []corev1.ContainerPort{
			{ContainerPort: containerPort},
		}
	}

	// 构建 Deployment 对象
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

	// 创建 Deployment
	created, err := clientset.Clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Deployment '%s/%s' 已存在", namespace, name)
		}
		return nil, fmt.Errorf("创建 Deployment 失败: %w", err)
	}

	return &CreateResult{
		Kind:      "Deployment",
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    "Created",
		Message:   fmt.Sprintf("Deployment '%s/%s' 创建成功，副本数: %d", namespace, name, replicas),
		CreatedAt: created.CreationTimestamp.Time,
	}, nil
}
