package biz

import (
	"context"
	"log/slog"
)

// Embedder 定义文本向量化能力。
// 输入: 一组 chunk 文本。
// 输出: 与输入文本一一对应的 embedding 向量。
// 示例: `embedder.Embed(ctx, []string{"RAG"})`。
type Embedder interface {
	// Embed 为多段文本生成 embedding 向量。
	// 输入: `ctx` 是请求上下文, `chunks` 是待向量化文本列表。
	// 输出: 返回向量列表; provider 调用失败时返回错误。
	// 示例: `Embed(ctx, []string{"chunk"})` -> `[][]float32`。
	Embed(ctx context.Context, chunks []string) ([][]float32, error)
}

// EmbedderUsecase 封装 embedding 能力，供 RAG 用例复用。
type EmbedderUsecase struct {
	repo Embedder
	log  *slog.Logger
}

// NewEmbedderUsecase 创建 embedding 用例。
// 输入: `repo` 是 embedding provider, `log` 是结构化日志对象。
// 输出: 返回可生成文本向量的 embedding 用例。
// 示例: `NewEmbedderUsecase(repo, slog.Default())`。
func NewEmbedderUsecase(repo Embedder, log *slog.Logger) *EmbedderUsecase {
	return &EmbedderUsecase{repo: repo, log: log}
}

// Embed 为多段文本生成 embedding 向量。
// 输入: `ctx` 是请求上下文, `chunks` 是待向量化文本列表。
// 输出: 返回与输入文本一一对应的向量列表。
// 示例: `usecase.Embed(ctx, []string{"chunk"})` -> `[][]float32`。
func (usecase *EmbedderUsecase) Embed(ctx context.Context, chunks []string) ([][]float32, error) {
	return usecase.repo.Embed(ctx, chunks)
}
