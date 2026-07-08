package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"kubectl-mcp/internal/k8s"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InspectStorageDiagnostics diagnoses PVC/PV/StorageClass and volume mount issues.
func InspectStorageDiagnostics(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	contextName, defaultNamespace, _ := getContextAndNamespace(args, k8sClient)
	namespace := strings.TrimSpace(stringArg(args, "namespace", defaultNamespace))
	topN := intArg(args, "topN", 20)
	if topN <= 0 {
		topN = 20
	}
	clientSet, err := getClientSet(contextName, k8sClient)
	if err != nil {
		return nil, err
	}
	cs := clientSet.Clientset

	pvcs, err := cs.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 PVC 失败: %w", err)
	}
	pvs, err := cs.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 PV 失败: %w", err)
	}
	scs, err := cs.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("查询 StorageClass 失败: %w", err)
	}
	events, _ := cs.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})

	scSet := map[string]struct{}{}
	for _, sc := range scs.Items {
		scSet[sc.Name] = struct{}{}
	}
	findings := make([]DiagnosticFinding, 0)
	for _, pvc := range pvcs.Items {
		if pvc.Status.Phase == corev1.ClaimPending {
			desc := fmt.Sprintf("PVC %s/%s 处于 Pending，storageClass=%s。", pvc.Namespace, pvc.Name, pvcStorageClass(pvc))
			findings = append(findings, newStorageFinding("PVCPending", "warning", "PVC 处于 Pending", desc, DiagnosticObjectRef{Kind: "PersistentVolumeClaim", Namespace: pvc.Namespace, Name: pvc.Name}, nil, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "检查 StorageClass、动态供给器、容量和访问模式。"))
		}
		if sc := pvcStorageClass(pvc); sc != "" && sc != "-" {
			if _, ok := scSet[sc]; !ok {
				desc := fmt.Sprintf("PVC %s/%s 引用的 StorageClass %s 不存在。", pvc.Namespace, pvc.Name, sc)
				findings = append(findings, newStorageFinding("StorageClassMissing", "warning", "StorageClass 缺失", desc, DiagnosticObjectRef{Kind: "PersistentVolumeClaim", Namespace: pvc.Namespace, Name: pvc.Name}, []DiagnosticObjectRef{{Kind: "StorageClass", Name: sc}}, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "创建 StorageClass 或修正 PVC storageClassName。"))
			}
		}
	}
	for _, pv := range pvs.Items {
		if pv.Status.Phase == corev1.VolumeReleased || pv.Status.Phase == corev1.VolumeFailed {
			desc := fmt.Sprintf("PV %s 处于 %s。", pv.Name, pv.Status.Phase)
			findings = append(findings, newStorageFinding("PV"+string(pv.Status.Phase), "warning", "PV 状态异常", desc, DiagnosticObjectRef{Kind: "PersistentVolume", Name: pv.Name}, nil, []DiagnosticEvidence{{Source: "resource", Message: desc, Count: 1}}, "检查 PV 回收策略、claimRef、底层存储状态。"))
		}
	}
	if events != nil {
		for _, event := range events.Items {
			if event.Reason != "FailedMount" && event.Reason != "FailedAttachVolume" {
				continue
			}
			ns := event.InvolvedObject.Namespace
			if ns == "" {
				ns = event.Namespace
			}
			desc := fmt.Sprintf("%s %s/%s: %s", event.Reason, ns, event.InvolvedObject.Name, event.Message)
			findings = append(findings, newStorageFinding(classifyStorageEvent(event.Reason), "warning", "卷挂载/附加失败", desc, DiagnosticObjectRef{Kind: event.InvolvedObject.Kind, Namespace: ns, Name: event.InvolvedObject.Name}, nil, []DiagnosticEvidence{{Source: "event", Message: desc, Count: normalizedEventCount(event.Count), LastSeen: event.LastTimestamp.Time.Format(time.RFC3339)}}, "检查 PVC/PV 绑定、节点挂载能力、CSI/存储插件和底层存储。"))
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].FindingType == findings[j].FindingType {
			return findings[i].ID < findings[j].ID
		}
		return findings[i].FindingType < findings[j].FindingType
	})
	if len(findings) > topN {
		findings = findings[:topN]
	}
	return DiagnosticReport{CheckTime: time.Now(), Scope: DiagnosticScope{Cluster: contextName, Namespace: namespace}, Summary: buildDiagnosticSummary(findings), Findings: findings}, nil
}

func newStorageFinding(findingType, severity, title, description string, affected DiagnosticObjectRef, related []DiagnosticObjectRef, evidence []DiagnosticEvidence, recommendation string) DiagnosticFinding {
	return DiagnosticFinding{ID: stableFindingID("storage-diagnostics", findingType, affected.Namespace, affected.Kind, affected.Name, description), Severity: severity, FindingType: findingType, Title: title, Description: description, AffectedObject: affected, RelatedObjects: related, Evidence: evidence, Recommendation: recommendation, SafeActions: []DiagnosticSafeAction{{Action: "describe", RiskLevel: "read", Reason: "查看 PVC/PV/Pod 详情和事件"}, {Action: "inspect_pod_diagnostics", RiskLevel: "read", Reason: "检查受影响 Pod 状态"}}}
}
func pvcStorageClass(pvc corev1.PersistentVolumeClaim) string {
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName == "" {
		return "-"
	}
	return *pvc.Spec.StorageClassName
}
func classifyStorageEvent(reason string) string {
	if reason == "FailedAttachVolume" {
		return "VolumeAttachFailure"
	}
	return "VolumeMountFailure"
}
