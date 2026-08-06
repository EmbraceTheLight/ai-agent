package tools

import (
	"context"
	"encoding/json"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/utils"
)

var DefaultExecutor *Executor

func NewDefaultExecutor() *Executor {
	DefaultExecutor = NewExecutor(config.MaxSteps)
	defaultTimeTool := NewTimeTool("time", "获取当前时间, 支持指定时区")
	defaultCalculatorTool := NewCalculatorTool("calculator", "计算器, 支持加, 减, 乘, 除")
	defaultHttpGetTool := NewHttpGetTool("http_get", "通过 http get 请求获取数据")
	DefaultExecutor.RegisterTool(NewTool(defaultTimeTool))
	DefaultExecutor.RegisterTool(NewTool(defaultCalculatorTool))
	DefaultExecutor.RegisterTool(NewTool(defaultHttpGetTool))
	return DefaultExecutor
}

// Executor 工具执行器, 包含工具注册, 工具查找, 工具执行功能
type Executor struct {
	// 工具函数映射, key 为 tool 名称, value 为工具函数
	toolMap map[string]*Tool

	// 最大调用次数
	MaxSteps int
}

// NewExecutor 初始化一个 Executor
// toolMap 用于注册 Tool
// maxSteps 用于限制工具调用次数
func NewExecutor(maxSteps int) *Executor {
	return &Executor{
		toolMap:  make(map[string]*Tool),
		MaxSteps: maxSteps,
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

// GetToolList 获取当前 Executor 中注册的所有工具
func (e *Executor) GetToolList() []*Tool {
	ret := make([]*Tool, 0, len(e.toolMap))
	for _, v := range e.toolMap {
		ret = append(ret, v)
	}
	return ret
}

// normalizeRequest 将请求参数转换为 ToolFunc 接收的 json.RawMessage 格式
// 如果传入的 req 已经是 JSON 格式, 则不再二次序列化
func (e *Executor) normalizeRequest(req any) (json.RawMessage, error) {
	var obj map[string]any
	switch raw := req.(type) {
	case []byte:
		if err := json.Unmarshal(raw, &obj); err == nil && json.Valid(raw) == true {
			return json.RawMessage(raw), nil
		}
		return json.Marshal(raw)
	case string:
		if err := json.Unmarshal([]byte(raw), &obj); err == nil && json.Valid([]byte(raw)) == true {
			return json.RawMessage(raw), nil
		}
		return json.Marshal(raw)
	default:
		return json.Marshal(raw)
	}
}
