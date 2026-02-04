package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"kubectl-mcp/internal/config"

	"github.com/spf13/viper"
)

// setupTest 为每个测试设置干净的环境
func setupTest(t *testing.T) func() {
	// 保存原始环境变量
	originalEnv := make(map[string]string)
	envVars := []string{
		"KUBECTL_MCP_HOST",
		"KUBECTL_MCP_PORT",
		"KUBECTL_MCP_KUBECONFIGPATH",
		"KUBECTL_MCP_DEFAULTCONTEXT",
		"KUBECTL_MCP_LOGLEVEL",
		"KUBECTL_MCP_LOGFORMAT",
		"KUBECTL_MCP_LOGFILE",
		"KUBECTL_MCP_MAXCONCURRENTREQUESTS",
		"KUBECTL_MCP_REQUESTTIMEOUT",
		"KUBECTL_MCP_APITOKEN",
		"KUBECTL_MCP_ENABLECACHE",
		"KUBECTL_MCP_CACHETTL",
	}

	for _, env := range envVars {
		originalEnv[env] = os.Getenv(env)
		os.Unsetenv(env)
	}

	// 重置 viper
	viper.Reset()

	// 返回清理函数
	return func() {
		// 恢复环境变量
		for env, val := range originalEnv {
			if val != "" {
				os.Setenv(env, val)
			} else {
				os.Unsetenv(env)
			}
		}
		viper.Reset()
	}
}

// createTempKubeconfig 创建临时的 kubeconfig 文件用于测试
func createTempKubeconfig(t *testing.T) string {
	// 在当前工作目录下创建临时目录，避免 Windows 权限问题
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	tmpDir, err := os.MkdirTemp(cwd, "kubectl-mcp-test-*")
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

// TestLoadConfig_Defaults 测试默认值设置
func TestLoadConfig_Defaults(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 创建临时 kubeconfig
	kubeconfigPath := createTempKubeconfig(t)

	// 设置必需的环境变量
	os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证默认值
	if cfg.Host != "0.0.0.0" {
		t.Errorf("期望 Host 为 '0.0.0.0'，实际为 '%s'", cfg.Host)
	}

	if cfg.Port != 8080 {
		t.Errorf("期望 Port 为 8080，实际为 %d", cfg.Port)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("期望 LogLevel 为 'info'，实际为 '%s'", cfg.LogLevel)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("期望 LogFormat 为 'json'，实际为 '%s'", cfg.LogFormat)
	}

	if cfg.MaxConcurrentRequests != 100 {
		t.Errorf("期望 MaxConcurrentRequests 为 100，实际为 %d", cfg.MaxConcurrentRequests)
	}

	if cfg.RequestTimeout != 30*time.Second {
		t.Errorf("期望 RequestTimeout 为 30s，实际为 %v", cfg.RequestTimeout)
	}

	if !cfg.EnableCache {
		t.Error("期望 EnableCache 为 true")
	}

	if cfg.CacheTTL != 5*time.Minute {
		t.Errorf("期望 CacheTTL 为 5m，实际为 %v", cfg.CacheTTL)
	}
}

// TestLoadConfig_FromEnv 测试从环境变量加载配置
func TestLoadConfig_FromEnv(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 创建临时 kubeconfig
	kubeconfigPath := createTempKubeconfig(t)

	// 设置环境变量
	os.Setenv("KUBECTL_MCP_HOST", "127.0.0.1")
	os.Setenv("KUBECTL_MCP_PORT", "9090")
	os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)
	os.Setenv("KUBECTL_MCP_DEFAULTCONTEXT", "test-context")
	os.Setenv("KUBECTL_MCP_LOGLEVEL", "debug")
	os.Setenv("KUBECTL_MCP_LOGFORMAT", "text")
	os.Setenv("KUBECTL_MCP_MAXCONCURRENTREQUESTS", "200")
	os.Setenv("KUBECTL_MCP_REQUESTTIMEOUT", "60s")
	os.Setenv("KUBECTL_MCP_APITOKEN", "test-token")
	os.Setenv("KUBECTL_MCP_ENABLECACHE", "false")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证环境变量配置
	if cfg.Host != "127.0.0.1" {
		t.Errorf("期望 Host 为 '127.0.0.1'，实际为 '%s'", cfg.Host)
	}

	if cfg.Port != 9090 {
		t.Errorf("期望 Port 为 9090，实际为 %d", cfg.Port)
	}

	if cfg.DefaultContext != "test-context" {
		t.Errorf("期望 DefaultContext 为 'test-context'，实际为 '%s'", cfg.DefaultContext)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("期望 LogLevel 为 'debug'，实际为 '%s'", cfg.LogLevel)
	}

	if cfg.LogFormat != "text" {
		t.Errorf("期望 LogFormat 为 'text'，实际为 '%s'", cfg.LogFormat)
	}

	if cfg.MaxConcurrentRequests != 200 {
		t.Errorf("期望 MaxConcurrentRequests 为 200，实际为 %d", cfg.MaxConcurrentRequests)
	}

	if cfg.RequestTimeout != 60*time.Second {
		t.Errorf("期望 RequestTimeout 为 60s，实际为 %v", cfg.RequestTimeout)
	}

	if cfg.APIToken != "test-token" {
		t.Errorf("期望 APIToken 为 'test-token'，实际为 '%s'", cfg.APIToken)
	}

	if cfg.EnableCache {
		t.Error("期望 EnableCache 为 false")
	}
}

// TestLoadConfig_FromConfigFile 测试从配置文件加载配置
func TestLoadConfig_FromConfigFile(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 创建临时目录和配置文件
	tmpDir := t.TempDir()
	kubeconfigPath := createTempKubeconfig(t)

	// 使用正斜杠，YAML 中更安全，并转义反斜杠
	kubeconfigPathForYAML := filepath.ToSlash(kubeconfigPath)

	configContent := `host: "192.168.1.1"
port: 7070
kubeconfigPath: ` + kubeconfigPathForYAML + `
defaultContext: "prod-context"
logLevel: "warn"
logFormat: "json"
maxConcurrentRequests: 150
requestTimeout: "45s"
apiToken: "file-token"
enableCache: true
cacheTTL: "10m"
`

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}

	// 切换到临时目录
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置文件配置
	if cfg.Host != "192.168.1.1" {
		t.Errorf("期望 Host 为 '192.168.1.1'，实际为 '%s'", cfg.Host)
	}

	if cfg.Port != 7070 {
		t.Errorf("期望 Port 为 7070，实际为 %d", cfg.Port)
	}

	if cfg.DefaultContext != "prod-context" {
		t.Errorf("期望 DefaultContext 为 'prod-context'，实际为 '%s'", cfg.DefaultContext)
	}

	if cfg.LogLevel != "warn" {
		t.Errorf("期望 LogLevel 为 'warn'，实际为 '%s'", cfg.LogLevel)
	}

	if cfg.MaxConcurrentRequests != 150 {
		t.Errorf("期望 MaxConcurrentRequests 为 150，实际为 %d", cfg.MaxConcurrentRequests)
	}

	if cfg.RequestTimeout != 45*time.Second {
		t.Errorf("期望 RequestTimeout 为 45s，实际为 %v", cfg.RequestTimeout)
	}

	if cfg.APIToken != "file-token" {
		t.Errorf("期望 APIToken 为 'file-token'，实际为 '%s'", cfg.APIToken)
	}

	if cfg.CacheTTL != 10*time.Minute {
		t.Errorf("期望 CacheTTL 为 10m，实际为 %v", cfg.CacheTTL)
	}
}

// TestLoadConfig_Priority 测试配置加载优先级：命令行 > 环境变量 > 配置文件
func TestLoadConfig_Priority(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	// 创建临时目录和配置文件
	tmpDir := t.TempDir()
	kubeconfigPath := createTempKubeconfig(t)

	// 使用正斜杠，YAML 中更安全
	kubeconfigPathForYAML := filepath.ToSlash(kubeconfigPath)

	// 1. 配置文件设置 port = 7070
	configContent := `host: "192.168.1.1"
port: 7070
kubeconfigPath: ` + kubeconfigPathForYAML + `
logLevel: "warn"
`

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("创建配置文件失败: %v", err)
	}

	// 切换到临时目录
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	// 2. 环境变量设置 port = 9090（应该覆盖配置文件）
	os.Setenv("KUBECTL_MCP_PORT", "9090")
	os.Setenv("KUBECTL_MCP_LOGLEVEL", "debug")

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证优先级：环境变量应该覆盖配置文件
	if cfg.Port != 9090 {
		t.Errorf("期望 Port 为 9090（环境变量），实际为 %d", cfg.Port)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("期望 LogLevel 为 'debug'（环境变量），实际为 '%s'", cfg.LogLevel)
	}

	// Host 没有环境变量，应该使用配置文件的值
	if cfg.Host != "192.168.1.1" {
		t.Errorf("期望 Host 为 '192.168.1.1'（配置文件），实际为 '%s'", cfg.Host)
	}
}

// TestValidate_InvalidPort 测试无效端口验证
func TestValidate_InvalidPort(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	kubeconfigPath := createTempKubeconfig(t)

	tests := []struct {
		name string
		port int
	}{
		{"端口为 0", 0},
		{"端口为负数", -1},
		{"端口超过最大值", 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("KUBECTL_MCP_PORT", "8080")
			os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)

			cfg, err := config.LoadConfig()
			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}

			cfg.Port = tt.port
			err = cfg.Validate()
			if err == nil {
				t.Errorf("期望验证失败，但成功了")
			}
		})
	}
}

// TestValidate_InvalidKubeconfig 测试无效 kubeconfig 验证
func TestValidate_InvalidKubeconfig(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	tests := []struct {
		name           string
		kubeconfigPath string
	}{
		{"kubeconfig 路径为空", ""},
		{"kubeconfig 文件不存在", "/nonexistent/path/config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("KUBECTL_MCP_PORT", "8080")
			os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", tt.kubeconfigPath)

			_, err := config.LoadConfig()
			if err == nil {
				t.Errorf("期望验证失败，但成功了")
			}
		})
	}
}

// TestValidate_InvalidLogLevel 测试无效日志级别验证
func TestValidate_InvalidLogLevel(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	kubeconfigPath := createTempKubeconfig(t)

	os.Setenv("KUBECTL_MCP_PORT", "8080")
	os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)
	os.Setenv("KUBECTL_MCP_LOGLEVEL", "invalid")

	_, err := config.LoadConfig()
	if err == nil {
		t.Errorf("期望验证失败，但成功了")
	}
}

// TestValidate_InvalidLogFormat 测试无效日志格式验证
func TestValidate_InvalidLogFormat(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	kubeconfigPath := createTempKubeconfig(t)

	os.Setenv("KUBECTL_MCP_PORT", "8080")
	os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)
	os.Setenv("KUBECTL_MCP_LOGFORMAT", "xml")

	_, err := config.LoadConfig()
	if err == nil {
		t.Errorf("期望验证失败，但成功了")
	}
}

// TestValidate_InvalidPerformanceConfig 测试无效性能配置验证
func TestValidate_InvalidPerformanceConfig(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	kubeconfigPath := createTempKubeconfig(t)

	tests := []struct {
		name    string
		envVar  string
		envVal  string
		wantErr bool
	}{
		{"最大并发请求数为 0", "KUBECTL_MCP_MAXCONCURRENTREQUESTS", "0", true},
		{"请求超时时间小于 1 秒", "KUBECTL_MCP_REQUESTTIMEOUT", "500ms", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Reset()
			os.Clearenv()

			os.Setenv("KUBECTL_MCP_PORT", "8080")
			os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)
			os.Setenv(tt.envVar, tt.envVal)

			_, err := config.LoadConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("期望错误 = %v，实际错误 = %v", tt.wantErr, err)
			}
		})
	}
}

// TestValidate_InvalidCacheConfig 测试无效缓存配置验证
func TestValidate_InvalidCacheConfig(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	kubeconfigPath := createTempKubeconfig(t)

	os.Setenv("KUBECTL_MCP_PORT", "8080")
	os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)
	os.Setenv("KUBECTL_MCP_ENABLECACHE", "true")
	os.Setenv("KUBECTL_MCP_CACHETTL", "500ms")

	_, err := config.LoadConfig()
	if err == nil {
		t.Errorf("期望验证失败（缓存 TTL 小于 1 秒），但成功了")
	}
}

// TestGetListenAddress 测试获取监听地址
func TestGetListenAddress(t *testing.T) {
	cleanup := setupTest(t)
	defer cleanup()

	kubeconfigPath := createTempKubeconfig(t)

	os.Setenv("KUBECTL_MCP_HOST", "127.0.0.1")
	os.Setenv("KUBECTL_MCP_PORT", "9090")
	os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	expected := "127.0.0.1:9090"
	if cfg.GetListenAddress() != expected {
		t.Errorf("期望监听地址为 '%s'，实际为 '%s'", expected, cfg.GetListenAddress())
	}
}

// TestIsDebugMode 测试调试模式判断
func TestIsDebugMode(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected bool
	}{
		{"debug 模式", "debug", true},
		{"info 模式", "info", false},
		{"warn 模式", "warn", false},
		{"error 模式", "error", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupTest(t)
			defer cleanup()

			kubeconfigPath := createTempKubeconfig(t)

			os.Setenv("KUBECTL_MCP_PORT", "8080")
			os.Setenv("KUBECTL_MCP_KUBECONFIGPATH", kubeconfigPath)
			os.Setenv("KUBECTL_MCP_LOGLEVEL", tt.logLevel)

			cfg, err := config.LoadConfig()
			if err != nil {
				t.Fatalf("加载配置失败: %v", err)
			}

			if cfg.IsDebugMode() != tt.expected {
				t.Errorf("期望 IsDebugMode() 返回 %v，实际返回 %v", tt.expected, cfg.IsDebugMode())
			}
		})
	}
}
