package rag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadRAGResourcesLoadsMarkdownAndTextRecursively 测试目录递归加载能力:
// 只加载 .md/.txt 文件, 跳过不支持的扩展名, 并保留文件内容和绝对路径。
func TestLoadRAGResourcesLoadsMarkdownAndTextRecursively(t *testing.T) {
	fmt.Println(len("你好"))
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

	docs, err := LoadRAGResources(root)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(docs))
	}

	gotContentByFile := make(map[string]string, len(docs))
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
		gotContentByFile[filepath.Base(doc.SourcePath)] = doc.Content
	}

	if gotContentByFile["rag.md"] != files[filepath.Join(root, "rag.md")] {
		t.Fatalf("unexpected markdown content: %q", gotContentByFile["rag.md"])
	}
	if gotContentByFile["todo.TXT"] != files[filepath.Join(subDir, "todo.TXT")] {
		t.Fatalf("unexpected text content: %q", gotContentByFile["todo.TXT"])
	}
}

// TestLoadRAGResourcesLoadsSingleFile 测试传入单个支持文件路径时,
// 能返回一个 Document, 且 SourcePath 和 Content 与源文件一致。
func TestLoadRAGResourcesLoadsSingleFile(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "single.md")
	content := "# Single\nonly one document"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("write test file failed: %v", err)
	}

	docs, err := LoadRAGResources(filePath)
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
}

// TestLoadRAGResourcesReturnsErrorWhenNoSupportedFiles 测试目录中没有 .md/.txt 文件时,
// 加载器会返回错误, 并且不返回任何文档。
func TestLoadRAGResourcesReturnsErrorWhenNoSupportedFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "meta.json"), []byte(`{"name":"skip"}`), 0644); err != nil {
		t.Fatalf("write unsupported file failed: %v", err)
	}

	docs, err := LoadRAGResources(root)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if docs != nil {
		t.Fatalf("expected nil documents, got %d", len(docs))
	}
}
