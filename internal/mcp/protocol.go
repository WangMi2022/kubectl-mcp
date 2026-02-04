package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/tools"
)

// MCPHandler MCP 协议处理器
// 负责处理 MCP 协议的请求验证、工具路由和响应格式化
type MCPHandler struct {
	toolRegistry *tools.ToolRegistry   // 工具注册表
	k8sManager   *k8s.K8SClientManager // K8S 客户端管理器
	auditLogger  *audit.AuditLogger    // 审计日志器
	version      string                // 服务器版本
	startTime    time.Time             // 服务器启动时间
}

// MCPHandlerConfig MCP 处理器配置
type MCPHandlerConfig struct {
	Version string // 服务器版本
}

// NewMCPHandler 创建新的 MCP 协议处理器
// 参数:
//   - toolRegistry: 工具注册表
//   - k8sManager: K8S 客户端管理器
//   - auditLogger: 审计日志器
//   - config: 处理器配置
//
// 返回:
//   - *MCPHandler: MCP 处理器实例
//   - error: 错误信息
func NewMCPHandler(
	toolRegistry *tools.ToolRegistry,
	k8sManager *k8s.K8SClientManager,
	auditLogger *audit.AuditLogger,
	config *MCPHandlerConfig,
) (*MCPHandler, error) {
	if toolRegistry == nil {
		return nil, fmt.Errorf("工具注册表不能为空")
	}
	if k8sManager == nil {
		return nil, fmt.Errorf("K8S 客户端管理器不能为空")
	}

	version := "1.0.0"
	if config != nil && config.Version != "" {
		version = config.Version
	}

	return &MCPHandler{
		toolRegistry: toolRegistry,
		k8sManager:   k8sManager,
		auditLogger:  auditLogger,
		version:      version,
		startTime:    time.Now(),
	}, nil
}

// HandleToolCall 处理工具调用请求
// 参数:
//   - ctx: 上下文
//   - req: 工具调用请求
//
// 返回:
//   - *ToolCallResponse: 工具调用响应
func (h *MCPHandler) HandleToolCall(ctx context.Context, req *ToolCallRequest) *ToolCallResponse {
	startTime := time.Now()

	// 验证请求
	if err := h.ValidateRequest(req); err != nil {
		return h.formatErrorResponse(err, req)
	}

	// 获取使用的 context
	targetContext := req.Context
	if targetContext == "" {
		targetContext = h.k8sManager.GetCurrentContext()
	}

	// 获取工具名称（兼容 tool 和 name 字段）
	toolName := req.GetToolName()

	// 将分页和详细程度参数注入到 arguments 中
	args := h.injectRequestOptions(req)

	// 执行工具
	result, err := h.toolRegistry.ExecuteTool(ctx, toolName, args, h.k8sManager)

	// 计算执行耗时
	duration := time.Since(startTime)

	// 记录审计日志
	h.logOperation(req, targetContext, result, err, duration)

	// 格式化响应
	response := h.FormatResponse(result, err)
	response.RequestID = req.RequestID
	response.Duration = duration.Milliseconds()
	response.Context = targetContext

	// 处理分页信息（如果结果包含分页数据）
	if paginatedResult, ok := result.(*PaginatedResult); ok {
		response.Data = paginatedResult.Items
		response.Pagination = paginatedResult.Pagination
	}

	return response
}

// injectRequestOptions 将请求选项注入到参数中
func (h *MCPHandler) injectRequestOptions(req *ToolCallRequest) map[string]interface{} {
	args := req.Arguments
	if args == nil {
		args = make(map[string]interface{})
	}

	// 注入详细程度
	if req.Verbosity != "" {
		args["_verbosity"] = string(req.Verbosity)
	}

	// 注入分页参数
	if req.Pagination != nil {
		normalized := NormalizePagination(req.Pagination)
		args["_page"] = normalized.Page
		args["_pageSize"] = normalized.PageSize
	}

	return args
}

// PaginatedResult 分页结果包装
type PaginatedResult struct {
	Items      interface{}     `json:"items"`
	Pagination *PaginationInfo `json:"pagination"`
}

// ValidateRequest 验证请求参数
// 参数:
//   - req: 工具调用请求
//
// 返回:
//   - error: 验证错误
func (h *MCPHandler) ValidateRequest(req *ToolCallRequest) error {
	if req == nil {
		return &ValidationError{
			Code:    ErrCodeInvalidRequest,
			Message: "请求不能为空",
		}
	}

	// 获取工具名称（兼容 tool 和 name 字段）
	toolName := req.GetToolName()

	// 验证工具名称
	if toolName == "" {
		return &ValidationError{
			Code:    ErrCodeInvalidToolName,
			Message: "工具名称不能为空",
		}
	}

	// 验证工具名称格式（只允许字母、数字、下划线和连字符）
	if !isValidToolName(toolName) {
		return &ValidationError{
			Code:    ErrCodeInvalidToolName,
			Message: fmt.Sprintf("工具名称 '%s' 格式无效，只允许字母、数字、下划线和连字符", toolName),
		}
	}

	// 检查工具是否存在
	if !h.toolRegistry.HasTool(toolName) {
		return &ValidationError{
			Code:    ErrCodeToolNotFound,
			Message: fmt.Sprintf("工具 '%s' 不存在", toolName),
		}
	}

	// 验证 context（如果指定）
	if req.Context != "" {
		contexts := h.k8sManager.ListContexts()
		found := false
		for _, ctx := range contexts {
			if ctx.Name == req.Context {
				found = true
				break
			}
		}
		if !found {
			return &ValidationError{
				Code:    ErrCodeContextNotFound,
				Message: fmt.Sprintf("context '%s' 不存在", req.Context),
			}
		}
	}

	// 验证输出详细程度（如果指定）
	if req.Verbosity != "" && !IsVerbosityValid(req.Verbosity) {
		return &ValidationError{
			Code:       ErrCodeInvalidArguments,
			Message:    fmt.Sprintf("无效的输出详细程度 '%s'，有效值为: brief, standard, detailed", req.Verbosity),
			Suggestion: "请使用 brief（简洁）、standard（标准）或 detailed（详细）",
		}
	}

	// 验证分页参数（如果指定）
	if req.Pagination != nil {
		if req.Pagination.Page < 0 {
			return &ValidationError{
				Code:       ErrCodeInvalidArguments,
				Message:    "页码不能为负数",
				Suggestion: "页码从 1 开始",
			}
		}
		if req.Pagination.PageSize < 0 {
			return &ValidationError{
				Code:       ErrCodeInvalidArguments,
				Message:    "每页数量不能为负数",
				Suggestion: fmt.Sprintf("每页数量范围为 1-%d", MaxPageSize),
			}
		}
		if req.Pagination.PageSize > MaxPageSize {
			return &ValidationError{
				Code:       ErrCodeInvalidArguments,
				Message:    fmt.Sprintf("每页数量不能超过 %d", MaxPageSize),
				Suggestion: fmt.Sprintf("请将每页数量设置为 %d 或更小", MaxPageSize),
			}
		}
	}

	return nil
}

// FormatResponse 格式化响应
// 参数:
//   - result: 执行结果
//   - err: 执行错误
//
// 返回:
//   - *ToolCallResponse: 格式化后的响应
func (h *MCPHandler) FormatResponse(result interface{}, err error) *ToolCallResponse {
	if err != nil {
		return h.formatErrorFromError(err)
	}

	// 将结果序列化为 JSON 文本，用于 MCP content 格式
	var contentText string
	if result != nil {
		jsonBytes, jsonErr := json.Marshal(result)
		if jsonErr == nil {
			contentText = string(jsonBytes)
		}
	}

	response := &ToolCallResponse{
		Success: true,
		Data:    result,
	}

	// 添加 MCP 标准 content 格式
	if contentText != "" {
		response.Content = []ContentItem{
			{
				Type: "text",
				Text: contentText,
			},
		}
	}

	return response
}

// GetToolList 获取工具列表
// 参数:
//   - category: 工具分类（可选，为空则返回所有）
//
// 返回:
//   - *ToolListResponse: 工具列表响应
func (h *MCPHandler) GetToolList(category string) *ToolListResponse {
	var toolList []*tools.Tool

	if category != "" {
		toolList = h.toolRegistry.GetToolsByCategory(tools.ToolCategory(category))
	} else {
		toolList = h.toolRegistry.GetAllTools()
	}

	// 转换为 ToolInfo
	toolInfos := make([]*ToolInfo, 0, len(toolList))
	for _, tool := range toolList {
		toolInfo := &ToolInfo{
			Name:                 tool.Name,
			Description:          tool.Description,
			Category:             string(tool.Category),
			RequiresConfirmation: tool.RequiresConfirmation,
			Example:              tool.Example,
			RiskLevel:            tool.RiskLevel,
		}

		// 转换 InputSchema
		if tool.InputSchema != nil {
			toolInfo.InputSchema = h.convertInputSchema(tool.InputSchema)
		}

		toolInfos = append(toolInfos, toolInfo)
	}

	// 获取所有分类
	categories := []string{
		string(tools.CategoryQuery),
		string(tools.CategoryCreate),
		string(tools.CategoryUpdate),
		string(tools.CategoryDelete),
	}

	return &ToolListResponse{
		Tools:      toolInfos,
		TotalCount: len(toolInfos),
		Categories: categories,
	}
}

// GetHealth 获取健康状态
// 返回:
//   - *HealthResponse: 健康检查响应
func (h *MCPHandler) GetHealth() *HealthResponse {
	contexts := h.k8sManager.ListContexts()
	contextNames := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		contextNames = append(contextNames, ctx.Name)
	}

	uptime := int64(time.Since(h.startTime).Seconds())

	return &HealthResponse{
		Status:    "healthy",
		Version:   h.version,
		Contexts:  contextNames,
		Current:   h.k8sManager.GetCurrentContext(),
		Timestamp: time.Now(),
		Uptime:    uptime,
	}
}

// GetContextList 获取 context 列表
// 返回:
//   - *ContextListResponse: context 列表响应
func (h *MCPHandler) GetContextList() *ContextListResponse {
	k8sContexts := h.k8sManager.ListContexts()
	currentContext := h.k8sManager.GetCurrentContext()

	contexts := make([]*ContextInfo, 0, len(k8sContexts))
	for _, ctx := range k8sContexts {
		contexts = append(contexts, &ContextInfo{
			Name:      ctx.Name,
			Cluster:   ctx.Cluster,
			User:      ctx.User,
			Namespace: ctx.Namespace,
			Current:   ctx.Name == currentContext,
		})
	}

	return &ContextListResponse{
		Contexts:   contexts,
		Current:    currentContext,
		TotalCount: len(contexts),
	}
}

// CreatePaginatedResult 创建分页结果
// 参数:
//   - items: 数据项切片
//   - page: 当前页码
//   - pageSize: 每页数量
//   - totalCount: 总数量
//
// 返回:
//   - *PaginatedResult: 分页结果
func CreatePaginatedResult(items interface{}, page, pageSize, totalCount int) *PaginatedResult {
	return &PaginatedResult{
		Items:      items,
		Pagination: CalculatePagination(page, pageSize, totalCount),
	}
}

// GetVersion 获取服务器版本
func (h *MCPHandler) GetVersion() string {
	return h.version
}

// GetUptime 获取服务器运行时间（秒）
func (h *MCPHandler) GetUptime() int64 {
	return int64(time.Since(h.startTime).Seconds())
}

// GetK8SManager 获取 K8S 客户端管理器（用于测试）
func (h *MCPHandler) GetK8SManager() *k8s.K8SClientManager {
	return h.k8sManager
}

// ========== 私有辅助方法 ==========

// logOperation 记录操作日志
func (h *MCPHandler) logOperation(req *ToolCallRequest, context string, result interface{}, err error, duration time.Duration) {
	if h.auditLogger == nil {
		return
	}

	// 构建操作日志
	opLog := &audit.OperationLog{
		Timestamp: time.Now(),
		Tool:      req.GetToolName(),
		Arguments: req.Arguments,
		Context:   context,
		Success:   err == nil,
		Duration:  duration,
	}

	// 添加用户信息
	if req.User != nil {
		opLog.User = &audit.UserInfo{
			ID:   req.User.ID,
			Name: req.User.Name,
			Role: req.User.Role,
		}
	}

	// 添加命名空间信息
	if ns, ok := req.Arguments["namespace"].(string); ok {
		opLog.Namespace = ns
	}

	// 添加资源信息
	if resourceType, ok := req.Arguments["resourceType"].(string); ok {
		opLog.ResourceType = resourceType
	}
	if name, ok := req.Arguments["name"].(string); ok {
		opLog.ResourceName = name
	}

	// 添加错误信息
	if err != nil {
		opLog.Error = err.Error()
	}

	// 记录日志
	_ = h.auditLogger.LogOperation(opLog)
}

// formatErrorResponse 格式化验证错误响应
func (h *MCPHandler) formatErrorResponse(err error, req *ToolCallRequest) *ToolCallResponse {
	response := h.formatErrorFromError(err)
	if req != nil {
		response.RequestID = req.RequestID
	}
	return response
}

// formatErrorFromError 从错误创建错误响应
func (h *MCPHandler) formatErrorFromError(err error) *ToolCallResponse {
	if err == nil {
		return &ToolCallResponse{Success: true}
	}

	// 检查是否为验证错误
	if validationErr, ok := err.(*ValidationError); ok {
		return &ToolCallResponse{
			Success: false,
			Error: &ErrorInfo{
				Type:       ErrTypeClient,
				Code:       validationErr.Code,
				Message:    validationErr.Message,
				Suggestion: validationErr.Suggestion,
			},
		}
	}

	// 分析错误类型
	errInfo := h.classifyError(err)

	return &ToolCallResponse{
		Success: false,
		Error:   errInfo,
	}
}

// classifyError 分类错误
func (h *MCPHandler) classifyError(err error) *ErrorInfo {
	errMsg := err.Error()

	// 检查常见错误模式
	switch {
	case strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "不存在"):
		return &ErrorInfo{
			Type:       ErrTypeNotFound,
			Code:       ErrCodeResourceNotFound,
			Message:    errMsg,
			Suggestion: "请检查资源名称和命名空间是否正确",
		}

	case strings.Contains(errMsg, "already exists") || strings.Contains(errMsg, "已存在"):
		return &ErrorInfo{
			Type:       ErrTypeConflict,
			Code:       ErrCodeResourceAlreadyExists,
			Message:    errMsg,
			Suggestion: "资源已存在，请使用更新操作或删除后重新创建",
		}

	case strings.Contains(errMsg, "forbidden") || strings.Contains(errMsg, "权限"):
		return &ErrorInfo{
			Type:       ErrTypeAuth,
			Code:       ErrCodePermissionDenied,
			Message:    errMsg,
			Suggestion: "请检查 kubeconfig 中的用户是否有足够的权限",
		}

	case strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "认证"):
		return &ErrorInfo{
			Type:       ErrTypeAuth,
			Code:       ErrCodeAuthFailed,
			Message:    errMsg,
			Suggestion: "请检查 kubeconfig 中的认证信息是否正确",
		}

	case strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "超时"):
		return &ErrorInfo{
			Type:       ErrTypeTimeout,
			Code:       ErrCodeConnectionTimeout,
			Message:    errMsg,
			Suggestion: "请检查网络连接和集群状态",
		}

	case strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "连接"):
		return &ErrorInfo{
			Type:       ErrTypeNetwork,
			Code:       ErrCodeClusterUnreachable,
			Message:    errMsg,
			Suggestion: "请检查集群是否可达，API server 是否正常运行",
		}

	case strings.Contains(errMsg, "参数") || strings.Contains(errMsg, "argument"):
		return &ErrorInfo{
			Type:       ErrTypeClient,
			Code:       ErrCodeInvalidArguments,
			Message:    errMsg,
			Suggestion: "请检查参数格式和类型是否正确",
		}

	default:
		return &ErrorInfo{
			Type:    ErrTypeServer,
			Code:    "INTERNAL_ERROR",
			Message: errMsg,
		}
	}
}

// convertInputSchema 转换 InputSchema 为 map
func (h *MCPHandler) convertInputSchema(schema *tools.InputSchema) map[string]interface{} {
	if schema == nil {
		return nil
	}

	result := map[string]interface{}{
		"type": schema.Type,
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	if len(schema.Properties) > 0 {
		properties := make(map[string]interface{})
		for name, prop := range schema.Properties {
			properties[name] = h.convertParameterSchema(prop)
		}
		result["properties"] = properties
	}

	return result
}

// convertParameterSchema 转换 ParameterSchema 为 map
func (h *MCPHandler) convertParameterSchema(schema *tools.ParameterSchema) map[string]interface{} {
	if schema == nil {
		return nil
	}

	result := map[string]interface{}{
		"type": schema.Type,
	}

	if schema.Description != "" {
		result["description"] = schema.Description
	}

	if schema.Default != nil {
		result["default"] = schema.Default
	}

	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	if schema.Items != nil {
		result["items"] = h.convertParameterSchema(schema.Items)
	}

	if len(schema.Properties) > 0 {
		properties := make(map[string]interface{})
		for name, prop := range schema.Properties {
			properties[name] = h.convertParameterSchema(prop)
		}
		result["properties"] = properties
	}

	return result
}

// isValidToolName 验证工具名称格式
func isValidToolName(name string) bool {
	if name == "" {
		return false
	}

	for _, c := range name {
		if !((c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '_' || c == '-') {
			return false
		}
	}

	return true
}

// ========== 验证错误类型 ==========

// ValidationError 验证错误
type ValidationError struct {
	Code       string // 错误码
	Message    string // 错误消息
	Suggestion string // 修复建议
}

// Error 实现 error 接口
func (e *ValidationError) Error() string {
	return e.Message
}
