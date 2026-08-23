package rag

/* Embedding API */

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

// vector 表示一个 embedding 向量。
// 输入: 通常来自 embedding provider 返回的 `[]float64`。
// 输出: 用于向量库保存和相似度计算。
// 示例: `Vector{0.1, 0.2, 0.3}`。
type vector []float64

// Embedding 存放 chunk 元数据及其向量。
// 输入: `Chunk` 是包含来源文件、序号和内容的 chunk, `Vector` 是该 chunk 对应的 embedding。
// 输出: 作为内存向量库中的一条记录。
// 示例: `Embedding{Chunk: chunk, Vector: Vector{0.1, 0.2}}`。
type Embedding struct {
	Chunk  *Chunk
	Vector vector
}

// Document 表示从本地文件加载得到的一篇文档。
// 输入: `SourcePath` 是源文件路径, `Content` 是完整文本内容。
// 输出: 供 chunker 切分为多个 `Chunk`。
// 示例: `Document{SourcePath: "notes/rag.md", Content: "# RAG"}`。
type Document struct {
	SourcePath string
	Content    string
}

// CosineSimilarity 表示一次向量检索命中的结果。
// 输入: `Embed` 是命中的向量库记录, `Score` 是 query 与记录向量的余弦相似度。
// 输出: 由 `VectorStore.Search` 按相似度降序返回。
// 示例: `CosineSimilarity{Embed: embedding, Score: 0.92}`。
type CosineSimilarity struct {
	Embed *Embedding
	Score float64 // 余弦相似度
}

// SmallRootCosineSimilarity 是按余弦相似度排序的小根堆。
// 输入: 堆中的元素是 `*CosineSimilarity`。
// 输出: 堆顶始终是当前候选集合中分数最低的结果。
// 示例: 用于保留 topK 个最高分检索结果。
type SmallRootCosineSimilarity []*CosineSimilarity

// Len 返回堆中元素数量。
// 输入: 当前堆。
// 输出: 堆长度。
// 示例: `pq.Len()` -> `3`。
func (bigPQ SmallRootCosineSimilarity) Len() int { return len(bigPQ) }

// Less 定义小根堆排序规则。
// 输入: 两个元素下标。
// 输出: 当 i 的分数小于 j 时返回 true。
// 示例: 分数更低的结果会排到堆顶。
func (bigPQ SmallRootCosineSimilarity) Less(i, j int) bool {
	return bigPQ[i].Score < bigPQ[j].Score
}

// Swap 交换堆中两个元素的位置。
// 输入: 两个元素下标。
// 输出: 原地修改堆切片。
// 示例: heap 调整过程中调用。
func (bigPQ SmallRootCosineSimilarity) Swap(i, j int) {
	bigPQ[i], bigPQ[j] = bigPQ[j], bigPQ[i]
}

// Push 向堆中追加一个检索结果。
// 输入: `x` 必须是 `*CosineSimilarity`。
// 输出: 原地扩展堆切片。
// 示例: `heap.Push(&pq, result)`。
func (bigPQ *SmallRootCosineSimilarity) Push(x any) {
	*bigPQ = append(*bigPQ, x.(*CosineSimilarity))
}

// Pop 从堆尾移除并返回一个检索结果。
// 输入: 当前堆。
// 输出: 被移除的 `*CosineSimilarity`。
// 示例: `heap.Pop(&pq)`。
func (bigPQ *SmallRootCosineSimilarity) Pop() any {
	old := *bigPQ
	n := len(old)
	x := old[n-1]
	*bigPQ = old[0 : n-1]
	return x
}
