# go-ai-agent

[English](./README.md) | 简体中文

这是一个用 Go 学习 AI Agent 工程化开发的项目。当前版本是一个最小 CLI AI 助手，可以调用 OpenAI 兼容的 LLM API，支持普通问答、流式输出和结构化 JSON 输出。

项目学习路线见 `../learning-roadmap`。

## 已支持功能

- 使用 `github.com/openai/openai-go/v3` 接入 OpenAI 兼容 API
- 使用 `github.com/joho/godotenv` 加载本地 `.env`
- 普通问答输出
- streaming 流式输出
- 基于 JSON Schema 的结构化 JSON 输出
- 支持四种意图分类：
  - `rag_question`
  - `tool_question`
  - `agent_question`
  - `general_question`
- 封装 `LLMClient` 接口
- 支持命令行参数控制输出模式、模型、system instruction 和 timeout
- 提供 makefile 常用命令

## 项目结构

```text
cmd/
  assistant-cli/
    assistant-cli.go
internal/
  config/
    config.go
    const.go
    flag.go
  llm/
    client.go
    openAI.go
    output_schema.go
    types.go
  tools/
    context_tool.go
    handle_input.go
docs/
testdata/
.env.example
makefile
go.mod
```

## 环境要求

- 与 `go.mod` 兼容的 Go 版本
- OpenAI 或 OpenAI 兼容中转站 API Key
- 可选：`make`

## 配置方式

在当前项目根目录创建 `.env`：

```env
OPENAI_API_KEY=<your-api-key>
OPENAI_BASE_URL=<your-openai-compatible-base-url>
OPENAI_MODEL=<model-name>
```

示例：

```env
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://example.com/v1
OPENAI_MODEL=gpt-5.5
```

不要提交 `.env`，真实 API Key 只保存在本地。项目中的 `.env.example` 只作为配置模板。

## 运行方式

启动 CLI：

```powershell
go run ./cmd/assistant-cli
```

通过命令行参数传入问题：

```powershell
go run ./cmd/assistant-cli "什么是 AI Agent？"
```

使用流式输出：

```powershell
go run ./cmd/assistant-cli --stream "解释一下 Agent Loop"
```

使用结构化 JSON 输出：

```powershell
go run ./cmd/assistant-cli --json "MCP 和 tool calling 有什么关系？"
```

指定模型：

```powershell
go run ./cmd/assistant-cli --model gpt-5.5 "解释一下 RAG"
```

指定 system instruction：

```powershell
go run ./cmd/assistant-cli --instruction "你是一个简洁的 Go 后端学习助手。" "解释 Go interface"
```

指定 timeout：

```powershell
go run ./cmd/assistant-cli --timeout 30s "解释 streaming 输出"
```

## Make 命令

```powershell
make run
make stream
make json
make test
```

在 Windows 上，如果 makefile 中直接写中文参数，可能因为 GNU Make / cmd / PowerShell 的编码链路导致乱码。当前项目的 make 命令尽量只负责选择运行模式，具体问题可以通过 stdin 输入。

## 输出模式说明

普通模式会等模型完整生成后一次性输出回答。

流式模式会在模型返回增量内容时立即打印：

```text
model delta -> onDelta -> fmt.Print(delta)
```

JSON 模式要求模型返回类似下面的结构：

```json
{
  "intent": "agent_question",
  "answer": "模型回答内容",
  "confidence": 0.86
}
```

CLI 会将模型返回的 JSON 解析为 `IntentResult`，并校验 intent 是否属于预定义类型、confidence 是否在 0 到 1 之间。

## 第 1 周学习总结

第 1 周完成了 Go 调用 LLM 的基础工程能力：

- 跑通第一次 LLM API 调用
- 将 API Key、Base URL、模型名放入环境配置
- 支持 CLI 输入
- 支持 system instruction
- 支持 streaming 输出
- 支持基于 JSON Schema 的结构化 JSON 输出
- 抽象出 `LLMClient` 接口
- 使用 `context.Context` 控制 timeout
- 接入本地 `.env`
- 增加 makefile 常用命令

## 下一步

第 2 周进入 tool calling：

- 定义 Go 工具
- 使用 JSON Schema 描述工具参数
- 让模型决定调用哪个工具
- Go 侧校验工具参数
- 实现基础 tool call loop
- 增加超时、allowlist、最大调用次数等安全边界

