package rag

import (
	"go-ai-agent/internal/utils"
)

/* Embedding API */

// embedderClient 是基于 HTTP 的 embedding 客户端实现。
// 输入: 保存 embedding 模型名称和通用 HTTP Client。
// 输出: 通过 `Embed` 方法调用外部 embedding 服务。
// 示例: `NewEmbeddingClient("http://localhost:11434", "qwen3-embedding:0.6b")`。
type embedderClient struct {
	model      string
	httpClient *utils.HttpClient
}

// EmbedReq 描述一次 embedding 请求。
// 输入: `Model` 是 embedding 模型名称, `Input` 是待向量化的文本列表。
// 输出: 作为 JSON 请求体发送给 embedding provider。
// 示例: `EmbedReq{Model: "qwen3-embedding:0.6b", Input: []string{"RAG"}}`。
type EmbedReq struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbedResp 描述 embedding provider 返回的响应结构。
// 输入: 由 HTTP JSON 响应反序列化得到。
// 输出: `EmbeddingsData` 保存与输入文本一一对应的向量。
// 示例: `EmbedResp{EmbeddingsData: [][]float64{{0.1, 0.2}}}`。
type EmbedResp struct {
	EmbeddingsData [][]float64 `json:"embeddings"`
}

/* other data structure */

// Vector 表示一个 embedding 向量。
// 输入: 通常来自 embedding provider 返回的 `[]float64`。
// 输出: 用于向量库保存和相似度计算。
// 示例: `Vector{0.1, 0.2, 0.3}`。
type Vector []float64

// Embedding 存放 chunk 元数据及其向量。
// 输入: `Chunk` 是包含来源文件、序号和内容的 chunk, `Vector` 是该 chunk 对应的 embedding。
// 输出: 作为内存向量库中的一条记录。
// 示例: `Embedding{Chunk: chunk, Vector: Vector{0.1, 0.2}}`。
type Embedding struct {
	Chunk  *Chunk
	Vector Vector
}

// Document 表示从本地文件加载得到的一篇文档。
// 输入: `SourcePath` 是源文件路径, `Content` 是完整文本内容。
// 输出: 供 chunker 切分为多个 `Chunk`。
// 示例: `Document{SourcePath: "notes/rag.md", Content: "# RAG"}`。
type Document struct {
	Title      string
	SourcePath string
	Content    string
}

// Chunk 表示文档切分后的一个片段。
// 输入: 来源文档内容经过 `ChunkDocument` 切分得到。
// 输出: 保存 chunk 文本、来源文件、序号和 rune 偏移。
// 示例: `Chunk{SourceFile: "notes/rag.md", ChunkIndex: 0, Content: "RAG"}`。
type Chunk struct {
	SourceFile string // 源文件路径
	Title      string // markdown 文件标题, Trilium 中只有标题格式为 `# <title>`
	ChunkIndex int    // Chunk 索引
	Content    string // 分块内容
	CreatedAt  int64  // 创建时间时间戳
	UpdatedAt  int64  // 更新时间时间戳

	// 分块起始偏移量, 该偏移量为形式上的偏移量, 并非按照字节偏移.
	// 对于存在 rune 的文本, 该偏移量可能无法直接用于切片操作
	// 如 "你好x", 其对应的 x 的 RuneStartOffset 偏移量为 2, 而非字节偏移 6。
	RuneStartOffset int
	RuneEndOffset   int // 分块终止偏移量
}

// SearchResult 表示一次向量检索命中的结果。
// 输入: `Chunk` 是命中的向量库记录, `Score` 是 query 与记录向量的余弦相似度。
// 输出: 由 `VectorStore.Search` 按相似度降序返回。
// 示例: `SearchResult{Chunk: chunk, Score: 0.92}`。
type SearchResult struct {
	Chunk *Chunk
	Score float64 // 余弦相似度
}

// SearchResultMinHeap 是按余弦相似度排序的小根堆。
// 输入: 堆中的元素是 `*SearchResult`。
// 输出: 堆顶始终是当前候选集合中分数最低的结果。
// 示例: 用于保留 topK 个最高分检索结果。
type SearchResultMinHeap []*SearchResult
