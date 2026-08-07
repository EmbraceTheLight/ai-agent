package session

import (
	"strings"
	"testing"
)

// 作用：验证 Parser 能把模型上下文、用户消息和 token_count 归入同一用户轮次。
// 入参：t 为 Go 测试框架提供的测试句柄。
// 返回值：无；断言失败时通过 t.Fatal 或 t.Fatalf 终止测试。
// 示例：go test ./internal/session -run TestParserGroupsTokenCountsByUserTurn。
func TestParserGroupsTokenCountsByUserTurn(t *testing.T) {
	input := strings.Join([]string{
		`{"timestamp":"2026-08-01T10:00:00Z","type":"turn_context","payload":{"model":"gpt-5.5"}}`,
		`{"timestamp":"2026-08-01T10:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"hello   world"}}`,
		`{"timestamp":"2026-08-01T10:00:02Z","type":"event_msg","payload":{"type":"token_count","info":{"context_window":1000,"last_token_usage":{"input_tokens":800,"cached_input_tokens":600,"output_tokens":50,"reasoning_output_tokens":10,"total_tokens":850}},"rate_limits":{"primary":{"used_percent":42.5,"resets_at":1785999600}}}}`,
	}, "\n")

	events, err := (&Parser{}).Parse(strings.NewReader(input), "session.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	ev := events[0]
	if ev.Model != "gpt-5.5" {
		t.Fatalf("model = %q", ev.Model)
	}
	if ev.NonCached != 200 {
		t.Fatalf("noncached = %d", ev.NonCached)
	}
	if ev.ContextUtilization == nil || *ev.ContextUtilization != 80 {
		t.Fatalf("context utilization = %#v", ev.ContextUtilization)
	}
	if ev.Prompt != "hello world" {
		t.Fatalf("prompt = %q", ev.Prompt)
	}
}
