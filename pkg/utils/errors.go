package utils

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"syscall"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// ========== 错误类型常量 ==========

// 错误类型
const (
	ErrTypeClient   = "CLIENT_ERROR"  // 客户端错误（4xx）
	ErrTypeServer   = "SERVER_ERROR"  // 服务端错误（5xx）
	ErrTypeNetwork  = "NETWORK_ERROR" // 网络错误
	ErrTypeAuth     = "AUTH_ERROR"    // 认证错误
	ErrTypeNotFound = "NOT_FOUND"     // 资源未找到
	ErrTypeConflict = "CONFLICT"      // 资源冲突
	ErrTypeTimeout  = "TIMEOUT"       // 超时错误
)

// ========== 错误码常量 ==========

const (
	// Kubeconfig 相关错误
	ErrCodeKubeconfigNotFound = "KUBECONFIG_NOT_FOUND" // kubeconfig 文件未找到
	ErrCodeKubeconfigInvalid  = "KUBECONFIG_INVALID"   // kubeconfig 文件无效
	ErrCodeContextNotFound    = "CONTEXT_NOT_FOUND"    // context 不存在

	// 连接相关错误
	ErrCodeClusterUnreachable = "CLUSTER_UNREACHABLE" // 集群不可达
	ErrCodeConnectionTimeout  = "CONNECTION_TIMEOUT"  // 连接超时
	ErrCodeAuthFailed         = "AUTH_FAILED"         // 认证失败
	ErrCodeConnectionRefused  = "CONNECTION_REFUSED"  // 连接被拒绝
	ErrCodeDNSError           = "DNS_ERROR"           // DNS 解析错误

	// 资源相关错误
	ErrCodeResourceNotFound      = "RESOURCE_NOT_FOUND"      // 资源未找到
	ErrCodeResourceAlreadyExists = "RESOURCE_ALREADY_EXISTS" // 资源已存在
	ErrCodeInvalidResource       = "INVALID_RESOURCE"        // 无效的资源定义
	ErrCodeResourceConflict      = "RESOURCE_CONFLICT"       // 资源冲突

	// 参数相关错误
	ErrCodeInvalidArguments = "INVALID_ARGUMENTS" // 无效的参数
	ErrCodeMissingArguments = "MISSING_ARGUMENTS" // 缺少必填参数
	ErrCodeInvalidToolName  = "INVALID_TOOL_NAME" // 无效的工具名称
	ErrCodeToolNotFound     = "TOOL_NOT_FOUND"    // 工具不存在
	ErrCodeInvalidNamespace = "INVALID_NAMESPACE" // 无效的命名空间
	ErrCodeInvalidYAML      = "INVALID_YAML"      // 无效的 YAML 格式

	// 权限相关错误
	ErrCodePermissionDenied = "PERMISSION_DENIED" // 权限不足
	ErrCodeUnauthorized     = "UNAUTHORIZED"      // 未授权
	ErrCodeForbidden        = "FORBIDDEN"         // 禁止访问

	// 请求相关错误
	ErrCodeInvalidRequest  = "INVALID_REQUEST"   // 无效的请求
	ErrCodeRequestTimeout  = "REQUEST_TIMEOUT"   // 请求超时
	ErrCodeTooManyRequests = "TOO_MANY_REQUESTS" // 请求过多

	// 服务器相关错误
	ErrCodeInternalError    = "INTERNAL_ERROR"    // 内部错误
	ErrCodeServiceUnavail   = "SERVICE_UNAVAIL"   // 服务不可用
	ErrCodeConfigError      = "CONFIG_ERROR"      // 配置错误
	ErrCodeInitializeFailed = "INITIALIZE_FAILED" // 初始化失败
)

// ========== 错误信息结构 ==========

// ErrorInfo 错误信息结构
type ErrorInfo struct {
	Type       string `json:"type"`                 // 错误类型
	Code       string `json:"code"`                 // 错误码
	Message    string `json:"message"`              // 错误消息
	Details    string `json:"details,omitempty"`    // 详细信息
	Suggestion string `json:"suggestion,omitempty"` // 修复建议
}

// Error 实现 error 接口
func (e *ErrorInfo) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s - %s", e.Type, e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Code, e.Message)
}

// NewErrorInfo 创建错误信息
func NewErrorInfo(errType, code, message string) *ErrorInfo {
	return &ErrorInfo{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// WithDetails 添加详细信息
func (e *ErrorInfo) WithDetails(details string) *ErrorInfo {
	e.Details = details
	return e
}

// WithSuggestion 添加修复建议
func (e *ErrorInfo) WithSuggestion(suggestion string) *ErrorInfo {
	e.Suggestion = suggestion
	return e
}

// Clone 克隆错误信息
func (e *ErrorInfo) Clone() *ErrorInfo {
	return &ErrorInfo{
		Type:       e.Type,
		Code:       e.Code,
		Message:    e.Message,
		Details:    e.Details,
		Suggestion: e.Suggestion,
	}
}

// ========== 错误分类逻辑 ==========

// ClassifyError 对错误进行分类，返回错误类型和错误码
// 参数:
//   - err: 原始错误
//
// 返回:
//   - errType: 错误类型
//   - errCode: 错误码
func ClassifyError(err error) (errType string, errCode string) {
	if err == nil {
		return "", ""
	}

	// 检查是否是 ErrorInfo 类型
	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type, errInfo.Code
	}

	// 检查 K8S API 错误
	if statusErr, ok := err.(*k8serrors.StatusError); ok {
		return classifyK8SStatusError(statusErr)
	}

	// 检查网络错误
	if netErr := classifyNetworkError(err); netErr != nil {
		return netErr.Type, netErr.Code
	}

	// 检查超时错误
	if isTimeoutError(err) {
		return ErrTypeTimeout, ErrCodeRequestTimeout
	}

	// 检查错误消息中的关键字
	errMsg := strings.ToLower(err.Error())
	return classifyByErrorMessage(errMsg)
}

// classifyK8SStatusError 分类 K8S API 状态错误
func classifyK8SStatusError(statusErr *k8serrors.StatusError) (string, string) {
	status := statusErr.Status()

	switch status.Reason {
	case "NotFound":
		return ErrTypeNotFound, ErrCodeResourceNotFound
	case "AlreadyExists":
		return ErrTypeConflict, ErrCodeResourceAlreadyExists
	case "Conflict":
		return ErrTypeConflict, ErrCodeResourceConflict
	case "Forbidden":
		return ErrTypeAuth, ErrCodeForbidden
	case "Unauthorized":
		return ErrTypeAuth, ErrCodeUnauthorized
	case "BadRequest":
		return ErrTypeClient, ErrCodeInvalidRequest
	case "Invalid":
		return ErrTypeClient, ErrCodeInvalidResource
	case "Timeout":
		return ErrTypeTimeout, ErrCodeRequestTimeout
	case "ServerTimeout":
		return ErrTypeTimeout, ErrCodeConnectionTimeout
	case "ServiceUnavailable":
		return ErrTypeServer, ErrCodeServiceUnavail
	case "InternalError":
		return ErrTypeServer, ErrCodeInternalError
	default:
		// 根据 HTTP 状态码分类
		code := status.Code
		if code >= 400 && code < 500 {
			return ErrTypeClient, ErrCodeInvalidRequest
		} else if code >= 500 {
			return ErrTypeServer, ErrCodeInternalError
		}
		return ErrTypeServer, ErrCodeInternalError
	}
}

// classifyNetworkError 分类网络错误
func classifyNetworkError(err error) *ErrorInfo {
	// 检查连接被拒绝
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			// 检查是否是连接被拒绝
			var syscallErr syscall.Errno
			if errors.As(opErr.Err, &syscallErr) {
				if syscallErr == syscall.ECONNREFUSED {
					return NewErrorInfo(ErrTypeNetwork, ErrCodeConnectionRefused, "连接被拒绝")
				}
			}
			return NewErrorInfo(ErrTypeNetwork, ErrCodeClusterUnreachable, "无法连接到集群")
		}
	}

	// 检查 DNS 错误
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return NewErrorInfo(ErrTypeNetwork, ErrCodeDNSError, "DNS 解析失败")
	}

	// 检查其他网络错误
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return NewErrorInfo(ErrTypeTimeout, ErrCodeConnectionTimeout, "连接超时")
		}
		return NewErrorInfo(ErrTypeNetwork, ErrCodeClusterUnreachable, "网络错误")
	}

	return nil
}

// isTimeoutError 检查是否是超时错误
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// 检查 net.Error 接口
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	// 检查错误消息
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "deadline exceeded") ||
		strings.Contains(errMsg, "context deadline exceeded")
}

// classifyByErrorMessage 根据错误消息分类
func classifyByErrorMessage(errMsg string) (string, string) {
	// 认证相关
	if strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "unauthenticated") {
		return ErrTypeAuth, ErrCodeUnauthorized
	}

	if strings.Contains(errMsg, "forbidden") ||
		strings.Contains(errMsg, "permission denied") ||
		strings.Contains(errMsg, "access denied") {
		return ErrTypeAuth, ErrCodePermissionDenied
	}

	// 资源相关
	if strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "doesn't exist") ||
		strings.Contains(errMsg, "does not exist") {
		return ErrTypeNotFound, ErrCodeResourceNotFound
	}

	if strings.Contains(errMsg, "already exists") ||
		strings.Contains(errMsg, "conflict") {
		return ErrTypeConflict, ErrCodeResourceAlreadyExists
	}

	// 网络相关
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no route to host") {
		return ErrTypeNetwork, ErrCodeConnectionRefused
	}

	if strings.Contains(errMsg, "unreachable") ||
		strings.Contains(errMsg, "no such host") {
		return ErrTypeNetwork, ErrCodeClusterUnreachable
	}

	// 配置相关
	if strings.Contains(errMsg, "kubeconfig") {
		if strings.Contains(errMsg, "not found") ||
			strings.Contains(errMsg, "no such file") {
			return ErrTypeClient, ErrCodeKubeconfigNotFound
		}
		return ErrTypeClient, ErrCodeKubeconfigInvalid
	}

	if strings.Contains(errMsg, "context") &&
		(strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "不存在")) {
		return ErrTypeClient, ErrCodeContextNotFound
	}

	// 参数相关
	if strings.Contains(errMsg, "invalid") ||
		strings.Contains(errMsg, "无效") {
		return ErrTypeClient, ErrCodeInvalidArguments
	}

	if strings.Contains(errMsg, "missing") ||
		strings.Contains(errMsg, "required") ||
		strings.Contains(errMsg, "缺少") {
		return ErrTypeClient, ErrCodeMissingArguments
	}

	// 默认为服务器错误
	return ErrTypeServer, ErrCodeInternalError
}

// ========== 错误响应格式化 ==========

// FormatError 格式化错误为 ErrorInfo
// 参数:
//   - err: 原始错误
//
// 返回:
//   - *ErrorInfo: 格式化后的错误信息
func FormatError(err error) *ErrorInfo {
	if err == nil {
		return nil
	}

	// 如果已经是 ErrorInfo 类型，直接返回
	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo
	}

	// 分类错误
	errType, errCode := ClassifyError(err)

	// 创建错误信息
	result := NewErrorInfo(errType, errCode, getErrorMessage(err, errCode))
	result.Details = err.Error()
	result.Suggestion = GetSuggestion(errCode)

	return result
}

// FormatK8SError 格式化 K8S API 错误
// 参数:
//   - err: K8S API 错误
//   - resourceType: 资源类型（如 Pod、Deployment）
//   - resourceName: 资源名称
//
// 返回:
//   - *ErrorInfo: 格式化后的错误信息
func FormatK8SError(err error, resourceType, resourceName string) *ErrorInfo {
	if err == nil {
		return nil
	}

	errInfo := FormatError(err)

	// 添加资源上下文信息
	if resourceType != "" || resourceName != "" {
		contextInfo := ""
		if resourceType != "" && resourceName != "" {
			contextInfo = fmt.Sprintf("资源: %s/%s", resourceType, resourceName)
		} else if resourceType != "" {
			contextInfo = fmt.Sprintf("资源类型: %s", resourceType)
		} else {
			contextInfo = fmt.Sprintf("资源: %s", resourceName)
		}

		if errInfo.Details != "" {
			errInfo.Details = fmt.Sprintf("%s; %s", contextInfo, errInfo.Details)
		} else {
			errInfo.Details = contextInfo
		}
	}

	return errInfo
}

// getErrorMessage 根据错误码获取用户友好的错误消息
func getErrorMessage(err error, errCode string) string {
	switch errCode {
	// Kubeconfig 相关
	case ErrCodeKubeconfigNotFound:
		return "kubeconfig 文件未找到"
	case ErrCodeKubeconfigInvalid:
		return "kubeconfig 文件格式无效"
	case ErrCodeContextNotFound:
		return "指定的 context 不存在"

	// 连接相关
	case ErrCodeClusterUnreachable:
		return "无法连接到 Kubernetes 集群"
	case ErrCodeConnectionTimeout:
		return "连接集群超时"
	case ErrCodeConnectionRefused:
		return "集群连接被拒绝"
	case ErrCodeDNSError:
		return "无法解析集群地址"

	// 认证相关
	case ErrCodeAuthFailed:
		return "集群认证失败"
	case ErrCodeUnauthorized:
		return "未授权访问"
	case ErrCodePermissionDenied:
		return "权限不足"
	case ErrCodeForbidden:
		return "禁止访问该资源"

	// 资源相关
	case ErrCodeResourceNotFound:
		return "资源不存在"
	case ErrCodeResourceAlreadyExists:
		return "资源已存在"
	case ErrCodeResourceConflict:
		return "资源冲突"
	case ErrCodeInvalidResource:
		return "资源定义无效"

	// 参数相关
	case ErrCodeInvalidArguments:
		return "参数无效"
	case ErrCodeMissingArguments:
		return "缺少必填参数"
	case ErrCodeInvalidToolName:
		return "工具名称无效"
	case ErrCodeToolNotFound:
		return "工具不存在"
	case ErrCodeInvalidNamespace:
		return "命名空间无效"
	case ErrCodeInvalidYAML:
		return "YAML 格式无效"

	// 请求相关
	case ErrCodeInvalidRequest:
		return "请求格式无效"
	case ErrCodeRequestTimeout:
		return "请求处理超时"
	case ErrCodeTooManyRequests:
		return "请求过于频繁"

	// 服务器相关
	case ErrCodeInternalError:
		return "服务器内部错误"
	case ErrCodeServiceUnavail:
		return "服务暂时不可用"
	case ErrCodeConfigError:
		return "配置错误"
	case ErrCodeInitializeFailed:
		return "初始化失败"

	default:
		// 使用原始错误消息
		if err != nil {
			return err.Error()
		}
		return "未知错误"
	}
}

// ========== 错误建议生成 ==========

// suggestionMap 错误码到建议的映射
var suggestionMap = map[string]string{
	// Kubeconfig 相关
	ErrCodeKubeconfigNotFound: "请检查 kubeconfig 文件路径是否正确，或设置 KUBECONFIG 环境变量",
	ErrCodeKubeconfigInvalid:  "请检查 kubeconfig 文件格式是否正确，可使用 kubectl config view 验证",
	ErrCodeContextNotFound:    "请使用 kubectl config get-contexts 查看可用的 context 列表",

	// 连接相关
	ErrCodeClusterUnreachable: "请检查集群地址是否正确，网络是否可达，可使用 kubectl cluster-info 测试连接",
	ErrCodeConnectionTimeout:  "请检查网络连接和集群状态，可能需要增加超时时间或检查防火墙设置",
	ErrCodeConnectionRefused:  "请确认 Kubernetes API Server 是否正在运行，端口是否正确",
	ErrCodeDNSError:           "请检查集群地址是否正确，DNS 服务是否正常",

	// 认证相关
	ErrCodeAuthFailed:       "请检查 kubeconfig 中的认证信息是否有效，证书是否过期",
	ErrCodeUnauthorized:     "请检查是否已正确配置认证信息，或联系管理员获取访问权限",
	ErrCodePermissionDenied: "当前用户没有执行此操作的权限，请联系集群管理员配置 RBAC 权限",
	ErrCodeForbidden:        "访问被禁止，请检查 RBAC 配置或联系管理员",

	// 资源相关
	ErrCodeResourceNotFound:      "请检查资源名称和命名空间是否正确，可使用 kubectl get 命令确认资源是否存在",
	ErrCodeResourceAlreadyExists: "资源已存在，如需更新请使用 update 或 apply 操作",
	ErrCodeResourceConflict:      "资源版本冲突，请获取最新版本后重试",
	ErrCodeInvalidResource:       "请检查资源定义是否符合 Kubernetes API 规范",

	// 参数相关
	ErrCodeInvalidArguments: "请检查参数格式和类型是否正确",
	ErrCodeMissingArguments: "请提供所有必填参数",
	ErrCodeInvalidToolName:  "请检查工具名称是否正确，可使用 GET /tools 获取可用工具列表",
	ErrCodeToolNotFound:     "请使用 GET /tools 获取可用工具列表",
	ErrCodeInvalidNamespace: "请检查命名空间名称是否正确，可使用 get_namespaces 工具查看可用命名空间",
	ErrCodeInvalidYAML:      "请检查 YAML 格式是否正确，确保缩进和语法无误",

	// 请求相关
	ErrCodeInvalidRequest:  "请检查请求格式是否符合 API 规范",
	ErrCodeRequestTimeout:  "请求处理超时，请稍后重试或检查集群状态",
	ErrCodeTooManyRequests: "请求过于频繁，请稍后重试",

	// 服务器相关
	ErrCodeInternalError:    "服务器内部错误，请查看服务器日志获取详细信息",
	ErrCodeServiceUnavail:   "服务暂时不可用，请稍后重试",
	ErrCodeConfigError:      "请检查服务器配置是否正确",
	ErrCodeInitializeFailed: "服务初始化失败，请检查配置和依赖服务",
}

// GetSuggestion 根据错误码获取修复建议
// 参数:
//   - errCode: 错误码
//
// 返回:
//   - string: 修复建议
func GetSuggestion(errCode string) string {
	if suggestion, ok := suggestionMap[errCode]; ok {
		return suggestion
	}
	return "请检查错误详情并重试，如问题持续请联系管理员"
}

// GetSuggestionForError 根据错误获取修复建议
// 参数:
//   - err: 错误
//
// 返回:
//   - string: 修复建议
func GetSuggestionForError(err error) string {
	if err == nil {
		return ""
	}

	_, errCode := ClassifyError(err)
	return GetSuggestion(errCode)
}

// ========== 便捷错误创建函数 ==========

// NewKubeconfigNotFoundError 创建 kubeconfig 未找到错误
func NewKubeconfigNotFoundError(path string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeKubeconfigNotFound, "kubeconfig 文件未找到").
		WithDetails(fmt.Sprintf("路径: %s", path)).
		WithSuggestion(GetSuggestion(ErrCodeKubeconfigNotFound))
}

// NewKubeconfigInvalidError 创建 kubeconfig 无效错误
func NewKubeconfigInvalidError(details string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeKubeconfigInvalid, "kubeconfig 文件格式无效").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeKubeconfigInvalid))
}

// NewContextNotFoundError 创建 context 未找到错误
func NewContextNotFoundError(contextName string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeContextNotFound, "指定的 context 不存在").
		WithDetails(fmt.Sprintf("context: %s", contextName)).
		WithSuggestion(GetSuggestion(ErrCodeContextNotFound))
}

// NewClusterUnreachableError 创建集群不可达错误
func NewClusterUnreachableError(details string) *ErrorInfo {
	return NewErrorInfo(ErrTypeNetwork, ErrCodeClusterUnreachable, "无法连接到 Kubernetes 集群").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeClusterUnreachable))
}

// NewConnectionTimeoutError 创建连接超时错误
func NewConnectionTimeoutError(details string) *ErrorInfo {
	return NewErrorInfo(ErrTypeTimeout, ErrCodeConnectionTimeout, "连接集群超时").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeConnectionTimeout))
}

// NewAuthFailedError 创建认证失败错误
func NewAuthFailedError(details string) *ErrorInfo {
	return NewErrorInfo(ErrTypeAuth, ErrCodeAuthFailed, "集群认证失败").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeAuthFailed))
}

// NewPermissionDeniedError 创建权限不足错误
func NewPermissionDeniedError(resource, action string) *ErrorInfo {
	return NewErrorInfo(ErrTypeAuth, ErrCodePermissionDenied, "权限不足").
		WithDetails(fmt.Sprintf("无法对资源 %s 执行 %s 操作", resource, action)).
		WithSuggestion(GetSuggestion(ErrCodePermissionDenied))
}

// NewResourceNotFoundError 创建资源未找到错误
func NewResourceNotFoundError(resourceType, resourceName, namespace string) *ErrorInfo {
	details := fmt.Sprintf("类型: %s, 名称: %s", resourceType, resourceName)
	if namespace != "" {
		details += fmt.Sprintf(", 命名空间: %s", namespace)
	}
	return NewErrorInfo(ErrTypeNotFound, ErrCodeResourceNotFound, "资源不存在").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeResourceNotFound))
}

// NewResourceAlreadyExistsError 创建资源已存在错误
func NewResourceAlreadyExistsError(resourceType, resourceName, namespace string) *ErrorInfo {
	details := fmt.Sprintf("类型: %s, 名称: %s", resourceType, resourceName)
	if namespace != "" {
		details += fmt.Sprintf(", 命名空间: %s", namespace)
	}
	return NewErrorInfo(ErrTypeConflict, ErrCodeResourceAlreadyExists, "资源已存在").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeResourceAlreadyExists))
}

// NewInvalidArgumentsError 创建参数无效错误
func NewInvalidArgumentsError(paramName, reason string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeInvalidArguments, "参数无效").
		WithDetails(fmt.Sprintf("参数: %s, 原因: %s", paramName, reason)).
		WithSuggestion(GetSuggestion(ErrCodeInvalidArguments))
}

// NewMissingArgumentsError 创建缺少参数错误
func NewMissingArgumentsError(paramNames ...string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeMissingArguments, "缺少必填参数").
		WithDetails(fmt.Sprintf("缺少参数: %s", strings.Join(paramNames, ", "))).
		WithSuggestion(GetSuggestion(ErrCodeMissingArguments))
}

// NewToolNotFoundError 创建工具未找到错误
func NewToolNotFoundError(toolName string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeToolNotFound, "工具不存在").
		WithDetails(fmt.Sprintf("工具名称: %s", toolName)).
		WithSuggestion(GetSuggestion(ErrCodeToolNotFound))
}

// NewInvalidYAMLError 创建 YAML 无效错误
func NewInvalidYAMLError(details string) *ErrorInfo {
	return NewErrorInfo(ErrTypeClient, ErrCodeInvalidYAML, "YAML 格式无效").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeInvalidYAML))
}

// NewRequestTimeoutError 创建请求超时错误
func NewRequestTimeoutError(operation string) *ErrorInfo {
	return NewErrorInfo(ErrTypeTimeout, ErrCodeRequestTimeout, "请求处理超时").
		WithDetails(fmt.Sprintf("操作: %s", operation)).
		WithSuggestion(GetSuggestion(ErrCodeRequestTimeout))
}

// NewInternalError 创建内部错误
func NewInternalError(details string) *ErrorInfo {
	return NewErrorInfo(ErrTypeServer, ErrCodeInternalError, "服务器内部错误").
		WithDetails(details).
		WithSuggestion(GetSuggestion(ErrCodeInternalError))
}

// ========== 错误判断辅助函数 ==========

// IsNotFoundError 判断是否是资源未找到错误
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeNotFound
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeNotFound
}

// IsConflictError 判断是否是冲突错误
func IsConflictError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeConflict
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeConflict
}

// IsAuthError 判断是否是认证错误
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeAuth
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeAuth
}

// IsTimeoutError 判断是否是超时错误
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeTimeout
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeTimeout
}

// IsNetworkError 判断是否是网络错误
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeNetwork
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeNetwork
}

// IsClientError 判断是否是客户端错误
func IsClientError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeClient
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeClient
}

// IsServerError 判断是否是服务器错误
func IsServerError(err error) bool {
	if err == nil {
		return false
	}

	var errInfo *ErrorInfo
	if errors.As(err, &errInfo) {
		return errInfo.Type == ErrTypeServer
	}

	errType, _ := ClassifyError(err)
	return errType == ErrTypeServer
}

// ========== 错误包装函数 ==========

// WrapError 包装错误，添加上下文信息
// 参数:
//   - err: 原始错误
//   - context: 上下文信息
//
// 返回:
//   - *ErrorInfo: 包装后的错误信息
func WrapError(err error, context string) *ErrorInfo {
	if err == nil {
		return nil
	}

	errInfo := FormatError(err)
	if context != "" {
		if errInfo.Details != "" {
			errInfo.Details = fmt.Sprintf("%s; %s", context, errInfo.Details)
		} else {
			errInfo.Details = context
		}
	}

	return errInfo
}

// WrapErrorWithCode 包装错误，指定错误码
// 参数:
//   - err: 原始错误
//   - errType: 错误类型
//   - errCode: 错误码
//   - message: 错误消息
//
// 返回:
//   - *ErrorInfo: 包装后的错误信息
func WrapErrorWithCode(err error, errType, errCode, message string) *ErrorInfo {
	errInfo := NewErrorInfo(errType, errCode, message)
	if err != nil {
		errInfo.Details = err.Error()
	}
	errInfo.Suggestion = GetSuggestion(errCode)
	return errInfo
}
