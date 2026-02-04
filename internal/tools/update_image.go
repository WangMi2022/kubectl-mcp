package tools

import (
	"context"
	"fmt"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UpdateDeploymentImage 更新 Deployment 镜像
// 参数:
//   - name: Deployment 名称（必填）
//   - namespace: 命名空间（可选，默认 default）
//   - image: 新镜像（必填）
//   - containerName: 容器名称（可选，默认更新第一个容器）
//   - context: K8S context 名称（可选）
func UpdateDeploymentImage(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
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

	containerName := getStringArg(args, "containerName", "")

	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}

	// 获取当前 Deployment
	deployment, err := clientset.Clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取 Deployment '%s/%s' 失败: %w", namespace, name, err)
	}

	// 查找并更新容器镜像
	containers := deployment.Spec.Template.Spec.Containers
	if len(containers) == 0 {
		return nil, fmt.Errorf("Deployment '%s/%s' 没有容器", namespace, name)
	}

	updated := false
	oldImage := ""
	targetContainer := ""

	// 如果指定了容器名称，更新指定容器
	if containerName != "" {
		for i := range containers {
			if containers[i].Name == containerName {
				oldImage = containers[i].Image
				containers[i].Image = image
				targetContainer = containerName
				updated = true
				break
			}
		}
		if !updated {
			return nil, fmt.Errorf("Deployment '%s/%s' 中未找到容器 '%s'", namespace, name, containerName)
		}
	} else {
		// 未指定容器名称，更新第一个容器
		oldImage = containers[0].Image
		containers[0].Image = image
		targetContainer = containers[0].Name
		updated = true
	}

	// 更新 Deployment
	deployment.Spec.Template.Spec.Containers = containers
	_, err = clientset.Clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("更新 Deployment '%s/%s' 镜像失败: %w", namespace, name, err)
	}

	return &UpdateResult{
		Kind:      "Deployment",
		Name:      name,
		Namespace: namespace,
		Action:    "UpdateImage",
		Status:    "Success",
		Message:   fmt.Sprintf("Deployment '%s/%s' 容器 '%s' 镜像从 '%s' 更新为 '%s'", namespace, name, targetContainer, oldImage, image),
		OldValue:  oldImage,
		NewValue:  image,
	}, nil
}
