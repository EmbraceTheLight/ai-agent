package rag

import "context"

// Embedder 定义文本向量化能力。
// 输入: 一组 chunk 文本。
// 输出: 与输入文本一一对应的 embedding 向量。
// 示例: `embedder.Embed(ctx, []string{"RAG"})`。
type Embedder interface {
	// Embed 为多段文本生成 embedding 向量。
	// 输入: `ctx` 是请求上下文, `chunks` 是待向量化文本列表。
	// 输出: 返回向量列表; provider 调用失败时返回错误。
	// 示例: `Embed(ctx, []string{"chunk"})` -> `[][]float64`。
	Embed(ctx context.Context, chunks []string) ([][]float64, error)
}

// VectorStore 定义 RAG 阶段最小内存向量库能力。
// 输入: chunk embedding 和 query embedding。
// 输出: 支持写入向量记录并按相似度检索 topK。
// 示例: `store := NewVectorStore(); store.Add(vec, chunk); store.Search(queryVec, 3)`。
type VectorStore interface {
	// Add 保存一个 chunk 及其 embedding 向量。
	// 输入: `Vector` 是 chunk embedding, `chunk` 是包含来源信息的 chunk。
	// 输出: 成功返回 nil; 向量非法或 chunk 为空时返回错误。
	// 示例: `Add(Vector{1, 0}, chunk)`。
	Add(vector Vector, chunk *Chunk) error

	// Search 检索与 queryVector 最相似的 topK 个 chunk。
	// 输入: `queryVector` 是问题 embedding, `topK` 是返回数量。
	// 输出: 返回按相似度降序排列的结果。
	// 示例: `Search(Vector{1, 0}, 2)`。
	Search(queryVector Vector, topK int) ([]*SearchResult, error)
}

// DocumentLoader 定义文档加载接口。
type DocumentLoader interface {
	// Load 解析并返回 path 下所有符合条件的文档。
	// 输入: `path` 是待加载的文件或目录路径。
	// 输出: 返回解析后的文档列表; 读取、过滤或解析失败时返回错误。
	// 示例: `Load("testdata/documents")`。
	Load(path string) ([]*Document, error)
}
