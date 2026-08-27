package rag

// Len 返回堆中元素数量。
// 输入: 当前堆。
// 输出: 堆长度。
// 示例: `pq.Len()` -> `3`。
func (minHeap SearchResultMinHeap) Len() int { return len(minHeap) }

// Less 定义小根堆排序规则。
// 输入: 两个元素下标。
// 输出: 当 i 的分数小于 j 时返回 true。
// 示例: 分数更低的结果会排到堆顶。
func (minHeap SearchResultMinHeap) Less(i, j int) bool {
	return minHeap[i].Score < minHeap[j].Score
}

// Swap 交换堆中两个元素的位置。
// 输入: 两个元素下标。
// 输出: 原地修改堆切片。
// 示例: heap 调整过程中调用。
func (minHeap SearchResultMinHeap) Swap(i, j int) {
	minHeap[i], minHeap[j] = minHeap[j], minHeap[i]
}

// Push 向堆中追加一个检索结果。
// 输入: `x` 必须是 `*SearchResult`。
// 输出: 原地扩展堆切片。
// 示例: `heap.Push(&pq, result)`。
func (minHeap *SearchResultMinHeap) Push(x any) {
	*minHeap = append(*minHeap, x.(*SearchResult))
}

// Pop 从堆尾移除并返回一个检索结果。
// 输入: 当前堆。
// 输出: 被移除的 `*SearchResult`。
// 示例: `heap.Pop(&pq)`。
func (minHeap *SearchResultMinHeap) Pop() any {
	old := *minHeap
	n := len(old)
	x := old[n-1]
	*minHeap = old[0 : n-1]
	return x
}
