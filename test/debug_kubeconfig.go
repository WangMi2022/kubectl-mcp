package main

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	kubeconfigPath := "./kubeconfig.yaml"
	if len(os.Args) > 1 {
		kubeconfigPath = os.Args[1]
	}

	fmt.Printf("使用 kubeconfig: %s\n", kubeconfigPath)

	// 加载 kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("加载 kubeconfig 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("API Server: %s\n", config.Host)

	// 创建客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		os.Exit(1)
	}

	// 获取所有 Service
	services, err := clientset.CoreV1().Services(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("获取 Service 失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n共找到 %d 个 Service:\n", len(services.Items))
	
	// 统计 NodePort 类型
	nodePortCount := 0
	for _, svc := range services.Items {
		if svc.Spec.Type == "NodePort" {
			nodePortCount++
			// 打印 NodePort 信息
			for _, port := range svc.Spec.Ports {
				if port.NodePort > 0 {
					fmt.Printf("  [NodePort] %s/%s - %d:%d\n", svc.Namespace, svc.Name, port.Port, port.NodePort)
				}
			}
		}
	}
	
	fmt.Printf("\n其中 NodePort 类型: %d 个\n", nodePortCount)
}
