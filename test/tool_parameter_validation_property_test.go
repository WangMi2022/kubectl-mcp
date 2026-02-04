package test

import (
	"context"
	"testing"

	"kubectl-mcp/internal/audit"
	"kubectl-mcp/internal/k8s"
	"kubectl-mcp/internal/mcp"
	"kubectl-mcp/internal/tools"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/require"
)

// TestProperty_ToolParameterValidation 测试工具参数验证属性
// Feature: kubectl-mcp-server, Property 6: 工具参数验证
// Validates: Requirements 16.2
//
// 属性：对于任何工具调用，系统必须根据工具的 inputSchema 验证参数的类型和必填性，
// 验证失败必须拒绝执行
func TestProperty_ToolParameterValidation(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	// 创建测试环境
	handler, toolRegistry := setupTestEnvironment(t)

	// 属性1: 缺少必填参数必须被拒绝
	properties.Property("缺少必填参数必须被拒绝", prop.ForAll(
		func(toolName string, missingParam string) bool {
			tool, exists := toolRegistry.GetTool(toolName)
			if !exists || tool.InputSchema == nil || len(tool.InputSchema.Required) == 0 {
				return true // 跳过没有必填参数的工具
			}

			// 检查 missingParam 是否是必填参数
			isRequired := false
			for _, required := range tool.InputSchema.Required {
				if required == missingParam {
					isRequired = true
					break
				}
			}

			if !isRequired {
				return true // 跳过非必填参数
			}

			// 构造缺少必填参数的请求
			args := make(map[string]interface{})
			for _, required := range tool.InputSchema.Required {
				if required != missingParam {
					// 为其他必填参数提供有效值
					if schema, ok := tool.InputSchema.Properties[required]; ok {
						args[required] = generateValidValue(schema)
					}
				}
			}

			req := &mcp.ToolCallRequest{
				Tool:      toolName,
				Arguments: args,
			}

			// 验证请求应该失败
			err := handler.ValidateRequest(req)
			if err == nil {
				t.Logf("期望验证失败（缺少必填参数 %s），但成功了", missingParam)
				return false
			}

			// 验证错误类型
			validationErr, ok := err.(*mcp.ValidationError)
			if !ok {
				t.Logf("期望返回 ValidationError，实际返回 %T", err)
				return false
			}

			// 验证错误码
			if validationErr.Code != mcp.ErrCodeInvalidArguments {
				t.Logf("期望错误码为 %s，实际为 %s", mcp.ErrCodeInvalidArguments, validationErr.Code)
				return false
			}

			return true
		},
		genRegisteredToolName(toolRegistry),
		gen.Identifier(),
	))

	// 属性2: 错误的参数类型必须被拒绝
	properties.Property("错误的参数类型必须被拒绝", prop.ForAll(
		func(toolName string, paramName string, wrongValue interface{}) bool {
			tool, exists := toolRegistry.GetTool(toolName)
			if !exists || tool.InputSchema == nil {
				return true
			}

			schema, exists := tool.InputSchema.Properties[paramName]
			if !exists {
				return true // 跳过不存在的参数
			}

			// 检查 wrongValue 是否与 schema 类型不匹配
			if isValueMatchingType(wrongValue, schema.Type) {
				return true // 跳过类型匹配的情况
			}

			// 构造包含错误类型参数的请求
			args := make(map[string]interface{})

			// 为所有必填参数提供有效值
			for _, required := range tool.InputSchema.Required {
				if required == paramName {
					args[required] = wrongValue // 使用错误类型的值
				} else if reqSchema, ok := tool.InputSchema.Properties[required]; ok {
					args[required] = generateValidValue(reqSchema)
				}
			}

			// 如果 paramName 不是必填参数，也添加它
			if _, exists := args[paramName]; !exists {
				args[paramName] = wrongValue
			}

			req := &mcp.ToolCallRequest{
				Tool:      toolName,
				Arguments: args,
			}

			// 验证请求应该失败
			err := handler.ValidateRequest(req)
			if err == nil {
				// 执行工具也应该失败
				ctx := context.Background()
				_, execErr := toolRegistry.ExecuteTool(ctx, toolName, args, handler.GetK8SManager())
				if execErr == nil {
					t.Logf("期望执行失败（参数 %s 类型错误），但成功了", paramName)
					return false
				}
			}

			return true
		},
		genRegisteredToolName(toolRegistry),
		gen.Identifier(),
		gen.OneGenOf(
			gen.Int(),
			gen.Bool(),
			gen.SliceOf(gen.Int()),
			gen.MapOf(gen.Identifier(), gen.Int()),
		),
	))

	// 属性3: 有效的参数必须被接受
	properties.Property("有效的参数必须被接受", prop.ForAll(
		func(toolName string) bool {
			tool, exists := toolRegistry.GetTool(toolName)
			if !exists || tool.InputSchema == nil {
				return true
			}

			// 构造包含所有必填参数的有效请求
			args := make(map[string]interface{})
			for _, required := range tool.InputSchema.Required {
				if schema, ok := tool.InputSchema.Properties[required]; ok {
					args[required] = generateValidValue(schema)
				}
			}

			req := &mcp.ToolCallRequest{
				Tool:      toolName,
				Arguments: args,
			}

			// 验证请求应该成功
			err := handler.ValidateRequest(req)
			if err != nil {
				t.Logf("期望验证成功，但失败了: %v", err)
				return false
			}

			return true
		},
		genRegisteredToolName(toolRegistry),
	))

	// 属性4: 枚举值验证
	properties.Property("枚举值必须在允许范围内", prop.ForAll(
		func(toolName string, paramName string, invalidEnumValue string) bool {
			tool, exists := toolRegistry.GetTool(toolName)
			if !exists || tool.InputSchema == nil {
				return true
			}

			schema, exists := tool.InputSchema.Properties[paramName]
			if !exists || len(schema.Enum) == 0 {
				return true // 跳过没有枚举限制的参数
			}

			// 检查 invalidEnumValue 是否在枚举范围内
			isValid := false
			for _, enumVal := range schema.Enum {
				if enumVal == invalidEnumValue {
					isValid = true
					break
				}
			}

			if isValid {
				return true // 跳过有效的枚举值
			}

			// 构造包含无效枚举值的请求
			args := make(map[string]interface{})
			for _, required := range tool.InputSchema.Required {
				if required == paramName {
					args[required] = invalidEnumValue
				} else if reqSchema, ok := tool.InputSchema.Properties[required]; ok {
					args[required] = generateValidValue(reqSchema)
				}
			}

			// 如果 paramName 不是必填参数，也添加它
			if _, exists := args[paramName]; !exists {
				args[paramName] = invalidEnumValue
			}

			// 执行工具应该失败
			ctx := context.Background()
			_, err := toolRegistry.ExecuteTool(ctx, toolName, args, handler.GetK8SManager())
			if err == nil {
				t.Logf("期望执行失败（枚举值 %s 无效），但成功了", invalidEnumValue)
				return false
			}

			return true
		},
		genRegisteredToolName(toolRegistry),
		gen.Identifier(),
		gen.Identifier(),
	))

	properties.TestingRun(t)
}

// TestToolParameterValidation_SpecificCases 测试特定的参数验证场景
func TestToolParameterValidation_SpecificCases(t *testing.T) {
	handler, toolRegistry := setupTestEnvironment(t)

	t.Run("字符串类型参数验证", func(t *testing.T) {
		// 注册一个需要字符串参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_string_param",
			Description: "测试字符串参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"name"},
				Properties: map[string]*tools.ParameterSchema{
					"name": {
						Type:        "string",
						Description: "名称参数",
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试有效的字符串参数
		req := &mcp.ToolCallRequest{
			Tool: "test_string_param",
			Arguments: map[string]interface{}{
				"name": "valid-name",
			},
		}
		err = handler.ValidateRequest(req)
		require.NoError(t, err, "有效的字符串参数应该通过验证")

		// 测试无效的类型（整数）
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_string_param", map[string]interface{}{
			"name": 123,
		}, handler.GetK8SManager())
		require.Error(t, err, "整数类型应该被拒绝")
		require.Contains(t, err.Error(), "应为字符串类型")
	})

	t.Run("整数类型参数验证", func(t *testing.T) {
		// 注册一个需要整数参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_integer_param",
			Description: "测试整数参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"count"},
				Properties: map[string]*tools.ParameterSchema{
					"count": {
						Type:        "integer",
						Description: "数量参数",
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试有效的整数参数
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_integer_param", map[string]interface{}{
			"count": 10,
		}, handler.GetK8SManager())
		require.NoError(t, err, "有效的整数参数应该通过验证")

		// 测试无效的类型（字符串）
		_, err = toolRegistry.ExecuteTool(ctx, "test_integer_param", map[string]interface{}{
			"count": "not-a-number",
		}, handler.GetK8SManager())
		require.Error(t, err, "字符串类型应该被拒绝")
		require.Contains(t, err.Error(), "应为整数类型")
	})

	t.Run("布尔类型参数验证", func(t *testing.T) {
		// 注册一个需要布尔参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_boolean_param",
			Description: "测试布尔参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"enabled"},
				Properties: map[string]*tools.ParameterSchema{
					"enabled": {
						Type:        "boolean",
						Description: "是否启用",
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试有效的布尔参数
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_boolean_param", map[string]interface{}{
			"enabled": true,
		}, handler.GetK8SManager())
		require.NoError(t, err, "有效的布尔参数应该通过验证")

		// 测试无效的类型（字符串）
		_, err = toolRegistry.ExecuteTool(ctx, "test_boolean_param", map[string]interface{}{
			"enabled": "yes",
		}, handler.GetK8SManager())
		require.Error(t, err, "字符串类型应该被拒绝")
		require.Contains(t, err.Error(), "应为布尔类型")
	})

	t.Run("数组类型参数验证", func(t *testing.T) {
		// 注册一个需要数组参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_array_param",
			Description: "测试数组参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"labels"},
				Properties: map[string]*tools.ParameterSchema{
					"labels": {
						Type:        "array",
						Description: "标签列表",
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试有效的数组参数
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_array_param", map[string]interface{}{
			"labels": []interface{}{"app=nginx", "env=prod"},
		}, handler.GetK8SManager())
		require.NoError(t, err, "有效的数组参数应该通过验证")

		// 测试无效的类型（字符串）
		_, err = toolRegistry.ExecuteTool(ctx, "test_array_param", map[string]interface{}{
			"labels": "not-an-array",
		}, handler.GetK8SManager())
		require.Error(t, err, "字符串类型应该被拒绝")
		require.Contains(t, err.Error(), "应为数组类型")
	})

	t.Run("对象类型参数验证", func(t *testing.T) {
		// 注册一个需要对象参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_object_param",
			Description: "测试对象参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"metadata"},
				Properties: map[string]*tools.ParameterSchema{
					"metadata": {
						Type:        "object",
						Description: "元数据对象",
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试有效的对象参数
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_object_param", map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": "test",
				"labels": map[string]interface{}{
					"app": "nginx",
				},
			},
		}, handler.GetK8SManager())
		require.NoError(t, err, "有效的对象参数应该通过验证")

		// 测试无效的类型（字符串）
		_, err = toolRegistry.ExecuteTool(ctx, "test_object_param", map[string]interface{}{
			"metadata": "not-an-object",
		}, handler.GetK8SManager())
		require.Error(t, err, "字符串类型应该被拒绝")
		require.Contains(t, err.Error(), "应为对象类型")
	})

	t.Run("枚举值参数验证", func(t *testing.T) {
		// 注册一个需要枚举参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_enum_param",
			Description: "测试枚举参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"level"},
				Properties: map[string]*tools.ParameterSchema{
					"level": {
						Type:        "string",
						Description: "日志级别",
						Enum:        []interface{}{"debug", "info", "warn", "error"},
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试有效的枚举值
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_enum_param", map[string]interface{}{
			"level": "info",
		}, handler.GetK8SManager())
		require.NoError(t, err, "有效的枚举值应该通过验证")

		// 测试无效的枚举值
		_, err = toolRegistry.ExecuteTool(ctx, "test_enum_param", map[string]interface{}{
			"level": "invalid",
		}, handler.GetK8SManager())
		require.Error(t, err, "无效的枚举值应该被拒绝")
		require.Contains(t, err.Error(), "不在允许的枚举范围内")
	})

	t.Run("缺少必填参数验证", func(t *testing.T) {
		// 注册一个有多个必填参数的测试工具
		testTool := &tools.Tool{
			Name:        "test_required_params",
			Description: "测试必填参数",
			Category:    tools.CategoryQuery,
			Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
				return map[string]string{"result": "success"}, nil
			},
			InputSchema: &tools.InputSchema{
				Type:     "object",
				Required: []string{"name", "namespace"},
				Properties: map[string]*tools.ParameterSchema{
					"name": {
						Type:        "string",
						Description: "资源名称",
					},
					"namespace": {
						Type:        "string",
						Description: "命名空间",
					},
					"labels": {
						Type:        "string",
						Description: "标签（可选）",
					},
				},
			},
		}
		err := toolRegistry.RegisterTool(testTool)
		require.NoError(t, err)

		// 测试缺少第一个必填参数
		ctx := context.Background()
		_, err = toolRegistry.ExecuteTool(ctx, "test_required_params", map[string]interface{}{
			"namespace": "default",
		}, handler.GetK8SManager())
		require.Error(t, err, "缺少必填参数 name 应该被拒绝")
		require.Contains(t, err.Error(), "缺少必填参数")

		// 测试缺少第二个必填参数
		_, err = toolRegistry.ExecuteTool(ctx, "test_required_params", map[string]interface{}{
			"name": "test-pod",
		}, handler.GetK8SManager())
		require.Error(t, err, "缺少必填参数 namespace 应该被拒绝")
		require.Contains(t, err.Error(), "缺少必填参数")

		// 测试提供所有必填参数
		_, err = toolRegistry.ExecuteTool(ctx, "test_required_params", map[string]interface{}{
			"name":      "test-pod",
			"namespace": "default",
		}, handler.GetK8SManager())
		require.NoError(t, err, "提供所有必填参数应该通过验证")

		// 测试提供所有必填参数和可选参数
		_, err = toolRegistry.ExecuteTool(ctx, "test_required_params", map[string]interface{}{
			"name":      "test-pod",
			"namespace": "default",
			"labels":    "app=nginx",
		}, handler.GetK8SManager())
		require.NoError(t, err, "提供所有必填参数和可选参数应该通过验证")
	})
}

// ========== 辅助函数 ==========

// setupTestEnvironment 创建测试环境
func setupTestEnvironment(t *testing.T) (*mcp.MCPHandler, *tools.ToolRegistry) {
	// 创建临时 kubeconfig
	kubeconfigPath := createTempKubeconfig(t)

	// 创建 K8S 客户端管理器
	k8sManager, err := k8s.NewK8SClientManager(kubeconfigPath)
	require.NoError(t, err)

	// 创建工具注册表
	toolRegistry := tools.NewToolRegistry()

	// 注册测试工具
	registerTestTools(t, toolRegistry)

	// 创建审计日志器
	auditLogger, err := audit.NewAuditLogger(&audit.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	require.NoError(t, err)

	// 创建 MCP 处理器
	handler, err := mcp.NewMCPHandler(
		toolRegistry,
		k8sManager,
		auditLogger,
		&mcp.MCPHandlerConfig{
			Version: "1.0.0-test",
		},
	)
	require.NoError(t, err)

	return handler, toolRegistry
}

// registerTestTools 注册测试工具
func registerTestTools(t *testing.T, registry *tools.ToolRegistry) {
	// 工具1: 需要字符串和整数参数
	tool1 := &tools.Tool{
		Name:        "test_tool_1",
		Description: "测试工具1",
		Category:    tools.CategoryQuery,
		Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
			return map[string]string{"result": "success"}, nil
		},
		InputSchema: &tools.InputSchema{
			Type:     "object",
			Required: []string{"name", "count"},
			Properties: map[string]*tools.ParameterSchema{
				"name": {
					Type:        "string",
					Description: "名称",
				},
				"count": {
					Type:        "integer",
					Description: "数量",
				},
				"enabled": {
					Type:        "boolean",
					Description: "是否启用（可选）",
				},
			},
		},
	}
	require.NoError(t, registry.RegisterTool(tool1))

	// 工具2: 需要枚举参数
	tool2 := &tools.Tool{
		Name:        "test_tool_2",
		Description: "测试工具2",
		Category:    tools.CategoryQuery,
		Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
			return map[string]string{"result": "success"}, nil
		},
		InputSchema: &tools.InputSchema{
			Type:     "object",
			Required: []string{"action"},
			Properties: map[string]*tools.ParameterSchema{
				"action": {
					Type:        "string",
					Description: "操作类型",
					Enum:        []interface{}{"create", "update", "delete"},
				},
			},
		},
	}
	require.NoError(t, registry.RegisterTool(tool2))

	// 工具3: 需要数组和对象参数
	tool3 := &tools.Tool{
		Name:        "test_tool_3",
		Description: "测试工具3",
		Category:    tools.CategoryQuery,
		Handler: func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
			return map[string]string{"result": "success"}, nil
		},
		InputSchema: &tools.InputSchema{
			Type:     "object",
			Required: []string{"labels", "metadata"},
			Properties: map[string]*tools.ParameterSchema{
				"labels": {
					Type:        "array",
					Description: "标签列表",
				},
				"metadata": {
					Type:        "object",
					Description: "元数据",
				},
			},
		},
	}
	require.NoError(t, registry.RegisterTool(tool3))
}

// genRegisteredToolName 生成已注册的工具名称
func genRegisteredToolName(registry *tools.ToolRegistry) gopter.Gen {
	toolNames := registry.GetToolNames()
	if len(toolNames) == 0 {
		return gen.Const("test_tool_1")
	}

	// 将字符串切片转换为 interface{} 切片
	interfaces := make([]interface{}, len(toolNames))
	for i, name := range toolNames {
		interfaces[i] = name
	}

	return gen.OneConstOf(interfaces...)
}

// generateValidValue 根据 schema 生成有效值
func generateValidValue(schema *tools.ParameterSchema) interface{} {
	switch schema.Type {
	case "string":
		if len(schema.Enum) > 0 {
			return schema.Enum[0]
		}
		return "test-value"
	case "integer":
		return 10
	case "boolean":
		return true
	case "array":
		return []interface{}{"item1", "item2"}
	case "object":
		return map[string]interface{}{"key": "value"}
	default:
		return "default-value"
	}
}

// isValueMatchingType 检查值是否匹配类型
func isValueMatchingType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		switch value.(type) {
		case int, int32, int64, float64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return false
	}
}
