# internal/analyzer

`internal/analyzer` 是离线诊断的统计聚合层。它接收 `internal/session` 解析出的 `TokenEvent`，生成面向报告的多维汇总结果。

## 包职责

- 扫描 sessions 目录下最近修改的 JSONL transcript。
- 调用 `session.ParseFile` 提取 token 事件。
- 对重复历史事件去重，避免 rollout/session 之间重复统计。
- 根据周额度 reset 时间推导 Codex 周窗口。
- 汇总全局 token 用量、模型用量、session 用量和用户轮次用量。
- 提取 rate-limit `used_percent` 的变化点。

## 核心类型

- `Result`：一次分析的完整结果。
- `Summary`：全局 token 汇总。
- `QuotaChange`：周额度使用率变化点。
- `ModelSummary`：按模型聚合的统计行。
- `SessionSummary`：按 session 聚合的统计行。
- `TurnSummary`：按用户轮次聚合的统计行。

## 常用入口

扫描目录：

```go
result, err := analyzer.AnalyzeDirectory(`C:\Users\me\.codex\sessions`, time.Now().AddDate(0, 0, -7))
```

分析指定文件：

```go
result, err := analyzer.AnalyzeFiles(root, []string{"a.jsonl", "b.jsonl"}, since)
```

## 去重策略

事件去重使用 `TokenEvent.EventKey`。如果重复事件中有的版本带有周额度字段，有的版本不带，则优先保留带周额度字段的事件，因为它对诊断额度变化更有价值。

## 输出边界

本包不负责格式化报告，也不直接输出到 stdout。文本展示由 `internal/report` 完成，实时 Hook 预警由 `internal/guard` 完成。

