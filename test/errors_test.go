package test

import (
	"errors"
	"net"
	"syscall"
	"testing"

	"kubectl-mcp/pkg/utils"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ========== 错误分类测试 ==========

// TestClassifyError_K8SStatusErrors 测试 K8S API 状态错误分类
func TestClassifyError_K8SStatusErrors(t *testing.T) {
	tests := []struct {
		name         string
		statusErr    *k8serrors.StatusError
		expectedType string
		expectedCode string
	}{
		{
			name: "NotFound 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonNotFound,
					Code:   404,
				},
			},
			expectedType: utils.ErrTypeNotFound,
			expectedCode: utils.ErrCodeResourceNotFound,
		},
		{
			name: "AlreadyExists 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonAlreadyExists,
					Code:   409,
				},
			},
			expectedType: utils.ErrTypeConflict,
			expectedCode: utils.ErrCodeResourceAlreadyExists,
		},
		{
			name: "Conflict 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonConflict,
					Code:   409,
				},
			},
			expectedType: utils.ErrTypeConflict,
			expectedCode: utils.ErrCodeResourceConflict,
		},
		{
			name: "Forbidden 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonForbidden,
					Code:   403,
				},
			},
			expectedType: utils.ErrTypeAuth,
			expectedCode: utils.ErrCodeForbidden,
		},
		{
			name: "Unauthorized 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonUnauthorized,
					Code:   401,
				},
			},
			expectedType: utils.ErrTypeAuth,
			expectedCode: utils.ErrCodeUnauthorized,
		},
		{
			name: "BadRequest 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonBadRequest,
					Code:   400,
				},
			},
			expectedType: utils.ErrTypeClient,
			expectedCode: utils.ErrCodeInvalidRequest,
		},
		{
			name: "Invalid 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonInvalid,
					Code:   422,
				},
			},
			expectedType: utils.ErrTypeClient,
			expectedCode: utils.ErrCodeInvalidResource,
		},
		{
			name: "Timeout 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonTimeout,
					Code:   504,
				},
			},
			expectedType: utils.ErrTypeTimeout,
			expectedCode: utils.ErrCodeRequestTimeout,
		},
		{
			name: "ServerTimeout 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonServerTimeout,
					Code:   504,
				},
			},
			expectedType: utils.ErrTypeTimeout,
			expectedCode: utils.ErrCodeConnectionTimeout,
		},
		{
			name: "ServiceUnavailable 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonServiceUnavailable,
					Code:   503,
				},
			},
			expectedType: utils.ErrTypeServer,
			expectedCode: utils.ErrCodeServiceUnavail,
		},
		{
			name: "InternalError 错误",
			statusErr: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonInternalError,
					Code:   500,
				},
			},
			expectedType: utils.ErrTypeServer,
			expectedCode: utils.ErrCodeInternalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType, errCode := utils.ClassifyError(tt.statusErr)
			if errType != tt.expectedType {
				t.Errorf("期望错误类型为 '%s'，实际为 '%s'", tt.expectedType, errType)
			}
			if errCode != tt.expectedCode {
				t.Errorf("期望错误码为 '%s'，实际为 '%s'", tt.expectedCode, errCode)
			}
		})
	}
}

// TestClassifyError_NetworkErrors 测试网络错误分类
func TestClassifyError_NetworkErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
		expectedCode string
	}{
		{
			name:         "连接被拒绝错误",
			err:          &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			expectedType: utils.ErrTypeNetwork,
			expectedCode: utils.ErrCodeConnectionRefused,
		},
		{
			name:         "DNS 解析错误",
			err:          &net.DNSError{Err: "no such host", Name: "invalid.cluster"},
			expectedType: utils.ErrTypeNetwork,
			expectedCode: utils.ErrCodeDNSError,
		},
		{
			name:         "网络超时错误",
			err:          &timeoutError{},
			expectedType: utils.ErrTypeTimeout,
			expectedCode: utils.ErrCodeConnectionTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType, errCode := utils.ClassifyError(tt.err)
			if errType != tt.expectedType {
				t.Errorf("期望错误类型为 '%s'，实际为 '%s'", tt.expectedType, errType)
			}
			if errCode != tt.expectedCode {
				t.Errorf("期望错误码为 '%s'，实际为 '%s'", tt.expectedCode, errCode)
			}
		})
	}
}

// timeoutError 模拟超时错误
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

// TestClassifyError_ByErrorMessage 测试根据错误消息分类
func TestClassifyError_ByErrorMessage(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedType string
		expectedCode string
	}{
		{
			name:         "unauthorized 错误",
			err:          errors.New("unauthorized access"),
			expectedType: utils.ErrTypeAuth,
			expectedCode: utils.ErrCodeUnauthorized,
		},
		{
			name:         "forbidden 错误",
			err:          errors.New("permission denied"),
			expectedType: utils.ErrTypeAuth,
			expectedCode: utils.ErrCodePermissionDenied,
		},
		{
			name:         "not found 错误",
			err:          errors.New("resource not found"),
			expectedType: utils.ErrTypeNotFound,
			expectedCode: utils.ErrCodeResourceNotFound,
		},
		{
			name:         "already exists 错误",
			err:          errors.New("resource already exists"),
			expectedType: utils.ErrTypeConflict,
			expectedCode: utils.ErrCodeResourceAlreadyExists,
		},
		{
			name:         "connection refused 错误",
			err:          errors.New("connection refused"),
			expectedType: utils.ErrTypeNetwork,
			expectedCode: utils.ErrCodeConnectionRefused,
		},
		{
			name:         "kubeconfig invalid 错误",
			err:          errors.New("kubeconfig is invalid"),
			expectedType: utils.ErrTypeClient,
			expectedCode: utils.ErrCodeKubeconfigInvalid,
		},
		{
			name:         "invalid arguments 错误",
			err:          errors.New("invalid parameter"),
			expectedType: utils.ErrTypeClient,
			expectedCode: utils.ErrCodeInvalidArguments,
		},
		{
			name:         "missing arguments 错误",
			err:          errors.New("missing required field"),
			expectedType: utils.ErrTypeClient,
			expectedCode: utils.ErrCodeMissingArguments,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errType, errCode := utils.ClassifyError(tt.err)
			if errType != tt.expectedType {
				t.Errorf("期望错误类型为 '%s'，实际为 '%s'", tt.expectedType, errType)
			}
			if errCode != tt.expectedCode {
				t.Errorf("期望错误码为 '%s'，实际为 '%s'", tt.expectedCode, errCode)
			}
		})
	}
}

// TestClassifyError_ErrorInfo 测试 ErrorInfo 类型错误分类
func TestClassifyError_ErrorInfo(t *testing.T) {
	errInfo := utils.NewErrorInfo(utils.ErrTypeClient, utils.ErrCodeInvalidArguments, "参数无效")

	errType, errCode := utils.ClassifyError(errInfo)

	if errType != utils.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeClient, errType)
	}
	if errCode != utils.ErrCodeInvalidArguments {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeInvalidArguments, errCode)
	}
}

// TestClassifyError_NilError 测试 nil 错误分类
func TestClassifyError_NilError(t *testing.T) {
	errType, errCode := utils.ClassifyError(nil)

	if errType != "" {
		t.Errorf("期望错误类型为空字符串，实际为 '%s'", errType)
	}
	if errCode != "" {
		t.Errorf("期望错误码为空字符串，实际为 '%s'", errCode)
	}
}

// ========== 错误响应格式化测试 ==========

// TestFormatError_StandardError 测试标准错误格式化
func TestFormatError_StandardError(t *testing.T) {
	err := errors.New("resource not found")

	errInfo := utils.FormatError(err)

	if errInfo == nil {
		t.Fatal("期望返回 ErrorInfo，实际为 nil")
	}

	if errInfo.Type != utils.ErrTypeNotFound {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNotFound, errInfo.Type)
	}

	if errInfo.Code != utils.ErrCodeResourceNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeResourceNotFound, errInfo.Code)
	}

	if errInfo.Message == "" {
		t.Error("期望错误消息不为空")
	}

	if errInfo.Details == "" {
		t.Error("期望错误详情不为空")
	}

	if errInfo.Suggestion == "" {
		t.Error("期望错误建议不为空")
	}
}

// TestFormatError_ErrorInfo 测试 ErrorInfo 类型格式化
func TestFormatError_ErrorInfo(t *testing.T) {
	originalErr := utils.NewErrorInfo(utils.ErrTypeClient, utils.ErrCodeInvalidArguments, "参数无效").
		WithDetails("参数 name 不能为空").
		WithSuggestion("请提供有效的参数")

	errInfo := utils.FormatError(originalErr)

	if errInfo == nil {
		t.Fatal("期望返回 ErrorInfo，实际为 nil")
	}

	if errInfo.Type != utils.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeClient, errInfo.Type)
	}

	if errInfo.Code != utils.ErrCodeInvalidArguments {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeInvalidArguments, errInfo.Code)
	}

	if errInfo.Message != "参数无效" {
		t.Errorf("期望错误消息为 '参数无效'，实际为 '%s'", errInfo.Message)
	}

	if errInfo.Details != "参数 name 不能为空" {
		t.Errorf("期望错误详情为 '参数 name 不能为空'，实际为 '%s'", errInfo.Details)
	}

	if errInfo.Suggestion != "请提供有效的参数" {
		t.Errorf("期望错误建议为 '请提供有效的参数'，实际为 '%s'", errInfo.Suggestion)
	}
}

// TestFormatError_NilError 测试 nil 错误格式化
func TestFormatError_NilError(t *testing.T) {
	errInfo := utils.FormatError(nil)

	if errInfo != nil {
		t.Errorf("期望返回 nil，实际返回了 ErrorInfo")
	}
}

// TestFormatK8SError 测试 K8S 错误格式化
func TestFormatK8SError(t *testing.T) {
	err := errors.New("pod not found")

	errInfo := utils.FormatK8SError(err, "Pod", "test-pod")

	if errInfo == nil {
		t.Fatal("期望返回 ErrorInfo，实际为 nil")
	}

	if errInfo.Type != utils.ErrTypeNotFound {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNotFound, errInfo.Type)
	}

	if errInfo.Details == "" {
		t.Error("期望错误详情不为空")
	}

	// 验证详情中包含资源信息
	expectedDetails := "资源: Pod/test-pod"
	if errInfo.Details[:len(expectedDetails)] != expectedDetails {
		t.Errorf("期望错误详情包含 '%s'，实际为 '%s'", expectedDetails, errInfo.Details)
	}
}

// TestFormatK8SError_WithNamespace 测试带命名空间的 K8S 错误格式化
func TestFormatK8SError_WithNamespace(t *testing.T) {
	err := &k8serrors.StatusError{
		ErrStatus: metav1.Status{
			Reason: metav1.StatusReasonNotFound,
			Code:   404,
		},
	}

	errInfo := utils.FormatK8SError(err, "Deployment", "nginx")

	if errInfo == nil {
		t.Fatal("期望返回 ErrorInfo，实际为 nil")
	}

	if errInfo.Type != utils.ErrTypeNotFound {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNotFound, errInfo.Type)
	}

	if errInfo.Code != utils.ErrCodeResourceNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeResourceNotFound, errInfo.Code)
	}
}

// ========== 错误信息结构测试 ==========

// TestErrorInfo_Error 测试 ErrorInfo.Error() 方法
func TestErrorInfo_Error(t *testing.T) {
	tests := []struct {
		name     string
		errInfo  *utils.ErrorInfo
		expected string
	}{
		{
			name: "包含详情的错误",
			errInfo: utils.NewErrorInfo(utils.ErrTypeClient, utils.ErrCodeInvalidArguments, "参数无效").
				WithDetails("参数 name 不能为空"),
			expected: "[CLIENT_ERROR] INVALID_ARGUMENTS: 参数无效 - 参数 name 不能为空",
		},
		{
			name:     "不包含详情的错误",
			errInfo:  utils.NewErrorInfo(utils.ErrTypeServer, utils.ErrCodeInternalError, "服务器内部错误"),
			expected: "[SERVER_ERROR] INTERNAL_ERROR: 服务器内部错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.errInfo.Error()
			if result != tt.expected {
				t.Errorf("期望错误字符串为 '%s'，实际为 '%s'", tt.expected, result)
			}
		})
	}
}

// TestErrorInfo_WithDetails 测试添加详情
func TestErrorInfo_WithDetails(t *testing.T) {
	errInfo := utils.NewErrorInfo(utils.ErrTypeClient, utils.ErrCodeInvalidArguments, "参数无效")

	result := errInfo.WithDetails("参数 name 不能为空")

	if result.Details != "参数 name 不能为空" {
		t.Errorf("期望详情为 '参数 name 不能为空'，实际为 '%s'", result.Details)
	}

	// 验证返回的是同一个对象
	if result != errInfo {
		t.Error("期望 WithDetails 返回同一个对象")
	}
}

// TestErrorInfo_WithSuggestion 测试添加建议
func TestErrorInfo_WithSuggestion(t *testing.T) {
	errInfo := utils.NewErrorInfo(utils.ErrTypeClient, utils.ErrCodeInvalidArguments, "参数无效")

	result := errInfo.WithSuggestion("请提供有效的参数")

	if result.Suggestion != "请提供有效的参数" {
		t.Errorf("期望建议为 '请提供有效的参数'，实际为 '%s'", result.Suggestion)
	}

	// 验证返回的是同一个对象
	if result != errInfo {
		t.Error("期望 WithSuggestion 返回同一个对象")
	}
}

// TestErrorInfo_Clone 测试克隆错误信息
func TestErrorInfo_Clone(t *testing.T) {
	original := utils.NewErrorInfo(utils.ErrTypeClient, utils.ErrCodeInvalidArguments, "参数无效").
		WithDetails("参数 name 不能为空").
		WithSuggestion("请提供有效的参数")

	cloned := original.Clone()

	// 验证克隆的内容相同
	if cloned.Type != original.Type {
		t.Errorf("期望类型为 '%s'，实际为 '%s'", original.Type, cloned.Type)
	}
	if cloned.Code != original.Code {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", original.Code, cloned.Code)
	}
	if cloned.Message != original.Message {
		t.Errorf("期望消息为 '%s'，实际为 '%s'", original.Message, cloned.Message)
	}
	if cloned.Details != original.Details {
		t.Errorf("期望详情为 '%s'，实际为 '%s'", original.Details, cloned.Details)
	}
	if cloned.Suggestion != original.Suggestion {
		t.Errorf("期望建议为 '%s'，实际为 '%s'", original.Suggestion, cloned.Suggestion)
	}

	// 验证是不同的对象
	if cloned == original {
		t.Error("期望 Clone 返回新对象")
	}

	// 修改克隆对象不应影响原对象
	cloned.Details = "修改后的详情"
	if original.Details == "修改后的详情" {
		t.Error("修改克隆对象不应影响原对象")
	}
}

// ========== 便捷错误创建函数测试 ==========

// TestNewKubeconfigNotFoundError 测试创建 kubeconfig 未找到错误
func TestNewKubeconfigNotFoundError(t *testing.T) {
	path := "/path/to/kubeconfig"
	errInfo := utils.NewKubeconfigNotFoundError(path)

	if errInfo.Type != utils.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeClient, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodeKubeconfigNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeKubeconfigNotFound, errInfo.Code)
	}
	if errInfo.Details == "" {
		t.Error("期望详情不为空")
	}
	if errInfo.Suggestion == "" {
		t.Error("期望建议不为空")
	}
}

// TestNewContextNotFoundError 测试创建 context 未找到错误
func TestNewContextNotFoundError(t *testing.T) {
	contextName := "prod-cluster"
	errInfo := utils.NewContextNotFoundError(contextName)

	if errInfo.Type != utils.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeClient, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodeContextNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeContextNotFound, errInfo.Code)
	}
	if errInfo.Details == "" {
		t.Error("期望详情不为空")
	}
}

// TestNewResourceNotFoundError 测试创建资源未找到错误
func TestNewResourceNotFoundError(t *testing.T) {
	errInfo := utils.NewResourceNotFoundError("Pod", "nginx", "default")

	if errInfo.Type != utils.ErrTypeNotFound {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNotFound, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodeResourceNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeResourceNotFound, errInfo.Code)
	}
	if errInfo.Details == "" {
		t.Error("期望详情不为空")
	}
}

// TestNewResourceAlreadyExistsError 测试创建资源已存在错误
func TestNewResourceAlreadyExistsError(t *testing.T) {
	errInfo := utils.NewResourceAlreadyExistsError("Deployment", "nginx", "default")

	if errInfo.Type != utils.ErrTypeConflict {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeConflict, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodeResourceAlreadyExists {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeResourceAlreadyExists, errInfo.Code)
	}
}

// TestNewInvalidArgumentsError 测试创建参数无效错误
func TestNewInvalidArgumentsError(t *testing.T) {
	errInfo := utils.NewInvalidArgumentsError("name", "不能为空")

	if errInfo.Type != utils.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeClient, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodeInvalidArguments {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeInvalidArguments, errInfo.Code)
	}
	if errInfo.Details == "" {
		t.Error("期望详情不为空")
	}
}

// TestNewMissingArgumentsError 测试创建缺少参数错误
func TestNewMissingArgumentsError(t *testing.T) {
	errInfo := utils.NewMissingArgumentsError("name", "namespace")

	if errInfo.Type != utils.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeClient, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodeMissingArguments {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeMissingArguments, errInfo.Code)
	}
	if errInfo.Details == "" {
		t.Error("期望详情不为空")
	}
}

// TestNewPermissionDeniedError 测试创建权限不足错误
func TestNewPermissionDeniedError(t *testing.T) {
	errInfo := utils.NewPermissionDeniedError("pods", "delete")

	if errInfo.Type != utils.ErrTypeAuth {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeAuth, errInfo.Type)
	}
	if errInfo.Code != utils.ErrCodePermissionDenied {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodePermissionDenied, errInfo.Code)
	}
}

// ========== 错误判断辅助函数测试 ==========

// TestIsNotFoundError 测试判断是否是资源未找到错误
func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 NotFound 错误",
			err:      utils.NewResourceNotFoundError("Pod", "nginx", "default"),
			expected: true,
		},
		{
			name: "K8S StatusError 类型的 NotFound 错误",
			err: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonNotFound,
					Code:   404,
				},
			},
			expected: true,
		},
		{
			name:     "标准错误包含 not found",
			err:      errors.New("resource not found"),
			expected: true,
		},
		{
			name:     "其他类型错误",
			err:      errors.New("invalid argument"),
			expected: false,
		},
		{
			name:     "nil 错误",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsNotFoundError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// TestIsConflictError 测试判断是否是冲突错误
func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 Conflict 错误",
			err:      utils.NewResourceAlreadyExistsError("Pod", "nginx", "default"),
			expected: true,
		},
		{
			name: "K8S StatusError 类型的 Conflict 错误",
			err: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonConflict,
					Code:   409,
				},
			},
			expected: true,
		},
		{
			name:     "标准错误包含 conflict",
			err:      errors.New("resource conflict"),
			expected: true,
		},
		{
			name:     "其他类型错误",
			err:      errors.New("invalid argument"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsConflictError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsConflictError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// TestIsAuthError 测试判断是否是认证错误
func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 Auth 错误",
			err:      utils.NewPermissionDeniedError("pods", "delete"),
			expected: true,
		},
		{
			name: "K8S StatusError 类型的 Unauthorized 错误",
			err: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonUnauthorized,
					Code:   401,
				},
			},
			expected: true,
		},
		{
			name:     "标准错误包含 unauthorized",
			err:      errors.New("unauthorized access"),
			expected: true,
		},
		{
			name:     "其他类型错误",
			err:      errors.New("resource not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsAuthError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// TestIsTimeoutError 测试判断是否是超时错误
func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 Timeout 错误",
			err:      utils.NewConnectionTimeoutError("连接超时"),
			expected: true,
		},
		{
			name:     "网络超时错误",
			err:      &timeoutError{},
			expected: true,
		},
		{
			name:     "标准错误包含 timeout",
			err:      errors.New("request timeout"),
			expected: true,
		},
		{
			name:     "其他类型错误",
			err:      errors.New("resource not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsTimeoutError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// TestIsNetworkError 测试判断是否是网络错误
func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 Network 错误",
			err:      utils.NewClusterUnreachableError("集群不可达"),
			expected: true,
		},
		{
			name:     "连接被拒绝错误",
			err:      &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
			expected: true,
		},
		{
			name:     "标准错误包含 connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "其他类型错误",
			err:      errors.New("resource not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsNetworkError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// TestIsClientError 测试判断是否是客户端错误
func TestIsClientError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 Client 错误",
			err:      utils.NewInvalidArgumentsError("name", "不能为空"),
			expected: true,
		},
		{
			name:     "kubeconfig invalid 错误",
			err:      errors.New("kubeconfig is invalid"),
			expected: true,
		},
		{
			name:     "其他类型错误",
			err:      errors.New("internal server error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsClientError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsClientError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// TestIsServerError 测试判断是否是服务器错误
func TestIsServerError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "ErrorInfo 类型的 Server 错误",
			err:      utils.NewInternalError("内部错误"),
			expected: true,
		},
		{
			name: "K8S StatusError 类型的 InternalError",
			err: &k8serrors.StatusError{
				ErrStatus: metav1.Status{
					Reason: metav1.StatusReasonInternalError,
					Code:   500,
				},
			},
			expected: true,
		},
		{
			name:     "未知错误默认为服务器错误",
			err:      errors.New("unknown error"),
			expected: true,
		},
		{
			name:     "客户端错误",
			err:      errors.New("invalid argument"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsServerError(tt.err)
			if result != tt.expected {
				t.Errorf("期望 IsServerError 返回 %v，实际返回 %v", tt.expected, result)
			}
		})
	}
}

// ========== 错误包装函数测试 ==========

// TestWrapError 测试包装错误
func TestWrapError(t *testing.T) {
	originalErr := errors.New("resource not found")
	context := "查询 Pod 时"

	errInfo := utils.WrapError(originalErr, context)

	if errInfo == nil {
		t.Fatal("期望返回 ErrorInfo，实际为 nil")
	}

	if errInfo.Type != utils.ErrTypeNotFound {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNotFound, errInfo.Type)
	}

	if errInfo.Details == "" {
		t.Error("期望详情不为空")
	}

	// 验证上下文信息被添加到详情中
	expectedPrefix := "查询 Pod 时"
	if len(errInfo.Details) < len(expectedPrefix) || errInfo.Details[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("期望详情以 '%s' 开头，实际为 '%s'", expectedPrefix, errInfo.Details)
	}
}

// TestWrapError_NilError 测试包装 nil 错误
func TestWrapError_NilError(t *testing.T) {
	errInfo := utils.WrapError(nil, "上下文")

	if errInfo != nil {
		t.Error("期望返回 nil，实际返回了 ErrorInfo")
	}
}

// TestWrapErrorWithCode 测试使用指定错误码包装错误
func TestWrapErrorWithCode(t *testing.T) {
	originalErr := errors.New("connection failed")

	errInfo := utils.WrapErrorWithCode(
		originalErr,
		utils.ErrTypeNetwork,
		utils.ErrCodeClusterUnreachable,
		"无法连接到集群",
	)

	if errInfo == nil {
		t.Fatal("期望返回 ErrorInfo，实际为 nil")
	}

	if errInfo.Type != utils.ErrTypeNetwork {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNetwork, errInfo.Type)
	}

	if errInfo.Code != utils.ErrCodeClusterUnreachable {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeClusterUnreachable, errInfo.Code)
	}

	if errInfo.Message != "无法连接到集群" {
		t.Errorf("期望消息为 '无法连接到集群'，实际为 '%s'", errInfo.Message)
	}

	if errInfo.Details != originalErr.Error() {
		t.Errorf("期望详情为 '%s'，实际为 '%s'", originalErr.Error(), errInfo.Details)
	}

	if errInfo.Suggestion == "" {
		t.Error("期望建议不为空")
	}
}

// ========== 错误建议生成测试 ==========

// TestGetSuggestion 测试获取错误建议
func TestGetSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		errCode  string
		hasValue bool
	}{
		{
			name:     "kubeconfig 未找到",
			errCode:  utils.ErrCodeKubeconfigNotFound,
			hasValue: true,
		},
		{
			name:     "context 未找到",
			errCode:  utils.ErrCodeContextNotFound,
			hasValue: true,
		},
		{
			name:     "集群不可达",
			errCode:  utils.ErrCodeClusterUnreachable,
			hasValue: true,
		},
		{
			name:     "权限不足",
			errCode:  utils.ErrCodePermissionDenied,
			hasValue: true,
		},
		{
			name:     "资源未找到",
			errCode:  utils.ErrCodeResourceNotFound,
			hasValue: true,
		},
		{
			name:     "未知错误码",
			errCode:  "UNKNOWN_ERROR",
			hasValue: true, // 应该返回默认建议
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := utils.GetSuggestion(tt.errCode)

			if tt.hasValue && suggestion == "" {
				t.Error("期望建议不为空")
			}
		})
	}
}

// TestGetSuggestionForError 测试根据错误获取建议
func TestGetSuggestionForError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		hasValue bool
	}{
		{
			name:     "kubeconfig invalid 错误",
			err:      errors.New("kubeconfig is invalid"),
			hasValue: true,
		},
		{
			name:     "权限错误",
			err:      errors.New("permission denied"),
			hasValue: true,
		},
		{
			name:     "nil 错误",
			err:      nil,
			hasValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestion := utils.GetSuggestionForError(tt.err)

			if tt.hasValue && suggestion == "" {
				t.Error("期望建议不为空")
			}

			if !tt.hasValue && suggestion != "" {
				t.Error("期望建议为空")
			}
		})
	}
}

// ========== 综合场景测试 ==========

// TestErrorHandling_CompleteFlow 测试完整的错误处理流程
func TestErrorHandling_CompleteFlow(t *testing.T) {
	// 1. 创建原始错误
	originalErr := errors.New("pod 'nginx' not found in namespace 'default'")

	// 2. 格式化错误
	errInfo := utils.FormatK8SError(originalErr, "Pod", "nginx")

	// 3. 验证错误分类
	if errInfo.Type != utils.ErrTypeNotFound {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", utils.ErrTypeNotFound, errInfo.Type)
	}

	// 4. 验证错误码
	if errInfo.Code != utils.ErrCodeResourceNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", utils.ErrCodeResourceNotFound, errInfo.Code)
	}

	// 5. 验证错误消息
	if errInfo.Message == "" {
		t.Error("期望错误消息不为空")
	}

	// 6. 验证错误详情
	if errInfo.Details == "" {
		t.Error("期望错误详情不为空")
	}

	// 7. 验证错误建议
	if errInfo.Suggestion == "" {
		t.Error("期望错误建议不为空")
	}

	// 8. 验证错误字符串格式
	errStr := errInfo.Error()
	if errStr == "" {
		t.Error("期望错误字符串不为空")
	}

	// 9. 验证错误判断函数
	if !utils.IsNotFoundError(errInfo) {
		t.Error("期望 IsNotFoundError 返回 true")
	}
}

// TestErrorHandling_MultipleErrorTypes 测试多种错误类型处理
func TestErrorHandling_MultipleErrorTypes(t *testing.T) {
	errorCases := []struct {
		name         string
		createError  func() error
		expectedType string
		expectedCode string
	}{
		{
			name: "kubeconfig 未找到",
			createError: func() error {
				return utils.NewKubeconfigNotFoundError("/path/to/config")
			},
			expectedType: utils.ErrTypeClient,
			expectedCode: utils.ErrCodeKubeconfigNotFound,
		},
		{
			name: "资源冲突",
			createError: func() error {
				return utils.NewResourceAlreadyExistsError("Pod", "nginx", "default")
			},
			expectedType: utils.ErrTypeConflict,
			expectedCode: utils.ErrCodeResourceAlreadyExists,
		},
		{
			name: "权限不足",
			createError: func() error {
				return utils.NewPermissionDeniedError("pods", "delete")
			},
			expectedType: utils.ErrTypeAuth,
			expectedCode: utils.ErrCodePermissionDenied,
		},
		{
			name: "连接超时",
			createError: func() error {
				return utils.NewConnectionTimeoutError("连接超时")
			},
			expectedType: utils.ErrTypeTimeout,
			expectedCode: utils.ErrCodeConnectionTimeout,
		},
		{
			name: "内部错误",
			createError: func() error {
				return utils.NewInternalError("服务器错误")
			},
			expectedType: utils.ErrTypeServer,
			expectedCode: utils.ErrCodeInternalError,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.createError()

			// 验证错误类型
			errType, errCode := utils.ClassifyError(err)
			if errType != tc.expectedType {
				t.Errorf("期望错误类型为 '%s'，实际为 '%s'", tc.expectedType, errType)
			}
			if errCode != tc.expectedCode {
				t.Errorf("期望错误码为 '%s'，实际为 '%s'", tc.expectedCode, errCode)
			}

			// 验证错误格式化
			errInfo := utils.FormatError(err)
			if errInfo == nil {
				t.Fatal("期望返回 ErrorInfo，实际为 nil")
			}

			// 验证错误信息完整性
			if errInfo.Type == "" {
				t.Error("期望错误类型不为空")
			}
			if errInfo.Code == "" {
				t.Error("期望错误码不为空")
			}
			if errInfo.Message == "" {
				t.Error("期望错误消息不为空")
			}
		})
	}
}
