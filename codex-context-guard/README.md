# codex-context-guard

`codex-context-guard` 提供 Go 实现的 `codex-usage` 命令行工具，用于分析 Codex transcript 的 token 用量，并在当前 session 接近上下文压力阈值时输出 Hook 预警。

项目第一版只使用 Go 标准库。由于 Codex transcript 内部格式不是稳定公共接口，所有 JSONL 兼容逻辑都集中在 `internal/session`，避免字段变化影响分析和 guard 业务逻辑。

## 命令

### 离线分析

```powershell
codex-usage analyze --sessions "$env:USERPROFILE\.codex\sessions" --since-days 7
```

报告内容包括：

- 周额度窗口和 rate-limit 变化点。
- input tokens、cached input tokens、non-cached input tokens、output tokens、reasoning tokens 和 total tokens。
- cache rate。
- 按模型统计。
- 按 session 统计。
- 按用户轮次统计。
- 高总 token 请求排行。
- 高非缓存输入请求排行。

### 实时 Guard

```powershell
codex-usage guard
```

`guard` 设计用于 Codex Hooks。它从 stdin 读取 Hook JSON，通过 `transcript_path` 检查当前 transcript，并且只在需要提示时输出带 `systemMessage` 的 Hook JSON 响应。

支持的输入字段保持最小且容错：

- `session_id`
- `transcript_path`
- `cwd`
- `model`
- `hook_event_name`
- `trigger` 或 `compact.trigger`

## 推荐 Hook

建议优先监听：

- `UserPromptSubmit`：新一轮用户请求开始前检查当前上下文。
- `PostCompact`：记录自动 compact，作为独立风险信号。
- `Stop`：后续可用于更新本地统计状态。

不建议默认在每个 `PostToolUse` 上做完整 transcript 分析。复杂任务中的工具调用非常频繁，guard 不应成为额外 I/O 压力来源。

## 包结构

- `cmd/codex-usage`：CLI 子命令入口。
- `internal/session`：transcript JSONL 兼容解析。
- `internal/analyzer`：离线用量统计和聚合。
- `internal/guard`：Hook 输入输出、风险策略和状态记录。
- `internal/report`：文本报告渲染。

## 开发

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go test ./...
```

如果当前机器的 Go SDK 不在 `PATH`，可以直接使用本机已安装的 SDK 路径执行 `go test`。

