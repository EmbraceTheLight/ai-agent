package guard

import (
	"fmt"
	"time"

	"codex-context-guard/internal/session"
)

// Policy 定义 Context Guard 的风险阈值。
type Policy struct {
	// WarnPercent 是输出普通上下文预警的占用百分比阈值。
	WarnPercent float64
	// HighPercent 是输出高风险上下文预警的占用百分比阈值。
	HighPercent float64
	// AutoCompactsPerDay 是统计窗口内触发 critical 自动 compact 提示的次数阈值。
	AutoCompactsPerDay int
	// AutoCompactInterval 是自动 compact 统计窗口，例如 24 小时。
	AutoCompactInterval time.Duration
}

// 作用：根据 token 事件中的上下文占用百分比生成风险提示。
// 入参：ev 为最近一次 token_count 事件。
// 返回值：返回适合写入 Hook systemMessage 的英文提示；无需提示时返回空字符串。
// 示例：msg := Policy{WarnPercent: 70, HighPercent: 85}.Evaluate(ev)。
func (p Policy) Evaluate(ev session.TokenEvent) string {
	if ev.ContextUtilization == nil {
		return ""
	}
	percent := *ev.ContextUtilization
	if percent >= p.HighPercent {
		return fmt.Sprintf("Context Guard: context usage is high at %.1f%%. Create a handoff and continue in a new session soon.", percent)
	}
	if percent >= p.WarnPercent {
		return fmt.Sprintf("Context Guard: context usage is %.1f%%. This session is getting long; consider compacting or preparing a handoff.", percent)
	}
	return ""
}
