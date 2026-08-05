package tools

import (
	"context"
	"encoding/json"
	"github.com/openai/openai-go/v3/packages/param"
)

type ITool interface {
	GetToolParameterJSONSchema() (jsonSchema map[string]any)
	GetToolName() string
	GetToolDescription() string
	GetToolHandler() ToolFunc
}

type ToolFunc func(ctx context.Context, req json.RawMessage) (string, error)

// ToolTemplate
// Tool 模板, Tool 应当将其嵌套到自身结构中
type ToolTemplate struct {
	Name        string
	Description string
}

func NewTemplate(name, description string) ToolTemplate {
	return ToolTemplate{
		Name:        name,
		Description: description,
	}
}

func (t *ToolTemplate) GetToolName() string {
	return t.Name
}

func (t *ToolTemplate) GetToolDescription() string {
	return t.Description
}

// Tool tool 完整定义
type Tool struct {
	// Name tool 名称, 用于调用 tool 时, 寻找指定 tool
	Name string `json:"name"`

	// Description tool 描述, 帮助模型判断什么时候调用该工具
	Description param.Opt[string] `json:"description"`

	// Strict 严格模式. 确保函数调用遵循定义的函数模式. 推荐将其设置为 true
	// 当 Strict 为 true 时, Parameter 参数中每个 object 的 additionalProperties 都必须设置为 false, 且其所有字段都必须标记为 required
	Strict param.Opt[bool] `json:"strict"`

	// Parameters tool 参数, json schema, 用于模型调用 tool 时, 约束参数结构
	Parameters map[string]any `json:"parameters"`

	Handler ToolFunc `json:"-"`
}

// NewTool 返回一个 tool 定义. Tool.Strict 固定为 true
func NewTool(iTool ITool) *Tool {
	return &Tool{
		Name:        iTool.GetToolName(),
		Description: param.NewOpt[string](iTool.GetToolDescription()),
		Strict:      param.NewOpt[bool](true),
		Parameters:  iTool.GetToolParameterJSONSchema(),
		Handler:     iTool.GetToolHandler(),
	}
}

func (t *Tool) validateRequest(req json.RawMessage) error {
	// TODO: 验证实际请求 req 与 Tool.Parameter 中对请求的约束是否一致
	return nil
}
