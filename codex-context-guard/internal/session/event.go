package session

import "time"

// TokenEvent 表示从 Codex transcript 中提取的一次 token_count 事件。
type TokenEvent struct {
	// EventKey 是去重用的稳定键，通常由时间戳和 token 数组合而成。
	EventKey string
	// TurnKey 是用户轮次聚合键，用于把同一轮内多次模型调用归到一起。
	TurnKey string
	// TurnTime 是触发该轮的用户消息时间；缺失时 analyzer 会回退到事件时间。
	TurnTime time.Time
	// Prompt 是用户消息的精简文本，便于报告展示高消耗轮次。
	Prompt string
	// Time 是 token_count 事件自身的时间戳。
	Time time.Time
	// Session 是 transcript 文件名推导出的 session 标识。
	Session string
	// Model 是事件发生时的模型名称，来自最近一次 turn_context。
	Model string
	// Input 是本次模型调用的输入 token 数。
	Input int64
	// Cached 是命中缓存的输入 token 数。
	Cached int64
	// NonCached 是未命中缓存的输入 token 数，等于 Input-Cached 且不会小于 0。
	NonCached int64
	// Output 是本次模型调用的输出 token 数。
	Output int64
	// Reasoning 是 reasoning output token 数。
	Reasoning int64
	// Total 是 transcript 中记录的本次调用总 token 数。
	Total int64
	// WeeklyUsed 是 Codex 周额度 used_percent；字段缺失时为 nil。
	WeeklyUsed *float64
	// WeeklyRemaining 是根据 WeeklyUsed 反推的剩余百分比；字段缺失时为 nil。
	WeeklyRemaining *float64
	// ResetTime 是周额度窗口重置时间；字段缺失时为 nil。
	ResetTime *time.Time
	// ContextWindow 是模型上下文窗口 token 上限；字段缺失时为 0。
	ContextWindow int64
	// ContextUtilization 是 Input/ContextWindow 得到的上下文占用百分比；无法计算时为 nil。
	ContextUtilization *float64
	// File 是事件来源 transcript 路径，用于排查兼容性问题。
	File string
}

// HasRateLimit 作用：判断该 token 事件是否携带周额度 used_percent 信息。
// 入参：无；方法接收者 e 为待检查的 TokenEvent。
// 返回值：如果 WeeklyUsed 不为 nil 返回 true，否则返回 false。
// 示例：if ev.HasRateLimit() { fmt.Println(*ev.WeeklyUsed) }。
func (e TokenEvent) HasRateLimit() bool {
	return e.WeeklyUsed != nil
}
