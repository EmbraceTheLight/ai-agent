# internal/guard

`internal/guard` 是实时 Context Guard 逻辑，面向 Codex Hooks 调用场景。

## 包职责

- 从 stdin 读取 Hook JSON 输入。
- 解析 `transcript_path` 指向的当前 transcript。
- 获取最近一次 token_count 事件的上下文占用率。
- 根据阈值策略输出 Hook 支持的 `systemMessage` JSON。
- 记录 `PostCompact` 中的自动 compact 事件。
- 根据短期自动 compact 次数生成更高风险提示。

## 核心类型

- `HookInput`：Codex Hook 输入字段。
- `Policy`：上下文预警阈值和自动 compact 统计窗口。
- `State`：本地持久化状态。
- `SessionState`：单个 session 的自动 compact 记录。

## 常用入口

```go
policy := guard.Policy{
    WarnPercent:         70,
    HighPercent:         85,
    AutoCompactsPerDay:  3,
    AutoCompactInterval: 24 * time.Hour,
}

err := guard.Run(os.Stdin, os.Stdout, "", statePath, policy)
```

## Hook 行为

- 输入为空且未指定 transcript 时，`Run` 安静返回，不输出内容。
- transcript 不存在时，视为当前还没有可分析 token 事件，不输出内容。
- 风险未达到阈值时，不输出 JSON，避免干扰 Codex UI。
- 达到阈值时，输出形如 `{"systemMessage":"..."}` 的 Hook 响应。

## 状态文件

状态文件默认保存自动 compact 记录。JSON 损坏时会降级为空状态，不阻断 Hook 主流程。

