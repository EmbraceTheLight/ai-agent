package analyzer

import (
	"io/fs"
	"path/filepath"
	"sort"
	"time"

	"codex-context-guard/internal/session"
)

// Result 是一次离线 transcript 分析的完整结果。
type Result struct {
	// SessionsDir 是被扫描的 Codex sessions 根目录。
	SessionsDir string
	// Since 是候选事件的最早时间，用于减少历史文件扫描范围。
	Since time.Time
	// WindowStart 是最终统计窗口起点，优先由周额度 reset 时间反推。
	WindowStart time.Time
	// WindowEnd 是周额度窗口结束时间；无法从 transcript 推导时为 nil。
	WindowEnd *time.Time
	// Events 是去重并按统计窗口过滤后的 token 事件。
	Events []session.TokenEvent
	// Changes 是周额度 used_percent 变化点序列。
	Changes []QuotaChange
	// Summary 是全局 token 汇总。
	Summary Summary
	// ModelSummaries 是按模型聚合后的 token 汇总。
	ModelSummaries []ModelSummary
	// TurnSummaries 是按用户轮次聚合后的 token 汇总。
	TurnSummaries []TurnSummary
	// SessionSummaries 是按 session 聚合后的 token 汇总。
	SessionSummaries []SessionSummary
}

// Summary 是 token 指标的通用汇总结构。
type Summary struct {
	// Input 是输入 token 总数。
	Input int64
	// Cached 是缓存命中的输入 token 总数。
	Cached int64
	// NonCached 是未命中缓存的输入 token 总数。
	NonCached int64
	// Output 是输出 token 总数。
	Output int64
	// Reasoning 是 reasoning output token 总数。
	Reasoning int64
	// Total 是 transcript 记录的总 token 数。
	Total int64
	// CacheRate 是 Cached/Input 的百分比；Input 为 0 时保持 0。
	CacheRate float64
}

// QuotaChange 表示周额度 used_percent 的一个变化点。
type QuotaChange struct {
	// Time 是变化点事件时间。
	Time time.Time
	// Used 是该事件记录的周额度已使用百分比。
	Used float64
	// Delta 是相对上一变化点的 used_percent 增量；首个变化点为 nil。
	Delta *float64
	// Input 是该变化点所在请求的输入 token 数。
	Input int64
	// Cached 是该变化点所在请求的缓存输入 token 数。
	Cached int64
	// NonCached 是该变化点所在请求的非缓存输入 token 数。
	NonCached int64
	// Output 是该变化点所在请求的输出 token 数。
	Output int64
	// Total 是该变化点所在请求的总 token 数。
	Total int64
	// Model 是该变化点所在请求使用的模型。
	Model string
	// Session 是该变化点所属 session。
	Session string
}

// ModelSummary 表示单个模型维度的用量汇总。
type ModelSummary struct {
	// Model 是模型名称；未知模型会归为 "(unknown)"。
	Model string
	// Requests 是该模型产生的 token_count 请求数。
	Requests int
	// Input 是该模型输入 token 总数。
	Input int64
	// Cached 是该模型缓存输入 token 总数。
	Cached int64
	// NonCached 是该模型非缓存输入 token 总数。
	NonCached int64
	// Output 是该模型输出 token 总数。
	Output int64
	// Reasoning 是该模型 reasoning output token 总数。
	Reasoning int64
	// Total 是该模型总 token 数。
	Total int64
}

// SessionSummary 表示单个 Codex session 维度的用量汇总。
type SessionSummary struct {
	// Session 是 transcript 文件名推导出的 session 标识。
	Session string
	// Requests 是该 session 中的 token_count 请求数。
	Requests int
	// Input 是该 session 输入 token 总数。
	Input int64
	// Cached 是该 session 缓存输入 token 总数。
	Cached int64
	// NonCached 是该 session 非缓存输入 token 总数。
	NonCached int64
	// Output 是该 session 输出 token 总数。
	Output int64
	// Total 是该 session 总 token 数。
	Total int64
}

// TurnSummary 表示单个用户轮次内多次模型调用的聚合结果。
type TurnSummary struct {
	// Time 是该用户轮次开始时间；缺失时回退到首个 token 事件时间。
	Time time.Time
	// Calls 是该轮内部模型调用次数。
	Calls int
	// Input 是该轮输入 token 总数。
	Input int64
	// Cached 是该轮缓存输入 token 总数。
	Cached int64
	// NonCached 是该轮非缓存输入 token 总数。
	NonCached int64
	// Output 是该轮输出 token 总数。
	Output int64
	// Reasoning 是该轮 reasoning output token 总数。
	Reasoning int64
	// Total 是该轮总 token 数。
	Total int64
	// UsedStart 是该轮内首个周额度 used_percent；缺失时为 nil。
	UsedStart *float64
	// UsedEnd 是该轮内最后一个周额度 used_percent；缺失时为 nil。
	UsedEnd *float64
	// QuotaDelta 是 UsedEnd-UsedStart；缺失任一端时为 nil。
	QuotaDelta *float64
	// Models 是该轮涉及的去重模型列表，保留首次出现顺序。
	Models []string
	// Prompt 是该轮用户消息摘要，用于报告定位高消耗轮次。
	Prompt string
}

// AnalyzeDirectory 作用：扫描目录下最近修改的 JSONL transcript，并生成完整用量分析结果。
// 入参：root 为 sessions 根目录；since 为候选文件和事件的最早时间。
// 返回值：返回 Result；目录遍历、文件信息读取或后续分析失败时返回 error。
// 示例：result, err := AnalyzeDirectory(`C:\Users\me\.codex\sessions`, time.Now().AddDate(0, 0, -7))。
func AnalyzeDirectory(root string, since time.Time) (Result, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().Before(since) {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return AnalyzeFiles(root, paths, since)
}

// AnalyzeFiles 作用：分析指定 transcript 文件列表，完成解析、去重、周窗口过滤和多维汇总。
// 入参：root 为报告中的 sessions 根目录；paths 为待分析 JSONL 文件路径；since 为事件最早时间。
// 返回值：返回 Result；任一文件解析失败时返回 error。
// 示例：result, err := AnalyzeFiles(root, []string{"a.jsonl", "b.jsonl"}, since)。
func AnalyzeFiles(root string, paths []string, since time.Time) (Result, error) {
	var events []session.TokenEvent
	for _, path := range paths {
		parsed, err := session.ParseFile(path)
		if err != nil {
			return Result{}, err
		}
		for _, ev := range parsed {
			if !ev.Time.Before(since) {
				events = append(events, ev)
			}
		}
	}

	events = dedup(events)
	windowStart, windowEnd := detectWindow(events, since)
	if windowEnd != nil {
		// 一旦 transcript 中出现周额度 reset 时间，优先按真实周窗口裁剪，而不是仅使用 since。
		filtered := events[:0]
		for _, ev := range events {
			if !ev.Time.Before(windowStart) && !ev.Time.After(*windowEnd) {
				filtered = append(filtered, ev)
			}
		}
		events = filtered
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Time.Before(events[j].Time) })

	result := Result{
		SessionsDir: root,
		Since:       since,
		WindowStart: windowStart,
		WindowEnd:   windowEnd,
		Events:      events,
	}
	result.Summary = summarize(events)
	result.Changes = quotaChanges(events)
	result.ModelSummaries = modelSummaries(events)
	result.SessionSummaries = sessionSummaries(events)
	result.TurnSummaries = turnSummaries(events)
	return result, nil
}

// 作用：按 EventKey 去重，并在重复事件中优先保留带 rate-limit 信息的版本。
// 入参：events 为解析出的 token 事件。
// 返回值：返回去重后按时间升序排列的事件切片。
// 示例：unique := dedup(events)。
func dedup(events []session.TokenEvent) []session.TokenEvent {
	best := make(map[string]session.TokenEvent, len(events))
	for _, ev := range events {
		prev, ok := best[ev.EventKey]
		if !ok || (!prev.HasRateLimit() && ev.HasRateLimit()) {
			// rollout/session 可能保存重复历史；带额度字段的副本信息更完整，所以优先保留。
			best[ev.EventKey] = ev
		}
	}
	out := make([]session.TokenEvent, 0, len(best))
	for _, ev := range best {
		out = append(out, ev)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	return out
}

// 作用：根据最近的 reset 时间推导 Codex 周额度窗口。
// 入参：events 为去重后的 token 事件；fallback 为无法推导窗口时的默认起点。
// 返回值：返回窗口起点和可选窗口终点；未发现 reset 时间时终点为 nil。
// 示例：start, end := detectWindow(events, since)。
func detectWindow(events []session.TokenEvent, fallback time.Time) (time.Time, *time.Time) {
	var latest *session.TokenEvent
	for i := range events {
		if events[i].ResetTime == nil {
			continue
		}
		if latest == nil || events[i].Time.After(latest.Time) {
			latest = &events[i]
		}
	}
	if latest == nil {
		return fallback, nil
	}
	end := *latest.ResetTime
	return end.AddDate(0, 0, -7), &end
}

// 作用：把一组 token 事件汇总为全局 token 指标。
// 入参：events 为待汇总事件。
// 返回值：返回 Summary，包含输入、缓存、输出、reasoning、总量和缓存率。
// 示例：summary := summarize(events)。
func summarize(events []session.TokenEvent) Summary {
	var s Summary
	for _, ev := range events {
		s.Input += ev.Input
		s.Cached += ev.Cached
		s.NonCached += ev.NonCached
		s.Output += ev.Output
		s.Reasoning += ev.Reasoning
		s.Total += ev.Total
	}
	if s.Input > 0 {
		s.CacheRate = float64(s.Cached) / float64(s.Input) * 100
	}
	return s
}

// 作用：提取周额度 used_percent 发生变化的事件序列。
// 入参：events 为按时间排序的 token 事件。
// 返回值：返回 QuotaChange 切片；没有 rate-limit 字段时返回空切片。
// 示例：changes := quotaChanges(events)。
func quotaChanges(events []session.TokenEvent) []QuotaChange {
	var changes []QuotaChange
	var previous *session.TokenEvent
	for i := range events {
		ev := events[i]
		if ev.WeeklyUsed == nil {
			continue
		}
		if previous == nil {
			changes = append(changes, newQuotaChange(ev, nil))
			previous = &events[i]
			continue
		}
		if *ev.WeeklyUsed != *previous.WeeklyUsed {
			delta := *ev.WeeklyUsed - *previous.WeeklyUsed
			changes = append(changes, newQuotaChange(ev, &delta))
			previous = &events[i]
		}
	}
	return changes
}

// 作用：把一个携带 rate-limit 信息的 token 事件转换为 QuotaChange 行。
// 入参：ev 为 token 事件；delta 为相对上一变化点的 used_percent 增量，可为 nil。
// 返回值：返回填充好的 QuotaChange。
// 示例：change := newQuotaChange(ev, &delta)。
func newQuotaChange(ev session.TokenEvent, delta *float64) QuotaChange {
	return QuotaChange{
		Time:      ev.Time,
		Used:      *ev.WeeklyUsed,
		Delta:     delta,
		Input:     ev.Input,
		Cached:    ev.Cached,
		NonCached: ev.NonCached,
		Output:    ev.Output,
		Total:     ev.Total,
		Model:     ev.Model,
		Session:   ev.Session,
	}
}

// 作用：按模型名称聚合 token 用量，并按总 token 降序排序。
// 入参：events 为待聚合 token 事件。
// 返回值：返回 ModelSummary 切片。
// 示例：rows := modelSummaries(events)。
func modelSummaries(events []session.TokenEvent) []ModelSummary {
	byModel := map[string]*ModelSummary{}
	for _, ev := range events {
		model := ev.Model
		if model == "" {
			model = "(unknown)"
		}
		row := byModel[model]
		if row == nil {
			row = &ModelSummary{Model: model}
			byModel[model] = row
		}
		row.Requests++
		row.Input += ev.Input
		row.Cached += ev.Cached
		row.NonCached += ev.NonCached
		row.Output += ev.Output
		row.Reasoning += ev.Reasoning
		row.Total += ev.Total
	}
	out := make([]ModelSummary, 0, len(byModel))
	for _, row := range byModel {
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// 作用：按 session 聚合 token 用量，并按总 token 降序排序。
// 入参：events 为待聚合 token 事件。
// 返回值：返回 SessionSummary 切片。
// 示例：rows := sessionSummaries(events)。
func sessionSummaries(events []session.TokenEvent) []SessionSummary {
	bySession := map[string]*SessionSummary{}
	for _, ev := range events {
		row := bySession[ev.Session]
		if row == nil {
			row = &SessionSummary{Session: ev.Session}
			bySession[ev.Session] = row
		}
		row.Requests++
		row.Input += ev.Input
		row.Cached += ev.Cached
		row.NonCached += ev.NonCached
		row.Output += ev.Output
		row.Total += ev.Total
	}
	out := make([]SessionSummary, 0, len(bySession))
	for _, row := range bySession {
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}

// 作用：按用户轮次聚合 token 用量、调用次数、模型列表和周额度变化。
// 入参：events 为待聚合 token 事件。
// 返回值：返回 TurnSummary 切片，按总 token 降序排序。
// 示例：rows := turnSummaries(events)。
func turnSummaries(events []session.TokenEvent) []TurnSummary {
	byTurn := map[string][]session.TokenEvent{}
	for _, ev := range events {
		byTurn[ev.TurnKey] = append(byTurn[ev.TurnKey], ev)
	}

	var out []TurnSummary
	for _, rows := range byTurn {
		sort.Slice(rows, func(i, j int) bool { return rows[i].Time.Before(rows[j].Time) })
		first := rows[0]
		turnTime := first.TurnTime
		if turnTime.IsZero() {
			// 兼容未捕获 user_message 的 token_count，仍用首个事件时间展示该轮。
			turnTime = first.Time
		}
		row := TurnSummary{Time: turnTime, Calls: len(rows), Prompt: first.Prompt}
		if row.Prompt == "" {
			row.Prompt = "(unattributed)"
		}

		seenModels := map[string]bool{}
		for _, ev := range rows {
			row.Input += ev.Input
			row.Cached += ev.Cached
			row.NonCached += ev.NonCached
			row.Output += ev.Output
			row.Reasoning += ev.Reasoning
			row.Total += ev.Total
			if ev.Model != "" && !seenModels[ev.Model] {
				row.Models = append(row.Models, ev.Model)
				seenModels[ev.Model] = true
			}
			if ev.WeeklyUsed != nil {
				if row.UsedStart == nil {
					start := *ev.WeeklyUsed
					row.UsedStart = &start
				}
				// 持续覆盖 UsedEnd，可得到该轮最后一次可见额度读数。
				end := *ev.WeeklyUsed
				row.UsedEnd = &end
			}
		}
		if row.UsedStart != nil && row.UsedEnd != nil {
			delta := *row.UsedEnd - *row.UsedStart
			row.QuotaDelta = &delta
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	return out
}
