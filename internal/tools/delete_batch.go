package tools

import (
	"context"
	"fmt"
	"sync"

	"kubectl-mcp/internal/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteResources 批量删除资源
func DeleteResources(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, namespace, _ := getContextAndNamespace(args, k8sClient)
	if namespace == "" {
		namespace = "default"
	}
	kind, ok := args["kind"].(string)
	if !ok || kind == "" {
		return nil, fmt.Errorf("参数 'kind' 是必填的")
	}
	labelSelector := ""
	if ls, ok := args["labelSelector"].(string); ok {
		labelSelector = ls
	}
	var names []string
	if namesArg, ok := args["names"].([]interface{}); ok {
		for _, n := range namesArg {
			if name, ok := n.(string); ok {
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 && labelSelector == "" {
		return nil, fmt.Errorf("必须指定 'names' 或 'labelSelector' 参数")
	}
	clientset, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	result := &BatchDeleteResult{
		Kind:          kind,
		Namespace:     namespace,
		LabelSelector: labelSelector,
		SuccessList:   []string{},
		FailureList:   []DeleteError{},
	}
	if len(names) > 0 {
		result.TotalCount = len(names)
		return deleteBatchByNames(ctx, clientset, kind, namespace, names, result)
	}
	return deleteBatchByLabelSelector(ctx, clientset, kind, namespace, labelSelector, result)
}

func deleteBatchByNames(ctx context.Context, clientset *k8s.ClientSet, kind, namespace string, names []string, result *BatchDeleteResult) (*BatchDeleteResult, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, name := range names {
		wg.Add(1)
		go func(resourceName string) {
			defer wg.Done()
			err := deleteResourceByKind(ctx, clientset, kind, namespace, resourceName)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.FailureCount++
				result.FailureList = append(result.FailureList, DeleteError{Name: resourceName, Error: err.Error()})
			} else {
				result.SuccessCount++
				result.SuccessList = append(result.SuccessList, resourceName)
			}
		}(name)
	}
	wg.Wait()
	result.Message = fmt.Sprintf("批量删除 %s 完成：成功 %d 个，失败 %d 个", kind, result.SuccessCount, result.FailureCount)
	return result, nil
}

func deleteBatchByLabelSelector(ctx context.Context, clientset *k8s.ClientSet, kind, namespace, labelSelector string, result *BatchDeleteResult) (*BatchDeleteResult, error) {
	names, err := listResourceNamesByKind(ctx, clientset, kind, namespace, labelSelector)
	if err != nil {
		return nil, fmt.Errorf("查询资源列表失败: %w", err)
	}
	if len(names) == 0 {
		result.Message = fmt.Sprintf("没有找到符合条件的 %s 资源", kind)
		return result, nil
	}
	result.TotalCount = len(names)
	return deleteBatchByNames(ctx, clientset, kind, namespace, names, result)
}

func deleteResourceByKind(ctx context.Context, clientset *k8s.ClientSet, kind, namespace, name string) error {
	deleteOptions := metav1.DeleteOptions{}
	switch kind {
	case "Pod":
		return clientset.Clientset.CoreV1().Pods(namespace).Delete(ctx, name, deleteOptions)
	case "Deployment":
		return clientset.Clientset.AppsV1().Deployments(namespace).Delete(ctx, name, deleteOptions)
	case "StatefulSet":
		return clientset.Clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, deleteOptions)
	case "DaemonSet":
		return clientset.Clientset.AppsV1().DaemonSets(namespace).Delete(ctx, name, deleteOptions)
	case "Service":
		return clientset.Clientset.CoreV1().Services(namespace).Delete(ctx, name, deleteOptions)
	case "ConfigMap":
		return clientset.Clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, deleteOptions)
	case "Secret":
		return clientset.Clientset.CoreV1().Secrets(namespace).Delete(ctx, name, deleteOptions)
	default:
		return fmt.Errorf("不支持的资源类型: %s", kind)
	}
}

func listResourceNamesByKind(ctx context.Context, clientset *k8s.ClientSet, kind, namespace, labelSelector string) ([]string, error) {
	listOptions := metav1.ListOptions{LabelSelector: labelSelector}
	var names []string
	switch kind {
	case "Pod":
		list, err := clientset.Clientset.CoreV1().Pods(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "Deployment":
		list, err := clientset.Clientset.AppsV1().Deployments(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "StatefulSet":
		list, err := clientset.Clientset.AppsV1().StatefulSets(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "DaemonSet":
		list, err := clientset.Clientset.AppsV1().DaemonSets(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "Service":
		list, err := clientset.Clientset.CoreV1().Services(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "ConfigMap":
		list, err := clientset.Clientset.CoreV1().ConfigMaps(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	case "Secret":
		list, err := clientset.Clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
		if err != nil {
			return nil, err
		}
		for _, item := range list.Items {
			names = append(names, item.Name)
		}
	default:
		return nil, fmt.Errorf("不支持的资源类型: %s", kind)
	}
	return names, nil
}
