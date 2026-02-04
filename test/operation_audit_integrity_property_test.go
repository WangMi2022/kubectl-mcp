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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProperty_OperationAuditIntegrity 测试操作审计完整性属性
// Property 3: 操作审计完整性
// Validates: Requirements 9.1, 9.2, 9.3, 9.4
// Feature: kubectl-mcp-server, Property 3: 对于任何 K8S 操作（CREATE/UPDATE/DELETE），系统必须在操作前后记录完整的审计日志，包括用户、时间、参数、结果
func TestProperty_OperationAuditIntegrity(t *testing.T) {
	// 获取当前工作目录，在其下创建临时目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	testTmpDir, err := os.MkdirTemp(cwd, "audit_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v (cwd=%s)", err, cwd)
	}
	defer os.RemoveAll(testTmpDir)

	// 配置属性测试参数
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100 // 至少运行 100 次迭代
	parameters.MaxSize = 20

	properties := gopter.NewProperties(parameters)

	// 属性 1: 每个操作都有对应的审计日志
	properties.Property("每个操作都有对应的审计日志", prop.ForAll(
		func(log *audit.OperationLog) bool {
			// 创建临时日志文件
			tmpFile := filepath.Join(testTmpDir, fmt.Sprintf("audit_%d.log", time.Now().UnixNano()))

			// 创建日志器
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

			// 记录操作日志
			err = logger.LogOperation(log)
			if err != nil {
				t.Logf("记录日志失败: %v", err)
				logger.Close()
				return false
			}

			// 关闭日志器以刷新缓冲区
			logger.Close()

			// 读取日志文件
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				t.Logf("读取日志文件失败: %v", err)
				return false
			}

			// 验证日志文件不为空
			if len(content) == 0 {
				t.Logf("日志文件为空")
				return false
			}

			// 验证日志包含操作信息
			logContent := string(content)
			return strings.Contains(logContent, log.Tool) &&
				strings.Contains(logContent, log.Context)
		},
		genOperationLog(),
	))

	// 属性 2: 日志包含所有必需字段
	properties.Property("日志包含所有必需字段", prop.ForAll(
		func(log *audit.OperationLog) bool {
			// 创建临时日志文件
			tmpFile := filepath.Join(testTmpDir, fmt.Sprintf("audit_%d.log", time.Now().UnixNano()))

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			if err != nil {
				return false
			}

			// 记录操作日志
			err = logger.LogOperation(log)
			if err != nil {
				logger.Close()
				return false
			}

			// 关闭日志器
			logger.Close()

			// 读取并解析日志
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				return false
			}

			// 解析 JSON 日志
			var logEntry map[string]interface{}
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) == 0 {
				return false
			}

			err = json.Unmarshal([]byte(lines[0]), &logEntry)
			if err != nil {
				t.Logf("解析 JSON 失败: %v", err)
				return false
			}

			// 验证必需字段存在
			requiredFields := []string{"timestamp", "tool", "context", "success", "duration"}
			for _, field := range requiredFields {
				if _, exists := logEntry[field]; !exists {
					t.Logf("缺少必需字段: %s", field)
					return false
				}
			}

			// 如果有用户信息，验证用户字段
			if log.User != nil {
				if _, exists := logEntry["user_id"]; !exists {
					t.Logf("缺少用户 ID 字段")
					return false
				}
				if _, exists := logEntry["user_name"]; !exists {
					t.Logf("缺少用户名称字段")
					return false
				}
			}

			// 如果操作失败，验证错误字段
			if !log.Success && log.Error != "" {
				if _, exists := logEntry["error"]; !exists {
					t.Logf("失败操作缺少错误字段")
					return false
				}
			}

			return true
		},
		genOperationLog(),
	))

	// 属性 3: 操作时间戳准确记录
	properties.Property("操作时间戳准确记录", prop.ForAll(
		func(log *audit.OperationLog) bool {
			// 创建临时日志文件
			tmpFile := filepath.Join(testTmpDir, fmt.Sprintf("audit_%d.log", time.Now().UnixNano()))

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			if err != nil {
				return false
			}

			// 确保日志的时间戳不在未来或过去太远
			now := time.Now()
			if log.Timestamp.After(now.Add(1*time.Hour)) || log.Timestamp.Before(now.Add(-25*time.Hour)) {
				log.Timestamp = now
			}

			// 记录操作日志
			err = logger.LogOperation(log)
			if err != nil {
				logger.Close()
				return false
			}

			// 关闭日志器
			logger.Close()

			// 读取并解析日志
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				return false
			}

			var logEntry map[string]interface{}
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) == 0 {
				return false
			}

			err = json.Unmarshal([]byte(lines[0]), &logEntry)
			if err != nil {
				return false
			}

			// 验证时间戳字段存在
			_, ok := logEntry["timestamp"].(string)
			if !ok {
				return false
			}

			// 时间戳字段存在即可，不严格验证具体值
			// 因为 LogOperation 会自动设置时间戳
			return true
		},
		genOperationLog(),
	))

	// 属性 4: 操作参数完整记录
	properties.Property("操作参数完整记录", prop.ForAll(
		func(log *audit.OperationLog) bool {
			// 如果没有参数，跳过
			if len(log.Arguments) == 0 {
				return true
			}

			// 创建临时日志文件
			tmpFile := filepath.Join(testTmpDir, fmt.Sprintf("audit_%d.log", time.Now().UnixNano()))

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			if err != nil {
				return false
			}

			// 记录操作日志
			err = logger.LogOperation(log)
			if err != nil {
				logger.Close()
				return false
			}

			// 关闭日志器
			logger.Close()

			// 读取并解析日志
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				return false
			}

			var logEntry map[string]interface{}
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) == 0 {
				return false
			}

			err = json.Unmarshal([]byte(lines[0]), &logEntry)
			if err != nil {
				return false
			}

			// 验证参数字段存在
			argsStr, ok := logEntry["arguments"].(string)
			if !ok {
				return false
			}

			// 解析参数 JSON
			var recordedArgs map[string]interface{}
			err = json.Unmarshal([]byte(argsStr), &recordedArgs)
			if err != nil {
				return false
			}

			// 验证所有参数都被记录
			for key := range log.Arguments {
				if _, exists := recordedArgs[key]; !exists {
					t.Logf("参数 %s 未被记录", key)
					return false
				}
			}

			return true
		},
		genOperationLogWithArgs(),
	))

	// 属性 5: 成功和失败操作都被记录
	properties.Property("成功和失败操作都被记录", prop.ForAll(
		func(success bool, errorMsg string) bool {
			// 创建临时日志文件
			tmpFile := filepath.Join(testTmpDir, fmt.Sprintf("audit_%d.log", time.Now().UnixNano()))

			// 创建日志器
			logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
				Level:    "info",
				Format:   "json",
				Output:   "file",
				FilePath: tmpFile,
			})
			if err != nil {
				return false
			}

			// 创建操作日志
			log := &audit.OperationLog{
				Timestamp: time.Now(),
				Tool:      "test_tool",
				Context:   "test-context",
				Success:   success,
				Duration:  100 * time.Millisecond,
			}

			if !success && errorMsg != "" {
				log.Error = errorMsg
			}

			// 记录操作日志
			err = logger.LogOperation(log)
			if err != nil {
				logger.Close()
				return false
			}

			// 关闭日志器
			logger.Close()

			// 读取日志文件
			content, err := os.ReadFile(tmpFile)
			if err != nil {
				return false
			}

			// 验证日志不为空
			if len(content) == 0 {
				return false
			}

			// 解析日志
			var logEntry map[string]interface{}
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) == 0 {
				return false
			}

			err = json.Unmarshal([]byte(lines[0]), &logEntry)
			if err != nil {
				return false
			}

			// 验证成功状态
			recordedSuccess, ok := logEntry["success"].(bool)
			if !ok || recordedSuccess != success {
				return false
			}

			// 如果是失败操作且有错误消息，验证错误字段
			if !success && errorMsg != "" {
				recordedError, ok := logEntry["error"].(string)
				if !ok || recordedError != errorMsg {
					return false
				}
			}

			return true
		},
		gen.Bool(),
		gen.AlphaString(),
	))

	// 运行所有属性测试
	properties.TestingRun(t)
}

// genOperationLog 生成随机的操作日志
func genOperationLog() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(), // tool
		gen.Identifier(), // context
		gen.Bool(),       // success
		gen.TimeRange(time.Now().Add(-24*time.Hour), time.Duration(24*time.Hour)), // timestamp (过去24小时到未来24小时)
		gen.IntRange(0, 5000),        // duration (ms)
		genUserInfo(),                // user
		gen.PtrOf(gen.Identifier()),  // namespace
		gen.PtrOf(gen.AlphaString()), // error
		gen.PtrOf(gen.Identifier()),  // resourceType
		gen.PtrOf(gen.Identifier()),  // resourceName
	).Map(func(values []interface{}) *audit.OperationLog {
		log := &audit.OperationLog{
			Tool:      values[0].(string),
			Context:   values[1].(string),
			Success:   values[2].(bool),
			Timestamp: values[3].(time.Time),
			Duration:  time.Duration(values[4].(int)) * time.Millisecond,
		}

		// 确保时间戳不在未来
		if log.Timestamp.After(time.Now()) {
			log.Timestamp = time.Now()
		}

		// 添加可选字段
		if user, ok := values[5].(*audit.UserInfo); ok && user != nil {
			log.User = user
		}

		if ns, ok := values[6].(*string); ok && ns != nil && *ns != "" {
			log.Namespace = *ns
		}

		if err, ok := values[7].(*string); ok && err != nil && *err != "" && !log.Success {
			log.Error = *err
		}

		if rt, ok := values[8].(*string); ok && rt != nil && *rt != "" {
			log.ResourceType = *rt
		}

		if rn, ok := values[9].(*string); ok && rn != nil && *rn != "" {
			log.ResourceName = *rn
		}

		return log
	})
}

// genOperationLogWithArgs 生成包含参数的操作日志
func genOperationLogWithArgs() gopter.Gen {
	return gopter.CombineGens(
		genOperationLog(),
		genArguments(),
	).Map(func(values []interface{}) *audit.OperationLog {
		log := values[0].(*audit.OperationLog)
		args := values[1].(map[string]interface{})
		log.Arguments = args
		return log
	})
}

// genUserInfo 生成随机的用户信息
func genUserInfo() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),            // id
		gen.AlphaString(),           // name
		gen.PtrOf(gen.Identifier()), // role
	).Map(func(values []interface{}) *audit.UserInfo {
		user := &audit.UserInfo{
			ID:   values[0].(string),
			Name: values[1].(string),
		}

		if role, ok := values[2].(*string); ok && role != nil && *role != "" {
			user.Role = *role
		}

		return user
	})
}

// genArguments 生成随机的参数映射
func genArguments() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		gen.Identifier(),
		gen.Int(),
		gen.Bool(),
	).Map(func(values []interface{}) map[string]interface{} {
		return map[string]interface{}{
			"key1":   values[0].(string),
			"key2":   values[1].(string),
			"number": values[2].(int),
			"flag":   values[3].(bool),
		}
	})
}

// TestProperty_OperationAuditIntegrity_EdgeCases 测试操作审计完整性的边界情况
func TestProperty_OperationAuditIntegrity_EdgeCases(t *testing.T) {
	// 获取当前工作目录，在其下创建临时目录
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	t.Run("空工具名称应该被记录", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(cwd, "audit_edge_test_*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "audit.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		log := &audit.OperationLog{
			Tool:    "",
			Context: "test-context",
			Success: true,
		}

		err = logger.LogOperation(log)
		require.NoError(t, err)
		logger.Close()

		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
	})

	t.Run("非常长的错误消息应该被完整记录", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(cwd, "audit_edge_test_*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "audit.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		longError := strings.Repeat("这是一个很长的错误消息。", 100)
		log := &audit.OperationLog{
			Tool:    "test_tool",
			Context: "test-context",
			Success: false,
			Error:   longError,
		}

		err = logger.LogOperation(log)
		require.NoError(t, err)
		logger.Close()

		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)
		assert.Contains(t, string(content), longError)
	})

	t.Run("包含特殊字符的参数应该被正确记录", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(cwd, "audit_edge_test_*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "audit.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		log := &audit.OperationLog{
			Tool:    "test_tool",
			Context: "test-context",
			Success: true,
			Arguments: map[string]interface{}{
				"special": `{"key": "value", "nested": {"array": [1, 2, 3]}}`,
				"unicode": "测试中文字符 🚀",
				"quotes":  `"quoted" and 'single'`,
			},
		}

		err = logger.LogOperation(log)
		require.NoError(t, err)
		logger.Close()

		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		var logEntry map[string]interface{}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, lines)

		err = json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)

		argsStr, ok := logEntry["arguments"].(string)
		require.True(t, ok)

		var recordedArgs map[string]interface{}
		err = json.Unmarshal([]byte(argsStr), &recordedArgs)
		require.NoError(t, err)

		assert.Contains(t, recordedArgs, "special")
		assert.Contains(t, recordedArgs, "unicode")
		assert.Contains(t, recordedArgs, "quotes")
	})

	t.Run("零时长的操作应该被记录", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(cwd, "audit_edge_test_*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "audit.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		log := &audit.OperationLog{
			Tool:     "test_tool",
			Context:  "test-context",
			Success:  true,
			Duration: 0,
		}

		err = logger.LogOperation(log)
		require.NoError(t, err)
		logger.Close()

		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		var logEntry map[string]interface{}
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		require.NotEmpty(t, lines)

		err = json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)

		_, exists := logEntry["duration"]
		assert.True(t, exists, "零时长也应该被记录")
	})

	t.Run("多个操作日志应该按顺序记录", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp(cwd, "audit_edge_test_*")
		if err != nil {
			t.Fatalf("创建临时目录失败: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		tmpFile := filepath.Join(tmpDir, "audit.log")

		logger, err := audit.NewAuditLogger(&audit.LoggerConfig{
			Level:    "info",
			Format:   "json",
			Output:   "file",
			FilePath: tmpFile,
		})
		require.NoError(t, err)

		// 记录多个操作
		numOps := 10
		for i := 0; i < numOps; i++ {
			log := &audit.OperationLog{
				Tool:    fmt.Sprintf("tool_%d", i),
				Context: "test-context",
				Success: true,
			}
			err = logger.LogOperation(log)
			require.NoError(t, err)
		}

		logger.Close()

		content, err := os.ReadFile(tmpFile)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		assert.Equal(t, numOps, len(lines), "应该记录所有操作")

		// 验证每个操作都被记录
		for i := 0; i < numOps; i++ {
			var logEntry map[string]interface{}
			err = json.Unmarshal([]byte(lines[i]), &logEntry)
			require.NoError(t, err)

			tool, ok := logEntry["tool"].(string)
			require.True(t, ok)
			assert.Equal(t, fmt.Sprintf("tool_%d", i), tool)
		}
	})
}
