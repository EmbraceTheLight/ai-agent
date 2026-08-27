package rag

import (
	"fmt"
	"strings"
	"time"
)

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
		timestamp := time.Now().UnixMilli()
		chunk := &Chunk{
			Title:           doc.Title,
			SourceFile:      doc.SourcePath,
			RuneStartOffset: start,
			RuneEndOffset:   end,
			ChunkIndex:      chunkIdx,
			CreatedAt:       timestamp,
			UpdatedAt:       timestamp,
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
