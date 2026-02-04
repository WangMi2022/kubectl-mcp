package test

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"syscall"
	"testing"
	"time"

	"kubectl-mcp/pkg/utils"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ========== 属性测试：错误信息明确性 ==========
// Feature: kubectl-mcp-server, Property 4: 错误信息明确性
// Validates: Requirements 10.1, 10.2, 10.3, 10.4, 10.5, 10.6
//
// 对于任何失败的操作，系统必须返回明确的错误类型（客户端错误、服务端错误、网络错误）
// 和可操作的错误信息

// TestProperty_ErrorClarityForAllErrorTypes 测试所有错误类型都有明确的类型和错误码
func TestProperty_ErrorClarityForAllErrorTypes(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < iterations; i++ {
		// 生成随机错误场景
		err := generateRandomError()

		// 格式化错误
		errInfo := utils.FormatError(err)

		// 验证错误信息不为 nil
		if errInfo == nil {
			t.Fatalf("迭代 %d: 期望返回 ErrorInfo，实际为 nil，原始错误: %v", i, err)
		}

		// Property 验证 1: 所有错误都有明确的类型
		if errInfo.Type == "" {
			t.Errorf("迭代 %d: 错误类型为空，原始错误: %v", i, err)
		}

		// Property 验证 2: 所有错误都有明确的错误码
		if errInfo.Code == "" {
			t.Errorf("迭代 %d: 错误码为空，原始错误: %v", i, err)
		}

		// Property 验证 3: 所有错误都有明确的错误消息
		if errInfo.Message == "" {
			t.Errorf("迭代 %d: 错误消息为空，原始错误: %v", i, err)
		}

		// Property 验证 4: 错误类型必须是预定义的类型之一
		validTypes := []string{
			utils.ErrTypeClient,
			utils.ErrTypeServer,
			utils.ErrTypeNetwork,
			utils.ErrTypeAuth,
			utils.ErrTypeNotFound,
			utils.ErrTypeConflict,
			utils.ErrTypeTimeout,
		}
		if !contains(validTypes, errInfo.Type) {
			t.Errorf("迭代 %d: 错误类型 '%s' 不在预定义类型列表中，原始错误: %v", i, errInfo.Type, err)
		}

		// Property 验证 5: 所有错误都有可操作的建议
		if errInfo.Suggestion == "" {
			t.Errorf("迭代 %d: 错误建议为空，错误码: %s，原始错误: %v", i, errInfo.Code, err)
		}
	}
}

// TestProperty_ErrorTypeConsistency 测试错误类型分类的一致性
func TestProperty_ErrorTypeConsistency(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < iterations; i++ {
		// 生成随机错误
		err := generateRandomError()

		// 多次分类同一个错误
		errType1, errCode1 := utils.ClassifyError(err)
		errType2, errCode2 := utils.ClassifyError(err)

		// Property 验证: 同一个错误多次分类应该得到相同的结果
		if errType1 != errType2 {
			t.Errorf("迭代 %d: 错误类型不一致，第一次: %s，第二次: %s，原始错误: %v", i, errType1, errType2, err)
		}

		if errCode1 != errCode2 {
			t.Errorf("迭代 %d: 错误码不一致，第一次: %s，第二次: %s，原始错误: %v", i, errCode1, errCode2, err)
		}
	}
}

// TestProperty_ErrorInfoCompleteness 测试 ErrorInfo 的完整性
func TestProperty_ErrorInfoCompleteness(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < iterations; i++ {
		// 生成随机错误
		err := generateRandomError()

		// 格式化错误
		errInfo := utils.FormatError(err)

		if errInfo == nil {
			t.Fatalf("迭代 %d: 期望返回 ErrorInfo，实际为 nil", i)
		}

		// Property 验证 1: Error() 方法应该返回非空字符串
		errStr := errInfo.Error()
		if errStr == "" {
			t.Errorf("迭代 %d: Error() 返回空字符串", i)
		}

		// Property 验证 2: Error() 字符串应该包含错误类型
		if !containsSubstring(errStr, errInfo.Type) {
			t.Errorf("迭代 %d: Error() 字符串不包含错误类型 '%s'，实际: %s", i, errInfo.Type, errStr)
		}

		// Property 验证 3: Error() 字符串应该包含错误码
		if !containsSubstring(errStr, errInfo.Code) {
			t.Errorf("迭代 %d: Error() 字符串不包含错误码 '%s'，实际: %s", i, errInfo.Code, errStr)
		}

		// Property 验证 4: Error() 字符串应该包含错误消息
		if !containsSubstring(errStr, errInfo.Message) {
			t.Errorf("迭代 %d: Error() 字符串不包含错误消息 '%s'，实际: %s", i, errInfo.Message, errStr)
		}
	}
}

// TestProperty_K8SErrorFormatting 测试 K8S 错误格式化的属性
func TestProperty_K8SErrorFormatting(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	resourceTypes := []string{"Pod", "Deployment", "Service", "ConfigMap", "Secret", "Namespace"}

	for i := 0; i < iterations; i++ {
		// 生成随机 K8S 错误
		err := generateRandomK8SError()
		resourceType := resourceTypes[rand.Intn(len(resourceTypes))]
		resourceName := fmt.Sprintf("resource-%d", rand.Intn(1000))

		// 格式化 K8S 错误
		errInfo := utils.FormatK8SError(err, resourceType, resourceName)

		if errInfo == nil {
			t.Fatalf("迭代 %d: 期望返回 ErrorInfo，实际为 nil", i)
		}

		// Property 验证 1: 错误详情应该包含资源类型
		if resourceType != "" && !containsSubstring(errInfo.Details, resourceType) {
			t.Errorf("迭代 %d: 错误详情不包含资源类型 '%s'，实际: %s", i, resourceType, errInfo.Details)
		}

		// Property 验证 2: 错误详情应该包含资源名称
		if resourceName != "" && !containsSubstring(errInfo.Details, resourceName) {
			t.Errorf("迭代 %d: 错误详情不包含资源名称 '%s'，实际: %s", i, resourceName, errInfo.Details)
		}

		// Property 验证 3: 所有必需字段都应该存在
		if errInfo.Type == "" || errInfo.Code == "" || errInfo.Message == "" {
			t.Errorf("迭代 %d: 错误信息不完整，Type: %s, Code: %s, Message: %s",
				i, errInfo.Type, errInfo.Code, errInfo.Message)
		}
	}
}

// TestProperty_ErrorSuggestionRelevance 测试错误建议的相关性
func TestProperty_ErrorSuggestionRelevance(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	// 定义错误码和相关关键词的映射
	relevantKeywords := map[string][]string{
		utils.ErrCodeKubeconfigNotFound:    {"kubeconfig", "文件", "路径", "KUBECONFIG"},
		utils.ErrCodeKubeconfigInvalid:     {"kubeconfig", "格式", "kubectl config"},
		utils.ErrCodeContextNotFound:       {"context", "kubectl config get-contexts"},
		utils.ErrCodeClusterUnreachable:    {"集群", "网络", "kubectl cluster-info"},
		utils.ErrCodeConnectionTimeout:     {"超时", "网络", "防火墙"},
		utils.ErrCodePermissionDenied:      {"权限", "RBAC", "管理员"},
		utils.ErrCodeResourceNotFound:      {"资源", "kubectl get", "命名空间"},
		utils.ErrCodeResourceAlreadyExists: {"已存在", "update", "apply"},
		utils.ErrCodeInvalidArguments:      {"参数", "格式", "类型"},
		utils.ErrCodeMissingArguments:      {"参数", "必填"},
		utils.ErrCodeInvalidYAML:           {"YAML", "格式", "缩进"},
	}

	for i := 0; i < iterations; i++ {
		// 随机选择一个错误码
		errCodes := []string{
			utils.ErrCodeKubeconfigNotFound,
			utils.ErrCodeKubeconfigInvalid,
			utils.ErrCodeContextNotFound,
			utils.ErrCodeClusterUnreachable,
			utils.ErrCodeConnectionTimeout,
			utils.ErrCodePermissionDenied,
			utils.ErrCodeResourceNotFound,
			utils.ErrCodeResourceAlreadyExists,
			utils.ErrCodeInvalidArguments,
			utils.ErrCodeMissingArguments,
			utils.ErrCodeInvalidYAML,
		}
		errCode := errCodes[rand.Intn(len(errCodes))]

		// 获取建议
		suggestion := utils.GetSuggestion(errCode)

		// Property 验证 1: 建议不应该为空
		if suggestion == "" {
			t.Errorf("迭代 %d: 错误码 '%s' 的建议为空", i, errCode)
			continue
		}

		// Property 验证 2: 建议应该包含至少一个相关关键词
		keywords, exists := relevantKeywords[errCode]
		if exists {
			hasRelevantKeyword := false
			for _, keyword := range keywords {
				if containsSubstring(suggestion, keyword) {
					hasRelevantKeyword = true
					break
				}
			}
			if !hasRelevantKeyword {
				t.Errorf("迭代 %d: 错误码 '%s' 的建议不包含任何相关关键词，建议: %s，期望关键词: %v",
					i, errCode, suggestion, keywords)
			}
		}
	}
}

// TestProperty_ErrorWrappingPreservesInformation 测试错误包装保留信息
func TestProperty_ErrorWrappingPreservesInformation(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	contexts := []string{
		"查询 Pod 时",
		"创建 Deployment 时",
		"删除 Service 时",
		"更新 ConfigMap 时",
		"扩缩容 StatefulSet 时",
	}

	for i := 0; i < iterations; i++ {
		// 生成随机错误
		originalErr := generateRandomError()
		context := contexts[rand.Intn(len(contexts))]

		// 包装错误
		wrappedErr := utils.WrapError(originalErr, context)

		if wrappedErr == nil {
			t.Fatalf("迭代 %d: 期望返回 ErrorInfo，实际为 nil", i)
		}

		// Property 验证 1: 包装后的错误应该保留原始错误的类型和错误码
		originalType, originalCode := utils.ClassifyError(originalErr)
		if wrappedErr.Type != originalType {
			t.Errorf("迭代 %d: 包装后错误类型改变，原始: %s，包装后: %s", i, originalType, wrappedErr.Type)
		}
		if wrappedErr.Code != originalCode {
			t.Errorf("迭代 %d: 包装后错误码改变，原始: %s，包装后: %s", i, originalCode, wrappedErr.Code)
		}

		// Property 验证 2: 包装后的错误应该包含上下文信息
		if context != "" && !containsSubstring(wrappedErr.Details, context) {
			t.Errorf("迭代 %d: 包装后的错误详情不包含上下文 '%s'，实际: %s", i, context, wrappedErr.Details)
		}

		// Property 验证 3: 包装后的错误应该仍然有建议
		if wrappedErr.Suggestion == "" {
			t.Errorf("迭代 %d: 包装后的错误建议为空", i)
		}
	}
}

// TestProperty_ErrorJudgmentFunctions 测试错误判断函数的正确性
func TestProperty_ErrorJudgmentFunctions(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < iterations; i++ {
		// 生成随机错误
		err := generateRandomError()

		// 获取错误类型
		errType, _ := utils.ClassifyError(err)

		// Property 验证: 错误判断函数应该与错误类型一致
		switch errType {
		case utils.ErrTypeNotFound:
			if !utils.IsNotFoundError(err) {
				t.Errorf("迭代 %d: IsNotFoundError 应该返回 true，错误: %v", i, err)
			}
		case utils.ErrTypeConflict:
			if !utils.IsConflictError(err) {
				t.Errorf("迭代 %d: IsConflictError 应该返回 true，错误: %v", i, err)
			}
		case utils.ErrTypeAuth:
			if !utils.IsAuthError(err) {
				t.Errorf("迭代 %d: IsAuthError 应该返回 true，错误: %v", i, err)
			}
		case utils.ErrTypeTimeout:
			if !utils.IsTimeoutError(err) {
				t.Errorf("迭代 %d: IsTimeoutError 应该返回 true，错误: %v", i, err)
			}
		case utils.ErrTypeNetwork:
			if !utils.IsNetworkError(err) {
				t.Errorf("迭代 %d: IsNetworkError 应该返回 true，错误: %v", i, err)
			}
		case utils.ErrTypeClient:
			if !utils.IsClientError(err) {
				t.Errorf("迭代 %d: IsClientError 应该返回 true，错误: %v", i, err)
			}
		case utils.ErrTypeServer:
			if !utils.IsServerError(err) {
				t.Errorf("迭代 %d: IsServerError 应该返回 true，错误: %v", i, err)
			}
		}
	}
}

// TestProperty_ErrorCloneIndependence 测试错误克隆的独立性
func TestProperty_ErrorCloneIndependence(t *testing.T) {
	const iterations = 100
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < iterations; i++ {
		// 创建随机错误信息
		original := createRandomErrorInfo()

		// 克隆错误
		cloned := original.Clone()

		// Property 验证 1: 克隆对象应该与原对象内容相同
		if cloned.Type != original.Type ||
			cloned.Code != original.Code ||
			cloned.Message != original.Message ||
			cloned.Details != original.Details ||
			cloned.Suggestion != original.Suggestion {
			t.Errorf("迭代 %d: 克隆对象内容与原对象不同", i)
		}

		// Property 验证 2: 克隆对象应该是独立的（修改克隆不影响原对象）
		cloned.Details = "修改后的详情"
		cloned.Suggestion = "修改后的建议"

		if original.Details == "修改后的详情" || original.Suggestion == "修改后的建议" {
			t.Errorf("迭代 %d: 修改克隆对象影响了原对象", i)
		}
	}
}

// ========== 辅助函数 ==========

// generateRandomError 生成随机错误
func generateRandomError() error {
	errorTypes := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	errorType := errorTypes[rand.Intn(len(errorTypes))]

	switch errorType {
	case 0:
		// K8S StatusError
		return generateRandomK8SError()
	case 1:
		// 网络错误
		return generateRandomNetworkError()
	case 2:
		// 超时错误
		return &timeoutError{}
	case 3:
		// ErrorInfo 类型
		return createRandomErrorInfo()
	case 4:
		// 标准错误 - not found
		messages := []string{"resource not found", "pod not found", "deployment doesn't exist"}
		return errors.New(messages[rand.Intn(len(messages))])
	case 5:
		// 标准错误 - unauthorized
		messages := []string{"unauthorized access", "authentication failed", "forbidden"}
		return errors.New(messages[rand.Intn(len(messages))])
	case 6:
		// 标准错误 - conflict
		messages := []string{"resource already exists", "conflict detected"}
		return errors.New(messages[rand.Intn(len(messages))])
	case 7:
		// 标准错误 - invalid
		messages := []string{"invalid parameter", "missing required field", "kubeconfig is invalid"}
		return errors.New(messages[rand.Intn(len(messages))])
	default:
		// 标准错误 - 其他
		messages := []string{"connection refused", "timeout", "internal server error"}
		return errors.New(messages[rand.Intn(len(messages))])
	}
}

// generateRandomK8SError 生成随机 K8S 错误
func generateRandomK8SError() error {
	reasons := []metav1.StatusReason{
		metav1.StatusReasonNotFound,
		metav1.StatusReasonAlreadyExists,
		metav1.StatusReasonConflict,
		metav1.StatusReasonForbidden,
		metav1.StatusReasonUnauthorized,
		metav1.StatusReasonBadRequest,
		metav1.StatusReasonInvalid,
		metav1.StatusReasonTimeout,
		metav1.StatusReasonServerTimeout,
		metav1.StatusReasonServiceUnavailable,
		metav1.StatusReasonInternalError,
	}

	codes := []int32{400, 401, 403, 404, 409, 422, 500, 503, 504}

	reason := reasons[rand.Intn(len(reasons))]
	code := codes[rand.Intn(len(codes))]

	return &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Reason:  reason,
			Code:    code,
			Message: fmt.Sprintf("K8S error: %s", reason),
		},
	}
}

// generateRandomNetworkError 生成随机网络错误
func generateRandomNetworkError() error {
	errorTypes := []int{0, 1, 2}
	errorType := errorTypes[rand.Intn(len(errorTypes))]

	switch errorType {
	case 0:
		// 连接被拒绝
		return &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}
	case 1:
		// DNS 错误
		hosts := []string{"invalid.cluster", "unknown.host", "bad.domain"}
		return &net.DNSError{Err: "no such host", Name: hosts[rand.Intn(len(hosts))]}
	default:
		// 其他网络错误
		return &net.OpError{Op: "dial", Err: errors.New("network unreachable")}
	}
}

// createRandomErrorInfo 创建随机 ErrorInfo
func createRandomErrorInfo() *utils.ErrorInfo {
	types := []string{
		utils.ErrTypeClient,
		utils.ErrTypeServer,
		utils.ErrTypeNetwork,
		utils.ErrTypeAuth,
		utils.ErrTypeNotFound,
		utils.ErrTypeConflict,
		utils.ErrTypeTimeout,
	}

	codes := []string{
		utils.ErrCodeKubeconfigNotFound,
		utils.ErrCodeContextNotFound,
		utils.ErrCodeClusterUnreachable,
		utils.ErrCodeResourceNotFound,
		utils.ErrCodeResourceAlreadyExists,
		utils.ErrCodeInvalidArguments,
		utils.ErrCodePermissionDenied,
		utils.ErrCodeInternalError,
	}

	messages := []string{
		"资源未找到",
		"参数无效",
		"权限不足",
		"连接失败",
		"资源已存在",
		"配置错误",
	}

	details := []string{
		"详细信息1",
		"详细信息2",
		"",
	}

	suggestions := []string{
		"请检查配置",
		"请联系管理员",
		"请重试",
	}

	errType := types[rand.Intn(len(types))]
	code := codes[rand.Intn(len(codes))]
	message := messages[rand.Intn(len(messages))]
	detail := details[rand.Intn(len(details))]
	suggestion := suggestions[rand.Intn(len(suggestions))]

	errInfo := utils.NewErrorInfo(errType, code, message)
	if detail != "" {
		errInfo.WithDetails(detail)
	}
	if suggestion != "" {
		errInfo.WithSuggestion(suggestion)
	}

	return errInfo
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// containsSubstring 检查字符串是否包含子字符串（不区分大小写）
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || findSubstring(str, substr))
}

// findSubstring 查找子字符串
func findSubstring(str, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(str) < len(substr) {
		return false
	}
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
