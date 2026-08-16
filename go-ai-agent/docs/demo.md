# go-ai-agent Demo 演示文档

本文档记录当前 CLI 的演示方式，以及第 2 周 Tool Agent 的固定验收流程。

当前阶段重点验收：

```text
Groq Chat Completions + Tool Calling
```

普通输出、流式输出和 JSON Schema 输出仍是第 1 周能力，但当前 Groq 改造只重点验证 `function_call` 路径。`groqClient` 暂时只重写了 `FunctionCall`，其他方法仍沿用 OpenAI Responses API 风格，不作为本轮 Groq 验收范围。

## 演示前准备

进入项目目录：

```powershell
cd D:\Go\WorkSpace\src\Go_Project\ai-agent\go-ai-agent
```

根据 `.env.example` 创建本地 `.env`：

```env
OPENAI_API_KEY=<your-groq-api-key>
OPENAI_BASE_URL=https://api.groq.com/openai/v1
OPENAI_MODEL=<groq-tool-use-model>
```

真实 API Key 只保存在本地，不要提交 `.env`。

当前 CLI 使用统一的 `--outputType` 参数：

```text
standard       普通完整输出
stream         流式输出
json           JSON Schema 结构化输出
function_call  工具调用
```

本轮固定 Demo 主要使用：

```powershell
go run ./cmd/assistant-cli --outputType function_call "<用户问题>"
```

## Demo 1：无需工具的普通问题

目的：验证 `ToolChoice=auto` 下，模型可以判断“不需要工具”，直接回答。

```powershell
go run ./cmd/assistant-cli --outputType function_call "请用一句话解释什么是 Tool Calling。"
```

预期：

```text
不触发工具调用
直接输出自然语言回答
```

这说明提供工具列表不等于模型必须调用工具。

## Demo 2：查询当前时间

```powershell
go run ./cmd/assistant-cli --outputType function_call "请告诉我现在广州的时间，并调用工具。"
```

预期调用链：

```text
模型选择 time
Go 执行 time
工具结果包含 time_zone 和 current_time
模型根据工具结果生成最终回答
```

广州没有单独的 IANA 时区，通常应使用：

```text
Asia/Shanghai
```

## Demo 3：基础计算

```powershell
go run ./cmd/assistant-cli --outputType function_call "请调用工具告诉我 4*5 的结果是多少。"
```

预期调用链：

```text
模型选择 calculator
arguments 表示乘法、4、5
Go 执行 calculator
工具结果为 20
模型生成最终回答
```

简单数学题模型可能直接回答。如果需要稳定验证 `calculator` 工具，可以在问题中明确要求“调用工具”。

## Demo 4：连续工具调用和跨工具调用

这是当前 Groq Tool Calling 的核心验收 Demo。

```powershell
go run ./cmd/assistant-cli --outputType function_call "请告诉我现在广州和洛杉矶的时间, 并调用工具告诉我 4*5 的结果是多少"
```

实际验证日志示例：

```text
第 1 次 tool 调用, tool 名称: time, 返回结果: {"current_time":"2026-08-16 22:21:39","time_zone":"Asia/Shanghai"}
第 2 次 tool 调用, tool 名称: time, 返回结果: {"current_time":"2026-08-16 07:21:39","time_zone":"America/Los_Angeles"}
第 3 次 tool 调用, tool 名称: calculator, 返回结果: 20.000000
```

该 Demo 验证了：

```text
连续调用同类工具：time -> time
跨工具调用：time -> calculator
messages history 能跨轮保存上下文
模型能在工具结果后继续决定下一步
```

最终回答应同时包含：

```text
广州当前时间
洛杉矶当前时间
4*5 的结果
```

## Demo 5：受限 HTTP GET allowlist 内请求

当前 `http_get` 只允许访问 allowlist 中的 hostname，并且只允许 `GET`。

```powershell
go run ./cmd/assistant-cli --outputType function_call "请调用 http_get，使用 GET 请求 https://uapis.cn/api/v1/misc/weather，query 参数为 city=anyang、adcode=410502、lang=zh，然后总结响应。"
```

预期调用参数包含：

```json
{
  "url": "https://uapis.cn/api/v1/misc/weather",
  "method": "GET",
  "query": {
    "city": "anyang",
    "adcode": "410502",
    "lang": "zh"
  }
}
```

预期：

```text
模型选择 http_get
Go 校验 hostname 命中 allowlist
Go 发起 GET 请求
模型总结响应内容
```

该 Demo 依赖网络和外部 API 状态。如果只验证 Go 工具行为，应优先运行本地 `httptest` 单元测试。

## Demo 6：受限 HTTP GET allowlist 外拒绝

```powershell
go run ./cmd/assistant-cli --outputType function_call "请调用 http_get，使用 GET 请求 https://example.com。"
```

预期：

```text
模型选择 http_get
Go 校验 hostname 不在 allowlist 中
工具执行失败
用户侧看到通用错误
后台日志记录具体错误原因
```

该 Demo 验证 `http_get` 不是通用无限制 HTTP 客户端，模型不能绕过 Go 侧安全边界访问任意 URL。

## Demo 7：错误场景

### 除零错误

```powershell
go run ./cmd/assistant-cli --outputType function_call "请调用 calculator 计算 10 除以 0。"
```

预期：

```text
calculator 返回除零错误
Agent Run 结束
用户侧不暴露内部细节
后台日志记录具体错误
```

### 非法时区

```powershell
go run ./cmd/assistant-cli --outputType function_call "请调用 time 工具查询 Invalid/Zone 当前时间。"
```

预期：

```text
time 工具返回时区错误
Agent Run 结束
后台可观察具体错误原因
```

### 最大工具调用次数

当模型在执行 `MaxSteps` 次工具后仍然请求工具，程序应返回工具调用次数超限错误。

真实模型的工具选择不稳定，因此该场景更适合后续通过 mock Chat Completions 响应做自动化测试。

## Demo 8：运行测试

运行工具测试：

```powershell
go test ./internal/tools
```

只运行 `http_get` 测试：

```powershell
go test ./internal/tools -run TestHttpGet -v
```

当前 `http_get` 测试覆盖：

```text
allowlist 内 URL 可以访问
query 参数会正确传给服务端
header 会正确传给服务端
allowlist 外 URL 被拒绝
非 GET 方法被拒绝
响应体超过限制会报错
```

运行全量测试：

```powershell
go test ./...
```

当前已知事项：`internal/llm/output_schema_test.go` 仍需完成 testcase 执行和断言；`time` 工具返回值已改为 JSON 字符串后，相关测试需要保持同步。

## 当前实现说明

### Groq Tool Calling

当前 Groq 版 Tool Calling 使用 Chat Completions messages history：

```text
system / user
assistant(tool_calls)
tool(tool_call_id, result)
assistant(final answer 或继续 tool_calls)
```

每轮工具执行后，Go 会把 assistant 的工具调用消息和 tool 结果追加到 history，再发起下一轮请求。

### 工具失败策略

当前阶段工具执行失败时，程序直接返回错误，不再把原始错误交给模型生成最终回答。

这样做的原因是：

```text
调试路径简单
错误边界清晰
避免把内部错误、请求细节、路径、上游响应等敏感信息暴露给用户
```

后续可以增强为：将脱敏后的结构化工具错误作为 tool result 回传模型，让模型生成更友好的自然语言回答。

### 工具自动选择

`ToolChoice=auto` 表示模型自己决定是否调用工具。它可能：

```text
不调用工具，直接回答
调用一个工具后结束
连续调用多个工具
```

因此，真实模型 Demo 适合做功能演示；稳定的边界验证仍需要单元测试或 mock 模型响应。

## 演示检查清单

- `.env` 已配置且不会提交。
- `function_call` 模式可运行。
- 不需要工具的问题不会触发工具调用。
- `time` 单工具调用可以完成。
- `calculator` 单工具调用可以完成。
- 多工具调用 Demo 可以触发 `time -> time -> calculator`。
- `http_get` allowlist 内请求可以完成。
- `http_get` allowlist 外请求会被拒绝。
- 工具调用 trace 能显示工具名称、结果和调用次数。
- 工具失败时用户侧不暴露内部敏感信息，后台能看到具体错误。
