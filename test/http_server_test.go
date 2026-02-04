package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/config"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/mcp"
	"kubectl-mcp/internal/server"
	"kubectl-mcp/internal/tools"
)

// setupHTTPServer 创建测试用的 HTTP 服务器
func setupHTTPServer(t *testing.T, cfg *config.ServerConfig) (*server.HTTPServer, *k8s.K8SClientManager) {
	// 创建临时 kubeconfig
	kubeconfigPath := createTempKubeconfig(t)

	// 创建 K8S 客户端管理器
	k8sManager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}

	// 创建工具注册表
	toolRegistry := tools.NewToolRegistry()

	// 注册测试工具
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
			},
		},
	}
	if err := toolRegistry.RegisterTool(testTool); err != nil {
		t.Fatalf("注册测试工具失败: %v", err)
	}

	// 创建审计日志器
	auditLogger, _ := audit.NewAuditLogger(&audit.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})

	// 创建性能指标收集器
	metrics := audit.NewMetricsCollector()

	// 使用默认配置（如果未提供）
	if cfg == nil {
		cfg = &config.ServerConfig{
			Host:                  "localhost",
			Port:                  8080,
			LogLevel:              "info",
			MaxConcurrentRequests: 100,
			RequestTimeout:        30 * time.Second,
			AllowedOrigins:        []string{"*"},
			EnableCache:           true,
			CacheTTL:              5 * time.Minute,
		}
	}

	// 创建 HTTP 服务器
	httpServer, err := server.NewHTTPServer(&server.HTTPServerConfig{
		Config:       cfg,
		ToolRegistry: toolRegistry,
		K8SManager:   k8sManager,
		AuditLogger:  auditLogger,
		Version:      "1.0.0-test",
		Metrics:      metrics,
	})
	if err != nil {
		t.Fatalf("创建 HTTP 服务器失败: %v", err)
	}

	return httpServer, k8sManager
}

// ========== POST /tool 测试 ==========

// TestHTTPHandleToolCall_ValidRequest 测试有效的工具调用请求
func TestHTTPHandleToolCall_ValidRequest(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	// 创建请求
	reqBody := mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
		RequestID: "test-request-1",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	httpServer.GetRouter().ServeHTTP(w, req)

	// 验证响应
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}

	var response mcp.ToolCallResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !response.Success {
		t.Errorf("期望请求成功，但失败了: %v", response.Error)
	}

	if response.RequestID != "test-request-1" {
		t.Errorf("期望 RequestID 为 'test-request-1'，实际为 '%s'", response.RequestID)
	}
}

// TestHTTPHandleToolCall_InvalidJSON 测试无效的 JSON 请求
func TestHTTPHandleToolCall_InvalidJSON(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码为 400，实际为 %d", w.Code)
	}

	var response mcp.ToolCallResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Success {
		t.Error("期望请求失败")
	}

	if response.Error == nil {
		t.Fatal("期望错误信息不为空")
	}

	if response.Error.Type != mcp.ErrTypeClient {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", mcp.ErrTypeClient, response.Error.Type)
	}
}

// TestHTTPHandleToolCall_WithUserHeaders 测试从 Header 中获取用户信息
func TestHTTPHandleToolCall_WithUserHeaders(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	reqBody := mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "user123")
	req.Header.Set("X-User-Name", "testuser")
	req.Header.Set("X-User-Role", "admin")

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}

	var response mcp.ToolCallResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if !response.Success {
		t.Errorf("期望请求成功，但失败了: %v", response.Error)
	}
}

// ========== GET /tools 测试 ==========

// TestHTTPHandleListTools_AllTools 测试获取所有工具列表
func TestHTTPHandleListTools_AllTools(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/tools", nil)
	w := httptest.NewRecorder()

	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}

	var response mcp.ToolListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.TotalCount == 0 {
		t.Error("期望工具总数大于 0")
	}

	if len(response.Tools) == 0 {
		t.Error("期望工具列表不为空")
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
			break
		}
	}

	if !found {
		t.Error("期望找到测试工具 'test_tool'")
	}
}

// TestHTTPHandleListTools_ByCategory 测试按分类获取工具列表
func TestHTTPHandleListTools_ByCategory(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/tools?category=query", nil)
	w := httptest.NewRecorder()

	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}

	var response mcp.ToolListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证所有返回的工具都属于 query 分类
	for _, tool := range response.Tools {
		if tool.Category != "query" {
			t.Errorf("期望工具分类为 'query'，实际为 '%s'", tool.Category)
		}
	}
}

// ========== GET /health 测试 ==========

// TestHTTPHandleHealthCheck 测试健康检查
func TestHTTPHandleHealthCheck(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	// 等待一小段时间以确保 uptime > 0
	time.Sleep(100 * time.Millisecond)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}

	var response mcp.HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
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

// ========== GET /contexts 测试 ==========

// TestHTTPHandleListContexts 测试获取 context 列表
func TestHTTPHandleListContexts(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/contexts", nil)
	w := httptest.NewRecorder()

	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}

	var response mcp.ContextListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
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

// ========== API Token 认证测试 ==========

// TestHTTPAuthMiddleware_WithValidToken 测试有效的 API Token
func TestHTTPAuthMiddleware_WithValidToken(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:                  "localhost",
		Port:                  8080,
		LogLevel:              "info",
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		AllowedOrigins:        []string{"*"},
		APIToken:              "test-token-123",
		EnableCache:           true,
		CacheTTL:              5 * time.Minute,
	}

	httpServer, _ := setupHTTPServer(t, cfg)

	reqBody := mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token-123")

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}
}

// TestHTTPAuthMiddleware_WithInvalidToken 测试无效的 API Token
func TestHTTPAuthMiddleware_WithInvalidToken(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:                  "localhost",
		Port:                  8080,
		LogLevel:              "info",
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		AllowedOrigins:        []string{"*"},
		APIToken:              "test-token-123",
		EnableCache:           true,
		CacheTTL:              5 * time.Minute,
	}

	httpServer, _ := setupHTTPServer(t, cfg)

	reqBody := mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 401，实际为 %d", w.Code)
	}

	var response mcp.ToolCallResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Success {
		t.Error("期望请求失败")
	}

	if response.Error == nil {
		t.Fatal("期望错误信息不为空")
	}

	if response.Error.Type != mcp.ErrTypeAuth {
		t.Errorf("期望错误类型为 '%s'，实际为 '%s'", mcp.ErrTypeAuth, response.Error.Type)
	}
}

// TestHTTPAuthMiddleware_MissingToken 测试缺少 API Token
func TestHTTPAuthMiddleware_MissingToken(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:                  "localhost",
		Port:                  8080,
		LogLevel:              "info",
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		AllowedOrigins:        []string{"*"},
		APIToken:              "test-token-123",
		EnableCache:           true,
		CacheTTL:              5 * time.Minute,
	}

	httpServer, _ := setupHTTPServer(t, cfg)

	reqBody := mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// 不设置 Authorization Header

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 401，实际为 %d", w.Code)
	}
}

// TestHTTPAuthMiddleware_InvalidFormat 测试无效的认证格式
func TestHTTPAuthMiddleware_InvalidFormat(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:                  "localhost",
		Port:                  8080,
		LogLevel:              "info",
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		AllowedOrigins:        []string{"*"},
		APIToken:              "test-token-123",
		EnableCache:           true,
		CacheTTL:              5 * time.Minute,
	}

	httpServer, _ := setupHTTPServer(t, cfg)

	reqBody := mcp.ToolCallRequest{
		Tool: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/tool", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "test-token-123") // 缺少 Bearer 前缀

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("期望状态码为 401，实际为 %d", w.Code)
	}
}

// TestHTTPAuthMiddleware_HealthCheckNoAuth 测试健康检查端点不需要认证
func TestHTTPAuthMiddleware_HealthCheckNoAuth(t *testing.T) {
	cfg := &config.ServerConfig{
		Host:                  "localhost",
		Port:                  8080,
		LogLevel:              "info",
		MaxConcurrentRequests: 100,
		RequestTimeout:        30 * time.Second,
		AllowedOrigins:        []string{"*"},
		APIToken:              "test-token-123",
		EnableCache:           true,
		CacheTTL:              5 * time.Minute,
	}

	httpServer, _ := setupHTTPServer(t, cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// 不设置 Authorization Header

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	// 健康检查端点应该不需要认证
	if w.Code != http.StatusOK {
		t.Errorf("期望状态码为 200，实际为 %d", w.Code)
	}
}

// ========== HTTP 方法限制测试 ==========

// TestHTTPMethodLimitMiddleware_AllowedMethods 测试允许的 HTTP 方法
func TestHTTPMethodLimitMiddleware_AllowedMethods(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"GET /tools", http.MethodGet, "/tools", http.StatusOK},
		{"GET /health", http.MethodGet, "/health", http.StatusOK},
		{"GET /contexts", http.MethodGet, "/contexts", http.StatusOK},
		{"OPTIONS /tool", http.MethodOptions, "/tool", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			httpServer.GetRouter().ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Errorf("期望状态码为 %d，实际为 %d", tt.want, w.Code)
			}
		})
	}
}

// TestHTTPMethodLimitMiddleware_DisallowedMethods 测试不允许的 HTTP 方法
func TestHTTPMethodLimitMiddleware_DisallowedMethods(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"PUT /tool", http.MethodPut, "/tool"},
		{"DELETE /tool", http.MethodDelete, "/tool"},
		{"PATCH /tool", http.MethodPatch, "/tool"},
		{"HEAD /tool", http.MethodHead, "/tool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			httpServer.GetRouter().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("期望状态码为 405，实际为 %d", w.Code)
			}

			var response mcp.ToolCallResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("解析响应失败: %v", err)
			}

			if response.Success {
				t.Error("期望请求失败")
			}

			if response.Error == nil {
				t.Fatal("期望错误信息不为空")
			}

			if response.Error.Type != mcp.ErrTypeClient {
				t.Errorf("期望错误类型为 '%s'，实际为 '%s'", mcp.ErrTypeClient, response.Error.Type)
			}
		})
	}
}

// ========== CORS 测试 ==========

// TestHTTPCORSMiddleware_AllowedOrigin 测试允许的来源
func TestHTTPCORSMiddleware_AllowedOrigin(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	// 验证 CORS 头
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("期望设置 Access-Control-Allow-Origin 头")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("期望设置 Access-Control-Allow-Methods 头")
	}

	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("期望设置 Access-Control-Allow-Headers 头")
	}
}

// TestHTTPCORSMiddleware_PreflightRequest 测试预检请求
func TestHTTPCORSMiddleware_PreflightRequest(t *testing.T) {
	httpServer, _ := setupHTTPServer(t, nil)

	req := httptest.NewRequest(http.MethodOptions, "/tool", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")

	w := httptest.NewRecorder()
	httpServer.GetRouter().ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("期望状态码为 204，实际为 %d", w.Code)
	}

	// 验证 CORS 头
	if w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("期望设置 Access-Control-Allow-Origin 头")
	}

	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("期望设置 Access-Control-Allow-Methods 头")
	}
}
