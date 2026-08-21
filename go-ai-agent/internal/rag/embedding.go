package rag

import (
	"context"
	"go-ai-agent/internal/utils"
)

type Embedder interface {
	Embed(ctx context.Context, chunks []string) ([][]float64, error)
}

type embedderClient struct {
	model      string
	httpClient *utils.HttpClient
}

func NewEmbeddingClient(url, model string) Embedder {
	return &embedderClient{
		model:      model,
		httpClient: utils.NewHttpClient(url),
	}
}

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
