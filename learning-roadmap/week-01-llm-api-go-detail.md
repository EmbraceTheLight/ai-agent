# 第 1 周细则：LLM API 和 Go 基础接入

适用时间：工作日每天 30-60 分钟，周末 2-4 小时。

本周目标：用 Go 跑通一次 LLM 调用，完成一个最小 CLI AI 助手，并为后续 tool calling、RAG、Agent Loop 打好工程基础。

本周最终交付：

- 一个可运行的 Go CLI 程序。
- 支持普通问答。
- 支持 streaming 输出。
- 支持结构化 JSON 输出。
- 有 `LLMClient` 接口封装。
- 有 `.env.example`、`README.md`、基础测试。

建议项目名：

```text
go-ai-agent-playground
```

## 本周推荐目录

```text
cmd/
  assistant-cli/
    main.go
internal/
  llm/
    client.go
    openai.go
    types.go
    client_test.go
internal/
  config/
    config.go
docs/
  week-01-notes.md
.env.example
README.md
Makefile
go.mod
```

## 开始前准备

如果你还没有做“准备日”，先完成这些：

1. 确认 Go 版本：

```powershell
go version
```

建议 Go 1.22+。

2. 初始化项目：

```powershell
go mod init github.com/yourname/go-ai-agent-playground
```

3. 创建 `.env.example`：

```text
OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_MODEL=
```

说明：

- 如果使用 OpenAI，`OPENAI_BASE_URL` 可以留空或使用官方地址。
- 如果使用 DeepSeek、阿里云百炼等 OpenAI 兼容接口，可以通过 `OPENAI_BASE_URL` 切换。
- 不要提交真实 API Key。

## 周一：跑通第一次 LLM 调用

时间预算：30-60 分钟。

### 今日目标

用 Go 发起一次最简单的 LLM 请求，并在终端打印回答。

### 今日步骤

1. 阅读官方文档 10 分钟：

- OpenAI Responses API：<https://platform.openai.com/docs/guides/responses>
- OpenAI Go SDK：<https://github.com/openai/openai-go>
- 如果用中文模型供应商：DeepSeek API 中文文档：<https://api-docs.deepseek.com/zh-cn/>

2. 安装 SDK：

```powershell
go get github.com/openai/openai-go
```

3. 新建 `cmd/assistant-cli/main.go`。

4. 从环境变量读取：

```text
OPENAI_API_KEY
OPENAI_BASE_URL
OPENAI_MODEL
```

5. 写死一个问题，例如：

```text
请用三句话解释什么是 AI Agent。
```

6. 调用模型并打印结果。

### 今日验收

运行：

```powershell
go run ./cmd/assistant-cli
```

你应该能在终端看到模型回答。

### 30 分钟版本

只要能跑通一次请求即可。

### 60 分钟加餐

- 把模型名、base URL、API Key 都改成环境变量。
- 给请求加 `context.WithTimeout`。
- 把错误信息打印清楚。

### 今日记录

在 `docs/week-01-notes.md` 记录：

```text
周一完成：
使用的模型：
遇到的问题：
明天要改进：
```

## 周二：理解消息角色，做成命令行问答

时间预算：30-60 分钟。

### 今日目标

把周一的固定问题改成命令行输入，让程序像一个最小聊天助手。

### 今日步骤

1. 理解三类常见消息：

- system：定义助手身份、边界、回答风格。
- user：用户输入。
- assistant：模型之前的回答，后续做多轮对话时会用到。

2. 在 `main.go` 中读取命令行参数：

```powershell
go run ./cmd/assistant-cli "请解释 Go interface 的用途"
```

3. 如果用户没有传参数，提示用法：

```text
Usage: assistant-cli "your question"
```

4. 加一个固定 system prompt：

```text
你是一个帮助 Go 后端开发者学习 AI Agent 的技术助手。回答要准确、简洁、偏工程实践。
```

5. 返回模型回答并打印。

### 今日验收

下面两条命令都能正常运行：

```powershell
go run ./cmd/assistant-cli "什么是 tool calling？"
go run ./cmd/assistant-cli "RAG 和普通搜索有什么区别？"
```

### 30 分钟版本

支持命令行参数即可。

### 60 分钟加餐

- 支持从 stdin 输入问题。
- 给输出加简单格式，例如 `Assistant:`。
- 把 system prompt 提取成常量。

### 今日记录

```text
周二完成：
我对 system prompt 的理解：
CLI 还有哪些不方便：
```

## 周三：加入 streaming 输出

时间预算：30-60 分钟。

### 今日目标

让模型回答逐步输出，而不是等完整结果返回后一次性打印。

### 今日步骤

1. 阅读 streaming 相关官方文档。

2. 在 `internal/llm/types.go` 定义基础类型：

```go
type Message struct {
    Role    string
    Content string
}
```

3. 在 `internal/llm/client.go` 定义接口：

```go
type Client interface {
    Generate(ctx context.Context, messages []Message) (string, error)
    Stream(ctx context.Context, messages []Message, onDelta func(string)) error
}
```

4. 在 `internal/llm/openai.go` 实现 `Stream`。

5. 在 CLI 中增加参数：

```powershell
go run ./cmd/assistant-cli --stream "解释一下 Agent Loop"
```

### 今日验收

运行 stream 模式时，回答应该逐步显示。

### 30 分钟版本

只在 `main.go` 里直接写 streaming 调用。

### 60 分钟加餐

- 把 streaming 封装进 `LLMClient`。
- 处理用户 Ctrl+C。
- 加请求超时。

### 今日记录

```text
周三完成：
streaming 和普通请求的区别：
封装时遇到的问题：
```

## 周四：结构化 JSON 输出

时间预算：30-60 分钟。

### 今日目标

让模型按固定 JSON 结构返回，训练自己不要依赖“自然语言随便解析”。

### 今日步骤

1. 阅读 structured output / JSON Schema 文档：

- OpenAI Structured Outputs：<https://platform.openai.com/docs/guides/structured-outputs>
- OpenAI Function Calling：<https://platform.openai.com/docs/guides/function-calling>

2. 定义输出结构：

```go
type IntentResult struct {
    Intent     string  `json:"intent"`
    Answer     string  `json:"answer"`
    Confidence float64 `json:"confidence"`
}
```

3. 让模型对用户问题做简单意图识别：

```text
rag_question
tool_question
agent_question
general_question
```

4. 解析 JSON 到 Go struct。

5. 如果解析失败，要打印原始响应并返回错误。

### 今日验收

运行：

```powershell
go run ./cmd/assistant-cli --json "MCP 和 tool calling 有什么关系？"
```

输出类似：

```json
{
  "intent": "tool_question",
  "answer": "...",
  "confidence": 0.82
}
```

### 30 分钟版本

用 prompt 约束模型返回 JSON，并用 `encoding/json` 解析。

### 60 分钟加餐

- 使用严格 schema / structured output。
- 对 `confidence` 做范围校验：0 到 1。
- 对 `intent` 做枚举校验。

### 今日记录

```text
周四完成：
JSON 输出是否稳定：
解析失败的原因：
```

## 周五：封装 LLMClient，整理工程结构

时间预算：30-60 分钟。

### 今日目标

把散落在 `main.go` 里的 LLM 调用整理到 `internal/llm`，为后续替换模型和做 Agent Loop 做准备。

### 今日步骤

1. 创建或完善：

```text
internal/llm/client.go
internal/llm/openai.go
internal/llm/types.go
internal/config/config.go
```

2. `main.go` 只保留 CLI 参数解析和调用逻辑。

3. `internal/config/config.go` 负责读取环境变量。

4. `internal/llm/openai.go` 负责具体模型调用。

5. 为 config 或 JSON 解析写一个小测试。

### 今日验收

运行：

```powershell
go test ./...
go run ./cmd/assistant-cli --stream "用 Go 怎么学习 AI Agent？"
```

两条命令都应该成功。

### 30 分钟版本

只完成 `LLMClient` 接口和 OpenAI 实现拆分。

### 60 分钟加餐

- 增加 `Provider` 字段，为以后兼容 DeepSeek / 阿里云百炼做准备。
- 增加 mock client，方便测试。

### 今日记录

```text
周五完成：
目前项目结构：
哪些代码还不够干净：
```

## 周六：做成本周可展示 Demo

时间预算：2-4 小时。

### 今日目标

把前 5 天的小功能整合成一个可演示 CLI。

### 今日步骤

1. 整理 CLI 参数：

```text
--stream      流式输出
--json        结构化输出
--model       指定模型
--timeout     指定超时时间
```

2. 给 `Makefile` 加命令：

```makefile
run:
	go run ./cmd/assistant-cli "什么是 AI Agent？"

stream:
	go run ./cmd/assistant-cli --stream "解释一下 Agent Loop"

json:
	go run ./cmd/assistant-cli --json "RAG 是什么？"

test:
	go test ./...
```

3. 写 `README.md`：

- 项目目标。
- 环境变量配置。
- 如何运行。
- 支持的功能。
- 本周学习成果。

4. 写 `docs/demo.md`，记录固定演示命令。

5. 跑一遍完整演示。

### 今日验收

下面命令都能运行：

```powershell
make run
make stream
make json
make test
```

如果 Windows 上没有 `make`，可以在 README 里同时写 PowerShell 等价命令。

### 今日记录

```text
周六完成：
Demo 是否能稳定复现：
README 是否足够让别人跑起来：
```

## 周日：复盘、补测试、准备下周 Tool Calling

时间预算：2-4 小时。

### 今日目标

把第 1 周成果收口，并为第 2 周 tool calling 做铺垫。

### 今日步骤

1. 跑测试：

```powershell
go test ./...
```

2. 检查 README 是否包含：

- 项目简介。
- 快速开始。
- 环境变量。
- 示例命令。
- 目录结构。
- 已完成功能。
- 下一步计划。

3. 补一个 mock client 测试：

```go
type MockClient struct {
    Response string
}
```

4. 在 `docs/week-01-notes.md` 写周复盘。

5. 预习第 2 周：

- tool calling 是什么。
- JSON Schema 为什么重要。
- Go 侧为什么必须做参数校验。

### 今日验收

你应该能回答这几个问题：

- 普通 LLM 请求和 streaming 请求有什么区别？
- 为什么要封装 `LLMClient` 接口？
- structured output 比普通自然语言输出稳定在哪里？
- 为什么不能把 API Key 写死在代码里？
- 第 2 周 tool calling 要解决什么问题？

## 本周完成标准

最低完成标准：

- `go run ./cmd/assistant-cli "问题"` 可以返回答案。
- 支持从环境变量读取 API Key 和模型名。
- 有 `README.md`。
- 有 `LLMClient` 接口。

推荐完成标准：

- 支持 streaming。
- 支持 JSON 结构化输出。
- 有 `Makefile` 或等价脚本。
- 有基础测试。
- 有 `docs/demo.md` 固定演示脚本。

优秀完成标准：

- 支持 OpenAI 兼容 base URL。
- 有 mock client。
- 所有请求有 timeout。
- 错误信息清晰。
- README 能让别人 5 分钟内跑起来。

## 本周不要做什么

- 不要急着接 RAG。
- 不要急着写 Agent Loop。
- 不要急着换多个模型供应商做兼容。
- 不要先研究 LangChain / LangGraph。
- 不要把时间花在 UI 上。

第 1 周的价值是把 LLM 当成一个可靠的外部服务来封装。这个基础打稳，后面的 tool calling、RAG、MCP 都会顺很多。

