package biz

import (
	"fmt"
	"strings"
	"time"
)

// Split 按固定 rune 数量和 overlap 将文档切分为多个 chunk。
// 输入: `doc` 是待切分文档, `config` 保存每个 chunk 的最大 rune 数和相邻 chunk 的重叠 rune 数。
// 输出: 返回带来源和偏移信息的 chunk 列表; 参数非法时返回错误。
// 示例: `doc.Split(ChunkConfig{Size: 500, Overlap: 100})` -> 返回按 500 rune 切分且重叠 100 rune 的片段。
func (doc *Document) Split(config ChunkConfig) ([]*Chunk, error) {
	var chunks []*Chunk
	if doc == nil {
		return nil, fmt.Errorf("doc 不能为 nil")
	}
	if config.Overlap < 0 {
		return nil, fmt.Errorf("overlap 不能小于 0, 当前 overlap: %d", config.Overlap)
	}
	if config.Size < 0 {
		return nil, fmt.Errorf("chunk size 不能小于 0, 当前 chunk size: %d", config.Size)
	}
	if config.Size <= config.Overlap {
		return nil, fmt.Errorf("要切分的 chunk 大小不能小于 overlap 大小. size 大小: %d, overlap 大小: %d", config.Size, config.Overlap)
	}

	runes := []rune(doc.Content)
	sb := strings.Builder{}
	for start, chunkIdx := 0, 0; start < len(runes); start, chunkIdx = start+config.Size-config.Overlap, chunkIdx+1 {
		end := start + config.Size
		if end > len(runes) {
			end = len(runes)
		}
		timestamp := time.Now().UnixMilli()
		chunk := &Chunk{
			Title:           doc.Title,
			SourceFile:      doc.SourcePath,
			RuneStartOffset: int64(start),
			RuneEndOffset:   int64(end),
			ChunkIndex:      int64(chunkIdx),
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
