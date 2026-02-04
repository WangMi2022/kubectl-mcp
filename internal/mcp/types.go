package mcp

import "time"

// ========== MCP 请求数据类型 ==========

// UserInfo 用户信息
type UserInfo struct {
	ID   string `json:"id"`             // 用户 ID
	Name string `json:"name"`           // 用户名称
	Role string `json:"role,omitempty"` // 用户角色
}

// OutputVerbosity 输出详细程度
type OutputVerbosity string

const (
	VerbosityBrief    OutputVerbosity = "brief"    // 简洁模式：只返回关键字段
	VerbosityStandard OutputVerbosity = "standard" // 标准模式：返回常用字段（默认）
	VerbosityDetailed OutputVerbosity = "detailed" // 详细模式：返回完整信息
)

// PaginationRequest 分页请求参数
type PaginationRequest struct {
	Page     int `json:"page,omitempty"`     // 页码（从 1 开始）
	PageSize int `json:"pageSize,omitempty"` // 每页数量（默认 50，最大 500）
}

// ToolCallRequest 工具调用请求
type ToolCallRequest struct {
	Tool       string                 `json:"tool"`                 // 工具名称（优先）
	Name       string                 `json:"name,omitempty"`       // 工具名称（兼容字段，与 tool 二选一）
	Arguments  map[string]interface{} `json:"arguments,omitempty"`  // 工具参数
	User       *UserInfo              `json:"user,omitempty"`       // 用户信息
	Context    string                 `json:"context,omitempty"`    // 指定的 K8S context
	RequestID  string                 `json:"requestId,omitempty"`  // 请求 ID（用于追踪）
	Verbosity  OutputVerbosity        `json:"verbosity,omitempty"`  // 输出详细程度
	Pagination *PaginationRequest     `json:"pagination,omitempty"` // 分页参数
}

// GetToolName 获取工具名称，优先使用 Tool 字段，其次使用 Name 字段
func (r *ToolCallRequest) GetToolName() string {
	if r.Tool != "" {
		return r.Tool
	}
	return r.Name
}

// ========== MCP 响应数据类型 ==========

// ErrorInfo 错误信息
type ErrorInfo struct {
	Type       string `json:"type"`                 // 错误类型: CLIENT_ERROR, SERVER_ERROR, NETWORK_ERROR, AUTH_ERROR, NOT_FOUND, CONFLICT, TIMEOUT
	Code       string `json:"code"`                 // 错误码
	Message    string `json:"message"`              // 错误消息
	Details    string `json:"details,omitempty"`    // 详细信息
	Suggestion string `json:"suggestion,omitempty"` // 修复建议
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Page       int  `json:"page"`       // 当前页码
	PageSize   int  `json:"pageSize"`   // 每页数量
	TotalCount int  `json:"totalCount"` // 总数量
	TotalPages int  `json:"totalPages"` // 总页数
	HasMore    bool `json:"hasMore"`    // 是否有更多数据
}

// ContentItem MCP 标准内容项
type ContentItem struct {
	Type string `json:"type"`           // 内容类型: text, image, etc.
	Text string `json:"text,omitempty"` // 文本内容
}

// ToolCallResponse 工具调用响应
type ToolCallResponse struct {
	Success    bool            `json:"success"`              // 是否成功
	Data       interface{}     `json:"data,omitempty"`       // 返回数据（原始格式）
	Content    []ContentItem   `json:"content,omitempty"`    // MCP 标准格式内容
	Error      *ErrorInfo      `json:"error,omitempty"`      // 错误信息
	RequestID  string          `json:"requestId,omitempty"`  // 请求 ID
	Duration   int64           `json:"duration,omitempty"`   // 执行耗时（毫秒）
	Context    string          `json:"context,omitempty"`    // 使用的 K8S context
	Pagination *PaginationInfo `json:"pagination,omitempty"` // 分页信息
}

// ========== 工具列表响应 ==========

// ToolInfo 工具信息（用于列表返回）
type ToolInfo struct {
	Name                 string                 `json:"name"`                 // 工具名称
	Description          string                 `json:"description"`          // 工具描述
	Category             string                 `json:"category"`             // 工具分类: query, create, update, delete
	RequiresConfirmation bool                   `json:"requiresConfirmation"` // 是否需要确认
	InputSchema          map[string]interface{} `json:"inputSchema"`          // 输入参数 Schema
	Example              string                 `json:"example,omitempty"`    // 使用示例
	RiskLevel            string                 `json:"riskLevel,omitempty"`  // 风险等级: low, medium, high
}

// ToolListResponse 工具列表响应
type ToolListResponse struct {
	Tools      []*ToolInfo `json:"tools"`                // 工具列表
	TotalCount int         `json:"totalCount"`           // 工具总数
	Categories []string    `json:"categories,omitempty"` // 可用分类
}

// ========== 健康检查响应 ==========

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string    `json:"status"`            // 状态: healthy, unhealthy, degraded
	Version   string    `json:"version"`           // 服务器版本
	Contexts  []string  `json:"contexts"`          // 可用的 K8S context 列表
	Current   string    `json:"current"`           // 当前使用的 context
	Timestamp time.Time `json:"timestamp"`         // 检查时间
	Uptime    int64     `json:"uptime,omitempty"`  // 运行时间（秒）
	Message   string    `json:"message,omitempty"` // 附加消息
}

// ========== Context 列表响应 ==========

// ContextInfo Context 信息
type ContextInfo struct {
	Name      string `json:"name"`      // context 名称
	Cluster   string `json:"cluster"`   // 集群名称
	User      string `json:"user"`      // 用户名称
	Namespace string `json:"namespace"` // 默认命名空间
	Current   bool   `json:"current"`   // 是否为当前 context
}

// ContextListResponse Context 列表响应
type ContextListResponse struct {
	Contexts   []*ContextInfo `json:"contexts"`   // context 列表
	Current    string         `json:"current"`    // 当前 context
	TotalCount int            `json:"totalCount"` // context 总数
}

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

// 错误码定义
const (
	// Kubeconfig 相关错误
	ErrCodeKubeconfigNotFound = "KUBECONFIG_NOT_FOUND" // kubeconfig 文件未找到
	ErrCodeKubeconfigInvalid  = "KUBECONFIG_INVALID"   // kubeconfig 文件无效
	ErrCodeContextNotFound    = "CONTEXT_NOT_FOUND"    // context 不存在

	// 连接相关错误
	ErrCodeClusterUnreachable = "CLUSTER_UNREACHABLE" // 集群不可达
	ErrCodeConnectionTimeout  = "CONNECTION_TIMEOUT"  // 连接超时
	ErrCodeAuthFailed         = "AUTH_FAILED"         // 认证失败

	// 资源相关错误
	ErrCodeResourceNotFound      = "RESOURCE_NOT_FOUND"      // 资源未找到
	ErrCodeResourceAlreadyExists = "RESOURCE_ALREADY_EXISTS" // 资源已存在
	ErrCodeInvalidResource       = "INVALID_RESOURCE"        // 无效的资源定义

	// 参数相关错误
	ErrCodeInvalidArguments = "INVALID_ARGUMENTS" // 无效的参数
	ErrCodeMissingArguments = "MISSING_ARGUMENTS" // 缺少必填参数
	ErrCodeInvalidToolName  = "INVALID_TOOL_NAME" // 无效的工具名称
	ErrCodeToolNotFound     = "TOOL_NOT_FOUND"    // 工具不存在

	// 权限相关错误
	ErrCodePermissionDenied = "PERMISSION_DENIED" // 权限不足
	ErrCodeUnauthorized     = "UNAUTHORIZED"      // 未授权

	// 请求相关错误
	ErrCodeInvalidRequest  = "INVALID_REQUEST"   // 无效的请求
	ErrCodeRequestTimeout  = "REQUEST_TIMEOUT"   // 请求超时
	ErrCodeTooManyRequests = "TOO_MANY_REQUESTS" // 请求过多
)

// ========== 辅助函数 ==========

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

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) *ToolCallResponse {
	return &ToolCallResponse{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(errInfo *ErrorInfo) *ToolCallResponse {
	return &ToolCallResponse{
		Success: false,
		Error:   errInfo,
	}
}

// WithRequestID 添加请求 ID
func (r *ToolCallResponse) WithRequestID(requestID string) *ToolCallResponse {
	r.RequestID = requestID
	return r
}

// WithDuration 添加执行耗时
func (r *ToolCallResponse) WithDuration(durationMs int64) *ToolCallResponse {
	r.Duration = durationMs
	return r
}

// WithContext 添加 context 信息
func (r *ToolCallResponse) WithContext(context string) *ToolCallResponse {
	r.Context = context
	return r
}

// WithPagination 添加分页信息
func (r *ToolCallResponse) WithPagination(pagination *PaginationInfo) *ToolCallResponse {
	r.Pagination = pagination
	return r
}

// ========== 分页辅助函数 ==========

// DefaultPageSize 默认每页数量
const DefaultPageSize = 50

// MaxPageSize 最大每页数量
const MaxPageSize = 500

// NormalizePagination 规范化分页参数
func NormalizePagination(req *PaginationRequest) *PaginationRequest {
	if req == nil {
		return &PaginationRequest{
			Page:     1,
			PageSize: DefaultPageSize,
		}
	}

	result := &PaginationRequest{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 规范化页码
	if result.Page < 1 {
		result.Page = 1
	}

	// 规范化每页数量
	if result.PageSize < 1 {
		result.PageSize = DefaultPageSize
	} else if result.PageSize > MaxPageSize {
		result.PageSize = MaxPageSize
	}

	return result
}

// CalculatePagination 计算分页信息
func CalculatePagination(page, pageSize, totalCount int) *PaginationInfo {
	totalPages := (totalCount + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return &PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

// PaginateSlice 对切片进行分页
// 返回分页后的起始和结束索引
func PaginateSlice(totalLen, page, pageSize int) (start, end int) {
	start = (page - 1) * pageSize
	if start >= totalLen {
		start = totalLen
	}

	end = start + pageSize
	if end > totalLen {
		end = totalLen
	}

	return start, end
}

// ========== 输出详细程度辅助函数 ==========

// GetVerbosity 获取输出详细程度，如果未指定则返回默认值
func GetVerbosity(v OutputVerbosity) OutputVerbosity {
	switch v {
	case VerbosityBrief, VerbosityStandard, VerbosityDetailed:
		return v
	default:
		return VerbosityStandard
	}
}

// IsVerbosityValid 检查输出详细程度是否有效
func IsVerbosityValid(v OutputVerbosity) bool {
	switch v {
	case VerbosityBrief, VerbosityStandard, VerbosityDetailed, "":
		return true
	default:
		return false
	}
}
