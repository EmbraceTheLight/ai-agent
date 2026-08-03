package tools

import (
	"context"
	"encoding/json"
	"github.com/openai/openai-go/v3/packages/param"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/utils"
)

type ITool interface {
	GetToolParameterJSONSchema() (jsonSchema map[string]any)
	GetToolHandler() ToolFunc
}

type ToolFunc func(ctx context.Context, req json.RawMessage) (string, error)

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
func NewTool(name, description string, iTool ITool) *Tool {
	return &Tool{
		Name:        name,
		Description: param.NewOpt[string](description),
		Strict:      param.NewOpt[bool](true),
		Parameters:  iTool.GetToolParameterJSONSchema(),
		Handler:     iTool.GetToolHandler(),
	}
}

func (t *Tool) validateRequest(req json.RawMessage) error {
	// TODO: 验证实际请求 req 与 Tool.Parameter 中对请求的约束是否一致
	return nil
}

// Executor 工具执行器, 包含工具注册, 工具查找, 工具执行功能
type Executor struct {
	// 工具函数映射, key 为 tool 名称, value 为工具函数
	toolMap map[string]*Tool
}

// NewExecutor 初始化一个 Executor
func NewExecutor() *Executor {
	return &Executor{
		toolMap: make(map[string]*Tool),
	}
}

func (e *Executor) GetTool(name string) *Tool {
	return e.toolMap[name]
}

// RegisterTool 注册工具函数
func (e *Executor) RegisterTool(t *Tool) {
	e.toolMap[t.Name] = t
}

// Execute 执行工具函数
// 该包装函数会在工具调用前做准备工作, 包括转换请求, 获取超时 context, 查找对应 tool
// 调用 tool 后, 会对其返回值做处理
// 只返回 string, 如果执行过程中出现错误, 则将错误转换为 string 格式并返回
func (e *Executor) Execute(ctx context.Context, toolName string, req any) (string, error) {
	jsonRequest, err := e.normalizeRequest(req)
	if err != nil {
		return "", errno.RequestInvalid.WithError(err)
	}

	tool := e.GetTool(toolName)
	if tool == nil || tool.Handler == nil {
		return "", errno.ErrToolNotFound.WithMsgf("工具 %s 未注册", toolName)
	}
	// TODO: 验证实际请求 req 与 Tool.Parameter 中对请求的约束是否一致
	err = tool.validateRequest(jsonRequest)
	if err != nil {
		return "", err
	}
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	res, err := tool.Handler(ctx, jsonRequest)
	if err != nil {
		return "", errno.ErrToolExecuteFailed.WithError(err)
	}
	return res, nil
}

// normalizeRequest 将请求参数转换为 ToolFunc 接收的 json.RawMessage 格式
// 目前接受 req 为 string
func (e *Executor) normalizeRequest(req any) (json.RawMessage, error) {
	return json.Marshal(req)
}
