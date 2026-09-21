package service

import (
	"context"
	v1 "go-ai-agent/app/rag-api/api/rag/v1"
	"go-ai-agent/app/rag-api/internal/biz"
	"log/slog"
)

type RAGService struct {
	embedderUsecase *biz.EmbedderUsecase
	log             *slog.Logger
}

func NewRagService(embeddingUsecase *biz.EmbedderUsecase, log *slog.Logger) *RAGService {
	return &RAGService{
		embedderUsecase: embeddingUsecase,
		log:             log,
	}
}

func (R *RAGService) Ask(ctx context.Context, request *v1.AskRequest) (*v1.AskResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (R *RAGService) Health(ctx context.Context, request *v1.HealthRequest) (*v1.HealthResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (R *RAGService) ImportDocument(ctx context.Context, request *v1.ImportDocumentRequest) (*v1.ImportDocumentResponse, error) {
	//TODO implement me
	panic("implement me")
}
