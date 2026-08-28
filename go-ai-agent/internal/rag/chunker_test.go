package rag

import (
	"path/filepath"
	"testing"
)

// TestChunkDocumentSplitsContentWithOverlap 测试普通文本按 size 切分,
// 且相邻 chunk 之间保留 overlap 个 rune 的重叠内容。
func TestChunkDocumentSplitsContentWithOverlap(t *testing.T) {
	doc := &Document{
		SourcePath: "notes/rag.md",
		Content:    "abcdefghijklmnopqrstuvwxyz",
	}

	chunks, err := ChunkDocument(doc, 10, 3)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []Chunk{
		{SourceFile: doc.SourcePath, ChunkIndex: 0, Content: "abcdefghij", RuneStartOffset: 0, RuneEndOffset: 10},
		{SourceFile: doc.SourcePath, ChunkIndex: 1, Content: "hijklmnopq", RuneStartOffset: 7, RuneEndOffset: 17},
		{SourceFile: doc.SourcePath, ChunkIndex: 2, Content: "opqrstuvwx", RuneStartOffset: 14, RuneEndOffset: 24},
		{SourceFile: doc.SourcePath, ChunkIndex: 3, Content: "vwxyz", RuneStartOffset: 21, RuneEndOffset: 26},
	}
	assertChunksEqual(t, chunks, want)
}

// TestChunkDocumentUsesRuneOffsets 测试中文内容不会按字节切坏,
// RuneStartOffset/RuneEndOffset 记录的是 rune 偏移量。
func TestChunkDocumentUsesRuneOffsets(t *testing.T) {
	doc := &Document{
		SourcePath: "notes/chinese.md",
		Content:    "你好世界abc",
	}

	chunks, err := ChunkDocument(doc, 3, 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []Chunk{
		{SourceFile: doc.SourcePath, ChunkIndex: 0, Content: "你好世", RuneStartOffset: 0, RuneEndOffset: 3},
		{SourceFile: doc.SourcePath, ChunkIndex: 1, Content: "世界a", RuneStartOffset: 2, RuneEndOffset: 5},
		{SourceFile: doc.SourcePath, ChunkIndex: 2, Content: "abc", RuneStartOffset: 4, RuneEndOffset: 7},
	}
	assertChunksEqual(t, chunks, want)
}

// TestChunkDocumentReturnsSingleChunkWhenContentShorterThanSize 测试文档长度不足一个 chunk 时,
// 会返回一个包含完整文档内容的 chunk。
func TestChunkDocumentReturnsSingleChunkWhenContentShorterThanSize(t *testing.T) {
	doc := &Document{
		SourcePath: "notes/short.txt",
		Content:    "short",
	}

	chunks, err := ChunkDocument(doc, 10, 2)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []Chunk{
		{SourceFile: doc.SourcePath, ChunkIndex: 0, Content: "short", RuneStartOffset: 0, RuneEndOffset: 5},
	}
	assertChunksEqual(t, chunks, want)
}

// TestChunkDocumentSupportsZeroOverlap 测试 overlap 为 0 时,
// chunk 之间不会出现重叠内容。
func TestChunkDocumentSupportsZeroOverlap(t *testing.T) {
	doc := &Document{
		SourcePath: "notes/no-overlap.txt",
		Content:    "abcdef",
	}

	chunks, err := ChunkDocument(doc, 3, 0)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	want := []Chunk{
		{SourceFile: doc.SourcePath, ChunkIndex: 0, Content: "abc", RuneStartOffset: 0, RuneEndOffset: 3},
		{SourceFile: doc.SourcePath, ChunkIndex: 1, Content: "def", RuneStartOffset: 3, RuneEndOffset: 6},
	}
	assertChunksEqual(t, chunks, want)
}

// TestChunkDocumentReturnsErrorForInvalidArguments 测试非法参数会返回错误,
// 并且不返回任何 chunk。
func TestChunkDocumentReturnsErrorForInvalidArguments(t *testing.T) {
	doc := &Document{SourcePath: "notes/rag.md", Content: "content"}
	tests := []struct {
		name    string
		doc     *Document
		size    int
		overlap int
	}{
		{name: "nil document", doc: nil, size: 10, overlap: 2},
		{name: "negative overlap", doc: doc, size: 10, overlap: -1},
		{name: "negative size", doc: doc, size: -1, overlap: 0},
		{name: "overlap equals size", doc: doc, size: 3, overlap: 3},
		{name: "overlap greater than size", doc: doc, size: 2, overlap: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks, err := ChunkDocument(tt.doc, tt.size, tt.overlap)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if chunks != nil {
				t.Fatalf("expected nil chunks, got %d", len(chunks))
			}
		})
	}
}

// TestChunkDocumentCarriesDocumentTitle 测试 chunker 会把 Document.Title 传递给每个 chunk。
// 输入: testdata 中一个包含标题的 Trilium 导出 Markdown 文档。
// 输出: 每个 chunk 的 Title 都应与 Document.Title 一致。
// 示例: `ChunkDocument(doc, 5000, 100)`。
func TestChunkDocumentCarriesDocumentTitle(t *testing.T) {
	filePath := filepath.Clean(filepath.Join("..", "..", "testdata", "documents", "work_notes_May", "五月", "第三周.md"))

	loader := NewTriliumDocumentLoader(nil, 0)
	docs, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	doc := docs[0]
	chunks, err := ChunkDocument(doc, 5000, 100)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk, got 0")
	}
	for i, chunk := range chunks {
		if chunk.Title != doc.Title {
			t.Fatalf("chunk %d expected title %q, got %q", i, doc.Title, chunk.Title)
		}
	}
}

// assertChunksEqual 对比实际 chunk 列表与期望 chunk 列表。
// 输入: `got` 是实际结果, `want` 是期望结果。
// 输出: 不匹配时通过 `t.Fatalf` 终止测试。
// 示例: `assertChunksEqual(t, chunks, want)`。
func assertChunksEqual(t *testing.T, got []*Chunk, want []Chunk) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d chunks, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] == nil {
			t.Fatalf("expected chunk %d, got nil", i)
		}
		if got[i].SourceFile != want[i].SourceFile {
			t.Fatalf("chunk %d expected source file %q, got %q", i, want[i].SourceFile, got[i].SourceFile)
		}
		if got[i].ChunkIndex != want[i].ChunkIndex {
			t.Fatalf("chunk %d expected index %d, got %d", i, want[i].ChunkIndex, got[i].ChunkIndex)
		}
		if got[i].Content != want[i].Content {
			t.Fatalf("chunk %d expected content %q, got %q", i, want[i].Content, got[i].Content)
		}
		if got[i].RuneStartOffset != want[i].RuneStartOffset {
			t.Fatalf("chunk %d expected start %d, got %d", i, want[i].RuneStartOffset, got[i].RuneStartOffset)
		}
		if got[i].RuneEndOffset != want[i].RuneEndOffset {
			t.Fatalf("chunk %d expected end %d, got %d", i, want[i].RuneEndOffset, got[i].RuneEndOffset)
		}
	}
}
