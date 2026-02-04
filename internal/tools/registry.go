package tools

import (
	"context"
	"fmt"
	"sync"

	"kubectl-mcp/internal/k8s"
)

// ToolCategory 工具分类
type ToolCategory string

const (
	CategoryQuery  ToolCategory = "query"  // 查询类工具
	CategoryCreate ToolCategory = "create" // 创建类工具
	CategoryUpdate ToolCategory = "update" // 修改类工具
	CategoryDelete ToolCategory = "delete" // 删除类工具
)

// ToolHandler 工具处理函数类型
type ToolHandler func(ctx context.Context, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error)

// ParameterSchema 参数 Schema 定义
type ParameterSchema struct {
	Type        string                      `json:"type"`                  // 参数类型: string, integer, boolean, array, object
	Description string                      `json:"description,omitempty"` // 参数描述
	Required    bool                        `json:"required,omitempty"`    // 是否必填
	Default     interface{}                 `json:"default,omitempty"`     // 默认值
	Enum        []interface{}               `json:"enum,omitempty"`        // 枚举值
	Items       *ParameterSchema            `json:"items,omitempty"`       // 数组元素类型
	Properties  map[string]*ParameterSchema `json:"properties,omitempty"`  // 对象属性
}

// InputSchema 工具输入参数 Schema
type InputSchema struct {
	Type       string                      `json:"type"`               // 固定为 "object"
	Properties map[string]*ParameterSchema `json:"properties"`         // 参数定义
	Required   []string                    `json:"required,omitempty"` // 必填参数列表
}

// Tool 工具定义
type Tool struct {
	Name                 string       `json:"name"`                 // 工具名称
	Description          string       `json:"description"`          // 工具描述
	Category             ToolCategory `json:"category"`             // 工具分类
	RequiresConfirmation bool         `json:"requiresConfirmation"` // 是否需要确认
	InputSchema          *InputSchema `json:"inputSchema"`          // 输入参数 Schema
	Handler              ToolHandler  `json:"-"`                    // 工具处理函数（不序列化）
	Example              string       `json:"example,omitempty"`    // 使用示例
	RiskLevel            string       `json:"riskLevel,omitempty"`  // 风险等级: low, medium, high
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewToolRegistry 创建新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

// RegisterTool 注册工具
// 参数:
//   - tool: 工具定义
//
// 返回:
//   - error: 错误信息
func (r *ToolRegistry) RegisterTool(tool *Tool) error {
	if tool == nil {
		return fmt.Errorf("工具定义不能为空")
	}

	if tool.Name == "" {
		return fmt.Errorf("工具名称不能为空")
	}

	if tool.Handler == nil {
		return fmt.Errorf("工具 '%s' 的处理函数不能为空", tool.Name)
	}

	if tool.InputSchema == nil {
		return fmt.Errorf("工具 '%s' 的输入 Schema 不能为空", tool.Name)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// 检查工具是否已存在
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("工具 '%s' 已存在", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

// GetTool 获取指定工具
// 参数:
//   - name: 工具名称
//
// 返回:
//   - *Tool: 工具定义
//   - bool: 是否存在
func (r *ToolRegistry) GetTool(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools 获取所有工具
// 返回:
//   - []*Tool: 工具列表
func (r *ToolRegistry) GetAllTools() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolsByCategory 按分类获取工具
// 参数:
//   - category: 工具分类
//
// 返回:
//   - []*Tool: 工具列表
func (r *ToolRegistry) GetToolsByCategory(category ToolCategory) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0)
	for _, tool := range r.tools {
		if tool.Category == category {
			tools = append(tools, tool)
		}
	}
	return tools
}

// ExecuteTool 执行工具
// 参数:
//   - ctx: 上下文
//   - toolName: 工具名称
//   - args: 工具参数
//   - k8sClient: K8S 客户端管理器
//
// 返回:
//   - interface{}: 执行结果
//   - error: 错误信息
func (r *ToolRegistry) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}, k8sClient *k8s.K8SClientManager) (interface{}, error) {
	tool, exists := r.GetTool(toolName)
	if !exists {
		return nil, fmt.Errorf("工具 '%s' 不存在", toolName)
	}

	// 验证参数
	if err := r.validateArgs(tool, args); err != nil {
		return nil, fmt.Errorf("参数验证失败: %w", err)
	}

	// 执行工具
	return tool.Handler(ctx, args, k8sClient)
}

// validateArgs 验证工具参数
// 参数:
//   - tool: 工具定义
//   - args: 工具参数
//
// 返回:
//   - error: 错误信息
func (r *ToolRegistry) validateArgs(tool *Tool, args map[string]interface{}) error {
	if tool.InputSchema == nil {
		return nil
	}

	// 检查必填参数
	for _, required := range tool.InputSchema.Required {
		if _, exists := args[required]; !exists {
			return fmt.Errorf("缺少必填参数: %s", required)
		}
	}

	// 验证参数类型
	for name, value := range args {
		schema, exists := tool.InputSchema.Properties[name]
		if !exists {
			// 允许额外参数，但不验证
			continue
		}

		if err := r.validateParamType(name, value, schema); err != nil {
			return err
		}
	}

	return nil
}

// validateParamType 验证参数类型
// 参数:
//   - name: 参数名称
//   - value: 参数值
//   - schema: 参数 Schema
//
// 返回:
//   - error: 错误信息
func (r *ToolRegistry) validateParamType(name string, value interface{}, schema *ParameterSchema) error {
	if value == nil {
		if schema.Required {
			return fmt.Errorf("参数 '%s' 不能为空", name)
		}
		return nil
	}

	switch schema.Type {
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("参数 '%s' 应为字符串类型", name)
		}
	case "integer":
		switch value.(type) {
		case int, int32, int64, float64:
			// float64 是 JSON 解析数字的默认类型
		default:
			return fmt.Errorf("参数 '%s' 应为整数类型", name)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("参数 '%s' 应为布尔类型", name)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("参数 '%s' 应为数组类型", name)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("参数 '%s' 应为对象类型", name)
		}
	}

	// 验证枚举值
	if len(schema.Enum) > 0 {
		found := false
		for _, enumVal := range schema.Enum {
			if value == enumVal {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("参数 '%s' 的值不在允许的枚举范围内", name)
		}
	}

	return nil
}

// ToolCount 获取已注册工具数量
// 返回:
//   - int: 工具数量
func (r *ToolRegistry) ToolCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// HasTool 检查工具是否存在
// 参数:
//   - name: 工具名称
//
// 返回:
//   - bool: 是否存在
func (r *ToolRegistry) HasTool(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.tools[name]
	return exists
}

// UnregisterTool 注销工具
// 参数:
//   - name: 工具名称
//
// 返回:
//   - error: 错误信息
func (r *ToolRegistry) UnregisterTool(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("工具 '%s' 不存在", name)
	}

	delete(r.tools, name)
	return nil
}

// GetToolNames 获取所有工具名称
// 返回:
//   - []string: 工具名称列表
func (r *ToolRegistry) GetToolNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
