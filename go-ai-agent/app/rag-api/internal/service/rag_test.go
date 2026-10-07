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

// VisitDocumentChunks 为 Ask 测试提供不执行遍历的向量库方法。
// 输入: 文档 ID 和批次处理函数。
// 输出: 始终返回 nil。
// 示例: `serviceStore{}.VisitDocumentChunks(ctx, id, handle)`。
func (serviceStore) VisitDocumentChunks(ctx context.Context, documentID string, handle func([]*biz.Chunk) error) error {
	return nil
}

// UpsertBatch 为 Ask 测试提供不执行写入的向量库方法。
// 输入: 待写入的向量记录。
// 输出: 始终返回 nil。
// 示例: `serviceStore{}.UpsertBatch(ctx, records)`。
func (serviceStore) UpsertBatch(ctx context.Context, records []*biz.Embedding) error { return nil }

// DeleteChunksFromIndex 为 Ask 测试提供不执行删除的向量库方法。
// 输入: 文档 ID 和删除起始序号。
// 输出: 始终返回 nil。
// 示例: `serviceStore{}.DeleteChunksFromIndex(ctx, id, 1)`。
func (serviceStore) DeleteChunksFromIndex(ctx context.Context, documentID string, from int64) error {
	return nil
}

// serviceMetadata 为 Ask 测试提供空的文档元数据仓库。
type serviceMetadata struct{}

// GetDocumentMetadataList 模拟未找到任何文档元数据。
// 输入: 文档 ID 列表。
// 输出: 返回 nil 列表和 nil 错误。
// 示例: `serviceMetadata{}.GetDocumentMetadataList(ctx, ids)`。
func (serviceMetadata) GetDocumentMetadataList(ctx context.Context, ids []string) ([]*biz.DocumentMetadataDO, error) {
	return nil, nil
}

// GetDocumentTitleByIdList 模拟未找到任何文档标题。
// 输入: 文档 ID 列表。
// 输出: 返回空映射和 nil 错误。
// 示例: `serviceMetadata{}.GetDocumentTitleByIdList(ctx, ids)`。
func (serviceMetadata) GetDocumentTitleByIdList(ctx context.Context, ids []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// UpsertDocumentMetadata 为 Ask 测试提供不执行写入的元数据方法。
// 输入: 待保存的文档元数据。
// 输出: 始终返回 nil。
// 示例: `serviceMetadata{}.UpsertDocumentMetadata(ctx, metadata)`。
func (serviceMetadata) UpsertDocumentMetadata(ctx context.Context, doc *biz.DocumentMetadataDO) error {
	return nil
}

// serviceLock 为 Ask 测试提供始终可获取的文档锁。
type serviceLock struct{}

// LockDocument 模拟成功取得文档锁。
// 输入: 文档 ID。
// 输出: 始终返回 true。
// 示例: `serviceLock{}.LockDocument(ctx, id)`。
func (serviceLock) LockDocument(ctx context.Context, id string) bool { return true }

// UnlockDocument 模拟释放文档锁, 不修改测试状态。
// 输入: 文档 ID。
// 输出: 无返回值。
// 示例: `serviceLock{}.UnlockDocument(ctx, id)`。
func (serviceLock) UnlockDocument(ctx context.Context, id string) {}

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
		serviceMetadata{}, serviceLock{},
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
