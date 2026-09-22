package data

import (
	"context"
	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/conf"
	"go-ai-agent/internal/llm"
)

type llmRepo struct {
	client llm.Client
}

// NewLLMRepo 创建面向 biz 的 OpenAI 兼容 LLM 适配器。
// 输入: `c` 是应用数据配置, 包含 provider、API key、base URL 和模型名称。
// 输出: 返回实现 `biz.LLM` 的 LLM provider。
// 示例: `NewLLMRepo(confData)`。
func NewLLMRepo(c *conf.Data) biz.LLM {
	var modelConf *conf.Data_Model
	if c != nil {
		modelConf = c.Model
	}
	if modelConf == nil {
		modelConf = &conf.Data_Model{}
	}
	return &llmRepo{
		client: llm.NewOpenAIClient(modelConf.ApiKey, modelConf.BaseUrl, modelConf.ModelName),
	}
}

// Generate 根据 biz 提供的系统 prompt 和用户问题生成回答。
// 输入: `ctx` 是请求上下文, `systemPrompt` 是资料约束 prompt, `userMessage` 是用户问题。
// 输出: 返回模型回答; LLM provider 调用失败时返回错误。
// 示例: `repo.Generate(ctx, prompt, "什么是 RAG?")`。
func (repo *llmRepo) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return repo.client.Generate(ctx, []llm.Message{
		{Role: llm.SystemMessage, Content: systemPrompt},
		{Role: llm.UserMessage, Content: userMessage},
	})
}
