package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"kubectl-mcp/internal/audit"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"
)

// TestLogFormatConsistencyProperty 测试日志格式一致性属性
// Feature: kubectl-mcp-server, Property 10: 日志格式一致性
// Validates: Requirements 9.6, 9.7
//
// 属性：对于任何操作日志，必须包含时间戳、用户信息、操作类型、context、结果等标准字段，
// 格式必须符合配置的日志格式（JSON/文本）
func TestLogFormatConsistencyProperty(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 属性1: JSON 格式日志必须包含所有标准字段
	properties.Property("JSON格式日志包含所有标准字段", prop.ForAll(
		func(log *audit.OperationLog) bool {
			// 创建临时日志文件 - 使用当前目录避免权限问题
			tmpDir, err := os.MkdirTemp(".", "kubectl-mcp-test-*")
			if err != nil {
				t.Logf("创建临时目录失败: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)
			tmpFile := filepath.Join(tmpDir, fmt.Sprintf("test_%d.log", time.Now().UnixNano()))

			// 创建 JSON 格式的日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			if err != nil {
				t.Logf("创建日志器失败: %v", err)
				return false
			}

			// 记录日志
			err = logger.LogOperation(log)
			if err != nil {
				logger.Close()
				t.Logf("记录日志失败: %v", err)
				return false
			}

			// 关闭日志器
			logger.Close()

			// 读取日志文件
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Logf("读取日志文件失败: %v", err)
				return false
			}

			logContent := string(content)
			if logContent == "" {
				t.Logf("日志文件为空")
				return false
			}

			// 解析 JSON 日志
			var logData map[string]interface{}
			lines := strings.Split(strings.TrimSpace(logContent), "\n")
			if len(lines) == 0 {
				t.Logf("日志文件没有内容行")
				return false
			}

			err = json.Unmarshal([]byte(lines[0]), &logData)
			if err != nil {
				t.Logf("解析 JSON 日志失败: %v, 内容: %s", err, lines[0])
				return false
			}

			// 验证标准字段存在
			requiredFields := []string{"timestamp", "tool", "context", "success", "duration"}
			for _, field := range requiredFields {
				if _, exists := logData[field]; !exists {
					t.Logf("缺少必需字段: %s", field)
					return false
				}
			}

			// 验证 tool 字段值正确
			if toolValue, ok := logData["tool"].(string); !ok || toolValue != log.Tool {
				t.Logf("tool 字段值不正确: 期望 %s, 实际 %v", log.Tool, logData["tool"])
				return false
			}

			// 验证 context 字段值正确
			if contextValue, ok := logData["context"].(string); !ok || contextValue != log.Context {
				t.Logf("context 字段值不正确: 期望 %s, 实际 %v", log.Context, logData["context"])
				return false
			}

			// 验证 success 字段值正确
			if successValue, ok := logData["success"].(bool); !ok || successValue != log.Success {
				t.Logf("success 字段值不正确: 期望 %v, 实际 %v", log.Success, logData["success"])
				return false
			}

			// 如果有用户信息，验证用户字段
			if log.User != nil {
				if _, exists := logData["user_id"]; !exists {
					t.Logf("缺少 user_id 字段")
					return false
				}
				if _, exists := logData["user_name"]; !exists {
					t.Logf("缺少 user_name 字段")
					return false
				}

				if userID, ok := logData["user_id"].(string); !ok || userID != log.User.ID {
					t.Logf("user_id 字段值不正确: 期望 %s, 实际 %v", log.User.ID, logData["user_id"])
					return false
				}

				if userName, ok := logData["user_name"].(string); !ok || userName != log.User.Name {
					t.Logf("user_name 字段值不正确: 期望 %s, 实际 %v", log.User.Name, logData["user_name"])
					return false
				}
			}

			// 如果有 namespace，验证 namespace 字段
			if log.Namespace != "" {
				if namespaceValue, ok := logData["namespace"].(string); !ok || namespaceValue != log.Namespace {
					t.Logf("namespace 字段值不正确: 期望 %s, 实际 %v", log.Namespace, logData["namespace"])
					return false
				}
			}

			// 如果有资源类型，验证 resource_type 字段
			if log.ResourceType != "" {
				if resourceType, ok := logData["resource_type"].(string); !ok || resourceType != log.ResourceType {
					t.Logf("resource_type 字段值不正确: 期望 %s, 实际 %v", log.ResourceType, logData["resource_type"])
					return false
				}
			}

			// 如果有资源名称，验证 resource_name 字段
			if log.ResourceName != "" {
				if resourceName, ok := logData["resource_name"].(string); !ok || resourceName != log.ResourceName {
					t.Logf("resource_name 字段值不正确: 期望 %s, 实际 %v", log.ResourceName, logData["resource_name"])
					return false
				}
			}

			// 如果操作失败，验证 error 字段
			if !log.Success && log.Error != "" {
				if errorValue, ok := logData["error"].(string); !ok || errorValue != log.Error {
					t.Logf("error 字段值不正确: 期望 %s, 实际 %v", log.Error, logData["error"])
					return false
				}
			}

			return true
		},
		genOperationLog(),
	))

	// 属性2: 文本格式日志必须包含关键信息
	properties.Property("文本格式日志包含关键信息", prop.ForAll(
		func(log *audit.OperationLog) bool {
			// 创建临时日志文件 - 使用当前目录避免权限问题
			tmpDir, err := os.MkdirTemp(".", "kubectl-mcp-test-*")
			if err != nil {
				t.Logf("创建临时目录失败: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)
			tmpFile := filepath.Join(tmpDir, fmt.Sprintf("test_%d.log", time.Now().UnixNano()))

			// 创建文本格式的日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "text",
				Output:   "file",
				FilePath: tmpFile,
			})
			if err != nil {
				t.Logf("创建日志器失败: %v", err)
				return false
			}

			// 记录日志
			err = logger.LogOperation(log)
			if err != nil {
				logger.Close()
				t.Logf("记录日志失败: %v", err)
				return false
			}

			// 关闭日志器
			logger.Close()

			// 读取日志文件
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Logf("读取日志文件失败: %v", err)
				return false
			}

			logContent := string(content)
			if logContent == "" {
				t.Logf("日志文件为空")
				return false
			}

			// 验证关键信息存在
			if !strings.Contains(logContent, log.Tool) {
				t.Logf("日志不包含 tool: %s", log.Tool)
				return false
			}

			if !strings.Contains(logContent, log.Context) {
				t.Logf("日志不包含 context: %s", log.Context)
				return false
			}

			if !strings.Contains(logContent, "timestamp") {
				t.Logf("日志不包含 timestamp 字段")
				return false
			}

			// 如果有用户信息，验证用户信息存在
			if log.User != nil {
				if !strings.Contains(logContent, log.User.ID) {
					t.Logf("日志不包含 user_id: %s", log.User.ID)
					return false
				}
			}

			// 如果有 namespace，验证 namespace 存在
			if log.Namespace != "" {
				if !strings.Contains(logContent, log.Namespace) {
					t.Logf("日志不包含 namespace: %s", log.Namespace)
					return false
				}
			}

			// 如果操作失败，验证错误信息存在
			if !log.Success && log.Error != "" {
				if !strings.Contains(logContent, log.Error) {
					t.Logf("日志不包含 error: %s", log.Error)
					return false
				}
			}

			return true
		},
		genOperationLog(),
	))

	// 属性3: 相同配置的日志器产生一致的格式
	properties.Property("相同配置的日志器产生一致的格式", prop.ForAll(
		func(log *audit.OperationLog, format string) bool {
			// 创建两个临时日志文件 - 使用当前目录避免权限问题
			tmpDir, err := os.MkdirTemp(".", "kubectl-mcp-test-*")
			if err != nil {
				t.Logf("创建临时目录失败: %v", err)
				return false
			}
			defer os.RemoveAll(tmpDir)
			tmpFile1 := filepath.Join(tmpDir, fmt.Sprintf("test1_%d.log", time.Now().UnixNano()))
			tmpFile2 := filepath.Join(tmpDir, fmt.Sprintf("test2_%d.log", time.Now().UnixNano()))

			// 创建第一个日志器
			logger1, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   format,
				Output:   "file",
				FilePath: tmpFile1,
			})
			if err != nil {
				t.Logf("创建第一个日志器失败: %v", err)
				return false
			}

			// 创建第二个日志器（相同配置）
			logger2, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   format,
				Output:   "file",
				FilePath: tmpFile2,
			})
			if err != nil {
				logger1.Close()
				t.Logf("创建第二个日志器失败: %v", err)
				return false
			}

			// 记录相同的日志
			err1 := logger1.LogOperation(log)
			err2 := logger2.LogOperation(log)

			logger1.Close()
			logger2.Close()

			if err1 != nil || err2 != nil {
				t.Logf("记录日志失败: err1=%v, err2=%v", err1, err2)
				return false
			}

			// 读取两个日志文件
			content1, err := os.ReadFile(tmpFile1)
			if err != nil {
				t.Logf("读取第一个日志文件失败: %v", err)
				return false
			}

			content2, err := os.ReadFile(tmpFile2)
			if err != nil {
				t.Logf("读取第二个日志文件失败: %v", err)
				return false
			}

			// 对于 JSON 格式，解析并比较关键字段
			if format == "json" {
				var log1Data, log2Data map[string]interface{}

				lines1 := strings.Split(strings.TrimSpace(string(content1)), "\n")
				lines2 := strings.Split(strings.TrimSpace(string(content2)), "\n")

				if len(lines1) == 0 || len(lines2) == 0 {
					t.Logf("日志文件没有内容行")
					return false
				}

				err1 := json.Unmarshal([]byte(lines1[0]), &log1Data)
				err2 := json.Unmarshal([]byte(lines2[0]), &log2Data)

				if err1 != nil || err2 != nil {
					t.Logf("解析 JSON 失败: err1=%v, err2=%v", err1, err2)
					return false
				}

				// 比较关键字段
				keyFields := []string{"tool", "context", "success"}
				for _, field := range keyFields {
					if log1Data[field] != log2Data[field] {
						t.Logf("字段 %s 不一致: %v vs %v", field, log1Data[field], log2Data[field])
						return false
					}
				}
			} else {
				// 对于文本格式，验证包含相同的关键信息
				content1Str := string(content1)
				content2Str := string(content2)

				if !strings.Contains(content1Str, log.Tool) || !strings.Contains(content2Str, log.Tool) {
					t.Logf("日志不包含 tool")
					return false
				}

				if !strings.Contains(content1Str, log.Context) || !strings.Contains(content2Str, log.Context) {
					t.Logf("日志不包含 context")
					return false
				}
			}

			return true
		},
		genOperationLog(),
		gen.OneConstOf("json", "text"),
	))

	properties.TestingRun(t)
}

// TestLogFormatConsistencyWithRealLogger 使用真实日志器测试格式一致性
func TestLogFormatConsistencyWithRealLogger(t *testing.T) {
	// 测试 JSON 格式
	t.Run("JSON格式一致性", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(".", "kubectl-mcp-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)
		tmpFile := filepath.Join(tmpDir, "json_test.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		// 记录多个不同的日志
		logs := []*audit.OperationLog{
			{
				Tool:    "get_pods",
				Context: "prod-cluster",
				Success: true,
				User: &audit.UserInfo{
					ID:   "user1",
					Name: "admin",
				},
			},
			{
				Tool:      "delete_pod",
				Context:   "dev-cluster",
				Namespace: "default",
				Success:   false,
				Error:     "pod not found",
			},
			{
				Tool:         "create_deployment",
				Context:      "test-cluster",
				Namespace:    "production",
				ResourceType: "Deployment",
				ResourceName: "nginx",
				Success:      true,
			},
		}

		for _, log := range logs {
			err := logger.LogOperation(log)
			require.NoError(t, err)
		}

		logger.Close()

		// 读取并验证所有日志行
		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Equal(t, len(logs), len(lines), "日志行数应该匹配")

		for i, line := range lines {
			var logData map[string]interface{}
			err := json.Unmarshal([]byte(line), &logData)
			require.NoError(t, err, "第 %d 行应该是有效的 JSON", i+1)

			// 验证标准字段存在
			require.Contains(t, logData, "timestamp", "第 %d 行应包含 timestamp", i+1)
			require.Contains(t, logData, "tool", "第 %d 行应包含 tool", i+1)
			require.Contains(t, logData, "context", "第 %d 行应包含 context", i+1)
			require.Contains(t, logData, "success", "第 %d 行应包含 success", i+1)
			require.Contains(t, logData, "duration", "第 %d 行应包含 duration", i+1)
		}
	})

	// 测试文本格式
	t.Run("文本格式一致性", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(".", "kubectl-mcp-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)
		tmpFile := filepath.Join(tmpDir, "text_test.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "text",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		// 记录多个不同的日志
		logs := []*audit.OperationLog{
			{
				Tool:    "get_nodes",
				Context: "prod-cluster",
				Success: true,
			},
			{
				Tool:    "list_namespaces",
				Context: "dev-cluster",
				Success: true,
			},
		}

		for _, log := range logs {
			err := logger.LogOperation(log)
			require.NoError(t, err)
		}

		logger.Close()

		// 读取并验证所有日志行
		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.Equal(t, len(logs), len(lines), "日志行数应该匹配")

		for i, line := range lines {
			// 验证关键信息存在
			require.Contains(t, line, "timestamp", "第 %d 行应包含 timestamp", i+1)
			require.Contains(t, line, logs[i].Tool, "第 %d 行应包含 tool", i+1)
			require.Contains(t, line, logs[i].Context, "第 %d 行应包含 context", i+1)
		}
	})
}
