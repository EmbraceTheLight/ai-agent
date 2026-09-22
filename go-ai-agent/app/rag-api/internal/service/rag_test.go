package service

import (
	"context"
	"log/slog"
	"testing"

	v1 "go-ai-agent/app/rag-api/api/rag/v1"
	"go-ai-agent/app/rag-api/internal/biz"
)

type serviceLoader struct{}

func (serviceLoader) Load(path string) ([]*biz.Document, error) {
	return []*biz.Document{{SourcePath: path, Content: "content"}}, nil
}

type serviceEmbedder struct{}

func (serviceEmbedder) Embed(ctx context.Context, chunks []string) ([][]float32, error) {
	if len(chunks) == 1 {
		return [][]float32{{1, 0}}, nil
	}
	return [][]float32{{1, 0}}, nil
}

type serviceStore struct{}

func (serviceStore) Add(ctx context.Context, vector biz.Vector, chunk *biz.Chunk) error {
	return nil
}

func (serviceStore) Search(ctx context.Context, queryVector biz.Vector, topK int) ([]*biz.SearchResult, error) {
	return []*biz.SearchResult{{
		Chunk: &biz.Chunk{SourceFile: "rag.md", Title: "RAG", ChunkIndex: 2, Content: "content"},
		Score: 0.95,
	}}, nil
}

type serviceLLM struct{}

func (serviceLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "answer", nil
}

// TestRAGServiceAskConvertsResponse 验证 Ask 返回的领域检索结果能正确转换为 citations 和 proto search results。
// 输入: 使用 fake provider 构造 RAGUsecase 和 RAGService。
// 输出: 断言回答、引用来源、chunk 索引和分数均被保留。
// 示例: `go test ./app/rag-api/internal/service -run TestRAGServiceAskConvertsResponse`。
func TestRAGServiceAskConvertsResponse(t *testing.T) {
	usecase := biz.NewRAGUsecase(
		serviceLoader{},
		serviceEmbedder{},
		serviceStore{},
		serviceLLM{},
		&biz.RAGConfig{ChunkSize: 10, Overlap: 2},
		slog.Default(),
	)
	service := NewRagService(usecase, slog.Default())

	response, err := service.Ask(context.Background(), &v1.AskRequest{Question: "question", TopK: 1})
	if err != nil {
		t.Fatalf("Ask() error = %v", err)
	}
	if response.GetAnswer() != "answer" {
		t.Fatalf("answer = %q, want answer", response.GetAnswer())
	}
	if len(response.GetCitations()) != 1 || len(response.GetSearchResults()) != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.GetCitations()[0].GetSourceFile() != "rag.md" || response.GetCitations()[0].GetChunkIndex() != 2 {
		t.Fatalf("unexpected citation: %+v", response.GetCitations()[0])
	}
	if response.GetSearchResults()[0].GetScore() != 0.95 {
		t.Fatalf("score = %f, want 0.95", response.GetSearchResults()[0].GetScore())
	}
}
