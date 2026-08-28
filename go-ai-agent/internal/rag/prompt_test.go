package rag

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildPromptIncludesChunkTitle 测试 prompt 会包含 chunk 的标题、来源和内容。
// 输入: testdata 中真实的 Trilium 导出 Markdown 文档。
// 输出: prompt 文本应包含标题、chunk 来源和正文内容。
// 示例: `BuildPrompt([]*SearchResult{result})`。
func TestBuildPromptIncludesChunkTitle(t *testing.T) {
	filePath := filepath.Clean(filepath.Join("..", "..", "testdata", "documents", "work_notes_May", "五月", "第三周.md"))

	loader := NewTriliumDocumentLoader(nil, 0)
	docs, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	chunks, err := ChunkDocument(docs[0], 5000, 100)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least 1 chunk, got 0")
	}

	prompt := BuildPrompt([]*SearchResult{{Chunk: chunks[0], Score: 0.9876}})
	if !strings.Contains(prompt, "[标题: 第三周, 来源: ") {
		t.Fatalf("expected prompt to contain title line, got %q", prompt)
	}
	if !strings.Contains(prompt, "得分: 0.987600") {
		t.Fatalf("expected prompt to contain formatted score, got %q", prompt)
	}
	if !strings.Contains(prompt, "# 第三周") {
		t.Fatalf("expected prompt to contain chunk content, got %q", prompt)
	}
}
