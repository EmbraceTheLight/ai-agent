# cmd/codex-usage

`cmd/codex-usage` 是命令行入口包，负责把用户输入的子命令和参数分发到内部业务包。

## 包职责

- 提供 `codex-usage analyze` 离线分析入口。
- 提供 `codex-usage guard` Codex Hook 实时检测入口。
- 解析命令行 flag，并把参数转换成 `internal/analyzer`、`internal/guard` 和 `internal/report` 需要的调用形式。
- 推导默认路径，例如 `$CODEX_HOME/sessions` 和 `$CODEX_HOME/context-guard-state.json`。

## 子命令

### analyze

用于扫描 Codex session transcript，并输出 token 用量、缓存命中、周额度变化、模型汇总、session 汇总和用户轮次排行。

```powershell
codex-usage analyze --sessions "$env:USERPROFILE\.codex\sessions" --since-days 7 --top 20
```

### guard

用于 Codex Hooks。默认从 stdin 读取 Hook JSON，通过 `transcript_path` 找到当前 transcript，并在风险达到阈值时输出 `systemMessage`。

```powershell
codex-usage guard --warn 70 --high 85 --auto-compact-limit 3
```

调试时也可以显式指定 transcript：

```powershell
codex-usage guard --transcript "C:\Users\me\.codex\sessions\active.jsonl"
```

## 设计边界

- 本包只做 CLI 参数处理，不直接解析 transcript。
- transcript 兼容逻辑放在 `internal/session`。
- token 聚合逻辑放在 `internal/analyzer`。
- Hook 输入输出、阈值策略和状态文件放在 `internal/guard`。
- 文本报告渲染放在 `internal/report`。

