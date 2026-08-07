package guard

import (
	"encoding/json"
	"os"
	"time"
)

// State 是 Context Guard 在本地保存的跨 Hook 调用状态。
type State struct {
	// Sessions 按 session_id 保存每个会话的 compact 相关状态。
	Sessions map[string]SessionState `json:"sessions"`
}

// SessionState 是单个 Codex session 的 guard 状态。
type SessionState struct {
	// AutoCompacts 记录该 session 自动 compact 的发生时间。
	AutoCompacts []time.Time `json:"auto_compacts"`
}

// 作用：从 JSON 文件加载 Context Guard 状态，并对缺失或损坏文件做容错降级。
// 入参：path 为状态文件路径。
// 返回值：返回 State；读取文件失败且不是“不存在”时返回 error，JSON 损坏时返回空状态和 nil。
// 示例：state, err := LoadState(`C:\Users\me\.codex\context-guard-state.json`)。
func LoadState(path string) (State, error) {
	state := State{Sessions: map[string]SessionState{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return state, err
	}
	if len(data) == 0 {
		return state, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		// 状态文件只用于风险增强，不应因为本地 JSON 损坏阻断 Hook 主流程。
		return State{Sessions: map[string]SessionState{}}, nil
	}
	if state.Sessions == nil {
		state.Sessions = map[string]SessionState{}
	}
	return state, nil
}

// 作用：把当前 guard 状态保存为格式化 JSON。
// 入参：path 为状态文件路径；方法接收者 s 为待保存状态。
// 返回值：保存成功返回 nil；创建目录、JSON 编码或写文件失败时返回 error。
// 示例：err := state.Save(`C:\Users\me\.codex\context-guard-state.json`)。
func (s *State) Save(path string) error {
	if err := ensureStateDir(path); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// 作用：记录某个 session 发生了一次自动 compact。
// 入参：sessionID 为 Codex session 标识，空值会归为 "unknown"；at 为 compact 发生时间。
// 返回值：无；该方法会原地更新 State。
// 示例：state.RecordAutoCompact("session-123", time.Now())。
func (s *State) RecordAutoCompact(sessionID string, at time.Time) {
	if sessionID == "" {
		sessionID = "unknown"
	}
	row := s.Sessions[sessionID]
	row.AutoCompacts = append(row.AutoCompacts, at)
	s.Sessions[sessionID] = row
}

// 作用：统计某个 session 在指定时间窗口内的自动 compact 次数，并顺便清理窗口外旧记录。
// 入参：sessionID 为 Codex session 标识，空值会归为 "unknown"；now 为统计基准时间；within 为回看窗口长度。
// 返回值：返回窗口内保留下来的自动 compact 次数。
// 示例：count := state.RecentAutoCompacts("session-123", time.Now(), 24*time.Hour)。
func (s *State) RecentAutoCompacts(sessionID string, now time.Time, within time.Duration) int {
	if sessionID == "" {
		sessionID = "unknown"
	}
	row := s.Sessions[sessionID]
	cutoff := now.Add(-within)
	kept := row.AutoCompacts[:0]
	for _, at := range row.AutoCompacts {
		if !at.Before(cutoff) {
			// 只保留窗口内记录，避免长期运行后状态文件无界增长。
			kept = append(kept, at)
		}
	}
	row.AutoCompacts = kept
	s.Sessions[sessionID] = row
	return len(kept)
}
