package biz

import "testing"

// TestDocumentSplit 验证 Document 领域方法按 rune 数量和 overlap 生成 chunk。
// 输入: 一个包含多个字符的 Document 和固定切分配置。
// 输出: 断言 chunk 内容、来源和索引被正确保留。
// 示例: `go test ./app/rag-api/internal/biz -run TestDocumentSplit`。
func TestDocumentSplit(t *testing.T) {
	doc := &Document{
		Title:      "RAG",
		SourcePath: "rag.md",
		Content:    "abcdefghij",
	}

	chunks, err := doc.Split(ChunkConfig{Size: 6, Overlap: 2})
	if err != nil {
		t.Fatalf("Split() error = %v", err)
	}
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	if chunks[0].Content != "abcdef" || chunks[1].Content != "efghij" {
		t.Fatalf("unexpected chunk contents: %q, %q", chunks[0].Content, chunks[1].Content)
	}
	if chunks[0].SourceFile != doc.SourcePath || chunks[0].Title != doc.Title || chunks[1].ChunkIndex != 1 {
		t.Fatalf("unexpected chunk metadata: %+v, %+v", chunks[0], chunks[1])
	}
}

// TestDocumentSplitRejectsInvalidConfig 验证 Document 领域方法保留原有非法参数校验。
// 输入: overlap 小于 0 或 size 不大于 overlap 的切分配置。
// 输出: 断言每种非法配置都会返回错误。
// 示例: `go test ./app/rag-api/internal/biz -run TestDocumentSplitRejectsInvalidConfig`。
func TestDocumentSplitRejectsInvalidConfig(t *testing.T) {
	doc := &Document{Content: "abcdef"}
	configs := []ChunkConfig{
		{Size: 6, Overlap: -1},
		{Size: -1, Overlap: 0},
		{Size: 3, Overlap: 3},
	}
	for _, config := range configs {
		if _, err := doc.Split(config); err == nil {
			t.Errorf("Split(%+v) error = nil, want error", config)
		}
	}
}
