package analyzer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 作用：验证 AnalyzeFiles 会去重重复 token 事件，并优先保留带周额度字段的版本。
// 入参：t 为 Go 测试框架提供的测试句柄。
// 返回值：无；断言失败时通过 t.Fatal 或 t.Fatalf 终止测试。
// 示例：go test ./internal/analyzer -run TestAnalyzeFilesDeduplicatesAndPrefersRateLimit。
func TestAnalyzeFilesDeduplicatesAndPrefersRateLimit(t *testing.T) {
	dir := t.TempDir()
	lineNoRate := `{"timestamp":"2026-08-01T10:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":20,"total_tokens":120}}}}`
	lineWithRate := `{"timestamp":"2026-08-01T10:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":100,"cached_input_tokens":40,"output_tokens":20,"total_tokens":120}},"rate_limits":{"primary":{"used_percent":12,"resets_at":1785999600}}}}`
	a := filepath.Join(dir, "a.jsonl")
	b := filepath.Join(dir, "b.jsonl")
	if err := os.WriteFile(a, []byte(lineNoRate+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(lineWithRate+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := AnalyzeFiles(dir, []string{a, b}, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("events = %d, want 1", len(result.Events))
	}
	if result.Events[0].WeeklyUsed == nil || *result.Events[0].WeeklyUsed != 12 {
		t.Fatalf("weekly used not preserved: %#v", result.Events[0].WeeklyUsed)
	}
	if result.Summary.NonCached != 60 {
		t.Fatalf("noncached summary = %d", result.Summary.NonCached)
	}
}
