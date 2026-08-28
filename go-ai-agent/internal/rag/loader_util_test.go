package rag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTriliumDocumentLoaderLoadsMarkdownAndTextRecursively 测试目录递归加载能力:
// 只加载 .md/.txt 文件, 跳过不支持的扩展名, 并保留文件内容、标题和绝对路径。
func TestTriliumDocumentLoaderLoadsMarkdownAndTextRecursively(t *testing.T) {
	root := t.TempDir()
	subDir := filepath.Join(root, "notes")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("mkdir test sub dir failed: %v", err)
	}

	files := map[string]string{
		filepath.Join(root, "rag.md"):       "# RAG\nretrieval augmented generation",
		filepath.Join(subDir, "todo.TXT"):   "chunk and embedding",
		filepath.Join(root, "ignored.json"): `{"skip": true}`,
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write test file %q failed: %v", path, err)
		}
	}

	loader := NewTriliumDocumentLoader(map[string]bool{".md": true, ".txt": true}, 0)
	docs, err := loader.Load(root)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}

	gotByFile := make(map[string]*Document, len(docs))
	for _, doc := range docs {
		if doc.SourcePath == "" {
			t.Fatal("expected source path, got empty string")
		}
		if !filepath.IsAbs(doc.SourcePath) {
			t.Fatalf("expected absolute source path, got %q", doc.SourcePath)
		}
		if strings.HasSuffix(doc.SourcePath, "ignored.json") {
			t.Fatalf("expected ignored.json to be skipped, got %q", doc.SourcePath)
		}
		gotByFile[filepath.Base(doc.SourcePath)] = doc
	}

	if gotByFile["rag.md"].Content != files[filepath.Join(root, "rag.md")] {
		t.Fatalf("unexpected markdown content: %q", gotByFile["rag.md"].Content)
	}
	if gotByFile["rag.md"].Title != "RAG" {
		t.Fatalf("unexpected markdown title: %q", gotByFile["rag.md"].Title)
	}
	if gotByFile["todo.TXT"].Content != files[filepath.Join(subDir, "todo.TXT")] {
		t.Fatalf("unexpected text content: %q", gotByFile["todo.TXT"].Content)
	}
	if gotByFile["todo.TXT"].Title != "chunk and embedding" {
		t.Fatalf("unexpected text title: %q", gotByFile["todo.TXT"].Title)
	}
}

// TestTriliumDocumentLoaderLoadsSingleFile 测试传入单个支持文件路径时,
// 能返回一个 Document, 且 SourcePath、Content 和 Title 与源文件一致。
func TestTriliumDocumentLoaderLoadsSingleFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "single.md")
	content := "# Single\nonly one document"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	loader := NewTriliumDocumentLoader(map[string]bool{".md": true}, 0)
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
	if docs[0].SourcePath != wantPath {
		t.Fatalf("expected source path %q, got %q", wantPath, docs[0].SourcePath)
	}
	if docs[0].Content != content {
		t.Fatalf("expected content %q, got %q", content, docs[0].Content)
	}
	if docs[0].Title != "Single" {
		t.Fatalf("expected title %q, got %q", "Single", docs[0].Title)
	}
}

// TestTriliumDocumentLoaderReturnsErrorWhenNoSupportedFiles 测试目录中没有 .md/.txt 文件时,
// 加载器会返回错误, 并且不返回任何文档。
func TestTriliumDocumentLoaderReturnsErrorWhenNoSupportedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "meta.json"), []byte(`{"name":"skip"}`), 0644); err != nil {
		t.Fatalf("write unsupported file failed: %v", err)
	}

	loader := NewTriliumDocumentLoader(map[string]bool{".md": true, ".txt": true}, 0)
	docs, err := loader.Load(root)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if docs != nil {
		t.Fatalf("expected nil documents, got %d", len(docs))
	}
}
