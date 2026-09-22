package biz

import "context"

// LLM 定义 RAG 生成回答所需的最小模型能力。
// 输入: 系统 prompt 和用户问题。
// 输出: 返回模型生成的回答文本。
// 示例: `llm.Generate(ctx, prompt, question)`。
type LLM interface {
	// Generate 根据系统 prompt 和用户问题生成回答。
	// 输入: `ctx` 是请求上下文, `systemPrompt` 是资料约束 prompt, `userMessage` 是用户问题。
	// 输出: 返回模型回答; provider 调用失败时返回错误。
	// 示例: `Generate(ctx, "只能基于资料回答", "什么是 RAG?")`。
	Generate(ctx context.Context, systemPrompt, userMessage string) (string, error)
}
