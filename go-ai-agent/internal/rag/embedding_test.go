package rag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// TestEmbeddingClientEmbedsLoadedWorkNoteChunks 测试使用真实 testdata 完成文档加载、chunk 切分和 embedding 请求。
// 输入: 从 `testdata/documents/work_notes_May/五月/第三周` 加载 markdown 文档, 并把前几个 chunk 发送给模拟 Ollama 服务。
// 输出: 返回的 embedding 数量应与输入 chunk 数量一致, 每个 embedding 应有固定维度。
// 示例: `LoadRAGResources(...) -> ChunkDocument(...) -> Embed(ctx, texts)` -> 返回 `[][]float64`。
func TestEmbeddingClientEmbedsLoadedWorkNoteChunks(t *testing.T) {
	documentsDir := filepath.Clean(filepath.Join("..", "..", "testdata", "documents", "work_notes_May", "五月", "第三周"))
	docs, err := LoadRAGResources(documentsDir)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected loaded documents, got 0")
	}

	texts := collectChunkTexts(t, docs, 200, 40, 3)
	model := "qwen3-embedding:0.6b"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Fatalf("expected path /api/embed, got %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("expected json content type, got %q", got)
		}

		var req EmbedReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		if req.Model != model {
			t.Fatalf("expected model %q, got %q", model, req.Model)
		}
		if len(req.Input) != len(texts) {
			t.Fatalf("expected %d input texts, got %d", len(texts), len(req.Input))
		}
		for i, text := range req.Input {
			if text != texts[i] {
				t.Fatalf("input %d expected %q, got %q", i, texts[i], text)
			}
		}

		embeddings := make([][]float64, len(req.Input))
		for i := range req.Input {
			embeddings[i] = []float64{float64(i), float64(len([]rune(req.Input[i]))), 1}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(EmbedResp{EmbeddingsData: embeddings}); err != nil {
			t.Fatalf("encode response failed: %v", err)
		}
	}))
	defer server.Close()

	embedder := NewEmbeddingClient(server.URL, model)
	embeddings, err := embedder.Embed(context.Background(), texts)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(embeddings) != len(texts) {
		t.Fatalf("expected %d embeddings, got %d", len(texts), len(embeddings))
	}
	for i, embedding := range embeddings {
		if len(embedding) != 3 {
			t.Fatalf("embedding %d expected dimension 3, got %d", i, len(embedding))
		}
	}
}

// TestEmbeddingClientReturnsHTTPError 测试 embedding 服务返回非 2xx 时会向调用方返回错误。
// 输入: 模拟 Ollama 服务返回 500。
// 输出: Embed 应返回错误, 且不返回 embedding。
// 示例: `Embed(ctx, []string{"text"})` -> 返回非 nil error。
func TestEmbeddingClientReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusInternalServerError)
	}))
	defer server.Close()

	embedder := NewEmbeddingClient(server.URL, "qwen3-embedding:0.6b")
	embeddings, err := embedder.Embed(context.Background(), []string{"RAG 文档问答"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if embeddings != nil {
		t.Fatalf("expected nil embeddings, got %d", len(embeddings))
	}
}

// collectChunkTexts 从文档列表中收集指定数量的 chunk 文本。
// 输入: `docs` 是文档列表, `size` 和 `overlap` 是 chunk 参数, `limit` 是最多收集数量。
// 输出: 返回非空 chunk 文本列表; 数量不足或切分失败时终止测试。
// 示例: `collectChunkTexts(t, docs, 200, 40, 3)`。
func collectChunkTexts(t *testing.T, docs []*Document, size, overlap, limit int) []string {
	t.Helper()

	var texts []string
	for _, doc := range docs {
		chunks, err := ChunkDocument(doc, size, overlap)
		if err != nil {
			t.Fatalf("chunk document %q failed: %v", doc.SourcePath, err)
		}
		for _, chunk := range chunks {
			if chunk.Content == "" {
				t.Fatalf("expected non-empty chunk content from %q", chunk.SourceFile)
			}
			texts = append(texts, chunk.Content)
			if len(texts) == limit {
				return texts
			}
		}
	}

	t.Fatalf("expected at least %d chunks, got %d", limit, len(texts))
	return nil
}
