package data

import (
	"context"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"log/slog"
)

type embedderRepo struct {
	data *Data
	log  *slog.Logger
}

func NewEmbedderRepo(data *Data, log *slog.Logger) biz.Embedder {
	if log == nil {
		log = slog.Default()
	}
	return &embedderRepo{
		data: data,
		log:  log,
	}
}

// EmbedResp 描述 embedding provider 返回的响应结构。
// 输入: 由 HTTP JSON 响应反序列化得到。
// 输出: `EmbeddingsData` 保存与输入文本一一对应的向量。
// 示例: `EmbedResp{EmbeddingsData: [][]float32{{0.1, 0.2}}}`。
type embedResp struct {
	EmbeddingsData [][]float32 `json:"embeddings"`
}

func (repo *embedderRepo) Embed(ctx context.Context, chunks []string) ([][]float32, error) {
	if repo == nil || repo.data == nil || repo.data.embedderData == nil {
		return nil, fmt.Errorf("embedding client 未初始化")
	}
	requestBody := map[string]any{
		"model": repo.data.embedderData.model,
		"input": chunks,
	}
	resp := &embedResp{}
	err := repo.data.embedderData.httpClient.HttpPostJSON(ctx, "/api/embed", nil, requestBody, resp)
	if err != nil {
		return nil, err
	}
	return resp.EmbeddingsData, nil
}
