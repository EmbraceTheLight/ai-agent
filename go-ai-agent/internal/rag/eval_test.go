package rag

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEvalCheckClassifiesResults 验证 EvalCheck 能区分错误 source、低分结果和合法结果。
// 输入: 临时目录中的期望文件、非期望文件和三条带分数的检索结果。
// 输出: 断言三类结果数量正确, 且存在合法结果时 EvalShouldAnswer 为 true。
// 示例: `go test ./internal/rag -run TestEvalCheckClassifiesResults`。
func TestEvalCheckClassifiesResults(t *testing.T) {
	root := t.TempDir()
	expectedPath := filepath.Join(root, "expected.md")
	otherPath := filepath.Join(root, "other.md")
	for _, path := range []string{expectedPath, otherPath} {
		if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
			t.Fatalf("write test file %q: %v", path, err)
		}
	}

	checkConf := &EvalCheckConfig{
		ExpectSources: []string{expectedPath},
		MinScore:      0.5,
		ShouldAnswer:  true,
	}
	results := []*SearchResult{
		{Chunk: &Chunk{SourceFile: expectedPath, ChunkIndex: 0}, Score: 0.8},
		{Chunk: &Chunk{SourceFile: expectedPath, ChunkIndex: 1}, Score: 0.4},
		{Chunk: &Chunk{SourceFile: otherPath, ChunkIndex: 0}, Score: 0.9},
	}

	got := checkConf.EvalCheck(results)
	if len(got.Legal) != 1 {
		t.Fatalf("expected one legal result, got %d", len(got.Legal))
	}
	if len(got.ScoreTooLow) != 1 {
		t.Fatalf("expected one low-score result, got %d", len(got.ScoreTooLow))
	}
	if len(got.NotFindInExpectSources) != 1 {
		t.Fatalf("expected one unexpected-source result, got %d", len(got.NotFindInExpectSources))
	}
	if !got.EvalShouldAnswer {
		t.Fatal("expected EvalShouldAnswer to be true when legal result exists")
	}
}
