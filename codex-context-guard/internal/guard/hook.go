package guard

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codex-context-guard/internal/session"
)

// HookInput 表示 Codex Hook 通过 stdin 传入的最小输入字段集合。
type HookInput struct {
	// SessionID 是 Codex 当前 session 标识，用于 compact 状态归档。
	SessionID string `json:"session_id"`
	// TranscriptPath 是当前 transcript JSONL 路径，是 guard 读取 token 状态的主要入口。
	TranscriptPath string `json:"transcript_path"`
	// CWD 是 Hook 触发时 Codex 所在工作目录；当前版本保留但不参与策略判断。
	CWD string `json:"cwd"`
	// Model 是 Hook 触发时的模型名称；当前版本保留但优先使用 transcript 中的模型。
	Model string `json:"model"`
	// HookEventName 是 Hook 事件名，例如 UserPromptSubmit 或 PostCompact。
	HookEventName string `json:"hook_event_name"`
	// Trigger 是 compact 触发来源，例如 auto 或 manual。
	Trigger string `json:"trigger"`
}

// hookOutput 表示写回 Codex Hook stdout 的 JSON 响应。
type hookOutput struct {
	// SystemMessage 是 Codex UI/event stream 会展示的系统提示；为空时省略。
	SystemMessage string `json:"systemMessage,omitempty"`
}

// 作用：执行一次 Context Guard Hook 检查，必要时向 stdout 写出 systemMessage JSON。
// 入参：stdin 为 Hook JSON 输入；stdout 为 Hook 响应输出；transcriptOverride 为命令行强制指定的 transcript；statePath 为状态文件；policy 为风险策略。
// 返回值：成功返回 nil；读取状态、解析 transcript 或保存状态失败时返回 error。
// 示例：err := Run(os.Stdin, os.Stdout, "", statePath, Policy{WarnPercent: 70, HighPercent: 85})。
func Run(stdin io.Reader, stdout io.Writer, transcriptOverride string, statePath string, policy Policy) error {
	input, _ := readHookInput(stdin)
	if transcriptOverride != "" {
		input.TranscriptPath = transcriptOverride
	}
	if input.TranscriptPath == "" {
		return nil
	}

	state, err := LoadState(statePath)
	if err != nil {
		return err
	}

	var message string
	eventName := strings.ToLower(input.HookEventName)
	if eventName == "postcompact" && strings.EqualFold(input.Trigger, "auto") {
		// 自动 compact 是独立风险信号：即使当前上下文刚下降，频繁自动压缩也说明会话过长。
		state.RecordAutoCompact(input.SessionID, time.Now())
		if state.RecentAutoCompacts(input.SessionID, time.Now(), policy.AutoCompactInterval) >= policy.AutoCompactsPerDay {
			message = "Context Guard: this session has auto-compacted repeatedly in the last 24h. Create a handoff and continue in a new session."
		}
	}

	if message == "" {
		ev, ok, err := latestTokenEvent(input.TranscriptPath)
		if err != nil {
			return err
		}
		if ok {
			message = policy.Evaluate(ev)
		}
	}

	if err := state.Save(statePath); err != nil {
		return err
	}
	if message == "" {
		return nil
	}
	return json.NewEncoder(stdout).Encode(hookOutput{SystemMessage: message})
}

// 作用：读取并容错解析 Codex Hook stdin JSON。
// 入参：r 为 Hook stdin 数据源，最多读取 4 MiB 以避免异常大输入拖慢 guard。
// 返回值：返回 HookInput；输入为空时返回零值；JSON 解析失败或读取失败时返回 error。
// 示例：input, err := readHookInput(strings.NewReader(`{"transcript_path":"session.jsonl"}`))。
func readHookInput(r io.Reader) (HookInput, error) {
	data, err := io.ReadAll(io.LimitReader(r, 4*1024*1024))
	if err != nil {
		return HookInput{}, err
	}
	data = []byte(strings.TrimSpace(string(data)))
	if len(data) == 0 {
		return HookInput{}, nil
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return HookInput{}, err
	}
	out := HookInput{
		SessionID:      stringAny(raw, "session_id"),
		TranscriptPath: stringAny(raw, "transcript_path"),
		CWD:            stringAny(raw, "cwd"),
		Model:          stringAny(raw, "model"),
		HookEventName:  stringAny(raw, "hook_event_name"),
		Trigger:        firstStringAny(raw, "trigger", "compact_trigger"),
	}
	if out.Trigger == "" {
		// 兼容 compact.trigger 这种嵌套结构，避免 Hook 字段形态微调导致自动 compact 丢失。
		if compact, ok := raw["compact"].(map[string]any); ok {
			out.Trigger = firstStringAny(compact, "trigger", "type")
		}
	}
	return out, nil
}

// 作用：读取 transcript 并返回最后一个 token_count 事件，作为当前上下文状态。
// 入参：path 为 transcript JSONL 路径。
// 返回值：返回事件、是否找到事件以及错误；文件不存在或无事件时 ok=false 且 error=nil。
// 示例：ev, ok, err := latestTokenEvent(`C:\Users\me\.codex\sessions\active.jsonl`)。
func latestTokenEvent(path string) (session.TokenEvent, bool, error) {
	events, err := session.ParseFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return session.TokenEvent{}, false, nil
		}
		return session.TokenEvent{}, false, err
	}
	if len(events) == 0 {
		return session.TokenEvent{}, false, nil
	}
	return events[len(events)-1], true, nil
}

// 作用：把 map 中指定 key 的值转换为字符串。
// 入参：m 为待读取对象；key 为字段名。
// 返回值：字段缺失或 nil 时返回空字符串；字符串以外类型使用 fmt.Sprint 转换。
// 示例：sessionID := stringAny(raw, "session_id")。
func stringAny(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// 作用：按候选 key 顺序读取第一个非空字符串。
// 入参：m 为待读取对象；keys 为候选字段名。
// 返回值：返回第一个非空字符串；全部缺失时返回空字符串。
// 示例：trigger := firstStringAny(raw, "trigger", "compact_trigger")。
func firstStringAny(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := stringAny(m, key); s != "" {
			return s
		}
	}
	return ""
}

// 作用：确保状态文件所在目录存在。
// 入参：path 为状态文件路径。
// 返回值：目录已存在或无需创建时返回 nil；创建失败时返回 error。
// 示例：err := ensureStateDir(`C:\Users\me\.codex\context-guard-state.json`)。
func ensureStateDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
