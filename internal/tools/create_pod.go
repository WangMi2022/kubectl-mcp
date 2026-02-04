package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePod 创建 Pod
// 参数:
//   - name: Pod 名称（必填）
//   - image: 容器镜像（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - containerName: 容器名称（可选，默认与 Pod 名称相同）
//   - command: 容器启动命令（可选）
//   - args: 容器启动参数（可选）
//   - env: 环境变量（可选）
//   - labels: 标签（可选）
//   - restartPolicy: 重启策略（可选，默认 Always）
//   - context: K8S context 名称（可选）
func CreatePod(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
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
	labels := getMapStringString(args, "labels")
	containerName := getStringArg(args, "containerName", name)
	command := getStringSlice(args, "command")
	argsSlice := getStringSlice(args, "args")
	restartPolicy := getStringArg(args, "restartPolicy", "Always")
	envVars := buildEnvVars(args)
	resources := buildResourceRequirements(args)

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 构建容器
	container := corev1.Container{
		Name:      containerName,
		Image:     image,
		Command:   command,
		Args:      argsSlice,
		Env:       envVars,
		Resources: resources,
	}

	// 构建 Pod 对象
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers:    []corev1.Container{container},
			RestartPolicy: corev1.RestartPolicy(restartPolicy),
		},
	}

	// 创建 Pod
	created, err := clientset.Clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("Pod '%s/%s' 已存在", namespace, name)
		}
		return nil, fmt.Errorf("创建 Pod 失败: %w", err)
	}

	return &CreateResult{
		Kind:      "Pod",
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    string(created.Status.Phase),
		Message:   fmt.Sprintf("Pod '%s/%s' 创建成功", namespace, name),
		CreatedAt: created.CreationTimestamp.Time,
	}, nil
}
