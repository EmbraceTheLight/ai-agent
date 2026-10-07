package biz

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeDocumentLoader struct {
	documents []*Document
	err       error
}

// testDocumentPath 在测试临时目录中创建一个可用于计算文档 ID 的文件。
// 输入: `t` 是当前测试, `name` 是临时文件名。
// 输出: 返回已存在的文件绝对路径; 创建失败时终止测试。
// 示例: `path := testDocumentPath(t, "rag.md")`。
func testDocumentPath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	return path
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
	if embedder.result == nil && embedder.err == nil {
		vectors := make([][]float32, len(chunks))
		for i := range vectors {
			vectors[i] = []float32{1, 0}
		}
		return vectors, nil
	}
	return embedder.result, embedder.err
}

// fakeVectorStore 记录批量写入及删除结果, 并可模拟指定批次失败。
type fakeVectorStore struct {
	added       []*Embedding
	rows        map[int64]*Embedding
	failUpsert  bool
	failOnCall  int
	upsertCalls int
	deletedFrom int64
}

func (store *fakeVectorStore) Add(ctx context.Context, vector Vector, chunk *Chunk) error {
	store.added = append(store.added, &Embedding{Vector: vector, Chunk: chunk})
	return nil
}

func (store *fakeVectorStore) Search(ctx context.Context, queryVector Vector, topK int) ([]*SearchResult, error) {
	return nil, nil
}

// VisitDocumentChunks 将 fake 中属于指定文档的旧 chunk 交给处理函数。
// 输入: `documentID` 是待查询的文档 ID, `handle` 接收匹配结果。
// 输出: 返回处理函数的错误。
// 示例: `store.VisitDocumentChunks(ctx, id, func(chunks []*Chunk) error { return nil })`。
func (store *fakeVectorStore) VisitDocumentChunks(ctx context.Context, documentID string, handle func([]*Chunk) error) error {
	batch := make([]*Chunk, 0)
	for _, row := range store.rows {
		if row.Chunk.DocumentId == documentID {
			batch = append(batch, row.Chunk)
		}
	}
	return handle(batch)
}

// UpsertBatch 按 chunk ID 保存测试记录, 并按配置模拟写入失败。
// 输入: `records` 是本批待写入的 chunk 向量。
// 输出: 命中失败条件时返回错误, 否则返回 nil。
// 示例: `store.UpsertBatch(ctx, records)`。
func (store *fakeVectorStore) UpsertBatch(ctx context.Context, records []*Embedding) error {
	store.upsertCalls++
	if store.failUpsert || store.failOnCall == store.upsertCalls {
		return errors.New("upsert failed")
	}
	if store.rows == nil {
		store.rows = make(map[int64]*Embedding)
	}
	for _, record := range records {
		store.rows[record.Chunk.Id] = record
		store.added = append(store.added, record)
	}
	return nil
}

// TestRAGImportResumesAfterLaterBatchFailure 验证首批已写入、次批失败后重跑能复用主键且最后才提交元数据。
// 输入: 70 个 chunk 的文档和在第二批写入时失败的 fake vector store。
// 输出: 断言失败时元数据未写入, 重试后 chunk 数量正确且已写主键不变。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGImportResumesAfterLaterBatchFailure`。
func TestRAGImportResumesAfterLaterBatchFailure(t *testing.T) {
	path := testDocumentPath(t, "large.md")
	loader := &fakeDocumentLoader{documents: []*Document{{SourcePath: path, Content: strings.Repeat("a", 70)}}}
	store := &fakeVectorStore{failOnCall: 2}
	meta := &fakeMetadata{}
	usecase := NewRAGUsecase(loader, &fakeEmbedder{}, store, fakeLLM{}, meta, fakeLock{}, &RAGConfig{ChunkSize: 1}, slog.Default())
	if _, err := usecase.ImportDocuments(context.Background(), path); err == nil {
		t.Fatal("expected second batch error")
	}
	if len(store.rows) != 32 || meta.writes != 0 {
		t.Fatalf("rows=%d metadata writes=%d", len(store.rows), meta.writes)
	}
	firstIDs := make(map[int64]struct{}, len(store.rows))
	for id := range store.rows {
		firstIDs[id] = struct{}{}
	}
	store.failOnCall = 0
	if _, err := usecase.ImportDocuments(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if len(store.rows) != 70 || meta.writes != 1 {
		t.Fatalf("rows=%d metadata writes=%d", len(store.rows), meta.writes)
	}
	for id := range firstIDs {
		if _, ok := store.rows[id]; !ok {
			t.Fatalf("existing primary key %d was not reused", id)
		}
	}
}

// DeleteChunksFromIndex 模拟删除指定文档中超出新切分范围的记录。
// 输入: `documentID` 是文档 ID, `from` 是删除起始序号。
// 输出: 删除完成返回 nil, 并记录删除起始序号供断言。
// 示例: `store.DeleteChunksFromIndex(ctx, id, 1)`。
func (store *fakeVectorStore) DeleteChunksFromIndex(ctx context.Context, documentID string, from int64) error {
	store.deletedFrom = from
	for id, row := range store.rows {
		if row.Chunk.DocumentId == documentID && row.Chunk.ChunkIndex >= from {
			delete(store.rows, id)
		}
	}
	return nil
}

// fakeMetadata 保存单篇文档的测试元数据并统计写入次数。
type fakeMetadata struct {
	row    *DocumentMetadataDO
	writes int
}

// GetDocumentMetadataList 返回 fake 中已保存的文档元数据。
// 输入: `ids` 是待查询的文档 ID 列表。
// 输出: 无记录时返回 nil, 否则返回当前测试记录。
// 示例: `metadata.GetDocumentMetadataList(ctx, []string{id})`。
func (m *fakeMetadata) GetDocumentMetadataList(ctx context.Context, ids []string) ([]*DocumentMetadataDO, error) {
	if m.row == nil {
		return nil, nil
	}
	return []*DocumentMetadataDO{m.row}, nil
}

// GetDocumentTitleByIdList 为 fake 查询返回空标题映射。
// 输入: `ids` 是待查询的文档 ID 列表。
// 输出: 返回空映射和 nil 错误。
// 示例: `metadata.GetDocumentTitleByIdList(ctx, []string{id})`。
func (m *fakeMetadata) GetDocumentTitleByIdList(ctx context.Context, ids []string) (map[string]string, error) {
	return map[string]string{}, nil
}

// UpsertDocumentMetadata 保存测试元数据并累计成功写入次数。
// 输入: `row` 是本次成功导入的文档元数据。
// 输出: 保存完成返回 nil。
// 示例: `metadata.UpsertDocumentMetadata(ctx, row)`。
func (m *fakeMetadata) UpsertDocumentMetadata(ctx context.Context, row *DocumentMetadataDO) error {
	m.row = row
	m.writes++
	return nil
}

// fakeLock 在业务测试中始终允许取得文档锁。
type fakeLock struct{}

// LockDocument 模拟成功取得指定文档的锁。
// 输入: `id` 是文档 ID。
// 输出: 始终返回 true。
// 示例: `fakeLock{}.LockDocument(ctx, id)`。
func (fakeLock) LockDocument(ctx context.Context, id string) bool { return true }

// UnlockDocument 模拟释放文档锁, 不修改测试状态。
// 输入: `id` 是文档 ID。
// 输出: 无返回值。
// 示例: `fakeLock{}.UnlockDocument(ctx, id)`。
func (fakeLock) UnlockDocument(ctx context.Context, id string) {}

type fakeLLM struct{}

func (fakeLLM) Generate(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	return "answer", nil
}

// TestRAGUsecaseImportDocuments 验证文档导入按 load、chunk、embed、批量写入顺序执行。
// 输入: 使用 fake loader、embedder 和 vector store 构造 RAGUsecase。
// 输出: 断言统计结果、embedding 输入和写入 chunk 数量。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGUsecaseImportDocuments`。
func TestRAGUsecaseImportDocuments(t *testing.T) {
	path := testDocumentPath(t, "rag.md")
	loader := &fakeDocumentLoader{documents: []*Document{{
		Title:      "RAG",
		SourcePath: path,
		Content:    "abcdefghij",
	}}}
	embedder := &fakeEmbedder{result: [][]float32{{1, 0}, {0, 1}}}
	store := &fakeVectorStore{}
	usecase := NewRAGUsecase(
		loader,
		embedder,
		store,
		fakeLLM{},
		&fakeMetadata{}, fakeLock{},
		&RAGConfig{ChunkSize: 6, Overlap: 2},
		slog.Default(),
	)

	result, err := usecase.ImportDocuments(context.Background(), path)
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
	path := testDocumentPath(t, "rag.md")
	loader := &fakeDocumentLoader{documents: []*Document{{SourcePath: path, Content: "abcdef"}}}
	embedder := &fakeEmbedder{result: [][]float32{{1, 0}}}
	store := &fakeVectorStore{}
	usecase := NewRAGUsecase(loader, embedder, store, fakeLLM{}, &fakeMetadata{}, fakeLock{}, &RAGConfig{ChunkSize: 3}, slog.Default())

	_, err := usecase.ImportDocuments(context.Background(), path)
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
	usecase := NewRAGUsecase(loader, embedder, &fakeVectorStore{}, fakeLLM{}, &fakeMetadata{}, fakeLock{}, &RAGConfig{ChunkSize: 3}, slog.Default())

	_, err := usecase.ImportDocuments(context.Background(), "rag.md")
	if err == nil {
		t.Fatal("ImportDocuments() error = nil, want loader error")
	}
	if len(embedder.inputs) != 0 {
		t.Fatalf("embed calls = %d, want 0", len(embedder.inputs))
	}
}

// TestRAGImportRetriesAfterFailedBatch 验证批量写入失败不推进元数据，重试后可成功完成。
// 输入: 首次批量写入失败、第二次恢复的 fake vector store。
// 输出: 断言重试后元数据只写一次, 再次导入不重复向量化。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGImportRetriesAfterFailedBatch`。
func TestRAGImportRetriesAfterFailedBatch(t *testing.T) {
	path := testDocumentPath(t, "retry.md")
	loader := &fakeDocumentLoader{documents: []*Document{{SourcePath: path, Content: "abcdef"}}}
	store := &fakeVectorStore{failUpsert: true}
	meta := &fakeMetadata{}
	usecase := NewRAGUsecase(loader, &fakeEmbedder{result: [][]float32{{1, 0}}}, store, fakeLLM{}, meta, fakeLock{}, &RAGConfig{ChunkSize: 6}, slog.Default())
	if _, err := usecase.ImportDocuments(context.Background(), path); err == nil {
		t.Fatal("expected upsert error")
	}
	if meta.writes != 0 {
		t.Fatalf("metadata writes = %d, want 0", meta.writes)
	}
	store.failUpsert = false
	if _, err := usecase.ImportDocuments(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if meta.writes != 1 || len(store.rows) != 1 {
		t.Fatalf("writes=%d rows=%d", meta.writes, len(store.rows))
	}
	previous := meta.row.LastIndexedAt
	result, err := usecase.ImportDocuments(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Embedded != 0 || meta.writes != 1 || !meta.row.LastIndexedAt.Equal(previous) {
		t.Fatalf("idempotent import: %+v, writes=%d", result, meta.writes)
	}
}

// TestRAGImportDeletesSurplusChunks 验证文档缩短时删除超出新切分范围的旧记录。
// 输入: 同一文档先导入两个 chunk, 再将内容缩短为一个 chunk。
// 输出: 断言旧尾部 chunk 被删除且只保留当前记录。
// 示例: `go test ./app/rag-api/internal/biz -run TestRAGImportDeletesSurplusChunks`。
func TestRAGImportDeletesSurplusChunks(t *testing.T) {
	path := testDocumentPath(t, "short.md")
	loader := &fakeDocumentLoader{documents: []*Document{{SourcePath: path, Content: "abcdef"}}}
	store := &fakeVectorStore{}
	meta := &fakeMetadata{}
	usecase := NewRAGUsecase(loader, &fakeEmbedder{result: [][]float32{{1, 0}, {0, 1}}}, store, fakeLLM{}, meta, fakeLock{}, &RAGConfig{ChunkSize: 3}, slog.Default())
	if _, err := usecase.ImportDocuments(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	loader.documents[0].Content = "abc"
	usecase.embedder = &fakeEmbedder{result: [][]float32{{1, 0}}}
	if _, err := usecase.ImportDocuments(context.Background(), path); err != nil {
		t.Fatal(err)
	}
	if store.deletedFrom != 1 || len(store.rows) != 1 {
		t.Fatalf("deletedFrom=%d rows=%d", store.deletedFrom, len(store.rows))
	}
}
