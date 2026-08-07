package report

import (
	"fmt"
	"strings"

	"codex-context-guard/internal/analyzer"
	"codex-context-guard/internal/session"
)

// 作用：把 analyzer.Result 渲染为命令行可阅读的纯文本报告。
// 入参：r 为分析结果；limit 控制高消耗请求、session 和用户轮次的展示行数。
// 返回值：返回完整报告字符串。
// 示例：fmt.Print(Text(result, 20))。
func Text(r analyzer.Result, limit int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Codex Usage Analysis\n")
	fmt.Fprintf(&b, "Scanning: %s\n", r.SessionsDir)
	fmt.Fprintf(&b, "Weekly window start: %s\n", r.WindowStart.Format("2006-01-02 15:04:05"))
	if r.WindowEnd != nil {
		fmt.Fprintf(&b, "Weekly window reset: %s\n", r.WindowEnd.Format("2006-01-02 15:04:05"))
	}
	fmt.Fprintf(&b, "Unique token events in window: %d\n\n", len(r.Events))

	fmt.Fprintf(&b, "1. Weekly quota change points\n")
	for _, c := range r.Changes {
		delta := "-"
		if c.Delta != nil {
			delta = fmt.Sprintf("+%.2f%%", *c.Delta)
		}
		fmt.Fprintf(&b, "%s used=%.2f%% delta=%s input=%d cached=%d noncached=%d output=%d total=%d model=%s session=%s\n",
			c.Time.Format("2006-01-02 15:04:05"), c.Used, delta, c.Input, c.Cached, c.NonCached, c.Output, c.Total, c.Model, c.Session)
	}

	fmt.Fprintf(&b, "\n2. Top %d requests by total tokens\n", limit)
	for _, ev := range topEvents(r.Events, limit, func(a, b session.TokenEvent) bool { return a.Total > b.Total }) {
		writeEvent(&b, ev)
	}

	fmt.Fprintf(&b, "\n3. Top %d requests by NON-CACHED input\n", limit)
	for _, ev := range topEvents(r.Events, limit, func(a, b session.TokenEvent) bool { return a.NonCached > b.NonCached }) {
		writeEvent(&b, ev)
	}

	fmt.Fprintf(&b, "\n4. Summary\n")
	fmt.Fprintf(&b, "Input:      %d\n", r.Summary.Input)
	fmt.Fprintf(&b, "Cached:     %d\n", r.Summary.Cached)
	fmt.Fprintf(&b, "Non-Cached: %d\n", r.Summary.NonCached)
	fmt.Fprintf(&b, "Output:     %d\n", r.Summary.Output)
	fmt.Fprintf(&b, "Reasoning:  %d\n", r.Summary.Reasoning)
	fmt.Fprintf(&b, "Total:      %d\n", r.Summary.Total)
	fmt.Fprintf(&b, "Cache Rate: %.2f%%\n", r.Summary.CacheRate)

	fmt.Fprintf(&b, "\n5. Usage by model\n")
	for _, row := range r.ModelSummaries {
		fmt.Fprintf(&b, "%s requests=%d input=%d cached=%d noncached=%d output=%d total=%d\n",
			row.Model, row.Requests, row.Input, row.Cached, row.NonCached, row.Output, row.Total)
	}

	fmt.Fprintf(&b, "\n6. Usage by session\n")
	for i, row := range r.SessionSummaries {
		if i >= limit {
			break
		}
		fmt.Fprintf(&b, "%s requests=%d input=%d cached=%d noncached=%d output=%d total=%d\n",
			row.Session, row.Requests, row.Input, row.Cached, row.NonCached, row.Output, row.Total)
	}

	fmt.Fprintf(&b, "\n7. Top %d user turns by token usage\n", limit)
	for i, row := range r.TurnSummaries {
		if i >= limit {
			break
		}
		usedStart := floatPtr(row.UsedStart)
		usedEnd := floatPtr(row.UsedEnd)
		quotaDelta := floatPtr(row.QuotaDelta)
		fmt.Fprintf(&b, "%s calls=%d input=%d cached=%d noncached=%d output=%d total=%d usedStart=%s usedEnd=%s quotaDelta=%s models=%s prompt=%s\n",
			row.Time.Format("2006-01-02 15:04:05"), row.Calls, row.Input, row.Cached, row.NonCached, row.Output, row.Total,
			usedStart, usedEnd, quotaDelta, strings.Join(row.Models, ","), row.Prompt)
	}

	fmt.Fprintf(&b, "\nAnalysis complete.\n")
	return b.String()
}

// 作用：向报告构造器写入单个 token 事件的一行摘要。
// 入参：b 为字符串构造器；ev 为待输出 token 事件。
// 返回值：无；该函数直接修改 b。
// 示例：writeEvent(&builder, ev)。
func writeEvent(b *strings.Builder, ev session.TokenEvent) {
	weekly := "-"
	if ev.WeeklyUsed != nil {
		weekly = fmt.Sprintf("%.2f%%", *ev.WeeklyUsed)
	}
	utilization := "-"
	if ev.ContextUtilization != nil {
		utilization = fmt.Sprintf("%.2f%%", *ev.ContextUtilization)
	}
	fmt.Fprintf(b, "%s input=%d cached=%d noncached=%d output=%d reasoning=%d total=%d weekly=%s context=%s model=%s session=%s\n",
		ev.Time.Format("2006-01-02 15:04:05"), ev.Input, ev.Cached, ev.NonCached, ev.Output, ev.Reasoning, ev.Total, weekly, utilization, ev.Model, ev.Session)
}

// 作用：按指定比较函数选出前 N 个 token 事件。
// 入参：events 为候选事件；limit 为最大返回数量；less 定义排序优先级。
// 返回值：返回重新排序后的前 N 个事件；候选数量不足时返回全部事件。
// 示例：rows := topEvents(events, 20, func(a, b session.TokenEvent) bool { return a.Total > b.Total })。
func topEvents(events []session.TokenEvent, limit int, less func(a, b session.TokenEvent) bool) []session.TokenEvent {
	out := append([]session.TokenEvent(nil), events...)
	for i := 0; i < len(out); i++ {
		best := i
		for j := i + 1; j < len(out); j++ {
			if less(out[j], out[best]) {
				best = j
			}
		}
		out[i], out[best] = out[best], out[i]
		if i+1 == limit {
			return out[:limit]
		}
	}
	return out
}

// 作用：把可选 float64 指针格式化为报告字段。
// 入参：v 为待格式化的浮点指针，可为 nil。
// 返回值：nil 返回 "-"；非 nil 返回保留两位小数的字符串。
// 示例：text := floatPtr(row.QuotaDelta)。
func floatPtr(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f", *v)
}
