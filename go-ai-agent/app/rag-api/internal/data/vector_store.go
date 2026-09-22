package data

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"math"
	"sync"
)

type memoryVectorStore struct {
	mu         sync.RWMutex
	embeddings []*biz.Embedding
}

// Add 向内存向量库中添加一条 chunk 向量记录。
// 输入: `Vector` 是 chunk 的 embedding 向量, `chunk` 是带来源文件和序号的 chunk。
// 输出: 成功时返回 nil; 向量为空或 chunk 为 nil 时返回错误。
// 示例: `store.Add(ctx, Vector{1, 0}, &Chunk{SourceFile: "notes/rag.md", ChunkIndex: 0, Content: "RAG"})`。
func (v *memoryVectorStore) Add(ctx context.Context, vector biz.Vector, chunk *biz.Chunk) error {
	if len(vector) == 0 {
		return errors.New("插入的向量维度为 0")
	}
	if chunk == nil {
		return errors.New("传入的 chunk 为 nil")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.embeddings = append(v.embeddings, &biz.Embedding{
		Chunk:  chunk,
		Vector: vector,
	})
	return nil
}

// Search 在内存向量库中检索与 queryVector 最相似的 topK 个 chunk。
// 输入: `queryVector` 是问题的 embedding 向量, `topK` 是需要返回的结果数量。
// 输出: 返回按余弦相似度降序排列的检索结果; 参数非法或向量无法比较时返回错误。
// 示例: `store.Search(ctx, Vector{1, 0}, 3)` -> 返回分数最高的 3 个 chunk。
func (v *memoryVectorStore) Search(ctx context.Context, queryVector biz.Vector, topK int) ([]*biz.SearchResult, error) {
	if topK <= 0 {
		return nil, fmt.Errorf("topK 必须大于 0")
	}
	v.mu.RLock()
	embeddings := append([]*biz.Embedding(nil), v.embeddings...)
	v.mu.RUnlock()

	var smallPQ biz.SearchResultMinHeap
	ret := make([]*biz.SearchResult, topK)

	heap.Init(&smallPQ)
	for i := 0; i < len(embeddings); i++ {
		cs, err := cosineSimilarity(queryVector, embeddings[i].Vector)
		if err != nil {
			return nil, err
		}
		heap.Push(&smallPQ, &biz.SearchResult{
			Chunk: embeddings[i].Chunk,
			Score: cs,
		})
		if len(smallPQ) > topK {
			heap.Pop(&smallPQ)
		}
	}
	if len(smallPQ) < topK {
		return nil, fmt.Errorf("topK 大于向量库大小")
	}

	// 从小根堆中取 topK, 逆序放入 ret 中, 使 ret 中的元素为按照余弦相似度降序排序
	for i := topK - 1; i >= 0; i-- {
		ret[i] = heap.Pop(&smallPQ).(*biz.SearchResult)
	}
	return ret, nil
}

// cosineSimilarity 计算两个向量的余弦相似度。
// 输入: `a` 和 `b` 是两个维度一致的非零向量。
// 输出: 返回二者的余弦相似度; 维度不一致、空向量或零向量时返回错误。
// 示例: `cosineSimilarity(Vector{1, 0}, Vector{1, 0})` -> `1`。
func cosineSimilarity(a, b biz.Vector) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("向量长度不一致. a: %d, b: %d", len(a), len(b))
	}
	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("存在长度为 0 的向量")
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i]) * float64(b[i])
	}
	normA = getVectorLength(a)
	normB = getVectorLength(b)
	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("向量长度为0")
	}
	return dotProduct / (normA * normB), nil
}

// getVectorLength 计算向量的欧几里得长度。
// 输入: `Vector` 是待计算的向量。
// 输出: 返回 `sqrt(sum(x_i^2)`); 空向量返回 0。
// 示例: `getVectorLength(Vector{3, 4})` -> `5`。
func getVectorLength(vector biz.Vector) float64 {
	if len(vector) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vector {
		sum += float64(v) * float64(v)
	}
	return math.Sqrt(sum)
}
