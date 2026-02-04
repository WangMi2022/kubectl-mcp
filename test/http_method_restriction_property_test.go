package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/config"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/server"
	"kubectl-mcp/internal/tools"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// TestProperty_HTTPMethodRestriction 测试 HTTP 方法限制属性
// Property 5: HTTP 方法限制
// Validates: Requirements 13.3
// Feature: kubectl-mcp-server, Property 5: 对于任何来自 admin-backend 的请求，系统仅接受 POST 和 GET 方法，其他 HTTP 方法必须返回 405 Method Not Allowed
func TestProperty_HTTPMethodRestriction(t *testing.T) {
	// 创建测试服务器
	httpServer := createTestHTTPServer(t)

	// 配置属性测试参数
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100 // 至少运行 100 次迭代
	parameters.MaxSize = 10

	properties := gopter.NewProperties(parameters)

	// 属性 1: 不支持的 HTTP 方法返回 405
	// 对于任何不是 GET、POST、OPTIONS 的 HTTP 方法，服务器应该返回 405 Method Not Allowed
	properties.Property("不支持的 HTTP 方法返回 405", prop.ForAll(
		func(method string, endpoint string) bool {
			// 跳过允许的方法
			if method == http.MethodGet || method == http.MethodPost || method == http.MethodOptions {
				return true
			}

			// 创建请求
			var req *http.Request
			var err error

			if endpoint == "/tool" {
				// POST 端点需要 JSON body
				body := map[string]interface{}{
					"tool":      "get_pods",
					"arguments": map[string]interface{}{},
				}
				bodyBytes, _ := json.Marshal(body)
				req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(bodyBytes))
			} else {
				req, err = http.NewRequest(method, endpoint, nil)
			}

			if err != nil {
				t.Logf("创建请求失败: %v", err)
				return false
			}

			req.Header.Set("Content-Type", "application/json")

			// 发送请求
			w := httptest.NewRecorder()
			httpServer.GetRouter().ServeHTTP(w, req)

			// 验证返回 405 Method Not Allowed
			if w.Code != http.StatusMethodNotAllowed {
				t.Logf("方法 %s 访问 %s 应该返回 405，实际返回 %d", method, endpoint, w.Code)
				return false
			}

			// 验证响应包含错误信息
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Logf("解析响应失败: %v", err)
				return false
			}

			// 验证 success 字段为 false
			if success, ok := response["success"].(bool); !ok || success {
				t.Logf("响应的 success 字段应该为 false")
				return false
			}

			// 验证包含 error 字段
			if _, ok := response["error"]; !ok {
				t.Logf("响应应该包含 error 字段")
				return false
			}

			return true
		},
		genHTTPMethod(),
		genEndpoint(),
	))

	// 属性 2: GET 方法被接受
	// 对于所有 GET 端点，GET 方法应该被接受（不返回 405）
	properties.Property("GET 方法被接受", prop.ForAll(
		func(endpoint string) bool {
			// 只测试 GET 端点
			if endpoint == "/tool" {
				return true // /tool 只支持 POST
			}

			req, err := http.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				t.Logf("创建请求失败: %v", err)
				return false
			}

			w := httptest.NewRecorder()
			httpServer.GetRouter().ServeHTTP(w, req)

			// 验证不返回 405
			if w.Code == http.StatusMethodNotAllowed {
				t.Logf("GET 方法访问 %s 不应该返回 405", endpoint)
				return false
			}

			return true
		},
		genGETEndpoint(),
	))

	// 属性 3: POST 方法被接受
	// 对于 /tool 端点，POST 方法应该被接受（不返回 405）
	properties.Property("POST 方法被接受", prop.ForAll(
		func(toolName string) bool {
			body := map[string]interface{}{
				"tool":      toolName,
				"arguments": map[string]interface{}{},
			}
			bodyBytes, _ := json.Marshal(body)

			req, err := http.NewRequest(http.MethodPost, "/tool", bytes.NewBuffer(bodyBytes))
			if err != nil {
				t.Logf("创建请求失败: %v", err)
				return false
			}

			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			httpServer.GetRouter().ServeHTTP(w, req)

			// 验证不返回 405
			// 注意：可能返回其他错误（如工具不存在），但不应该是 405
			if w.Code == http.StatusMethodNotAllowed {
				t.Logf("POST 方法访问 /tool 不应该返回 405")
				return false
			}

			return true
		},
		genToolNameForHTTP(),
	))

	// 属性 4: OPTIONS 方法被接受（CORS 预检）
	// 对于所有端点，OPTIONS 方法应该被接受（用于 CORS 预检）
	properties.Property("OPTIONS 方法被接受", prop.ForAll(
		func(endpoint string) bool {
			req, err := http.NewRequest(http.MethodOptions, endpoint, nil)
			if err != nil {
				t.Logf("创建请求失败: %v", err)
				return false
			}

			req.Header.Set("Origin", "http://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")

			w := httptest.NewRecorder()
			httpServer.GetRouter().ServeHTTP(w, req)

			// 验证不返回 405
			// OPTIONS 请求应该返回 204 No Content
			if w.Code == http.StatusMethodNotAllowed {
				t.Logf("OPTIONS 方法访问 %s 不应该返回 405", endpoint)
				return false
			}

			return true
		},
		genEndpoint(),
	))

	// 属性 5: 错误响应格式一致性
	// 对于所有不支持的方法，错误响应格式应该一致
	properties.Property("不支持方法的错误响应格式一致", prop.ForAll(
		func(method string, endpoint string) bool {
			// 跳过允许的方法
			if method == http.MethodGet || method == http.MethodPost || method == http.MethodOptions {
				return true
			}

			req, err := http.NewRequest(method, endpoint, nil)
			if err != nil {
				return false
			}

			w := httptest.NewRecorder()
			httpServer.GetRouter().ServeHTTP(w, req)

			// 验证返回 405
			if w.Code != http.StatusMethodNotAllowed {
				return false
			}

			// 解析响应
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				return false
			}

			// 验证响应结构
			if _, ok := response["success"]; !ok {
				return false
			}
			if _, ok := response["error"]; !ok {
				return false
			}

			// 验证 error 字段包含必要信息
			errorInfo, ok := response["error"].(map[string]interface{})
			if !ok {
				return false
			}

			// 验证包含 type、code、message 字段
			if _, ok := errorInfo["type"]; !ok {
				return false
			}
			if _, ok := errorInfo["code"]; !ok {
				return false
			}
			if _, ok := errorInfo["message"]; !ok {
				return false
			}

			return true
		},
		genHTTPMethod(),
		genEndpoint(),
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// 辅助函数：生成 HTTP 方法
func genHTTPMethod() gopter.Gen {
	// 生成各种 HTTP 方法，包括标准方法和非标准方法
	methods := []interface{}{
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodConnect,
		http.MethodTrace,
		"CUSTOM",
		"INVALID",
		"UNKNOWN",
	}

	return gen.OneConstOf(methods...).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：生成端点
func genEndpoint() gopter.Gen {
	endpoints := []interface{}{
		"/tool",
		"/tools",
		"/health",
		"/contexts",
		"/metrics",
		"/cache/stats",
	}

	return gen.OneConstOf(endpoints...).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：生成 GET 端点
func genGETEndpoint() gopter.Gen {
	endpoints := []interface{}{
		"/tools",
		"/health",
		"/contexts",
		"/metrics",
		"/cache/stats",
	}

	return gen.OneConstOf(endpoints...).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：生成工具名称
func genToolNameForHTTP() gopter.Gen {
	toolNames := []interface{}{
		"get_pods",
		"get_nodes",
		"get_namespaces",
		"get_deployments",
		"create_pod",
		"delete_pod",
		"invalid_tool",
	}

	return gen.OneConstOf(toolNames...).WithShrinker(gopter.NoShrinker)
}

// 辅助函数：创建测试 HTTP 服务器
func createTestHTTPServer(t *testing.T) *server.HTTPServer {
	// 创建临时 kubeconfig
	kubeconfigPath := createTestKubeconfigForHTTPMethod(t)

	// 创建配置
	cfg := &config.ServerConfig{
		Host:                  "localhost",
		Port:                  8080,
		KubeconfigPath:        kubeconfigPath,
		LogLevel:              "info",
		LogFormat:             "json",
		MaxConcurrentRequests: 100,
		RequestTimeout:        30000000000, // 30 秒
		AllowedOrigins:        []string{"*"},
		EnableCache:           true,
		CacheTTL:              300000000000, // 5 分钟
	}

	// 创建 K8S 客户端管理器
	k8sManager, err := k8s.NewK8SClientManager(kubeconfigPath)
	if err != nil {
		t.Fatalf("创建 K8S 客户端管理器失败: %v", err)
	}

	// 创建工具注册表
	registry := tools.NewToolRegistry()
	tools.RegisterQueryTools(registry)
	tools.RegisterCreateTools(registry)
	tools.RegisterUpdateTools(registry)
	tools.RegisterDeleteTools(registry)

	// 创建审计日志器
	auditLogger, err := audit.NewAuditLogger(&audit.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("创建审计日志器失败: %v", err)
	}

	// 创建指标收集器
	metrics := audit.NewMetricsCollector()

	// 创建 HTTP 服务器
	httpServer, err := server.NewHTTPServer(&server.HTTPServerConfig{
		Config:       cfg,
		ToolRegistry: registry,
		K8SManager:   k8sManager,
		AuditLogger:  auditLogger,
		Version:      "test",
		Metrics:      metrics,
	})
	if err != nil {
		t.Fatalf("创建 HTTP 服务器失败: %v", err)
	}

	return httpServer
}

// 辅助函数：创建测试用的 kubeconfig 文件
func createTestKubeconfigForHTTPMethod(t *testing.T) string {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	tmpDir, err := os.MkdirTemp(cwd, "kubectl-mcp-http-method-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	kubeconfigPath := filepath.Join(tmpDir, "config")

	kubeconfigContent := `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://test-cluster:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
    namespace: default
  name: test-context
current-context: test-context
users:
- name: test-user
  user:
    token: test-token
`

	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("创建临时 kubeconfig 失败: %v", err)
	}

	return kubeconfigPath
}

// TestProperty_HTTPMethodRestriction_EdgeCases 测试 HTTP 方法限制的边界情况
func TestProperty_HTTPMethodRestriction_EdgeCases(t *testing.T) {
	httpServer := createTestHTTPServer(t)

	t.Run("PUT 方法返回 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/tool", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("PUT 方法应该返回 405，实际返回 %d", w.Code)
		}
	})

	t.Run("DELETE 方法返回 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, "/tools", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("DELETE 方法应该返回 405，实际返回 %d", w.Code)
		}
	})

	t.Run("PATCH 方法返回 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPatch, "/health", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("PATCH 方法应该返回 405，实际返回 %d", w.Code)
		}
	})

	t.Run("HEAD 方法返回 405", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodHead, "/contexts", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("HEAD 方法应该返回 405，实际返回 %d", w.Code)
		}
	})

	t.Run("GET 方法访问 /tools 成功", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/tools", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code == http.StatusMethodNotAllowed {
			t.Error("GET 方法访问 /tools 不应该返回 405")
		}
	})

	t.Run("GET 方法访问 /health 成功", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code == http.StatusMethodNotAllowed {
			t.Error("GET 方法访问 /health 不应该返回 405")
		}
	})

	t.Run("POST 方法访问 /tool 成功", func(t *testing.T) {
		body := map[string]interface{}{
			"tool":      "get_pods",
			"arguments": map[string]interface{}{},
		}
		bodyBytes, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/tool", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		if w.Code == http.StatusMethodNotAllowed {
			t.Error("POST 方法访问 /tool 不应该返回 405")
		}
	})

	t.Run("OPTIONS 方法用于 CORS 预检", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodOptions, "/tool", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")

		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		// OPTIONS 请求应该返回 204 No Content
		if w.Code == http.StatusMethodNotAllowed {
			t.Error("OPTIONS 方法不应该返回 405")
		}

		// 验证 CORS 头
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("OPTIONS 响应应该包含 Access-Control-Allow-Methods 头")
		}
	})

	t.Run("不支持的方法错误响应包含建议", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPut, "/tool", nil)
		w := httptest.NewRecorder()
		httpServer.GetRouter().ServeHTTP(w, req)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		errorInfo, ok := response["error"].(map[string]interface{})
		if !ok {
			t.Fatal("响应应该包含 error 字段")
		}

		// 验证包含建议
		if _, ok := errorInfo["suggestion"]; !ok {
			t.Error("错误响应应该包含 suggestion 字段")
		}
	})

	t.Run("所有端点都应用方法限制", func(t *testing.T) {
		endpoints := []string{"/tool", "/tools", "/health", "/contexts", "/metrics"}

		for _, endpoint := range endpoints {
			req, _ := http.NewRequest(http.MethodPut, endpoint, nil)
			w := httptest.NewRecorder()
			httpServer.GetRouter().ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("端点 %s 应该拒绝 PUT 方法，实际返回 %d", endpoint, w.Code)
			}
		}
	})

	t.Run("并发请求方法限制正常工作", func(t *testing.T) {
		results := make(chan int, 10)

		// 并发发送不支持的方法请求
		for i := 0; i < 10; i++ {
			go func() {
				req, _ := http.NewRequest(http.MethodPut, "/tool", nil)
				w := httptest.NewRecorder()
				httpServer.GetRouter().ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		// 收集结果
		for i := 0; i < 10; i++ {
			code := <-results
			if code != http.StatusMethodNotAllowed {
				t.Errorf("并发请求应该返回 405，实际返回 %d", code)
			}
		}
	})
}
