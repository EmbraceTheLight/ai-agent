package rag

import (
	"context"
	"go-ai-agent/internal/utils"
)

// Embedder 定义文本向量化能力。
// 输入: 一组 chunk 文本。
// 输出: 与输入文本一一对应的 embedding 向量。
// 示例: `embedder.Embed(ctx, []string{"RAG"})`。
type Embedder interface {
	// Embed 为多段文本生成 embedding 向量。
	// 输入: `ctx` 是请求上下文, `chunks` 是待向量化文本列表。
	// 输出: 返回向量列表; provider 调用失败时返回错误。
	// 示例: `Embed(ctx, []string{"chunk"})` -> `[][]float64`。
	Embed(ctx context.Context, chunks []string) ([][]float64, error)
}

// embedderClient 是基于 HTTP 的 embedding 客户端实现。
// 输入: 保存 embedding 模型名称和通用 HTTP Client。
// 输出: 通过 `Embed` 方法调用外部 embedding 服务。
// 示例: `NewEmbeddingClient("http://localhost:11434", "qwen3-embedding:0.6b")`。
type embedderClient struct {
	model      string
	httpClient *utils.HttpClient
}

// NewEmbeddingClient 创建 embedding 客户端。
// 输入: `url` 是 embedding 服务基础地址, `model` 是 embedding 模型名称。
// 输出: 返回实现 `Embedder` 的客户端。
// 示例: `NewEmbeddingClient("http://localhost:11434", "qwen3-embedding:0.6b")`。
func NewEmbeddingClient(url, model string) Embedder {
	return &embedderClient{
		model:      model,
		httpClient: utils.NewHttpClient(url),
	}
}

// Embed 调用 embedding 服务为多段 chunk 文本生成向量。
// 输入: `ctx` 是请求上下文, `chunks` 是待生成 embedding 的文本列表。
// 输出: 返回与输入文本一一对应的向量列表; HTTP 调用或响应解析失败时返回错误。
// 示例: `embedder.Embed(ctx, []string{"RAG 文档问答"})` -> 返回 `[][]float64`。
func (ec *embedderClient) Embed(ctx context.Context, chunks []string) ([][]float64, error) {
	requestBody := map[string]any{
		"model": ec.model,
		"input": chunks,
	}
	resp := &EmbedResp{}
	err := ec.httpClient.HttpPostJSON(ctx, "/api/embed", nil, requestBody, resp)
	if err != nil {
		return nil, err
	}
	return resp.EmbeddingsData, nil
}
