# internal/report

`internal/report` 负责把 `internal/analyzer` 的结构化结果渲染为人类可读文本。

## 包职责

- 输出分析范围和周额度窗口。
- 输出 rate-limit `used_percent` 变化点。
- 输出高 token 请求排行。
- 输出高非缓存输入请求排行。
- 输出全局 token 汇总和 cache rate。
- 输出模型、session 和用户轮次维度排行。

## 常用入口

```go
text := report.Text(result, 20)
fmt.Print(text)
```

## 输入与输出

- 输入：`analyzer.Result` 和展示行数 `limit`。
- 输出：纯文本字符串，不直接写 stdout。

## 设计边界

- 本包不扫描文件。
- 本包不解析 transcript。
- 本包不判断 guard 风险等级。
- 本包只负责展示，便于未来替换为 Markdown、JSON 或其他格式。

