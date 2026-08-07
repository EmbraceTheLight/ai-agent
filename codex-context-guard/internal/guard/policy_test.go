package guard

import (
	"strings"
	"testing"

	"codex-context-guard/internal/session"
)

// 作用：验证上下文占用达到 High 阈值时 Policy 会输出高风险提示。
// 入参：t 为 Go 测试框架提供的测试句柄。
// 返回值：无；断言失败时通过 t.Fatalf 终止测试。
// 示例：go test ./internal/guard -run TestPolicyWarnsOnHighContext。
func TestPolicyWarnsOnHighContext(t *testing.T) {
	util := 90.0
	msg := (Policy{WarnPercent: 70, HighPercent: 85}).Evaluate(session.TokenEvent{ContextUtilization: &util})
	if !strings.Contains(msg, "high") {
		t.Fatalf("message = %q", msg)
	}
}
