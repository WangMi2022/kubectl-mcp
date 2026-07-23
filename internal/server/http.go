package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/config"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/mcp"
	"kubectl-mcp/internal/tools"
)

// HTTPServer HTTP 服务器结构
type HTTPServer struct {
	router       *gin.Engine
	config       *config.ServerConfig
	mcpHandler   *mcp.MCPHandler
	httpServer   *http.Server
	auditLogger  *audit.AuditLogger
	shutdownOnce sync.Once

	// 性能优化组件
	rateLimiter *CompositeRateLimiter   // 限流器
	queryCache  *QueryCache             // 查询缓存
	metrics     *audit.MetricsCollector // 性能指标收集器
}

// HTTPServerConfig HTTP 服务器配置
type HTTPServerConfig struct {
	Config       *config.ServerConfig
	ToolRegistry *tools.ToolRegistry
	K8SManager   *k8s.K8SClientManager
	AuditLogger  *audit.AuditLogger
	Version      string
	Metrics      *audit.MetricsCollector // 性能指标收集器
}

// cachedToolCallResult 只保存可跨请求复用的查询结果。
// requestId、context 和 duration 属于单次请求 envelope，不能进入缓存。
type cachedToolCallResult struct {
	Data       interface{}
	Content    []mcp.ContentItem
	Pagination *mcp.PaginationInfo
}

func newCachedToolCallResult(response *mcp.ToolCallResponse) cachedToolCallResult {
	result := cachedToolCallResult{
		Data:    response.Data,
		Content: append([]mcp.ContentItem(nil), response.Content...),
	}
	if response.Pagination != nil {
		pagination := *response.Pagination
		result.Pagination = &pagination
	}
	return result
}

func (r cachedToolCallResult) response(requestID, contextName string) *mcp.ToolCallResponse {
	response := &mcp.ToolCallResponse{
		Success:   true,
		Data:      r.Data,
		Content:   append([]mcp.ContentItem(nil), r.Content...),
		RequestID: requestID,
		Context:   contextName,
	}
	if r.Pagination != nil {
		pagination := *r.Pagination
		response.Pagination = &pagination
	}
	return response
}

// NewHTTPServer 创建新的 HTTP 服务器
// 参数:
//   - cfg: HTTP 服务器配置
//
// 返回:
//   - *HTTPServer: HTTP 服务器实例
//   - error: 错误信息
func NewHTTPServer(cfg *HTTPServerConfig) (*HTTPServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("服务器配置不能为空")
	}
	if cfg.Config == nil {
		return nil, fmt.Errorf("服务器配置不能为空")
	}
	if cfg.ToolRegistry == nil {
		return nil, fmt.Errorf("工具注册表不能为空")
	}
	if cfg.K8SManager == nil {
		return nil, fmt.Errorf("K8S 客户端管理器不能为空")
	}

	// 创建 MCP 处理器
	mcpHandler, err := mcp.NewMCPHandler(
		cfg.ToolRegistry,
		cfg.K8SManager,
		cfg.AuditLogger,
		&mcp.MCPHandlerConfig{
			Version: cfg.Version,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("创建 MCP 处理器失败: %w", err)
	}

	// 设置 Gin 模式
	if cfg.Config.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 路由
	router := gin.New()

	// 创建性能优化组件
	// 1. 创建限流器
	var rateLimiter *CompositeRateLimiter
	if cfg.Config.MaxConcurrentRequests > 0 {
		// 令牌桶限流器：每秒允许的请求数 = 最大并发数 * 2
		tokenBucket := NewTokenBucketLimiter(float64(cfg.Config.MaxConcurrentRequests*2), cfg.Config.MaxConcurrentRequests*4)
		// 并发限制器
		concurrencyLimiter := NewConcurrencyLimiter(cfg.Config.MaxConcurrentRequests)
		rateLimiter = NewCompositeRateLimiter(tokenBucket, concurrencyLimiter)
	}

	// 2. 创建查询缓存
	queryCache := NewQueryCache(&CacheConfig{
		Enabled:    cfg.Config.EnableCache,
		DefaultTTL: cfg.Config.CacheTTL,
		MaxSize:    1000,
	})

	// 3. 使用传入的指标收集器或创建新的
	metrics := cfg.Metrics
	if metrics == nil {
		metrics = audit.NewMetricsCollector()
	}

	server := &HTTPServer{
		router:      router,
		config:      cfg.Config,
		mcpHandler:  mcpHandler,
		auditLogger: cfg.AuditLogger,
		rateLimiter: rateLimiter,
		queryCache:  queryCache,
		metrics:     metrics,
	}

	// 配置中间件和路由
	server.setupMiddleware()
	server.setupRoutes()

	return server, nil
}

// setupMiddleware 配置中间件
func (s *HTTPServer) setupMiddleware() {
	// 恢复中间件（处理 panic）
	s.router.Use(gin.Recovery())

	// 日志中间件
	s.router.Use(s.loggingMiddleware())

	// CORS 中间件
	s.router.Use(s.corsMiddleware())

	// HTTP 方法限制中间件
	s.router.Use(s.methodLimitMiddleware())

	// 限流中间件
	if s.rateLimiter != nil {
		s.router.Use(s.rateLimitMiddleware())
	}

	// API Token 认证中间件（如果配置了 Token）
	if s.config.APIToken != "" {
		s.router.Use(s.authMiddleware())
	}

	// 请求超时中间件
	s.router.Use(s.timeoutMiddleware())

	// 指标收集中间件
	if s.metrics != nil {
		s.router.Use(s.metricsMiddleware())
	}
}

// setupRoutes 配置路由
func (s *HTTPServer) setupRoutes() {
	// POST /tool - 执行工具调用
	s.router.POST("/tool", s.handleToolCall)

	// GET /tools - 获取工具列表
	s.router.GET("/tools", s.handleListTools)

	// GET /health - 健康检查
	s.router.GET("/health", s.handleHealthCheck)

	// GET /contexts - 获取 context 列表
	s.router.GET("/contexts", s.handleListContexts)

	// GET /metrics - Prometheus 格式的性能指标
	if s.metrics != nil {
		s.router.GET("/metrics", gin.WrapF(s.metrics.PrometheusHandler()))
	}

	// GET /cache/stats - 缓存统计
	if s.queryCache != nil {
		s.router.GET("/cache/stats", s.handleCacheStats)
	}
}

// ========== HTTP 处理器 ==========

// handleToolCall 处理工具调用请求
// POST /tool
func (s *HTTPServer) handleToolCall(c *gin.Context) {
	var req mcp.ToolCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, mcp.NewErrorResponse(
			mcp.NewErrorInfo(mcp.ErrTypeClient, mcp.ErrCodeInvalidRequest, "请求格式无效").
				WithDetails(err.Error()).
				WithSuggestion("请检查请求 JSON 格式是否正确"),
		))
		return
	}

	// 从 Header 中获取用户信息（如果存在）
	if req.User == nil {
		userID := c.GetHeader("X-User-ID")
		userName := c.GetHeader("X-User-Name")
		userRole := c.GetHeader("X-User-Role")
		if userID != "" || userName != "" {
			req.User = &mcp.UserInfo{
				ID:   userID,
				Name: userName,
				Role: userRole,
			}
		}
	}

	// 从 Header 中获取请求 ID（如果存在）
	if req.RequestID == "" {
		req.RequestID = c.GetHeader("X-Request-ID")
	}

	// 获取工具名称（兼容 tool 和 name 字段）
	toolName := req.GetToolName()

	// 获取上下文（带超时）
	ctx := c.Request.Context()

	// 检查缓存（仅对可缓存的查询工具）
	cacheKey := ""
	if s.queryCache != nil && IsCacheable(toolName) {
		cacheKey = s.toolCallCacheKey(&req)
		if cachedData, found := s.queryCache.Get(cacheKey); found {
			if cachedResult, ok := cachedData.(cachedToolCallResult); ok {
				// 命中时使用当前请求的 correlation envelope，绝不回放旧 requestId。
				if s.metrics != nil {
					s.metrics.RecordCacheHit()
				}
				c.JSON(http.StatusOK, cachedResult.response(req.RequestID, s.toolCallResponseContext(&req)))
				return
			}
			// 未知缓存类型 fail closed 为 miss，避免返回无法重新绑定的旧 envelope。
			s.queryCache.Delete(cacheKey)
		}
		// 缓存未命中
		if s.metrics != nil {
			s.metrics.RecordCacheMiss()
		}
	}

	// 执行工具调用
	startTime := time.Now()
	response := s.mcpHandler.HandleToolCall(ctx, &req)
	duration := time.Since(startTime)

	// 记录工具调用指标
	if s.metrics != nil {
		s.metrics.RecordToolCall(toolName, response.Success, duration)
	}

	// 缓存成功的查询结果
	if s.queryCache != nil && response.Success && IsCacheable(toolName) {
		if cacheKey == "" {
			cacheKey = s.toolCallCacheKey(&req)
		}
		s.queryCache.Set(cacheKey, newCachedToolCallResult(response))
	}

	// 如果是修改操作，失效相关缓存
	if s.queryCache != nil && response.Success {
		invalidatedTools := GetInvalidatedTools(toolName)
		if len(invalidatedTools) == 0 && (toolName == "apply_yaml" || toolName == "patch_resource") {
			// apply_yaml 和 patch_resource 清除所有缓存
			s.queryCache.Clear()
		} else {
			for _, tool := range invalidatedTools {
				s.queryCache.InvalidateByTool(tool)
			}
		}
	}

	// 根据结果返回相应的 HTTP 状态码
	statusCode := http.StatusOK
	if !response.Success && response.Error != nil {
		statusCode = s.getHTTPStatusFromError(response.Error)
	}

	c.JSON(statusCode, response)
}

// handleListTools 处理获取工具列表请求
// GET /tools
func (s *HTTPServer) handleListTools(c *gin.Context) {
	// 获取可选的分类过滤参数
	category := c.Query("category")

	response := s.mcpHandler.GetToolList(category)
	c.JSON(http.StatusOK, response)
}

// handleHealthCheck 处理健康检查请求
// GET /health
func (s *HTTPServer) handleHealthCheck(c *gin.Context) {
	response := s.mcpHandler.GetHealth()
	c.JSON(http.StatusOK, response)
}

// handleListContexts 处理获取 context 列表请求
// GET /contexts
func (s *HTTPServer) handleListContexts(c *gin.Context) {
	response := s.mcpHandler.GetContextList()
	c.JSON(http.StatusOK, response)
}

// ========== 中间件 ==========

// loggingMiddleware 日志中间件
func (s *HTTPServer) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		requestID := c.GetHeader("X-Request-ID")

		// 处理请求
		c.Next()

		// 记录详细的请求日志
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		if s.auditLogger != nil {
			_ = s.auditLogger.LogOperation(&audit.OperationLog{
				Timestamp: startTime,
				Tool:      fmt.Sprintf("%s %s", method, path),
				Context:   requestID,
				Success:   statusCode < 400,
				Duration:  duration,
				Arguments: map[string]interface{}{
					"client_ip":   clientIP,
					"user_agent":  userAgent,
					"status_code": statusCode,
					"query":       c.Request.URL.RawQuery,
				},
			})
		}
	}
}

// corsMiddleware CORS 中间件
func (s *HTTPServer) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许该来源
		allowed := false
		for _, allowedOrigin := range s.config.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if allowed {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-User-ID, X-User-Name, X-User-Role, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")
		c.Header("Access-Control-Allow-Credentials", "true")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// methodLimitMiddleware HTTP 方法限制中间件
// 仅允许 GET、POST 和 OPTIONS 方法
func (s *HTTPServer) methodLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// 允许的方法：GET、POST、OPTIONS
		if method != http.MethodGet && method != http.MethodPost && method != http.MethodOptions {
			c.JSON(http.StatusMethodNotAllowed, mcp.NewErrorResponse(
				mcp.NewErrorInfo(mcp.ErrTypeClient, "METHOD_NOT_ALLOWED",
					fmt.Sprintf("不支持的 HTTP 方法: %s", method)).
					WithSuggestion("仅支持 GET 和 POST 方法"),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// authMiddleware API Token 认证中间件
func (s *HTTPServer) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 健康检查端点不需要认证
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// OPTIONS 请求不需要认证（CORS 预检）
		if c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// 获取 Authorization Header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, mcp.NewErrorResponse(
				mcp.NewErrorInfo(mcp.ErrTypeAuth, mcp.ErrCodeUnauthorized, "缺少认证信息").
					WithSuggestion("请在 Authorization Header 中提供 Bearer Token"),
			))
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, mcp.NewErrorResponse(
				mcp.NewErrorInfo(mcp.ErrTypeAuth, mcp.ErrCodeUnauthorized, "认证格式无效").
					WithSuggestion("请使用 Bearer Token 格式: Authorization: Bearer <token>"),
			))
			c.Abort()
			return
		}

		token := parts[1]

		// 验证 Token
		if token != s.config.APIToken {
			c.JSON(http.StatusUnauthorized, mcp.NewErrorResponse(
				mcp.NewErrorInfo(mcp.ErrTypeAuth, mcp.ErrCodeAuthFailed, "认证失败").
					WithSuggestion("请检查 API Token 是否正确"),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// timeoutMiddleware 请求超时中间件
func (s *HTTPServer) timeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), s.config.RequestTimeout)
		defer cancel()

		// 替换请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 使用 channel 来处理超时
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			// 请求正常完成
		case <-ctx.Done():
			// 请求超时
			if ctx.Err() == context.DeadlineExceeded {
				c.JSON(http.StatusGatewayTimeout, mcp.NewErrorResponse(
					mcp.NewErrorInfo(mcp.ErrTypeTimeout, mcp.ErrCodeRequestTimeout, "请求超时").
						WithSuggestion(fmt.Sprintf("请求处理时间超过 %v，请稍后重试或检查集群状态", s.config.RequestTimeout)),
				))
				c.Abort()
			}
		}
	}
}

// rateLimitMiddleware 限流中间件
func (s *HTTPServer) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 健康检查和指标端点不限流
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/metrics" {
			c.Next()
			return
		}

		// 尝试获取许可
		ctx := c.Request.Context()
		if err := s.rateLimiter.Acquire(ctx); err != nil {
			if s.metrics != nil {
				s.metrics.RecordError("RATE_LIMIT_EXCEEDED")
			}
			c.JSON(http.StatusTooManyRequests, mcp.NewErrorResponse(
				mcp.NewErrorInfo(mcp.ErrTypeClient, "RATE_LIMIT_EXCEEDED", "请求过于频繁").
					WithSuggestion("请稍后重试，或联系管理员调整限流配置"),
			))
			c.Abort()
			return
		}

		// 请求完成后释放许可
		defer s.rateLimiter.Release()

		c.Next()
	}
}

// metricsMiddleware 指标收集中间件
func (s *HTTPServer) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// 增加并发计数
		s.metrics.IncrementConcurrent()
		defer s.metrics.DecrementConcurrent()

		// 处理请求
		c.Next()

		// 记录请求指标
		duration := time.Since(startTime)
		success := c.Writer.Status() < 400
		s.metrics.RecordRequest(success, duration)

		// 记录错误类型
		if !success {
			statusCode := c.Writer.Status()
			var errType string
			switch {
			case statusCode >= 500:
				errType = "SERVER_ERROR"
			case statusCode == 429:
				errType = "RATE_LIMIT"
			case statusCode == 408 || statusCode == 504:
				errType = "TIMEOUT"
			case statusCode == 401 || statusCode == 403:
				errType = "AUTH_ERROR"
			case statusCode == 404:
				errType = "NOT_FOUND"
			default:
				errType = "CLIENT_ERROR"
			}
			s.metrics.RecordError(errType)
		}
	}
}

// handleCacheStats 处理缓存统计请求
// GET /cache/stats
func (s *HTTPServer) handleCacheStats(c *gin.Context) {
	stats := s.queryCache.Stats()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// ========== 辅助方法 ==========

// getHTTPStatusFromError 根据错误类型获取 HTTP 状态码
func (s *HTTPServer) getHTTPStatusFromError(errInfo *mcp.ErrorInfo) int {
	if errInfo == nil {
		return http.StatusInternalServerError
	}

	switch errInfo.Type {
	case mcp.ErrTypeClient:
		return http.StatusBadRequest
	case mcp.ErrTypeAuth:
		if errInfo.Code == mcp.ErrCodeUnauthorized {
			return http.StatusUnauthorized
		}
		return http.StatusForbidden
	case mcp.ErrTypeNotFound:
		return http.StatusNotFound
	case mcp.ErrTypeConflict:
		return http.StatusConflict
	case mcp.ErrTypeTimeout:
		return http.StatusGatewayTimeout
	case mcp.ErrTypeNetwork:
		return http.StatusBadGateway
	case mcp.ErrTypeServer:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func (s *HTTPServer) toolCallCacheKey(req *mcp.ToolCallRequest) string {
	args := make(map[string]interface{}, len(req.Arguments)+3)
	for name, value := range req.Arguments {
		args[name] = value
	}
	if req.Verbosity != "" {
		args["_verbosity"] = string(req.Verbosity)
	}
	if req.Pagination != nil {
		normalized := mcp.NormalizePagination(req.Pagination)
		args["_page"] = normalized.Page
		args["_pageSize"] = normalized.PageSize
	}
	return s.queryCache.GenerateKey(req.GetToolName(), args, s.toolCallResponseContext(req))
}

func (s *HTTPServer) toolCallResponseContext(req *mcp.ToolCallRequest) string {
	if req.Context != "" {
		return req.Context
	}
	return s.mcpHandler.GetK8SManager().GetCurrentContext()
}

// ========== 服务器生命周期 ==========

// Start 启动 HTTP 服务器
// 返回:
//   - error: 错误信息
func (s *HTTPServer) Start() error {
	addr := s.config.GetListenAddress()

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.RequestTimeout,
		WriteTimeout: s.config.RequestTimeout + 5*time.Second, // 写超时稍长一些
		IdleTimeout:  120 * time.Second,
	}

	// 启动服务器（阻塞）
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP 服务器启动失败: %w", err)
	}

	return nil
}

// StartAsync 异步启动 HTTP 服务器
// 返回:
//   - error: 启动错误（如果有）
func (s *HTTPServer) StartAsync() error {
	addr := s.config.GetListenAddress()

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.RequestTimeout,
		WriteTimeout: s.config.RequestTimeout + 5*time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if s.auditLogger != nil {
				_ = s.auditLogger.LogError(err, "HTTP 服务器异常退出")
			}
		}
	}()

	return nil
}

// Shutdown 优雅关闭服务器
// 参数:
//   - ctx: 上下文（用于控制关闭超时）
//
// 返回:
//   - error: 错误信息
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	var shutdownErr error

	s.shutdownOnce.Do(func() {
		// 关闭限流器
		if s.rateLimiter != nil {
			s.rateLimiter.Close()
		}

		// 关闭 HTTP 服务器
		if s.httpServer != nil {
			shutdownErr = s.httpServer.Shutdown(ctx)
		}
	})

	return shutdownErr
}

// GetRouter 获取 Gin 路由器（用于测试）
func (s *HTTPServer) GetRouter() *gin.Engine {
	return s.router
}

// GetConfig 获取服务器配置
func (s *HTTPServer) GetConfig() *config.ServerConfig {
	return s.config
}

// GetMCPHandler 获取 MCP 处理器
func (s *HTTPServer) GetMCPHandler() *mcp.MCPHandler {
	return s.mcpHandler
}

// GetMetrics 获取性能指标收集器
func (s *HTTPServer) GetMetrics() *audit.MetricsCollector {
	return s.metrics
}

// GetCache 获取查询缓存
func (s *HTTPServer) GetCache() *QueryCache {
	return s.queryCache
}

// GetRateLimiter 获取限流器
func (s *HTTPServer) GetRateLimiter() *CompositeRateLimiter {
	return s.rateLimiter
}
