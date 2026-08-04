# 第 2 周细则：Tool Calling 和最小安全底座

适用时间：工作日每天 30-60 分钟，周末 2-4 小时。

本周目标：理解 Agent 的地基，让模型能决定调用 Go 写的工具，同时从一开始建立工具安全边界。

本周最终交付：

- 一个可运行的 Tool Agent CLI。
- 支持 `get_current_time` 工具。
- 支持 `calculator` 工具。
- 支持受限 `http_get` 工具。
- 有基础 tool call loop。
- 工具参数有 JSON Schema。
- Go 侧有参数校验、错误处理、超时和最大调用次数。
- 有固定 Demo 和学习复盘。

## 本周核心理解

Tool calling 的关键不是“模型执行工具”，而是：

```text
模型决定要调用哪个工具，以及传什么参数。
Go 程序负责校验参数、执行工具、处理错误、返回工具结果。
模型再基于工具结果生成最终回答。
```

本周必须建立三个意识：

- 工具参数必须做 JSON Schema 约束。
- Go 侧必须做参数校验，不能完全信任模型。
- Agent loop 必须有超时、最大调用次数和安全边界。

## 本周待办事项

- 在工具 executor 中预留 JSON Schema 参数校验入口。
  - Executor 使用 `Tool.Parameters` 校验模型传入的 raw arguments。
  - 这层校验只负责通用结构约束，例如字段是否存在、类型是否正确、枚举是否合法、是否包含额外字段。
  - 具体工具的业务校验仍然放在各自的 `ToolFunc` / handler 中，例如 timezone 是否有效、除数是否为 0、URL 是否命中 allowlist。
  - 第二周可以先保留 `validateArguments(schema, rawArgs)` 这类函数入口，后续再实现完整 JSON Schema validator。

- 后续实现完整 JSON Schema validator 时，优先参考官方规范：
  - JSON Schema 规范总入口：<https://json-schema.org/specification>
  - JSON Schema Core：<https://json-schema.org/draft/2020-12/json-schema-core>
  - JSON Schema Validation：<https://json-schema.org/draft/2020-12/json-schema-validation>

- 有时间时阅读类似实现的 Go 框架源码，重点观察工具定义、参数 schema、executor、handler、agent loop 的边界：
  - CloudWeGo Eino：<https://github.com/cloudwego/eino>
  - Firebase Genkit Go：<https://github.com/firebase/genkit>
  - LangChainGo：<https://github.com/tmc/langchaingo>
  - nlpodyssey/openai-agents-go：<https://github.com/nlpodyssey/openai-agents-go>
  - Microsoft Agent Framework Go：<https://github.com/microsoft/agent-framework-go>

## 本周推荐目录

可以在第 1 周项目结构上继续演进：

```text
cmd/
  assistant-cli/
    main.go
internal/
  llm/
    client.go
    openai.go
    types.go
internal/
  tools/
    registry.go
    types.go
    executor.go
    time.go
    calculator.go
    http_get.go
internal/
  agent/
    loop.go
    trace.go
internal/
  config/
    config.go
docs/
  week-02-notes.md
  demo.md
```

不必一开始就强行调整成完整目录。可以随着每天功能增加逐步拆分。

## 周一：理解 Tool Calling

时间预算：30-60 分钟。

### 今日目标

理解 tool calling 的本质：模型不执行动作，只决定调用哪个工具。

### 今日步骤

1. 阅读 tool calling / function calling 文档。

推荐阅读：

- OpenAI Function Calling：<https://platform.openai.com/docs/guides/function-calling>
- OpenAI Responses API：<https://platform.openai.com/docs/guides/responses>

2. 理解完整流程：

```text
用户问题
  -> 模型判断是否需要工具
  -> 模型输出 tool call
  -> Go 程序根据 tool name 找到工具
  -> Go 程序校验参数
  -> Go 程序执行工具
  -> 工具结果回传模型
  -> 模型生成最终回答
```

3. 写一份笔记，回答：

- tool calling 和普通 API 调用有什么区别？
- 为什么说模型没有真的执行工具？
- 为什么工具参数需要 JSON Schema？
- 为什么 Go 侧还要做参数校验？

### 今日验收

你应该能画出 tool calling 流程，并能解释：

```text
模型负责决策，Go 负责执行。
```

### 30 分钟版本

只写流程笔记，不写代码。

### 60 分钟加餐

- 设计一个 `Tool` 类型草图。
- 思考工具至少需要哪些字段：名称、描述、参数 schema、执行函数。

### 今日记录

```text
周一完成：
我对 tool calling 的理解：
模型和 Go 程序的职责边界：
今天仍然模糊的问题：
```

## 周二：写第一个工具 get_current_time

时间预算：30-60 分钟。

### 今日目标

实现第一个低风险工具：`get_current_time`。

### 今日步骤

1. 定义工具基本结构。

先思考一个工具需要什么：

```text
name
description
parameters schema
execute function
```

2. 实现 `get_current_time`。

工具功能：

```text
返回当前时间。
```

可以先只返回本地时间。

3. 为工具参数设计 JSON Schema。

第一版可以没有参数，或者支持一个可选参数：

```text
timezone
```

不要一开始做复杂时区转换。核心是跑通工具定义和执行流程。

4. 手动调用工具执行函数，确认能返回时间。

### 今日验收

你应该能在 Go 代码中执行：

```text
get_current_time
```

并得到当前时间字符串。

### 30 分钟版本

只实现无参数版本。

### 60 分钟加餐

- 支持固定时间格式，例如 RFC3339。
- 给无效 timezone 返回清晰错误。
- 给工具执行加 context timeout。
- 设计一个很薄的工具 executor 草图，先明确它负责：查找工具、解析参数、设置 timeout、调用执行函数、统一返回错误。

### 今日记录

```text
周二完成：
工具定义包含哪些部分：
时间工具有哪些边界：
executor 应该负责哪些通用逻辑：
```

## 周三：写 calculator 工具

时间预算：30-60 分钟。

### 今日目标

实现一个带参数校验的计算工具：`calculator`。

### 今日步骤

1. 定义工具参数。

可以设计为：

```json
{
  "operation": "add",
  "a": 3,
  "b": 5
}
```

2. 支持四种操作：

```text
add
subtract
multiply
divide
```

3. 用 JSON Schema 限制：

- `operation` 必须是枚举值。
- `a` 和 `b` 必须是 number。
- 必须包含 `operation`、`a`、`b`。
- 不允许额外字段。

4. Go 侧做参数校验。

重点处理：

- operation 不合法。
- 参数缺失。
- 参数类型错误。
- 除以 0。

5. 如果周二已经设计了 executor，可以尝试让 `calculator` 也通过同一个 executor 执行。

此时 executor 只需要保持很薄，不需要接入模型，也不需要处理完整 Agent Loop。它的目标是让不同工具共享同一套执行边界。

### 今日验收

工具能够正确处理：

```text
3 + 5
10 - 4
6 * 7
8 / 2
8 / 0
```

### 30 分钟版本

只实现 Go 函数和基础参数校验。

### 60 分钟加餐

- 用 `encoding/json` 解析工具参数。
- 为 calculator 写一个小测试。
- 返回结构化工具结果，而不是随意字符串。
- 让 `get_current_time` 和 `calculator` 都能通过同一个 executor 被调用。

### 今日记录

```text
周三完成：
JSON Schema 限制了什么：
Go 侧额外校验了什么：
```

## 周四：写受限 http_get 工具

时间预算：30-60 分钟。

### 今日目标

实现一个受限 HTTP 工具，理解工具安全边界。

### 今日步骤

1. 实现 `http_get` 工具。

工具功能：

```text
请求一个允许访问的 URL，并返回响应摘要。
```

2. 不允许访问任意 URL。

必须先设计 allowlist，例如：

```text
localhost
127.0.0.1
example.com
```

或只允许本地 mock API。

3. 加基础安全限制：

- 只允许 `http` / `https`。
- 必须命中 allowlist。
- 设置请求超时。
- 限制响应体大小。

4. 测试允许和拒绝场景。

### 今日验收

下面两类场景都应该有明确结果：

```text
allowlist 内 URL -> 可以请求
allowlist 外 URL -> 被拒绝
```

### 30 分钟版本

只实现 allowlist 检查，不接真实复杂服务。

### 60 分钟加餐

- 写一个本地 mock API。
- 限制最大响应体大小。
- 记录 HTTP 状态码和耗时。

### 今日记录

```text
周四完成：
为什么 http_get 不能开放任意 URL：
allowlist 如何设计：
```

## 周五：实现基础 Tool Call Loop

时间预算：30-60 分钟。

### 今日目标

实现基础 tool call loop：模型请求工具，Go 执行工具，结果回传模型。

### 今日步骤

1. 建立工具注册表。

你需要能根据 tool name 找到对应工具：

```text
get_current_time -> 时间工具
calculator -> 计算工具
http_get -> HTTP 工具
```

2. 让模型知道有哪些工具。

工具定义应该包含：

- name
- description
- parameters schema

3. 处理模型返回的 tool call。

核心逻辑：

```text
读取 tool name
读取 arguments
交给 executor
executor 查找工具
executor 解析参数
executor 执行工具
拿到工具结果
回传模型
生成最终回答
```

4. 加最大调用次数。

第一版可以设置：

```text
maxSteps = 3
```

避免模型无限请求工具。

### 今日验收

至少能跑通：

```text
用户：现在几点？
模型：调用 get_current_time
Go：执行时间工具
模型：基于结果回答
```

以及：

```text
用户：3 加 5 等于多少？
模型：调用 calculator
Go：执行计算工具
模型：基于结果回答
```

### 30 分钟版本

只跑通一个工具的 loop。

### 60 分钟加餐

- 支持两个以上工具。
- 给每一步打印 trace。
- 工具失败时让模型给出友好回答。
- 将 timeout、参数解析失败、工具不存在等通用错误收口到 executor。

### 今日记录

```text
周五完成：
tool call loop 的步骤：
遇到的最难点：
```

## 周六：做 Tool Agent Demo

时间预算：2-4 小时。

### 今日目标

把本周工具整合成一个可演示的 Tool Agent CLI。

### 今日步骤

1. 整合三个工具：

```text
get_current_time
calculator
http_get
```

2. 准备固定演示问题：

```text
现在几点？
3 加 5 乘 2 是多少？
请求这个 mock API 并总结结果
```

3. 打印关键 trace。

建议至少打印：

```text
User:
Tool selected:
Tool args:
Tool result:
Assistant:
```

4. 写 `docs/demo.md` 或记录到学习笔记。

### 今日验收

能稳定演示：

- 查询时间。
- 简单计算。
- 受限 HTTP 请求。
- 工具调用链路可观察。

### 30 分钟版本

只整理一个可跑通的 demo 命令。

### 2-4 小时版本

- 三个工具都接入。
- 有固定 demo 文档。
- 有基础 trace。
- 有错误场景演示。

### 今日记录

```text
周六完成：
Demo 是否稳定：
哪些工具调用最容易失败：
```

## 周日：错误处理、安全限制和复盘

时间预算：2-4 小时。

### 今日目标

把第 2 周成果收口，补齐最小安全底座。

### 今日步骤

1. 补错误处理：

- 工具不存在。
- 参数 JSON 解析失败。
- 参数校验失败。
- 工具执行超时。
- HTTP 请求失败。
- 除零错误。

2. 补安全限制：

- 最大工具调用次数。
- HTTP allowlist。
- 请求超时。
- 响应体大小限制。

3. 补 trace。

至少记录：

```text
step
tool name
arguments
result
error
```

4. 写周复盘。

### 今日验收

你应该能回答：

- tool calling 为什么不是模型真的执行工具？
- 工具参数为什么需要 JSON Schema？
- 模型传错参数怎么办？
- 工具调用失败怎么处理？
- 如何避免无限调用工具？
- `http_get` 为什么不能访问任意 URL？

### 30 分钟版本

只写复盘和补最大调用次数。

### 2-4 小时版本

- 补齐主要错误处理。
- 补基础测试。
- 固定 demo 可以稳定复现。

### 今日记录

```text
周日完成：
本周完成：
本周最重要的理解：
还不稳定的地方：
第 3 周 RAG 前需要准备什么：
```

## 本周完成标准

最低完成标准：

- 有 `get_current_time` 工具。
- 有 `calculator` 工具。
- 有基础 tool call loop。
- 工具参数有 JSON Schema。
- Go 侧有基本参数校验。

推荐完成标准：

- 有受限 `http_get` 工具。
- 有 URL allowlist。
- 有最大调用次数。
- 有工具错误处理。
- CLI 能展示完整工具调用链路。

优秀完成标准：

- 有 trace 日志。
- 有 mock 工具测试。
- 有固定 demo。
- 有周复盘笔记。
- 能清楚解释 tool calling、JSON Schema、Go 参数校验、安全边界之间的关系。

## 本周不要做什么

- 不要急着接 RAG。
- 不要让 `http_get` 访问任意 URL。
- 不要跳过 Go 侧参数校验。
- 不要只依赖模型的 JSON Schema。
- 不要做复杂 UI。
- 不要一开始就追求多模型供应商兼容。

第 2 周的价值是建立 Agent 最小行动能力：模型会决策，Go 会执行，工具有边界，错误能收口。这个基础打稳，后面的 RAG、MCP 和 Agent Loop 扩展都会更安全。
