# go-ai-agent Demo 演示文档

本文档记录当前 CLI 已支持的输出模式，以及第二周 Tool Agent 的固定演示流程。

## 演示前准备

进入项目目录：

```powershell
cd D:\Go\WorkSpace\src\Go_Project\ai-agent\go-ai-agent
```

根据 `.env.example` 创建本地 `.env`：

```env
OPENAI_API_KEY=<your-api-key>
OPENAI_BASE_URL=<your-openai-compatible-base-url>
OPENAI_MODEL=<model-name>
```

真实 API Key 只保存在本地，不要提交 `.env`。

当前 CLI 使用统一的 `--outputType` 参数：

```text
standard       普通完整输出
stream         流式输出
json           JSON Schema 结构化输出
function_call  工具调用
```

为了让演示输入保持明确，下面的命令会显式传入 `--instruction` 和用户问题。

## Demo 1：普通输出

```powershell
go run ./cmd/assistant-cli --outputType standard --instruction "你是一个简洁的 Go AI Agent 学习助手。" "什么是 Agent Loop？"
```

预期结果：模型生成完整回答后一次性打印。

## Demo 2：流式输出

```powershell
go run ./cmd/assistant-cli --outputType stream --instruction "你是一个简洁的 Go AI Agent 学习助手。" "解释 Tool Calling 的工作流程。"
```

预期结果：模型回答以增量文本逐步打印。

## Demo 3：结构化 JSON 输出

```powershell
go run ./cmd/assistant-cli --outputType json --instruction "判断用户问题的意图并按指定 JSON Schema 回答。" "Tool Calling 和普通 API 请求有什么区别？"
```

预期结果：程序将模型 JSON 解析为 `IntentResult`，检查 intent 和 confidence，并打印回答及完整结构。

## Demo 4：查询当前时间

```powershell
go run ./cmd/assistant-cli --outputType function_call --instruction "当问题需要实时信息时使用已提供的工具。" "请调用 time 工具查询 Asia/Shanghai 当前时间。"
```

预期调用链：

```text
模型选择 time
Go 执行 time
Go 打印工具结果
模型根据结果生成最终回答
```

当前 trace 会显示类似内容：

```text
tool time Resp: 2026-08-10 10:30:00
工具被调用了 1 次
被调用的工具:
[time]
输出
<模型最终回答>
```

具体时间以实际运行结果为准。

## Demo 5：基础计算

默认 `tool_choice` 为自动选择，模型可能直接完成简单计算。为了验证 `calculator` 工具，提示词需要明确要求使用工具：

```powershell
go run ./cmd/assistant-cli --outputType function_call --instruction "所有数字计算必须调用 calculator 工具，不允许自行计算。" "请调用 calculator 计算 125 除以 5。"
```

预期调用链：

```text
模型选择 calculator
arguments 包含 divide、125、5
Go 执行 calculator
工具结果为 25
模型生成最终回答
```

模型是否选择工具仍具有一定不确定性。稳定验证工具执行逻辑应使用单元测试，稳定验证 Agent Loop 应使用模拟模型响应。

## Demo 6：受限 HTTP GET

当前 `http_get` 只允许访问 allowlist 中的 hostname，并且只允许 `GET`。

```powershell
go run ./cmd/assistant-cli --outputType function_call --instruction "需要外部 HTTP 数据时必须调用 http_get，并严格使用用户给出的 URL 和 query。" "请调用 http_get，使用 GET 请求 https://uapis.cn/api/v1/misc/weather，query 参数为 city=anyang、adcode=410502、lang=zh，然后总结响应。"
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

该 Demo 依赖网络和外部 API 状态。如果只验证 Go 工具行为，应运行本地 `httptest` 测试。

## Demo 7：错误场景

### 除零错误

```powershell
go run ./cmd/assistant-cli --outputType function_call --instruction "所有数字计算必须调用 calculator 工具。" "请调用 calculator 计算 10 除以 0。"
```

预期：Calculator 返回除零错误，Agent Run 结束并输出错误。

### URL allowlist 拒绝

```powershell
go run ./cmd/assistant-cli --outputType function_call --instruction "HTTP 请求必须调用 http_get。" "请调用 http_get，使用 GET 请求 https://example.com。"
```

预期：`http_get` 拒绝访问 allowlist 外的 hostname。

### 最大工具调用次数

当模型在执行 `MaxSteps` 次工具后仍然请求工具，程序应返回工具调用次数超限错误。由于真实模型的工具选择不稳定，该场景更适合通过模拟 Responses API 的自动化测试验证。

## Demo 8：运行测试

运行工具测试：

```powershell
go test ./internal/tools
```

当前已覆盖：

- time 的正常时区和非法时区。
- calculator 的加减乘除、除零、非法运算符和非法 JSON。
- http_get 的 allowlist、method、query、header 和响应大小限制。

运行全量测试：

```powershell
go test ./...
```

当前已知事项：`internal/llm/output_schema_test.go` 仍需完成 testcase 执行和断言。在修正前，全量测试会因为未使用的 `testcases` 变量而构建失败。

## Make 命令

```powershell
make run
make stream
make json
make function_call
make test
```

Makefile 只负责选择输出模式。当前默认 system instruction 的回退逻辑尚未启用，因此使用 Make 命令时，需要按程序读取顺序输入 system instruction 和用户问题。固定演示优先使用本文档中显式传入 `--instruction` 的命令。

## 当前已知限制

### 中转站上下文衔接

当前中转站能够完成单次工具调用，但使用 `PreviousResponseID` 进行连续工具调用时，模型可能遗忘原始任务。例如同时查询上海和纽约时间时，只查询上海后便生成最终回答。

即使显式设置 `Store=true`，该现象仍然存在。后续计划由 Go 手动保存原始输入、模型全部 output、reasoning item、function call 和 function call output。

因此，当前固定 Demo 以单工具调用为主，多轮工具调用暂不作为中转站环境下的稳定演示项。

### 工具自动选择

`ParallelToolCalls=false` 表示每轮最多调用一个工具，不表示模型必须调用工具。默认自动模式下，模型可以直接回答。需要确定性验证时，应使用模拟响应测试，而不是只依赖提示词。

## 演示检查清单

- `.env` 已配置且不会提交。
- `standard`、`stream`、`json`、`function_call` 参数可以识别。
- time 单工具调用能够完成。
- calculator 工具函数测试通过。
- http_get 本地 mock 测试通过。
- allowlist 和除零错误能够观察到。
- trace 能显示工具名称、结果和调用次数。
- 已知的中转站上下文限制已明确说明。
