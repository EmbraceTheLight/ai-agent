# internal/session

`internal/session` 负责读取 Codex transcript JSONL，并把其中稳定可用的 token 信息转换成内部结构 `TokenEvent`。

## 包职责

- 流式读取大型 JSONL transcript。
- 识别 `turn_context`，为后续 token 事件补充模型名称。
- 识别用户消息，生成用户轮次键和 prompt 摘要。
- 识别 `token_count`，提取 input、cached input、output、reasoning、total、rate-limit 和 context window 信息。
- 对未知事件、损坏 JSON 行、缺失字段做容错降级。

## 核心类型

- `TokenEvent`：一次模型调用的 token 统计事件。
- `Parser`：带状态的流式解析器，用于把用户轮次和后续 token_count 关联起来。

## 常用入口

```go
events, err := session.ParseFile(`C:\Users\me\.codex\sessions\session.jsonl`)
```

或者从任意 `io.Reader` 解析：

```go
events, err := (&session.Parser{}).Parse(reader, "session.jsonl")
```

## 兼容性原则

Codex Hooks 官方提供 `transcript_path`，但 transcript 内部字段不是稳定公共接口。因此本包必须集中处理字段兼容：

- 未知事件直接忽略。
- 单行 JSON 损坏直接跳过。
- token 字段缺失时使用 0 或 nil 降级。
- 上下文窗口字段名通过多候选路径读取。
- 解析器变化不应泄漏到 `internal/analyzer` 或 `internal/guard`。

