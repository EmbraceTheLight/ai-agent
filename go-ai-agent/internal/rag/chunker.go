package rag

import (
	"fmt"
	"strings"
)

// Chunk 表示文档切分后的一个片段。
// 输入: 来源文档内容经过 `ChunkDocument` 切分得到。
// 输出: 保存 chunk 文本、来源文件、序号和 rune 偏移。
// 示例: `Chunk{SourceFile: "notes/rag.md", ChunkIndex: 0, Content: "RAG"}`。
type Chunk struct {
	SourceFile string // 源文件路径
	ChunkIndex int    // Chunk 索引
	Content    string // 分块内容

	// 分块起始偏移量, 该偏移量为形式上的偏移量, 并非按照字节偏移.
	// 对于存在 rune 的文本, 该偏移量可能无法直接用于切片操作
	// 如 "你好x", 其对应的 x 的 Start 偏移量为 2, 而非字节偏移 6。
	Start int
	End   int // 分块终止偏移量
}

// ChunkDocument 按固定 rune 数量和 overlap 将文档切分为多个 chunk。
// 输入: `doc` 是待切分文档, `size` 是每个 chunk 的最大 rune 数, `overlap` 是相邻 chunk 的重叠 rune 数。
// 输出: 返回带来源和偏移信息的 chunk 列表; 参数非法时返回错误。
// 示例: `ChunkDocument(doc, 500, 100)` -> 返回按 500 rune 切分且重叠 100 rune 的片段。
func ChunkDocument(doc *Document, size, overlap int) ([]*Chunk, error) {
	var chunks []*Chunk
	if doc == nil {
		return nil, fmt.Errorf("doc 不能为 nil")
	}
	if overlap < 0 {
		return nil, fmt.Errorf("overlap 不能小于 0, 当前 overlap: %d", overlap)
	}
	if size < 0 {
		return nil, fmt.Errorf("chunk size 不能小于 0, 当前 chunk size: %d", size)
	}
	if size <= overlap {
		return nil, fmt.Errorf("要切分的 chunk 大小不能小于 overlap 大小. size 大小: %d, overlap 大小: %d", size, overlap)
	}

	runes := []rune(doc.Content)
	sb := strings.Builder{}
	for start, chunkIdx := 0, 0; start < len(runes); start, chunkIdx = start+size-overlap, chunkIdx+1 {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := &Chunk{
			SourceFile: doc.SourcePath,
			Start:      start,
			End:        end,
			ChunkIndex: chunkIdx,
		}
		for i := start; i < end; i++ {
			sb.WriteRune(runes[i])
		}

		chunk.Content = sb.String()
		sb.Reset()
		chunks = append(chunks, chunk)

		// chunk 已切分完毕
		if end == len(runes) {
			break
		}
	}

	return chunks, nil
}
