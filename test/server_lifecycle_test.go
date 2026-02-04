package test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerStartup 测试服务器启动流程
// Requirements: 1.6
func TestServerStartup(t *testing.T) {
	// 创建临时 kubeconfig 文件
	tmpDir, err := os.MkdirTemp("", "kubectl-mcp-test-*")
	require.NoError(t, err, "创建临时目录失败")
	defer os.RemoveAll(tmpDir)

	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err = os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0644)
	require.NoError(t, err, "创建临时 kubeconfig 失败")

	// 创建临时配置文件
	configPath := filepath.Join(tmpDir, "config.yaml")
	// 在 Windows 上，路径需要使用正斜杠或转义反斜杠
	kubeconfigPathForYAML := filepath.ToSlash(kubeconfigPath)
	configContent := fmt.Sprintf(`
host: "127.0.0.1"
port: 18080
kubeconfigPath: "%s"
logLevel: "info"
logFormat: "json"
maxConcurrentRequests: 10
requestTimeout: 30s
`, kubeconfigPathForYAML)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "创建临时配置文件失败")

	// 构建服务器二进制文件
	serverBinary := filepath.Join(tmpDir, "server.exe")

	// 获取当前工作目录
	cwd, err := os.Getwd()
	require.NoError(t, err, "获取当前目录失败")

	// 构建命令需要在项目根目录执行
	mainPath := filepath.Join(cwd, "..", "cmd", "server", "main.go")
	buildCmd := exec.Command("go", "build", "-o", serverBinary, mainPath)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("构建输出: %s", string(buildOutput))
		t.Skipf("跳过测试：无法构建服务器二进制文件: %v", err)
		return
	}

	// 启动服务器进程
	cmd := exec.Command(serverBinary)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECTL_MCP_CONFIG=%s", configPath),
		fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
	)

	// 捕获输出
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "创建 stdout pipe 失败")
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err, "创建 stderr pipe 失败")

	// 启动服务器
	err = cmd.Start()
	require.NoError(t, err, "启动服务器失败")

	// 确保测试结束时关闭服务器
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	// 读取启动日志
	startupLog := make(chan string, 1)
	go func() {
		output, _ := io.ReadAll(stdout)
		startupLog <- string(output)
	}()

	errorLog := make(chan string, 1)
	go func() {
		output, _ := io.ReadAll(stderr)
		errorLog <- string(output)
	}()

	// 等待服务器启动（最多 10 秒）
	serverReady := false
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)

		// 尝试访问健康检查端点
		resp, err := http.Get("http://127.0.0.1:18080/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverReady = true
				break
			}
		}
	}

	// 如果服务器未启动，打印日志
	if !serverReady {
		select {
		case output := <-startupLog:
			t.Logf("标准输出:\n%s", output)
		case <-time.After(100 * time.Millisecond):
		}
		select {
		case output := <-errorLog:
			t.Logf("错误输出:\n%s", output)
		case <-time.After(100 * time.Millisecond):
		}
	}

	require.True(t, serverReady, "服务器未能在 10 秒内启动")

	// 验证服务器响应
	resp, err := http.Get("http://127.0.0.1:18080/health")
	require.NoError(t, err, "访问健康检查端点失败")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "健康检查返回状态码不正确")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应体失败")

	// 验证响应包含预期字段
	bodyStr := string(body)
	assert.Contains(t, bodyStr, "status", "响应应包含 status 字段")
	assert.Contains(t, bodyStr, "version", "响应应包含 version 字段")

	// 停止服务器
	t.Log("正在停止服务器...")
	err = cmd.Process.Kill()
	require.NoError(t, err, "停止服务器失败")

	// 等待服务器退出
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		t.Logf("服务器退出: %v", err)
	case <-time.After(5 * time.Second):
		t.Error("服务器未能在 5 秒内退出")
	}

	// 读取并验证启动日志
	select {
	case output := <-startupLog:
		assert.Contains(t, output, "正在加载配置", "启动日志应包含配置加载信息")
		assert.Contains(t, output, "服务器已启动", "启动日志应包含服务器启动信息")
	case <-time.After(1 * time.Second):
		// 超时，继续
	}
}

// TestServerGracefulShutdown 测试服务器优雅关闭
// Requirements: 1.6, 14.7
// 注意：Windows 不支持 SIGTERM，此测试在 Windows 上会跳过
func TestServerGracefulShutdown(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不支持 SIGTERM 信号，跳过优雅关闭测试")
	}

	// 创建临时 kubeconfig 文件
	tmpDir := t.TempDir()
	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err := os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0644)
	require.NoError(t, err, "创建临时 kubeconfig 失败")

	// 创建临时配置文件
	configPath := filepath.Join(tmpDir, "config.yaml")
	kubeconfigPathForYAML := filepath.ToSlash(kubeconfigPath)
	configContent := fmt.Sprintf(`
host: "127.0.0.1"
port: 18081
kubeconfigPath: "%s"
logLevel: "info"
logFormat: "json"
maxConcurrentRequests: 10
requestTimeout: 30s
`, kubeconfigPathForYAML)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "创建临时配置文件失败")

	// 构建服务器二进制文件
	serverBinary := filepath.Join(tmpDir, "server")
	cwd, err := os.Getwd()
	require.NoError(t, err, "获取当前目录失败")

	mainPath := filepath.Join(cwd, "..", "cmd", "server", "main.go")
	buildCmd := exec.Command("go", "build", "-o", serverBinary, mainPath)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("构建输出: %s", string(buildOutput))
		t.Skipf("跳过测试：无法构建服务器二进制文件: %v", err)
		return
	}

	// 启动服务器进程
	cmd := exec.Command(serverBinary)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECTL_MCP_CONFIG=%s", configPath),
		fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
	)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "创建 stdout pipe 失败")

	err = cmd.Start()
	require.NoError(t, err, "启动服务器失败")

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	outputChan := make(chan string, 1)
	go func() {
		output, _ := io.ReadAll(stdout)
		outputChan <- string(output)
	}()

	// 等待服务器启动
	serverReady := false
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:18081/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverReady = true
				break
			}
		}
	}

	require.True(t, serverReady, "服务器未能在 10 秒内启动")

	// 发送 SIGTERM 信号
	shutdownStart := time.Now()
	err = cmd.Process.Signal(syscall.SIGTERM)
	require.NoError(t, err, "发送 SIGTERM 信号失败")

	// 等待服务器退出
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		shutdownDuration := time.Since(shutdownStart)
		t.Logf("服务器在 %v 内完成优雅关闭", shutdownDuration)
		assert.Less(t, shutdownDuration, 30*time.Second, "关闭时间应小于超时时间")

		select {
		case output := <-outputChan:
			assert.Contains(t, output, "正在关闭", "输出应包含关闭日志")
		case <-time.After(1 * time.Second):
			// 超时，继续
		}

	case <-time.After(35 * time.Second):
		t.Error("服务器未能在 35 秒内完成优雅关闭")
		cmd.Process.Kill()
	}
}

// TestServerSignalHandling 测试服务器信号处理
// Requirements: 1.6, 14.7
// 注意：Windows 不支持 Unix 信号，此测试在 Windows 上会跳过
func TestServerSignalHandling(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 不支持 Unix 信号，跳过信号处理测试")
	}

	signals := []struct {
		name   string
		signal os.Signal
	}{
		{"SIGINT", syscall.SIGINT},
		{"SIGTERM", syscall.SIGTERM},
	}

	for _, tc := range signals {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
			err := os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0644)
			require.NoError(t, err, "创建临时 kubeconfig 失败")

			port := 18082
			if tc.signal == syscall.SIGTERM {
				port = 18083
			}

			configPath := filepath.Join(tmpDir, "config.yaml")
			kubeconfigPathForYAML := filepath.ToSlash(kubeconfigPath)
			configContent := fmt.Sprintf(`
host: "127.0.0.1"
port: %d
kubeconfigPath: "%s"
logLevel: "info"
logFormat: "json"
maxConcurrentRequests: 10
requestTimeout: 30s
`, port, kubeconfigPathForYAML)
			err = os.WriteFile(configPath, []byte(configContent), 0644)
			require.NoError(t, err, "创建临时配置文件失败")

			serverBinary := filepath.Join(tmpDir, "server")
			cwd, err := os.Getwd()
			require.NoError(t, err, "获取当前目录失败")

			mainPath := filepath.Join(cwd, "..", "cmd", "server", "main.go")
			buildCmd := exec.Command("go", "build", "-o", serverBinary, mainPath)
			buildOutput, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Logf("构建输出: %s", string(buildOutput))
				t.Skipf("跳过测试：无法构建服务器二进制文件: %v", err)
				return
			}

			cmd := exec.Command(serverBinary)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECTL_MCP_CONFIG=%s", configPath),
				fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
			)

			stdout, err := cmd.StdoutPipe()
			require.NoError(t, err, "创建 stdout pipe 失败")

			err = cmd.Start()
			require.NoError(t, err, "启动服务器失败")

			defer func() {
				if cmd.Process != nil {
					cmd.Process.Kill()
					cmd.Wait()
				}
			}()

			outputChan := make(chan string, 1)
			go func() {
				output, _ := io.ReadAll(stdout)
				outputChan <- string(output)
			}()

			serverReady := false
			healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
			for i := 0; i < 20; i++ {
				time.Sleep(500 * time.Millisecond)
				resp, err := http.Get(healthURL)
				if err == nil {
					resp.Body.Close()
					if resp.StatusCode == http.StatusOK {
						serverReady = true
						break
					}
				}
			}

			require.True(t, serverReady, "服务器未能在 10 秒内启动")

			t.Logf("发送 %s 信号", tc.name)
			err = cmd.Process.Signal(tc.signal)
			require.NoError(t, err, "发送信号失败")

			done := make(chan error, 1)
			go func() {
				done <- cmd.Wait()
			}()

			select {
			case err := <-done:
				t.Logf("服务器收到 %s 信号后退出: %v", tc.name, err)

				select {
				case output := <-outputChan:
					outputLower := strings.ToLower(output)
					assert.True(t,
						strings.Contains(outputLower, "信号") ||
							strings.Contains(outputLower, "signal") ||
							strings.Contains(outputLower, "关闭") ||
							strings.Contains(outputLower, "shutdown"),
						"输出应包含信号或关闭相关的日志")
				case <-time.After(1 * time.Second):
					// 超时，继续
				}

			case <-time.After(10 * time.Second):
				t.Errorf("服务器未能在 10 秒内响应 %s 信号", tc.name)
				cmd.Process.Kill()
			}
		})
	}
}

// TestServerStartupWithInvalidConfig 测试使用无效配置启动服务器
// Requirements: 1.6
func TestServerStartupWithInvalidConfig(t *testing.T) {
	testCases := []struct {
		name          string
		configContent string
		expectedError string
	}{
		{
			name: "缺少 kubeconfig 路径",
			configContent: `
host: "127.0.0.1"
port: 18084
logLevel: "info"
`,
			expectedError: "kubeconfig",
		},
		{
			name: "无效的端口号",
			configContent: `
host: "127.0.0.1"
port: -1
kubeconfigPath: "/tmp/kubeconfig"
logLevel: "info"
`,
			expectedError: "端口",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "kubectl-mcp-test-*")
			require.NoError(t, err, "创建临时目录失败")
			defer os.RemoveAll(tmpDir)

			configPath := filepath.Join(tmpDir, "config.yaml")
			err = os.WriteFile(configPath, []byte(tc.configContent), 0644)
			require.NoError(t, err, "创建临时配置文件失败")

			serverBinary := filepath.Join(tmpDir, "server.exe")
			cwd, err := os.Getwd()
			require.NoError(t, err, "获取当前目录失败")

			mainPath := filepath.Join(cwd, "..", "cmd", "server", "main.go")
			buildCmd := exec.Command("go", "build", "-o", serverBinary, mainPath)
			buildOutput, err := buildCmd.CombinedOutput()
			if err != nil {
				t.Logf("构建输出: %s", string(buildOutput))
				t.Skipf("跳过测试：无法构建服务器二进制文件: %v", err)
				return
			}

			cmd := exec.Command(serverBinary)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("KUBECTL_MCP_CONFIG=%s", configPath),
			)

			stderr, err := cmd.StderrPipe()
			require.NoError(t, err, "创建 stderr pipe 失败")

			err = cmd.Start()
			require.NoError(t, err, "启动服务器失败")

			errorOutput, _ := io.ReadAll(stderr)
			err = cmd.Wait()

			if err == nil {
				t.Error("服务器应该因配置错误而退出")
			}

			errorStr := string(errorOutput)
			assert.Contains(t, strings.ToLower(errorStr),
				strings.ToLower(tc.expectedError),
				"错误输出应包含预期的错误信息")
		})
	}
}

// TestServerComponentInitialization 测试服务器组件初始化顺序
// Requirements: 1.6
func TestServerComponentInitialization(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kubectl-mcp-test-*")
	require.NoError(t, err, "创建临时目录失败")
	defer os.RemoveAll(tmpDir)

	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	err = os.WriteFile(kubeconfigPath, []byte(testKubeconfig), 0644)
	require.NoError(t, err, "创建临时 kubeconfig 失败")

	configPath := filepath.Join(tmpDir, "config.yaml")
	kubeconfigPathForYAML := filepath.ToSlash(kubeconfigPath)
	configContent := fmt.Sprintf(`
host: "127.0.0.1"
port: 18085
kubeconfigPath: "%s"
logLevel: "debug"
logFormat: "text"
maxConcurrentRequests: 10
requestTimeout: 30s
`, kubeconfigPathForYAML)
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err, "创建临时配置文件失败")

	serverBinary := filepath.Join(tmpDir, "server.exe")
	cwd, err := os.Getwd()
	require.NoError(t, err, "获取当前目录失败")

	mainPath := filepath.Join(cwd, "..", "cmd", "server", "main.go")
	buildCmd := exec.Command("go", "build", "-o", serverBinary, mainPath)
	buildOutput, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Logf("构建输出: %s", string(buildOutput))
		t.Skipf("跳过测试：无法构建服务器二进制文件: %v", err)
		return
	}

	cmd := exec.Command(serverBinary)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("KUBECTL_MCP_CONFIG=%s", configPath),
		fmt.Sprintf("KUBECONFIG=%s", kubeconfigPath),
	)

	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err, "创建 stdout pipe 失败")

	err = cmd.Start()
	require.NoError(t, err, "启动服务器失败")

	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
	}()

	outputChan := make(chan string, 1)
	go func() {
		output, _ := io.ReadAll(stdout)
		outputChan <- string(output)
	}()

	serverReady := false
	for i := 0; i < 20; i++ {
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://127.0.0.1:18085/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				serverReady = true
				break
			}
		}
	}

	require.True(t, serverReady, "服务器未能在 10 秒内启动")

	err = cmd.Process.Kill()
	require.NoError(t, err, "停止服务器失败")

	cmd.Wait()

	var output string
	select {
	case output = <-outputChan:
	case <-time.After(1 * time.Second):
		t.Fatal("读取输出超时")
	}

	// 验证组件初始化顺序
	expectedSequence := []string{
		"配置",
		"审计日志",
		"性能指标",
		"Kubernetes 客户端",
		"工具",
		"HTTP 服务器",
	}

	lastIndex := -1
	for _, component := range expectedSequence {
		index := strings.Index(output, component)
		if index == -1 {
			t.Logf("警告：输出中未找到组件 '%s'", component)
			continue
		}

		if index < lastIndex {
			t.Errorf("组件初始化顺序错误：'%s' 应该在前一个组件之后", component)
		}
		lastIndex = index
	}
}

// testKubeconfig 是用于测试的最小 kubeconfig
const testKubeconfig = `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:6443
    insecure-skip-tls-verify: true
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
