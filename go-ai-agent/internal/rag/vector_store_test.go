package rag

import (
	"math"
	"testing"
)

// TestVectorStoreSearchReturnsTopKInDescendingScore 测试内存向量库会返回相似度最高的 topK 结果,
// 且结果按余弦相似度从高到低排序。
func TestVectorStoreSearchReturnsTopKInDescendingScore(t *testing.T) {
	store := NewVectorStore()
	mustAddVector(t, store, vector{1, 0}, testChunk("notes/a.md", 0, "A exact match"))
	mustAddVector(t, store, vector{0.8, 0.2}, testChunk("notes/b.md", 1, "B close match"))
	mustAddVector(t, store, vector{0, 1}, testChunk("notes/c.md", 2, "C orthogonal"))
	mustAddVector(t, store, vector{-1, 0}, testChunk("notes/d.md", 3, "D opposite"))

	results, err := store.Search(vector{1, 0}, 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Embed.Chunk.Content != "A exact match" {
		t.Fatalf("expected first result A exact match, got %q", results[0].Embed.Chunk.Content)
	}
	if results[1].Embed.Chunk.Content != "B close match" {
		t.Fatalf("expected second result B close match, got %q", results[1].Embed.Chunk.Content)
	}
	if results[0].Score < results[1].Score {
		t.Fatalf("expected descending scores, got %.6f then %.6f", results[0].Score, results[1].Score)
	}
	if math.Abs(results[0].Score-1) > 1e-9 {
		t.Fatalf("expected first score close to 1, got %.12f", results[0].Score)
	}
	if results[0].Embed.Chunk.SourceFile != "notes/a.md" {
		t.Fatalf("expected first source file notes/a.md, got %q", results[0].Embed.Chunk.SourceFile)
	}
	if results[0].Embed.Chunk.ChunkIndex != 0 {
		t.Fatalf("expected first chunk index 0, got %d", results[0].Embed.Chunk.ChunkIndex)
	}
}

// TestVectorStoreAddRejectsEmptyVector 测试插入空向量时会返回错误,
// 避免后续相似度计算出现无效数据。
func TestVectorStoreAddRejectsEmptyVector(t *testing.T) {
	store := NewVectorStore()
	if err := store.Add(vector{}, testChunk("notes/empty.md", 0, "empty vector")); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestVectorStoreAddRejectsNilChunk 测试插入 nil chunk 时会返回错误,
// 避免检索结果缺失来源文件和 chunk 序号等引用信息。
func TestVectorStoreAddRejectsNilChunk(t *testing.T) {
	store := NewVectorStore()
	if err := store.Add(vector{1, 0}, nil); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// TestVectorStoreSearchReturnsErrorForInvalidTopK 测试 topK 非法或超过向量库大小时,
// Search 会返回错误而不是返回不明确的结果。
func TestVectorStoreSearchReturnsErrorForInvalidTopK(t *testing.T) {
	store := NewVectorStore()
	mustAddVector(t, store, vector{1, 0}, testChunk("notes/a.md", 0, "A"))

	tests := []struct {
		name string
		topK int
	}{
		{name: "zero topK", topK: 0},
		{name: "negative topK", topK: -1},
		{name: "topK greater than store size", topK: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := store.Search(vector{1, 0}, tt.topK)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if results != nil {
				t.Fatalf("expected nil results, got %d", len(results))
			}
		})
	}
}

// TestVectorStoreSearchReturnsErrorForInvalidVectors 测试 queryVector 与已存向量无法比较时,
// Search 会返回错误, 覆盖维度不一致、空 query 和零向量场景。
func TestVectorStoreSearchReturnsErrorForInvalidVectors(t *testing.T) {
	tests := []struct {
		name   string
		stored vector
		query  vector
	}{
		{name: "dimension mismatch", stored: vector{1, 0}, query: vector{1, 0, 0}},
		{name: "empty query vector", stored: vector{1, 0}, query: vector{}},
		{name: "zero query vector", stored: vector{1, 0}, query: vector{0, 0}},
		{name: "zero stored vector", stored: vector{0, 0}, query: vector{1, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewVectorStore()
			err := store.Add(tt.stored, testChunk("notes/chunk.md", 0, "chunk"))
			if err != nil {
				t.Fatalf("add vector failed: %v", err)
			}

			results, err := store.Search(tt.query, 1)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if results != nil {
				t.Fatalf("expected nil results, got %d", len(results))
			}
		})
	}
}

// TestCosineSimilarity 测试余弦相似度的基础数学结果。
// 输入: 相同方向、正交方向和相反方向的向量。
// 输出: 分数应分别接近 1、0 和 -1。
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a    vector
		b    vector
		want float64
	}{
		{name: "same direction", a: vector{1, 0}, b: vector{1, 0}, want: 1},
		{name: "orthogonal", a: vector{1, 0}, b: vector{0, 1}, want: 0},
		{name: "opposite direction", a: vector{1, 0}, b: vector{-1, 0}, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cosineSimilarity(tt.a, tt.b)
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if math.Abs(got-tt.want) > 1e-9 {
				t.Fatalf("expected %.6f, got %.6f", tt.want, got)
			}
		})
	}
}

// testChunk 创建带来源信息的测试 chunk。
// 输入: `sourceFile` 是源文件路径, `index` 是 chunk 序号, `content` 是 chunk 文本。
// 输出: 返回可写入向量库的 `*Chunk`。
// 示例: `testChunk("notes/rag.md", 0, "RAG")`。
func testChunk(sourceFile string, index int, content string) *Chunk {
	return &Chunk{
		SourceFile: sourceFile,
		ChunkIndex: index,
		Content:    content,
		Start:      0,
		End:        len([]rune(content)),
	}
}

// mustAddVector 向向量库添加测试向量, 添加失败时立即终止测试。
// 输入: `store` 是待写入的向量库, `vec` 是测试向量, `chunk` 是测试 chunk。
// 输出: 成功时无返回; 失败时调用 `t.Fatalf`。
// 示例: `mustAddVector(t, store, vector{1, 0}, testChunk("notes/rag.md", 0, "chunk"))`。
func mustAddVector(t *testing.T, store VectorStore, vec vector, chunk *Chunk) {
	t.Helper()
	if err := store.Add(vec, chunk); err != nil {
		t.Fatalf("add vector failed: %v", err)
	}
}
