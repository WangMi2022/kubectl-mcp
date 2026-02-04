package test

import (
	"context"
	"testing"
	"time"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/mcp"
	"kubectl-mcp/internal/tools"
)

// setupMCPHandler 创建测试用的 MCP 处理器
func setupMCPHandler(t *testing.T) (*mcp.MCPHandler, *k8s.K8SClientManager) {
	// 创建临时 kubeconfig
	kubeconfigPath := createTempKubeconfig(t)

	// 创建 K8S 客户端管理器
	k8sManager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}

	// 创建工具注册表
	toolRegistry := tools.NewToolRegistry()

	// 注册一个测试工具
	testTool := &tools.Tool{
		Name:        "test_tool",
		Description: "测试工具",
		Category:    tools.CategoryQuery,
		Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
			return map[string]string{"result": "success"}, nil
		},
		InputSchema: &tools.InputSchema{
			Type:     "object",
			Required: []string{"param1"},
			Properties: map[string]*tools.ParameterSchema{
				"param1": {
					Type:        "string",
					Description: "必填参数",
				},
				"param2": {
					Type:        "integer",
					Description: "可选参数",
				},
			},
		},
	}
	if err := toolRegistry.RegisterTool(testTool); err != nil {
		t.Fatalf("注册测试工具失败: %v", err)
	}

	// 创建审计日志器（可选）
	auditLogger, _ := audit.NewAuditLogger(&audit.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})

	// 创建 MCP 处理器
	handler, err := mcp.NewMCPHandler(
		toolRegistry,
		k8sManager,
		auditLogger,
		&mcp.MCPHandlerConfig{
			Version: "1.0.0-test",
		},
	)
	if err != nil {
		t.Fatalf("创建 MCP 处理器失败: %v", err)
	}

	return handler, k8sManager
}

// ========== 请求解析测试 ==========

// TestHandleToolCall_ValidRequest 测试有效的工具调用请求
func TestHandleToolCall_ValidRequest(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
			"param2": 123,
		},
		RequestID: "test-request-1",
	}

	ctx := context.Background()
	response := handler.HandleToolCall(ctx, req)

	// 验证响应
	if !response.Success {
		t.Errorf("期望请求成功，但失败了: %v", response.Error)
	}

	if response.RequestID != "test-request-1" {
		t.Errorf("期望 RequestID 为 'test-request-1'，实际为 '%s'", response.RequestID)
	}

	if response.Duration < 0 {
		t.Error("期望 Duration 不为负数")
	}

	if response.Context == "" {
		t.Error("期望 Context 不为空")
	}

	// 验证返回数据
	if response.Data == nil {
		t.Fatal("期望返回数据不为空")
	}

	dataMap, ok := response.Data.(map[string]string)
	if !ok {
		t.Fatal("期望返回数据为 map[string]string 类型")
	}

	if dataMap["result"] != "success" {
		t.Errorf("期望返回结果为 'success'，实际为 '%s'", dataMap["result"])
	}
}

// TestHandleToolCall_WithContext 测试指定 context 的工具调用
func TestHandleToolCall_WithContext(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
		Context: "test-context",
	}

	ctx := context.Background()
	response := handler.HandleToolCall(ctx, req)

	if !response.Success {
		t.Errorf("期望请求成功，但失败了: %v", response.Error)
	}

	if response.Context != "test-context" {
		t.Errorf("期望 Context 为 'test-context'，实际为 '%s'", response.Context)
	}
}

// TestHandleToolCall_WithUser 测试包含用户信息的工具调用
func TestHandleToolCall_WithUser(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
		User: &mcp.UserInfo{
			ID:   "user123",
			Name: "testuser",
			Role: "admin",
		},
	}

	ctx := context.Background()
	response := handler.HandleToolCall(ctx, req)

	if !response.Success {
		t.Errorf("期望请求成功，但失败了: %v", response.Error)
	}
}

// TestHandleToolCall_WithVerbosity 测试指定输出详细程度的工具调用
func TestHandleToolCall_WithVerbosity(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	tests := []struct {
		name      string
		verbosity mcp.OutputVerbosity
		wantErr   bool
	}{
		{"简洁模式", mcp.VerbosityBrief, false},
		{"标准模式", mcp.VerbosityStandard, false},
		{"详细模式", mcp.VerbosityDetailed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.ToolCallRequest{
				Tool: "test_tool",
				Arguments: map[string]interface{}{
					"param1": "value1",
				},
				Verbosity: tt.verbosity,
			}

			ctx := context.Background()
			response := handler.HandleToolCall(ctx, req)

			if response.Success == tt.wantErr {
				t.Errorf("期望成功 = %v，实际成功 = %v", !tt.wantErr, response.Success)
			}
		})
	}
}

// TestHandleToolCall_WithPagination 测试包含分页参数的工具调用
func TestHandleToolCall_WithPagination(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
		Pagination: &mcp.PaginationRequest{
			Page:     2,
			PageSize: 20,
		},
	}

	ctx := context.Background()
	response := handler.HandleToolCall(ctx, req)

	if !response.Success {
		t.Errorf("期望请求成功，但失败了: %v", response.Error)
	}
}

// ========== 参数验证测试 ==========

// TestValidateRequest_NilRequest 测试空请求验证
func TestValidateRequest_NilRequest(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	err := handler.ValidateRequest(nil)
	if err == nil {
		t.Error("期望验证失败，但成功了")
	}

	validationErr, ok := err.(*mcp.ValidationError)
	if !ok {
		t.Fatal("期望返回 ValidationError 类型")
	}

	if validationErr.Code != mcp.ErrCodeInvalidRequest {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeInvalidRequest, validationErr.Code)
	}
}

// TestValidateRequest_EmptyToolName 测试空工具名称验证
func TestValidateRequest_EmptyToolName(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}

	err := handler.ValidateRequest(req)
	if err == nil {
		t.Error("期望验证失败，但成功了")
	}

	validationErr, ok := err.(*mcp.ValidationError)
	if !ok {
		t.Fatal("期望返回 ValidationError 类型")
	}

	if validationErr.Code != mcp.ErrCodeInvalidToolName {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeInvalidToolName, validationErr.Code)
	}
}

// TestValidateRequest_InvalidToolName 测试无效工具名称格式验证
func TestValidateRequest_InvalidToolName(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	tests := []struct {
		name     string
		toolName string
	}{
		{"包含空格", "test tool"},
		{"包含特殊字符", "test@tool"},
		{"包含中文", "测试工具"},
		{"包含点号", "test.tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.ToolCallRequest{
				Tool: tt.toolName,
				Arguments: map[string]interface{}{
					"param1": "value1",
				},
			}

			err := handler.ValidateRequest(req)
			if err == nil {
				t.Errorf("期望验证失败（工具名称: %s），但成功了", tt.toolName)
			}

			validationErr, ok := err.(*mcp.ValidationError)
			if !ok {
				t.Fatal("期望返回 ValidationError 类型")
			}

			if validationErr.Code != mcp.ErrCodeInvalidToolName {
				t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeInvalidToolName, validationErr.Code)
			}
		})
	}
}

// TestValidateRequest_ToolNotFound 测试工具不存在验证
func TestValidateRequest_ToolNotFound(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "nonexistent_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}

	err := handler.ValidateRequest(req)
	if err == nil {
		t.Error("期望验证失败，但成功了")
	}

	validationErr, ok := err.(*mcp.ValidationError)
	if !ok {
		t.Fatal("期望返回 ValidationError 类型")
	}

	if validationErr.Code != mcp.ErrCodeToolNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeToolNotFound, validationErr.Code)
	}
}

// TestValidateRequest_ContextNotFound 测试 context 不存在验证
func TestValidateRequest_ContextNotFound(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
		Context: "nonexistent-context",
	}

	err := handler.ValidateRequest(req)
	if err == nil {
		t.Error("期望验证失败，但成功了")
	}

	validationErr, ok := err.(*mcp.ValidationError)
	if !ok {
		t.Fatal("期望返回 ValidationError 类型")
	}

	if validationErr.Code != mcp.ErrCodeContextNotFound {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeContextNotFound, validationErr.Code)
	}
}

// TestValidateRequest_InvalidVerbosity 测试无效输出详细程度验证
func TestValidateRequest_InvalidVerbosity(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	req := &mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
		Verbosity: "invalid",
	}

	err := handler.ValidateRequest(req)
	if err == nil {
		t.Error("期望验证失败，但成功了")
	}

	validationErr, ok := err.(*mcp.ValidationError)
	if !ok {
		t.Fatal("期望返回 ValidationError 类型")
	}

	if validationErr.Code != mcp.ErrCodeInvalidArguments {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeInvalidArguments, validationErr.Code)
	}

	if validationErr.Suggestion == "" {
		t.Error("期望包含修复建议")
	}
}

// TestValidateRequest_InvalidPagination 测试无效分页参数验证
func TestValidateRequest_InvalidPagination(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	tests := []struct {
		name       string
		pagination *mcp.PaginationRequest
		wantErr    bool
	}{
		{
			name: "页码为负数",
			pagination: &mcp.PaginationRequest{
				Page:     -1,
				PageSize: 10,
			},
			wantErr: true,
		},
		{
			name: "每页数量为负数",
			pagination: &mcp.PaginationRequest{
				Page:     1,
				PageSize: -10,
			},
			wantErr: true,
		},
		{
			name: "每页数量超过最大值",
			pagination: &mcp.PaginationRequest{
				Page:     1,
				PageSize: 1000,
			},
			wantErr: true,
		},
		{
			name: "有效的分页参数",
			pagination: &mcp.PaginationRequest{
				Page:     1,
				PageSize: 50,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &mcp.ToolCallRequest{
				Tool: "test_tool",
				Arguments: map[string]interface{}{
					"param1": "value1",
				},
				Pagination: tt.pagination,
			}

			err := handler.ValidateRequest(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("期望错误 = %v，实际错误 = %v", tt.wantErr, err != nil)
			}

			if err != nil {
				validationErr, ok := err.(*mcp.ValidationError)
				if !ok {
					t.Fatal("期望返回 ValidationError 类型")
				}

				if validationErr.Code != mcp.ErrCodeInvalidArguments {
					t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeInvalidArguments, validationErr.Code)
				}

				if validationErr.Suggestion == "" {
					t.Error("期望包含修复建议")
				}
			}
		})
	}
}

// ========== 响应格式化测试 ==========

// TestFormatResponse_Success 测试成功响应格式化
func TestFormatResponse_Success(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	result := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	response := handler.FormatResponse(result, nil)

	if !response.Success {
		t.Error("期望响应成功")
	}

	if response.Data == nil {
		t.Fatal("期望返回数据不为空")
	}

	if response.Error != nil {
		t.Error("期望错误信息为空")
	}

	dataMap, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatal("期望返回数据为 map[string]interface{} 类型")
	}

	if dataMap["key1"] != "value1" {
		t.Errorf("期望 key1 为 'value1'，实际为 '%v'", dataMap["key1"])
	}

	if dataMap["key2"] != 123 {
		t.Errorf("期望 key2 为 123，实际为 '%v'", dataMap["key2"])
	}
}

// TestFormatResponse_ValidationError 测试验证错误响应格式化
func TestFormatResponse_ValidationError(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	err := &mcp.ValidationError{
		Code:       mcp.ErrCodeInvalidArguments,
		Message:    "参数无效",
		Suggestion: "请检查参数格式",
	}

	response := handler.FormatResponse(nil, err)

	if response.Success {
		t.Error("期望响应失败")
	}

	if response.Error == nil {
		t.Fatal("期望错误信息不为空")
	}

	if response.Error.Type != mcp.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", mcp.ErrTypeClient, response.Error.Type)
	}

	if response.Error.Code != mcp.ErrCodeInvalidArguments {
		t.Errorf("期望错误码为 '%s'，实际为 '%s'", mcp.ErrCodeInvalidArguments, response.Error.Code)
	}

	if response.Error.Message != "参数无效" {
		t.Errorf("期望错误消息为 '参数无效'，实际为 '%s'", response.Error.Message)
	}

	if response.Error.Suggestion != "请检查参数格式" {
		t.Errorf("期望修复建议为 '请检查参数格式'，实际为 '%s'", response.Error.Suggestion)
	}
}

// TestFormatResponse_GenericError 测试通用错误响应格式化
func TestFormatResponse_GenericError(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	tests := []struct {
		name         string
		err          error
		expectedType string
		expectedCode string
	}{
		{
			name:         "资源未找到错误",
			err:          &mcp.ValidationError{Code: mcp.ErrCodeResourceNotFound, Message: "Pod not found"},
			expectedType: mcp.ErrTypeClient,
			expectedCode: mcp.ErrCodeResourceNotFound,
		},
		{
			name:         "权限不足错误",
			err:          &mcp.ValidationError{Code: mcp.ErrCodePermissionDenied, Message: "forbidden"},
			expectedType: mcp.ErrTypeClient,
			expectedCode: mcp.ErrCodePermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := handler.FormatResponse(nil, tt.err)

			if response.Success {
				t.Error("期望响应失败")
			}

			if response.Error == nil {
				t.Fatal("期望错误信息不为空")
			}

			if response.Error.Type != tt.expectedType {
				t.Errorf("期望错误类型为 '%s'，实际为 '%s'", tt.expectedType, response.Error.Type)
			}

			if response.Error.Code != tt.expectedCode {
				t.Errorf("期望错误码为 '%s'，实际为 '%s'", tt.expectedCode, response.Error.Code)
			}
		})
	}
}

// ========== 工具列表测试 ==========

// TestGetToolList_AllTools 测试获取所有工具列表
func TestGetToolList_AllTools(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	response := handler.GetToolList("")

	if response == nil {
		t.Fatal("期望响应不为空")
	}

	if response.TotalCount == 0 {
		t.Error("期望工具总数大于 0")
	}

	if len(response.Tools) == 0 {
		t.Error("期望工具列表不为空")
	}

	if len(response.Categories) == 0 {
		t.Error("期望分类列表不为空")
	}

	// 验证测试工具存在
	found := false
	for _, tool := range response.Tools {
		if tool.Name == "test_tool" {
			found = true
			if tool.Description == "" {
				t.Error("期望工具描述不为空")
			}
			if tool.Category == "" {
				t.Error("期望工具分类不为空")
			}
			if tool.InputSchema == nil {
				t.Error("期望工具 InputSchema 不为空")
			}
			break
		}
	}

	if !found {
		t.Error("期望找到测试工具 'test_tool'")
	}
}

// TestGetToolList_ByCategory 测试按分类获取工具列表
func TestGetToolList_ByCategory(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	response := handler.GetToolList("query")

	if response == nil {
		t.Fatal("期望响应不为空")
	}

	// 验证所有返回的工具都属于 query 分类
	for _, tool := range response.Tools {
		if tool.Category != "query" {
			t.Errorf("期望工具分类为 'query'，实际为 '%s'", tool.Category)
		}
	}
}

// ========== 健康检查测试 ==========

// TestGetHealth 测试健康检查
func TestGetHealth(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	// 等待一小段时间以确保 uptime > 0
	time.Sleep(100 * time.Millisecond)

	response := handler.GetHealth()

	if response == nil {
		t.Fatal("期望响应不为空")
	}

	if response.Status != "healthy" {
		t.Errorf("期望状态为 'healthy'，实际为 '%s'", response.Status)
	}

	if response.Version == "" {
		t.Error("期望版本不为空")
	}

	if len(response.Contexts) == 0 {
		t.Error("期望 context 列表不为空")
	}

	if response.Current == "" {
		t.Error("期望当前 context 不为空")
	}

	if response.Uptime < 0 {
		t.Errorf("期望运行时间不为负数，实际为 %d", response.Uptime)
	}

	if response.Timestamp.IsZero() {
		t.Error("期望时间戳不为零值")
	}
}

// ========== Context 列表测试 ==========

// TestGetContextList 测试获取 context 列表
func TestGetContextList(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	response := handler.GetContextList()

	if response == nil {
		t.Fatal("期望响应不为空")
	}

	if response.TotalCount == 0 {
		t.Error("期望 context 总数大于 0")
	}

	if len(response.Contexts) == 0 {
		t.Error("期望 context 列表不为空")
	}

	if response.Current == "" {
		t.Error("期望当前 context 不为空")
	}

	// 验证至少有一个 context 被标记为 current
	foundCurrent := false
	for _, ctx := range response.Contexts {
		if ctx.Name == "" {
			t.Error("期望 context 名称不为空")
		}
		if ctx.Cluster == "" {
			t.Error("期望集群名称不为空")
		}
		if ctx.User == "" {
			t.Error("期望用户名称不为空")
		}
		if ctx.Current {
			foundCurrent = true
		}
	}

	if !foundCurrent {
		t.Error("期望至少有一个 context 被标记为 current")
	}
}

// ========== 辅助方法测试 ==========

// TestGetVersion 测试获取版本
func TestGetVersion(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	version := handler.GetVersion()
	if version != "1.0.0-test" {
		t.Errorf("期望版本为 '1.0.0-test'，实际为 '%s'", version)
	}
}

// TestGetUptime 测试获取运行时间
func TestGetUptime(t *testing.T) {
	handler, _ := setupMCPHandler(t)

	// 等待一小段时间
	time.Sleep(100 * time.Millisecond)

	uptime := handler.GetUptime()
	if uptime < 0 {
		t.Errorf("期望运行时间不为负数，实际为 %d", uptime)
	}
}

// TestNewMCPHandler_NilToolRegistry 测试创建处理器时工具注册表为空
func TestNewMCPHandler_NilToolRegistry(t *testing.T) {
	kubeconfigPath := createTempKubeconfig(t)
	k8sManager, _ := k8s.NewK8SClientManager(kubeconfigPath)

	_, err := mcp.NewMCPHandler(nil, k8sManager, nil, nil)
	if err == nil {
		t.Error("期望创建失败，但成功了")
	}
}

// TestNewMCPHandler_NilK8SManager 测试创建处理器时 K8S 管理器为空
func TestNewMCPHandler_NilK8SManager(t *testing.T) {
	toolRegistry := tools.NewToolRegistry()

	_, err := mcp.NewMCPHandler(toolRegistry, nil, nil, nil)
	if err == nil {
		t.Error("期望创建失败，但成功了")
	}
}
