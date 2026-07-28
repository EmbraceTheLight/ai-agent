# go-ai-agent

English | [简体中文](README_zh-CN.md)

A Go learning project for building an AI Agent backend step by step. The current version is a small CLI assistant that calls an OpenAI-compatible LLM API, supports normal text generation, streaming output, and structured JSON output.

This project follows the learning roadmap in `../learning-roadmap`.

## Features

- OpenAI-compatible API access through `github.com/openai/openai-go/v3`
- Local `.env` configuration through `github.com/joho/godotenv`
- Normal question answering
- Streaming response output
- Structured JSON output with JSON Schema
- Intent classification into four types:
  - `rag_question`
  - `tool_question`
  - `agent_question`
  - `general_question`
- `LLMClient` interface abstraction
- CLI flags for output mode, model, instruction, and timeout
- Makefile commands for common run modes

## Project Structure

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

## Requirements

- Go version compatible with `go.mod`
- An OpenAI or OpenAI-compatible API key
- Optional: `make`

## Configuration

Create a local `.env` file in this project root:

```env
OPENAI_API_KEY=<your-api-key>
OPENAI_BASE_URL=<your-openai-compatible-base-url>
OPENAI_MODEL=<model-name>
```

Example:

```env
OPENAI_API_KEY=sk-...
OPENAI_BASE_URL=https://example.com/v1
OPENAI_MODEL=gpt-5.5
```

Do not commit `.env`. The repository keeps `.env.example` as the shareable template.

## Usage

Run the CLI:

```powershell
go run ./cmd/assistant-cli
```

Run with a question argument:

```powershell
go run ./cmd/assistant-cli "What is an AI Agent?"
```

Use streaming output:

```powershell
go run ./cmd/assistant-cli --stream "Explain Agent Loop"
```

Use structured JSON output:

```powershell
go run ./cmd/assistant-cli --json "What is the relationship between MCP and tool calling?"
```

Override the model:

```powershell
go run ./cmd/assistant-cli --model gpt-5.5 "Explain RAG"
```

Override the system instruction:

```powershell
go run ./cmd/assistant-cli --instruction "You are a concise Go backend tutor." "Explain Go interfaces"
```

Set a timeout:

```powershell
go run ./cmd/assistant-cli --timeout 30s "Explain streaming output"
```

## Make Commands

```powershell
make run
make stream
make json
make test
```

On Windows, avoid putting Chinese text directly in `makefile` commands if your shell or GNU Make uses a legacy code page. This project keeps make commands simple and lets the program read questions from stdin when needed.

## Output Modes

Normal mode returns a full answer after the model finishes generating.

Streaming mode prints each delta as soon as the model returns it:

```text
model delta -> onDelta -> fmt.Print(delta)
```

JSON mode asks the model to return an object like:

```json
{
  "intent": "agent_question",
  "answer": "The answer text",
  "confidence": 0.86
}
```

The CLI parses this JSON into `IntentResult` and validates the intent and confidence range.

## Week 1 Summary

During week 1, this project completed the foundation for LLM API access in Go:

- Ran the first LLM request from Go
- Moved API key, base URL, and model into environment configuration
- Added CLI input
- Added system instruction support
- Added streaming output
- Added structured JSON output with JSON Schema
- Extracted an `LLMClient` interface
- Added timeout handling with `context.Context`
- Added local `.env` loading
- Added Makefile shortcuts

## Next Step

Week 2 focuses on tool calling:

- Define Go tools
- Describe tool parameters with JSON Schema
- Let the model decide which tool to call
- Validate tool arguments in Go
- Build a basic tool call loop
- Add safety boundaries such as timeout, allowlist, and maximum steps

