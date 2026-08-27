package rag

import (
	"path/filepath"
	"testing"
)

// TestTriliumDocumentLoaderParsesTitle 测试 Trilium 文档加载器会解析 Markdown 第一行标题。
// 输入: testdata 中一个首行为 `# 第三周` 的 Trilium 导出 Markdown 文件。
// 输出: 返回的 Document 应包含文件内容、绝对路径和解析得到的标题。
// 示例: `NewTriliumDocumentLoader(nil, 0).Load(filePath)`。
func TestTriliumDocumentLoaderParsesTitle(t *testing.T) {
	filePath := filepath.Clean(filepath.Join("..", "..", "testdata", "documents", "work_notes_May", "五月", "第三周.md"))

	loader := NewTriliumDocumentLoader(nil, 0)
	docs, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	wantPath, err := filepath.Abs(filePath)
	if err != nil {
		t.Fatalf("get absolute path failed: %v", err)
	}
	doc := docs[0]
	if doc.SourcePath != wantPath {
		t.Fatalf("expected source path %q, got %q", wantPath, doc.SourcePath)
	}
	if doc.Content == "" {
		t.Fatal("expected non-empty content, got empty string")
	}
	if doc.Title != "第三周" {
		t.Fatalf("expected title %q, got %q", "第三周", doc.Title)
	}
}

// TestTriliumDocumentLoaderFallbacksTitleToFileName 测试 Trilium 文档缺少标题时回退到文件名。
// 输入: testdata 中一个空 Markdown 文件。
// 输出: 返回的 Document 标题应为去掉 `.md` 后缀的文件名。
// 示例: `NewTriliumDocumentLoader(nil, 0).Load(filePath)`。
func TestTriliumDocumentLoaderFallbacksTitleToFileName(t *testing.T) {
	filePath := filepath.Clean(filepath.Join("..", "..", "testdata", "documents", "work_notes_May", "五月", "第三周", "周末.md"))

	loader := NewTriliumDocumentLoader(nil, 0)
	docs, err := loader.Load(filePath)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("expected 1 document, got %d", len(docs))
	}

	doc := docs[0]
	if doc.Title != "周末" {
		t.Fatalf("expected title %q, got %q", "周末", doc.Title)
	}
	if doc.Content != "" {
		t.Fatalf("expected empty content, got %q", doc.Content)
	}
}
