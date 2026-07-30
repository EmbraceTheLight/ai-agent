package tools

import "context"

// Tool tool 完整定义
type Tool struct {
	// Name tool 名称, 用于调用 tool 时, 寻找指定 tool
	Name string `json:"name"`

	// Description tool 描述, 帮助模型判断什么时候调用该工具
	Description string `json:"description"`

	// Parameters tool 参数, json schema, 用于模型调用 tool 时, 约束参数结构
	Parameters map[string]any                                                `json:"parameters"`
	Execute    func(ctx context.Context, req map[string]any) (string, error) `json:"execute"`
}

func ExecuteFunc(ctx context.Context, req map[string]any) (string, error) {
	return "", nil
}
