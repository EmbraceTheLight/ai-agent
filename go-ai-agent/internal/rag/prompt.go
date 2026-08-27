package rag

import (
	"fmt"
	"strings"
)

// BuildPrompt 根据 topK 相关 chunk 构造 prompt 信息
// 输入: `topKCS` 是 topK 相关 chunk 的列表。
// 输出: `prompt` 是构造好的 prompt 信息。
// 示例: `BuildPrompt([]*SearchResult{cs1, cs2, cs3})` --> <基于 cs1、cs2 和 cs3 的 prompt 文本>。
func BuildPrompt(topKCS []*SearchResult) string {
	var sb strings.Builder
	writeLine(&sb, "只能基于下面给出的资料回答，并给出相关引用。当资料不足时要说不知道，明确禁止编造回答与来源。")
	writeLine(&sb, "下面是相关资料, 已经按照相关性得分进行了降序排序:")

	for _, cs := range topKCS {
		writeLine(&sb, fmt.Sprintf("[来源: %s#chunk-%d, 得分: %f]", cs.Chunk.SourceFile, cs.Chunk.ChunkIndex, cs.Score))
		writeLine(&sb, cs.Chunk.Content)
	}
	return sb.String()
}
func writeLine(sb *strings.Builder, str string) {
	sb.WriteString(str)
	sb.WriteString("\n")
}
