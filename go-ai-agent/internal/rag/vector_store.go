package rag

import (
	"container/heap"
	"errors"
	"fmt"
	"math"
)

// VectorStore 定义 RAG 阶段最小内存向量库能力。
// 输入: chunk embedding 和 query embedding。
// 输出: 支持写入向量记录并按相似度检索 topK。
// 示例: `store := NewVectorStore(); store.Add(vec, chunk); store.Search(queryVec, 3)`。
type VectorStore interface {
	// Add 保存一个 chunk 及其 embedding 向量。
	// 输入: `vector` 是 chunk embedding, `chunk` 是 chunk 文本。
	// 输出: 成功返回 nil; 向量非法时返回错误。
	// 示例: `Add(vector{1, 0}, "chunk text")`。
	Add(vector vector, chunk string) error

	// Search 检索与 queryVector 最相似的 topK 个 chunk。
	// 输入: `queryVector` 是问题 embedding, `topK` 是返回数量。
	// 输出: 返回按相似度降序排列的结果。
	// 示例: `Search(vector{1, 0}, 2)`。
	Search(queryVector vector, topK int) ([]*CosineSimilarity, error)
}

type defaultVectorStore []*Embedding

// NewVectorStore 创建内存向量库。
// 输入: 无。
// 输出: 返回一个实现 `VectorStore` 的内存存储。
// 示例: `store := NewVectorStore()`。
func NewVectorStore() VectorStore {
	return &defaultVectorStore{}
}

// Add 向内存向量库中添加一条 chunk 向量记录。
// 输入: `vector` 是 chunk 的 embedding 向量, `chunk` 是原始 chunk 文本。
// 输出: 成功时返回 nil; 向量为空时返回错误。
// 示例: `store.Add(vector{1, 0}, "RAG 文档问答")`。
func (v *defaultVectorStore) Add(vector vector, chunk string) error {
	if len(vector) == 0 {
		return errors.New("插入的向量维度为 0")
	}

	*v = append(*v, &Embedding{
		Chunk:  chunk,
		vector: vector,
	})
	return nil
}

// Search 在内存向量库中检索与 queryVector 最相似的 topK 个 chunk。
// 输入: `queryVector` 是问题的 embedding 向量, `topK` 是需要返回的结果数量。
// 输出: 返回按余弦相似度降序排列的检索结果; 参数非法或向量无法比较时返回错误。
// 示例: `store.Search(vector{1, 0}, 3)` -> 返回分数最高的 3 个 chunk。
func (v *defaultVectorStore) Search(queryVector vector, topK int) ([]*CosineSimilarity, error) {
	if topK <= 0 {
		return nil, fmt.Errorf("topK 必须大于 0")
	}
	var smallPQ SmallRootCosineSimilarity
	ret := make([]*CosineSimilarity, topK)

	heap.Init(&smallPQ)
	for i := 0; i < len(*v); i++ {
		cs, err := cosineSimilarity(queryVector, (*v)[i].vector)
		if err != nil {
			return nil, err
		}
		heap.Push(&smallPQ, &CosineSimilarity{
			Embed: (*v)[i],
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
		ret[i] = heap.Pop(&smallPQ).(*CosineSimilarity)
	}
	return ret, nil
}

// cosineSimilarity 计算两个向量的余弦相似度
// 输入: `a` 和 `b` 是两个维度一致的非零向量。
// 输出: 返回二者的余弦相似度; 维度不一致、空向量或零向量时返回错误。
// 示例: `cosineSimilarity(vector{1, 0}, vector{1, 0})` -> `1`。
func cosineSimilarity(a, b vector) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("向量长度不一致. a: %d, b: %d", len(a), len(b))
	}
	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("存在长度为 0 的向量")
	}
	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
	}
	normA = getVectorLength(a)
	normB = getVectorLength(b)
	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("向量长度为0")
	}
	return dotProduct / (normA * normB), nil
}

// getVectorLength 计算向量的欧几里得长度。
// 输入: `vector` 是待计算的向量。
// 输出: 返回 `sqrt(sum(x_i^2))`; 空向量返回 0。
// 示例: `getVectorLength(vector{3, 4})` -> `5`。
func getVectorLength(vector vector) float64 {
	if len(vector) == 0 {
		return 0
	}
	var sum float64
	for _, v := range vector {
		sum += v * v
	}
	return math.Sqrt(sum)
}
