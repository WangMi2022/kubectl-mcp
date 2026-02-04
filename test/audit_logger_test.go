package test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kubectl-mcp/internal/audit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLogOperationCompleteness 测试日志记录完整性
// Requirements: 9.1, 9.2
func TestLogOperationCompleteness(t *testing.T) {
	tests := []struct {
		name        string
		log         *audit.OperationLog
		expectError bool
		description string
	}{
		{
			name: "完整的成功操作日志",
			log: &audit.OperationLog{
				Timestamp: time.Now(),
				User: &audit.UserInfo{
					ID:   "user123",
					Name: "admin",
					Role: "administrator",
				},
				Tool:      "get_pods",
				Arguments: map[string]interface{}{"namespace": "default"},
				Context:   "prod-cluster",
				Namespace: "default",
				Success:   true,
				Duration:  100 * time.Millisecond,
			},
			expectError: false,
			description: "应该成功记录包含所有字段的操作日志",
		},
		{
			name: "完整的失败操作日志",
			log: &audit.OperationLog{
				Timestamp: time.Now(),
				User: &audit.UserInfo{
					ID:   "user456",
					Name: "operator",
				},
				Tool:      "delete_pod",
				Arguments: map[string]interface{}{"name": "test-pod", "namespace": "default"},
				Context:   "dev-cluster",
				Namespace: "default",
				Success:   false,
				Error:     "pod not found",
				Duration:  50 * time.Millisecond,
			},
			expectError: false,
			description: "应该成功记录失败的操作日志并包含错误信息",
		},
		{
			name: "最小必需字段的操作日志",
			log: &audit.OperationLog{
				Tool:    "get_nodes",
				Context: "test-cluster",
				Success: true,
			},
			expectError: false,
			description: "应该成功记录只包含必需字段的操作日志",
		},
		{
			name: "包含资源信息的操作日志",
			log: &audit.OperationLog{
				Tool:         "create_deployment",
				Context:      "prod-cluster",
				Namespace:    "production",
				ResourceType: "Deployment",
				ResourceName: "nginx-deployment",
				Success:      true,
				Duration:     200 * time.Millisecond,
			},
			expectError: false,
			description: "应该成功记录包含资源类型和名称的操作日志",
		},
		{
			name:        "空日志对象",
			log:         nil,
			expectError: true,
			description: "空日志对象应该返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时日志文件
			tmpFile := filepath.Join(t.TempDir(), "test.log")

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			require.NoError(t, err)

			// 记录日志
			err = logger.LogOperation(tt.log)

			// 立即关闭日志器以释放文件句柄
			logger.Close()

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)

				// 验证日志文件内容
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)

				logContent := string(content)
				assert.NotEmpty(t, logContent, "日志文件不应为空")

				// 验证必需字段存在
				if tt.log != nil {
					assert.Contains(t, logContent, tt.log.Tool, "日志应包含工具名称")
					assert.Contains(t, logContent, tt.log.Context, "日志应包含 context")

					// 验证用户信息
					if tt.log.User != nil {
						assert.Contains(t, logContent, tt.log.User.ID, "日志应包含用户 ID")
						assert.Contains(t, logContent, tt.log.User.Name, "日志应包含用户名称")
					}

					// 验证错误信息
					if !tt.log.Success && tt.log.Error != "" {
						assert.Contains(t, logContent, tt.log.Error, "失败日志应包含错误信息")
					}

					// 验证资源信息
					if tt.log.ResourceType != "" {
						assert.Contains(t, logContent, tt.log.ResourceType, "日志应包含资源类型")
					}
					if tt.log.ResourceName != "" {
						assert.Contains(t, logContent, tt.log.ResourceName, "日志应包含资源名称")
					}
				}
			}
		})
	}
}

// TestLogFormatting 测试日志格式化
// Requirements: 9.1, 9.2
func TestLogFormatting(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedFormat string
		description    string
	}{
		{
			name:           "JSON 格式",
			format:         "json",
			expectedFormat: "json",
			description:    "应该以 JSON 格式输出日志",
		},
		{
			name:           "文本格式",
			format:         "text",
			expectedFormat: "text",
			description:    "应该以文本格式输出日志",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时日志文件
			tmpFile := filepath.Join(t.TempDir(), "test.log")

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   tt.format,
				Output:   "file",
				FilePath: tmpFile,
			})
			require.NoError(t, err)

			// 记录测试日志
			testLog := &audit.OperationLog{
				Timestamp: time.Now(),
				User: &audit.UserInfo{
					ID:   "test-user",
					Name: "Test User",
				},
				Tool:      "test_tool",
				Context:   "test-context",
				Success:   true,
				Duration:  100 * time.Millisecond,
				Arguments: map[string]interface{}{"key": "value"},
			}

			err = logger.LogOperation(testLog)
			require.NoError(t, err)

			// 立即关闭日志器以释放文件句柄
			logger.Close()

			// 读取日志文件
			content, err := os.ReadFile(tmpFile)
			require.NoError(t, err)

			logContent := string(content)
			assert.NotEmpty(t, logContent, "日志文件不应为空")

			// 验证格式
			if tt.expectedFormat == "json" {
				// JSON 格式应该包含 JSON 结构
				assert.Contains(t, logContent, `"tool":"test_tool"`, "JSON 日志应包含正确的字段格式")
				assert.Contains(t, logContent, `"context":"test-context"`, "JSON 日志应包含 context 字段")
				assert.Contains(t, logContent, `"user_id":"test-user"`, "JSON 日志应包含用户 ID")
				assert.Contains(t, logContent, `"success":true`, "JSON 日志应包含成功状态")
			} else {
				// 文本格式应该是可读的
				assert.Contains(t, logContent, "test_tool", "文本日志应包含工具名称")
				assert.Contains(t, logContent, "test-context", "文本日志应包含 context")
				assert.Contains(t, logContent, "test-user", "文本日志应包含用户 ID")
			}

			// 验证时间戳格式
			assert.Contains(t, logContent, "timestamp", "日志应包含时间戳字段")
		})
	}
}

// TestLogOutputTargets 测试日志输出到不同目标
// Requirements: 9.1, 9.2
func TestLogOutputTargets(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		filePath    string
		expectError bool
		description string
	}{
		{
			name:        "输出到文件",
			output:      "file",
			filePath:    "test_file.log",
			expectError: false,
			description: "应该成功将日志输出到文件",
		},
		{
			name:        "输出到 stdout",
			output:      "stdout",
			filePath:    "",
			expectError: false,
			description: "应该成功将日志输出到 stdout",
		},
		{
			name:        "同时输出到文件和 stdout",
			output:      "both",
			filePath:    "test_both.log",
			expectError: false,
			description: "应该成功将日志同时输出到文件和 stdout",
		},
		{
			name:        "文件输出但未指定路径",
			output:      "file",
			filePath:    "",
			expectError: true,
			description: "文件输出时未指定路径应该返回错误",
		},
		{
			name:        "both 输出但未指定路径",
			output:      "both",
			filePath:    "",
			expectError: true,
			description: "both 输出时未指定路径应该返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fullPath string
			if tt.filePath != "" {
				fullPath = filepath.Join(t.TempDir(), tt.filePath)
			}

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   tt.output,
				FilePath: fullPath,
			})

			if tt.expectError {
				assert.Error(t, err, tt.description)
				return
			}

			require.NoError(t, err, tt.description)

			// 记录测试日志
			testLog := &audit.OperationLog{
				Tool:    "test_output",
				Context: "test-context",
				Success: true,
			}

			err = logger.LogOperation(testLog)
			require.NoError(t, err)

			// 立即关闭日志器以释放文件句柄
			logger.Close()

			// 验证文件输出
			if tt.output == "file" || tt.output == "both" {
				// 验证文件存在
				_, err := os.Stat(fullPath)
				assert.NoError(t, err, "日志文件应该存在")

				// 验证文件内容
				content, err := os.ReadFile(fullPath)
				require.NoError(t, err)
				assert.NotEmpty(t, content, "日志文件不应为空")
				assert.Contains(t, string(content), "test_output", "日志文件应包含工具名称")
			}
		})
	}
}

// TestLogError 测试错误日志记录
// Requirements: 9.1, 9.2
func TestLogError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		context     string
		expectError bool
		description string
	}{
		{
			name:        "记录标准错误",
			err:         errors.New("test error"),
			context:     "test context",
			expectError: false,
			description: "应该成功记录标准错误",
		},
		{
			name:        "记录格式化错误",
			err:         fmt.Errorf("formatted error: %s", "details"),
			context:     "error context",
			expectError: false,
			description: "应该成功记录格式化错误",
		},
		{
			name:        "空错误对象",
			err:         nil,
			context:     "test context",
			expectError: true,
			description: "空错误对象应该返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时日志文件
			tmpFile := filepath.Join(t.TempDir(), "error.log")

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "error",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			require.NoError(t, err)

			// 记录错误
			err = logger.LogError(tt.err, tt.context)

			// 立即关闭日志器以释放文件句柄
			logger.Close()

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)

				// 验证日志文件内容
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)

				logContent := string(content)
				assert.NotEmpty(t, logContent, "错误日志文件不应为空")
				assert.Contains(t, logContent, tt.context, "错误日志应包含上下文信息")
				if tt.err != nil {
					assert.Contains(t, logContent, tt.err.Error(), "错误日志应包含错误消息")
				}
			}
		})
	}
}

// TestLogMetrics 测试性能指标记录
// Requirements: 9.1, 9.2
func TestLogMetrics(t *testing.T) {
	tests := []struct {
		name        string
		metrics     *audit.Metrics
		expectError bool
		description string
	}{
		{
			name: "完整的性能指标",
			metrics: &audit.Metrics{
				Timestamp:       time.Now(),
				Operation:       "get_pods",
				Duration:        150 * time.Millisecond,
				Success:         true,
				ConcurrentCount: 5,
			},
			expectError: false,
			description: "应该成功记录完整的性能指标",
		},
		{
			name: "最小性能指标",
			metrics: &audit.Metrics{
				Operation: "list_nodes",
				Duration:  50 * time.Millisecond,
				Success:   true,
			},
			expectError: false,
			description: "应该成功记录最小性能指标",
		},
		{
			name:        "空指标对象",
			metrics:     nil,
			expectError: true,
			description: "空指标对象应该返回错误",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时日志文件
			tmpFile := filepath.Join(t.TempDir(), "metrics.log")

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			require.NoError(t, err)

			// 记录指标
			err = logger.LogMetrics(tt.metrics)

			// 立即关闭日志器以释放文件句柄
			logger.Close()

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)

				// 验证日志文件内容
				content, err := os.ReadFile(tmpFile)
				require.NoError(t, err)

				logContent := string(content)
				assert.NotEmpty(t, logContent, "指标日志文件不应为空")

				if tt.metrics != nil {
					assert.Contains(t, logContent, tt.metrics.Operation, "指标日志应包含操作名称")
					assert.Contains(t, logContent, "duration", "指标日志应包含持续时间")
					assert.Contains(t, logContent, "success", "指标日志应包含成功状态")
				}
			}
		})
	}
}

// TestLoggerConfiguration 测试日志器配置
// Requirements: 9.1, 9.2
func TestLoggerConfiguration(t *testing.T) {
	t.Run("默认配置", func(t *testing.T) {
		logger, err := audit.NewAuditLogger(nil)
		require.NoError(t, err)
		defer logger.Close()

		config := logger.GetConfig()
		assert.Equal(t, "info", config.Level, "默认日志级别应为 info")
		assert.Equal(t, "json", config.Format, "默认格式应为 json")
		assert.Equal(t, "stdout", config.Output, "默认输出应为 stdout")
	})

	t.Run("自定义配置", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "custom.log")

		customConfig := &audit.LoggerConfig{
			Level:      "debug",
			Format:     "text",
			Output:     "file",
			FilePath:   tmpFile,
			MaxSize:    50,
			MaxBackups: 5,
			MaxAge:     14,
			Compress:   false,
		}

		logger, err := audit.NewAuditLogger(customConfig)
		require.NoError(t, err)

		config := logger.GetConfig()
		assert.Equal(t, "debug", config.Level)
		assert.Equal(t, "text", config.Format)
		assert.Equal(t, "file", config.Output)
		assert.Equal(t, tmpFile, config.FilePath)

		// 立即关闭日志器以释放文件句柄
		logger.Close()
	})

	t.Run("动态设置日志级别", func(t *testing.T) {
		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		})
		require.NoError(t, err)
		defer logger.Close()

		// 测试有效的日志级别
		validLevels := []string{"debug", "info", "warn", "error"}
		for _, level := range validLevels {
			err := logger.SetLevel(level)
			assert.NoError(t, err, "应该成功设置有效的日志级别: %s", level)

			config := logger.GetConfig()
			assert.Equal(t, level, config.Level, "日志级别应该被更新")
		}

		// 测试无效的日志级别
		err = logger.SetLevel("invalid")
		assert.Error(t, err, "无效的日志级别应该返回错误")
	})
}

// TestConcurrentLogging 测试并发日志记录
// Requirements: 9.1, 9.2
func TestConcurrentLogging(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "concurrent.log")

	logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: tmpFile,
	})
	require.NoError(t, err)

	// 并发写入日志
	const numGoroutines = 10
	const logsPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < logsPerGoroutine; j++ {
				log := &audit.OperationLog{
					Tool:    fmt.Sprintf("tool_%d_%d", id, j),
					Context: fmt.Sprintf("context_%d", id),
					Success: true,
				}
				err := logger.LogOperation(log)
				assert.NoError(t, err)
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// 立即关闭日志器以释放文件句柄
	logger.Close()

	// 验证日志文件
	content, err := os.ReadFile(tmpFile)
	require.NoError(t, err)

	// 计算日志行数
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	expectedLines := numGoroutines * logsPerGoroutine
	assert.Equal(t, expectedLines, len(lines), "应该记录所有并发日志")
}
