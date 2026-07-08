package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"kubectl-mcp/internal/k8s"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TopMetricsReport struct {
	CheckTime        time.Time   `json:"checkTime"`
	MetricsAvailable bool        `json:"metricsAvailable"`
	Error            string      `json:"error,omitempty"`
	Items            interface{} `json:"items"`
}
type TopPodMetricItem struct {
	Namespace   string `json:"namespace"`
	Pod         string `json:"pod"`
	CPU         string `json:"cpu"`
	CPUMilli    int64  `json:"cpuMilli"`
	Memory      string `json:"memory"`
	MemoryBytes int64  `json:"memoryBytes"`
}
type TopNodeMetricItem struct {
	Node        string `json:"node"`
	CPU         string `json:"cpu"`
	CPUMilli    int64  `json:"cpuMilli"`
	Memory      string `json:"memory"`
	MemoryBytes int64  `json:"memoryBytes"`
}
type metricsList struct {
	Items []json.RawMessage `json:"items"`
}
type podMetric struct {
	Metadata   metav1.ObjectMeta `json:"metadata"`
	Containers []struct {
		Usage map[string]string `json:"usage"`
	} `json:"containers"`
}
type nodeMetric struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Usage    map[string]string `json:"usage"`
}

func TopPods(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, defaultNS, _ := getContextAndNamespace(args, k8sClient)
	ns := strings.TrimSpace(stringArg(args, "namespace", defaultNS))
	limit := intArg(args, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	cs, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	path := "/apis/metrics.k8s.io/v1beta1/pods"
	if ns != "" {
		path = "/apis/metrics.k8s.io/v1beta1/namespaces/" + ns + "/pods"
	}
	raw, err := cs.Clientset.CoreV1().RESTClient().Get().AbsPath(path).DoRaw(ctx)
	if err != nil {
		return TopMetricsReport{CheckTime: time.Now(), MetricsAvailable: false, Error: fmt.Sprintf("metrics.k8s.io unavailable: %v", err), Items: []TopPodMetricItem{}}, nil
	}
	items, err := parseTopPods(raw, limit)
	if err != nil {
		return nil, err
	}
	return TopMetricsReport{CheckTime: time.Now(), MetricsAvailable: true, Items: items}, nil
}
func TopNodes(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, _, _ := getContextAndNamespace(args, k8sClient)
	limit := intArg(args, "limit", 20)
	if limit <= 0 {
		limit = 20
	}
	cs, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	raw, err := cs.Clientset.CoreV1().RESTClient().Get().AbsPath("/apis/metrics.k8s.io/v1beta1/nodes").DoRaw(ctx)
	if err != nil {
		return TopMetricsReport{CheckTime: time.Now(), MetricsAvailable: false, Error: fmt.Sprintf("metrics.k8s.io unavailable: %v", err), Items: []TopNodeMetricItem{}}, nil
	}
	items, err := parseTopNodes(raw, limit)
	if err != nil {
		return nil, err
	}
	return TopMetricsReport{CheckTime: time.Now(), MetricsAvailable: true, Items: items}, nil
}

func parseTopPods(raw []byte, limit int) ([]TopPodMetricItem, error) {
	var list metricsList
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	out := []TopPodMetricItem{}
	for _, r := range list.Items {
		var pm podMetric
		if err := json.Unmarshal(r, &pm); err != nil {
			return nil, err
		}
		var cpu, mem int64
		for _, c := range pm.Containers {
			cpu += quantityMilli(c.Usage["cpu"])
			mem += quantityBytes(c.Usage["memory"])
		}
		out = append(out, TopPodMetricItem{Namespace: pm.Metadata.Namespace, Pod: pm.Metadata.Name, CPU: fmt.Sprintf("%dm", cpu), CPUMilli: cpu, Memory: fmt.Sprintf("%d", mem), MemoryBytes: mem})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUMilli > out[j].CPUMilli })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func parseTopNodes(raw []byte, limit int) ([]TopNodeMetricItem, error) {
	var list metricsList
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, err
	}
	out := []TopNodeMetricItem{}
	for _, r := range list.Items {
		var nm nodeMetric
		if err := json.Unmarshal(r, &nm); err != nil {
			return nil, err
		}
		cpu := quantityMilli(nm.Usage["cpu"])
		mem := quantityBytes(nm.Usage["memory"])
		out = append(out, TopNodeMetricItem{Node: nm.Metadata.Name, CPU: fmt.Sprintf("%dm", cpu), CPUMilli: cpu, Memory: fmt.Sprintf("%d", mem), MemoryBytes: mem})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CPUMilli > out[j].CPUMilli })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
func quantityMilli(v string) int64 {
	if v == "" {
		return 0
	}
	q, err := resource.ParseQuantity(v)
	if err != nil {
		return 0
	}
	return q.MilliValue()
}
func quantityBytes(v string) int64 {
	if v == "" {
		return 0
	}
	q, err := resource.ParseQuantity(v)
	if err != nil {
		return 0
	}
	return q.Value()
}
