package rag

import (
	"context"
	"go-ai-agent/internal/config"
	"net/http"
	"time"
)

type Embedder interface {
	Embed(ctx context.Context, chunks []string) ([][]float64, error)
}

type embedderClient struct {
	url        string
	model      string
	httpClient *http.Client
}

func NewEmbeddingClient(url, model string) Embedder {
	return &embedderClient{
		url:   url,
		model: model,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

func (ec *embedderClient) Embed(ctx context.Context, chunks []string) ([][]float64, error) {
	request := &http.Request{}
	ec.httpClient.Do()
}

func (ec *embedderClient) buildPostRequest(text string) *http.Request {

}
