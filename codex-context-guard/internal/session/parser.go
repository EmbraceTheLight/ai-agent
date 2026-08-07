package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var whitespace = regexp.MustCompile(`\s+`)

// Parser 维护 transcript 流式解析过程中的上下文状态。
type Parser struct {
	// currentModel 是最近一次 turn_context 事件声明的模型名称。
	currentModel string
	// currentTurnKey 是最近一次用户消息生成的轮次键。
	currentTurnKey string
	// currentTurnTime 是最近一次用户消息的时间。
	currentTurnTime time.Time
	// currentPrompt 是最近一次用户消息的短文本摘要。
	currentPrompt string
}

// ParseFile 作用：打开并解析一个 Codex transcript JSONL 文件。
// 入参：path 为 transcript 文件路径。
// 返回值：返回提取到的 TokenEvent 切片；打开或扫描失败时返回 error。
// 示例：events, err := ParseFile(`C:\Users\me\.codex\sessions\session.jsonl`)。
func ParseFile(path string) ([]TokenEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	p := Parser{}
	return p.Parse(f, path)
}

// Parse 作用：从任意 io.Reader 流式解析 JSONL transcript，并提取 token_count 事件。
// 入参：r 为 JSONL 输入流；sourcePath 为来源路径或名称，用于生成 session 名与错误信息。
// 返回值：返回按读取顺序提取的 TokenEvent；扫描器底层出错时返回 error。
// 示例：events, err := (&Parser{}).Parse(strings.NewReader(jsonl), "session.jsonl")。
func (p *Parser) Parse(r io.Reader, sourcePath string) ([]TokenEvent, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)

	var events []TokenEvent
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			// transcript 是非稳定内部格式，单行损坏或未知时跳过，避免整个分析失败。
			continue
		}

		ev, ok := p.parseLine(raw, sourcePath)
		if ok {
			events = append(events, ev)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%s: line %d: %w", sourcePath, lineNo, err)
	}
	return events, nil
}

// parseLine 作用：解析单行 transcript JSON 对象，并在遇到 token_count 时生成 TokenEvent。
// 入参：raw 为单行 JSON 解码后的对象；sourcePath 为来源路径或名称。
// 返回值：第一个返回值为解析出的 TokenEvent；第二个返回值表示该行是否产生事件。
// 示例：ev, ok := p.parseLine(raw, "session.jsonl")。
func (p *Parser) parseLine(raw map[string]any, sourcePath string) (TokenEvent, bool) {
	typ, _ := stringAt(raw, "type")
	payload, _ := mapAt(raw, "payload")
	timestamp, ok := timeAt(raw, "timestamp")
	if !ok {
		timestamp = time.Now()
	}

	if typ == "turn_context" {
		// turn_context 通常早于 token_count，用来给后续事件补上模型名称。
		if model, ok := stringAt(payload, "model"); ok {
			p.currentModel = model
		}
		return TokenEvent{}, false
	}

	payloadType, _ := stringAt(payload, "type")
	if typ == "event_msg" && payloadType == "user_message" {
		p.currentTurnKey = timestamp.Format(time.RFC3339Nano)
		p.currentTurnTime = timestamp
		p.currentPrompt = firstString(payload, "message", "text")
		if p.currentPrompt == "" {
			p.currentPrompt = "(message text unavailable)"
		} else {
			// 报告只需要定位高消耗轮次，压缩空白并截断可以避免输出整段长 prompt。
			p.currentPrompt = whitespace.ReplaceAllString(p.currentPrompt, " ")
			p.currentPrompt = strings.TrimSpace(p.currentPrompt)
			if len([]rune(p.currentPrompt)) > 120 {
				p.currentPrompt = string([]rune(p.currentPrompt)[:120]) + "..."
			}
		}
		return TokenEvent{}, false
	}

	if typ != "event_msg" || payloadType != "token_count" {
		// 未知事件属于兼容性边界，按设计忽略而不是失败。
		return TokenEvent{}, false
	}

	last, ok := mapAt(raw, "payload", "info", "last_token_usage")
	if !ok {
		return TokenEvent{}, false
	}

	input := intAt(last, "input_tokens")
	cached := intAt(last, "cached_input_tokens")
	output := intAt(last, "output_tokens")
	reasoning := intAt(last, "reasoning_output_tokens")
	total := intAt(last, "total_tokens")
	if input == 0 && output == 0 {
		return TokenEvent{}, false
	}

	nonCached := input - cached
	if nonCached < 0 {
		// 某些未来字段变更可能导致 cached 大于 input；守住统计下限，避免负数污染报告。
		nonCached = 0
	}

	var weeklyUsed *float64
	var weeklyRemaining *float64
	if used, ok := floatAt(raw, "payload", "rate_limits", "primary", "used_percent"); ok {
		weeklyUsed = &used
		remaining := 100 - used
		weeklyRemaining = &remaining
	}

	var resetTime *time.Time
	if resetUnix, ok := intAnyAt(raw, "payload", "rate_limits", "primary", "resets_at"); ok && resetUnix > 0 {
		t := time.Unix(resetUnix, 0)
		resetTime = &t
	}

	contextWindow := firstIntPath(raw,
		// Codex transcript 字段不是稳定接口，这里集中兼容已见过的多个上下文窗口字段名。
		[]string{"payload", "info", "context_window"},
		[]string{"payload", "info", "context_window_tokens"},
		[]string{"payload", "info", "model_context_window"},
		[]string{"payload", "info", "total_token_limit"},
		[]string{"payload", "info", "max_context_tokens"},
		[]string{"payload", "info", "last_token_usage", "context_window"},
	)

	var utilization *float64
	if contextWindow > 0 {
		u := float64(input) / float64(contextWindow) * 100
		utilization = &u
	}

	turnKey := p.currentTurnKey
	if turnKey == "" {
		// 如果 token_count 之前没有用户消息，则仍保留事件并归入来源文件的未归因轮次。
		turnKey = "unattributed|" + strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	}

	return TokenEvent{
		EventKey:           fmt.Sprintf("%s|%d|%d|%d|%d|%d", timestamp.Format(time.RFC3339Nano), input, cached, output, reasoning, total),
		TurnKey:            turnKey,
		TurnTime:           p.currentTurnTime,
		Prompt:             p.currentPrompt,
		Time:               timestamp,
		Session:            strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath)),
		Model:              p.currentModel,
		Input:              input,
		Cached:             cached,
		NonCached:          nonCached,
		Output:             output,
		Reasoning:          reasoning,
		Total:              total,
		WeeklyUsed:         weeklyUsed,
		WeeklyRemaining:    weeklyRemaining,
		ResetTime:          resetTime,
		ContextWindow:      contextWindow,
		ContextUtilization: utilization,
		File:               sourcePath,
	}, true
}

// 作用：按路径读取 map 中的字符串字段。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回字符串值以及是否成功读取。
// 示例：model, ok := stringAt(raw, "payload", "model")。
func stringAt(m map[string]any, path ...string) (string, bool) {
	v, ok := anyAt(m, path...)
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// 作用：按候选键顺序读取第一个非空字符串。
// 入参：m 为待读取对象；keys 为同一层级下的候选字段名。
// 返回值：返回第一个成功读取的字符串；全部失败时返回空字符串。
// 示例：prompt := firstString(payload, "message", "text")。
func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := stringAt(m, key); ok {
			return s
		}
	}
	return ""
}

// 作用：按路径读取 map 中的嵌套对象。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回 map[string]any 以及是否成功读取。
// 示例：payload, ok := mapAt(raw, "payload")。
func mapAt(m map[string]any, path ...string) (map[string]any, bool) {
	v, ok := anyAt(m, path...)
	if !ok {
		return nil, false
	}
	out, ok := v.(map[string]any)
	return out, ok
}

// 作用：按路径读取 RFC3339Nano 时间字符串并解析为 time.Time。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回解析后的时间以及是否成功。
// 示例：ts, ok := timeAt(raw, "timestamp")。
func timeAt(m map[string]any, path ...string) (time.Time, bool) {
	s, ok := stringAt(m, path...)
	if !ok {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// 作用：按路径读取整数；读取失败时使用 0 作为降级值。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回 int64 数值；字段缺失、类型不兼容或解析失败时返回 0。
// 示例：input := intAt(lastUsage, "input_tokens")。
func intAt(m map[string]any, path ...string) int64 {
	v, _ := intAnyAt(m, path...)
	return v
}

// 作用：按路径读取整数，并兼容 JSON number、float64、int64 和数字字符串。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回 int64 数值以及是否成功读取。
// 示例：resetUnix, ok := intAnyAt(raw, "payload", "rate_limits", "primary", "resets_at")。
func intAnyAt(m map[string]any, path ...string) (int64, bool) {
	v, ok := anyAt(m, path...)
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

// 作用：按路径读取浮点数，并兼容 JSON number、float64、int64 和数字字符串。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回 float64 数值以及是否成功读取。
// 示例：used, ok := floatAt(raw, "payload", "rate_limits", "primary", "used_percent")。
func floatAt(m map[string]any, path ...string) (float64, bool) {
	v, ok := anyAt(m, path...)
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(n, 64)
		return f, err == nil
	default:
		return 0, false
	}
}

// 作用：按候选路径顺序读取第一个可用整数，适配 transcript 字段名变化。
// 入参：m 为待读取对象；paths 为多组嵌套字段路径。
// 返回值：返回第一个成功读取的 int64；全部失败时返回 0。
// 示例：window := firstIntPath(raw, []string{"payload", "info", "context_window"})。
func firstIntPath(m map[string]any, paths ...[]string) int64 {
	for _, path := range paths {
		if v, ok := intAnyAt(m, path...); ok {
			return v
		}
	}
	return 0
}

// 作用：按嵌套路径读取任意类型字段，是本包所有安全取值辅助函数的底层入口。
// 入参：m 为待读取对象；path 为一段或多段嵌套字段名。
// 返回值：返回读取到的值以及是否存在且路径类型匹配。
// 示例：value, ok := anyAt(raw, "payload", "info", "last_token_usage")。
func anyAt(m map[string]any, path ...string) (any, bool) {
	var cur any = m
	for _, key := range path {
		asMap, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = asMap[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}
