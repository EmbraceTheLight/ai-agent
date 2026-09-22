package biz

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

type fakeDocumentLoader struct {
	documents []*Document
	err       error
}

func (loader *fakeDocumentLoader) Load(path string) ([]*Document, error) {
	return loader.documents, loader.err
}

type fakeEmbedder struct {
	inputs [][]string
	result [][]float32
	err    error
}

func (embedder *fakeEmbedder) Embed(ctx context.Context, chunks []string) ([][]float32, error) {
	embedder.inputs = append(embedder.inputs, append([]string(nil), chunks...))
	return embedder.result, embedder.err
}

type fakeVectorStore struct {
	added []*Embedding
}

func (store *fakeVectorStore) Add(ctx context.Context, vector Vector, chunk *Chunk) error {
	store.added = append(store.added, &Embedding{Vector: vector, Chunk: chunk})
	return nil
}

func (store *fakeVectorStore) Search(ctx context.Context, queryVector Vector, topK int) ([]*SearchResult, error) {
	return nil, nil
}

type fakeLLM struct{}

func (fakeLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "answer", nil
}

// TestRAGUsecaseImportDocuments 验证文档导入按 load、chunk、embed、store 顺序执行。
// 输入: 使用 fake loader、embedder 和 vector store 构造 RAGUsecase。
// 输出: 断言统计结果、embedding 输入和写入 chunk 数量。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGUsecaseImportDocuments`。
func TestRAGUsecaseImportDocuments(t *testing.T) {
	loader := &fakeDocumentLoader{documents: []*Document{{
		Title:      "RAG",
		SourcePath: "rag.md",
		Content:    "abcdefghij",
	}}}
	embedder := &fakeEmbedder{result: [][]float32{{1, 0}, {0, 1}}}
	store := &fakeVectorStore{}
	usecase := NewRAGUsecase(
		loader,
		embedder,
		store,
		fakeLLM{},
		&RAGConfig{ChunkSize: 6, Overlap: 2},
		slog.Default(),
	)

	result, err := usecase.ImportDocuments(context.Background(), "rag.md")
	if err != nil {
		t.Fatalf("ImportDocuments() error = %v", err)
	}
	if result.Documents != 1 || result.Chunks != 2 || result.Embedded != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(embedder.inputs) != 1 || len(embedder.inputs[0]) != 2 {
		t.Fatalf("unexpected embed inputs: %+v", embedder.inputs)
	}
	if len(store.added) != 2 {
		t.Fatalf("stored chunks = %d, want 2", len(store.added))
	}
	if store.added[0].Chunk.Content != "abcdef" || store.added[1].Chunk.Content != "efghij" {
		t.Fatalf("unexpected chunk contents: %q, %q", store.added[0].Chunk.Content, store.added[1].Chunk.Content)
	}
}

// TestRAGUsecaseImportDocumentsRejectsEmbeddingCountMismatch 验证 embedding 数量不匹配时不会写入向量库。
// 输入: fake embedder 返回少于 chunk 数量的向量。
// 输出: 断言导入失败且 vector store 没有新增记录。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGUsecaseImportDocumentsRejectsEmbeddingCountMismatch`。
func TestRAGUsecaseImportDocumentsRejectsEmbeddingCountMismatch(t *testing.T) {
	loader := &fakeDocumentLoader{documents: []*Document{{SourcePath: "rag.md", Content: "abcdef"}}}
	embedder := &fakeEmbedder{result: [][]float32{{1, 0}}}
	store := &fakeVectorStore{}
	usecase := NewRAGUsecase(loader, embedder, store, fakeLLM{}, &RAGConfig{ChunkSize: 3}, slog.Default())

	_, err := usecase.ImportDocuments(context.Background(), "rag.md")
	if err == nil {
		t.Fatal("ImportDocuments() error = nil, want embedding count mismatch")
	}
	if len(store.added) != 0 {
		t.Fatalf("stored chunks = %d, want 0", len(store.added))
	}
}

// TestRAGUsecaseImportDocumentsPropagatesLoaderError 验证文档加载失败时不会继续执行后续步骤。
// 输入: fake loader 返回错误。
// 输出: 断言导入返回错误且 embedding 没有收到输入。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGUsecaseImportDocumentsPropagatesLoaderError`。
func TestRAGUsecaseImportDocumentsPropagatesLoaderError(t *testing.T) {
	loader := &fakeDocumentLoader{err: errors.New("load failed")}
	embedder := &fakeEmbedder{result: [][]float32{{1, 0}}}
	usecase := NewRAGUsecase(loader, embedder, &fakeVectorStore{}, fakeLLM{}, &RAGConfig{ChunkSize: 3}, slog.Default())

	_, err := usecase.ImportDocuments(context.Background(), "rag.md")
	if err == nil {
		t.Fatal("ImportDocuments() error = nil, want loader error")
	}
	if len(embedder.inputs) != 0 {
		t.Fatalf("embed calls = %d, want 0", len(embedder.inputs))
	}
}
