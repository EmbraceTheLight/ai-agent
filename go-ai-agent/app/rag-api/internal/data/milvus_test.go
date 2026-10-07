package data

import (
	"testing"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// TestParseSearchResToRagChunkWithoutScore 验证 Query 返回无 Scores 时仍能解析主键和文档 ID。
// 输入: 模拟 Query 与 Search 两种 ResultSet 主键、分数布局。
// 输出: 断言 chunk ID、document ID 与相似度均正确解析。
// 示例: `go test ./app/rag-api/internal/data -run TestParseSearchResToRagChunkWithoutScore`。
func TestParseSearchResToRagChunkWithoutScore(t *testing.T) {
	res := milvusclient.ResultSet{
		ResultCount: 1,
		Fields: milvusclient.DataSet{
			column.NewColumnInt64("id", []int64{42}),
			column.NewColumnVarChar("document_id", []string{"doc-1"}),
			column.NewColumnVarChar("source_file_path", []string{"notes.md"}),
			column.NewColumnVarChar("chunk_hash", []string{"hash"}),
			column.NewColumnInt32("chunk_index", []int32{2}),
			column.NewColumnVarChar("content", []string{"content"}),
			column.NewColumnInt64("created_at", []int64{100}),
			column.NewColumnInt64("updated_at", []int64{101}),
			column.NewColumnInt32("rune_start_offset", []int32{3}),
			column.NewColumnInt32("rune_end_offset", []int32{10}),
		},
	}
	parsed, err := parseSearchResToRagChunk(&res)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed) != 1 || parsed[0].Chunk.Id != 42 || parsed[0].Chunk.DocumentId != "doc-1" || parsed[0].Score != 0 {
		t.Fatalf("unexpected query result: %+v", parsed)
	}
	res.Scores = []float32{0.8}
	parsed, err = parseSearchResToRagChunk(&res)
	if err != nil || parsed[0].Score != float64(res.Scores[0]) {
		t.Fatalf("unexpected search result: %+v, err=%v", parsed, err)
	}
	res.IDs = column.NewColumnInt64("id", []int64{42})
	res.Fields = res.Fields[1:]
	parsed, err = parseSearchResToRagChunk(&res)
	if err != nil || parsed[0].Chunk.Id != 42 {
		t.Fatalf("unexpected search IDs result: %+v, err=%v", parsed, err)
	}
}
