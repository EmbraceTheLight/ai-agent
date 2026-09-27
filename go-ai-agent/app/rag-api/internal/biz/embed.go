package biz

import (
	"context"
	"log"
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

type EmbedderUsecase struct {
	repo Embedder
	log  *log.Logger
}

func NewEmbedderUsecase(repo Embedder, log *log.Logger) *EmbedderUsecase {
	return &EmbedderUsecase{repo: repo, log: log}
}

func (usecase *EmbedderUsecase) Embed(ctx context.Context, chunks []string) ([][]float32, error) {
	return usecase.repo.Embed(ctx, chunks)
}
