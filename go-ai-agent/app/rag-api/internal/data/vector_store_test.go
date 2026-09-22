package data

import (
	"context"
	"testing"

	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/conf"
)

// TestNewVectorStoreMemory 验证 memory 配置不创建 Milvus 连接且能够完成向量写入和检索。
// 输入: 使用 memory vector_store 配置创建 Data 和 VectorStore。
// 输出: 断言向量库能返回按相似度降序排列的结果。
// 示例: `go test ./app/rag-api/internal/data -run TestNewVectorStoreMemory`。
func TestNewVectorStoreMemory(t *testing.T) {
	data, cleanup, err := NewData(&conf.Data{
		Embedder: &conf.Data_Embedder{BaseUrl: "http://localhost:11434", Model: "embedding", Dim: 2},
		Rag:      &conf.Data_Rag{VectorStore: "memory"},
	})
	if err != nil {
		t.Fatalf("NewData() error = %v", err)
	}
	defer cleanup()

	store, err := NewVectorStore(data, &conf.Data{
		Embedder: &conf.Data_Embedder{Dim: 2},
		Rag:      &conf.Data_Rag{VectorStore: "memory"},
	})
	if err != nil {
		t.Fatalf("NewVectorStore() error = %v", err)
	}
	if err := store.Add(context.Background(), biz.Vector{1, 0}, &biz.Chunk{SourceFile: "first.md", Content: "first"}); err != nil {
		t.Fatalf("Add(first) error = %v", err)
	}
	if err := store.Add(context.Background(), biz.Vector{0, 1}, &biz.Chunk{SourceFile: "second.md", Content: "second"}); err != nil {
		t.Fatalf("Add(second) error = %v", err)
	}

	results, err := store.Search(context.Background(), biz.Vector{1, 0}, 2)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 2 || results[0].Chunk.SourceFile != "first.md" {
		t.Fatalf("unexpected search results: %+v", results)
	}
}
