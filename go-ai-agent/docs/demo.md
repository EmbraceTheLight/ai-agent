# 第 1 周 Demo 演示脚本

本文档记录第 1 周 CLI AI 助手的固定演示流程。

## 演示前准备

请在项目根目录执行命令：

```powershell
cd <path-to-project>\go-ai-agent
```

确认当前目录下存在 `.env`：

```env
OPENAI_API_KEY=<your-api-key>
OPENAI_BASE_URL=<your-openai-compatible-base-url>
OPENAI_MODEL=<model-name>
```

注意：不要提交 `.env`，真实 API Key 只保存在本地。

## Demo 1：普通输出

命令：

```powershell
make run
```

PowerShell 等价命令：

```powershell
go run ./cmd/assistant-cli
```

程序提示输入问题时，输入：

```text
什么是 AI Agent？
```

预期结果：

```text
输出
<模型生成的完整回答>
```

这个 Demo 验证：

- CLI 可以读取用户输入。
- 程序可以调用 OpenAI 兼容 LLM API。
- 普通模式会等模型完整生成后一次性输出回答。

## Demo 2：流式输出

命令：

```powershell
make stream
```

PowerShell 等价命令：

```powershell
go run ./cmd/assistant-cli --stream
```

程序提示输入问题时，输入：

```text
解释一下 Agent Loop
```

预期结果：

```text
<模型回答会逐步显示>
```

这个 Demo 验证：

- CLI 支持 streaming 模式。
- `Stream` 方法可以逐步接收模型返回的 delta。
- `onDelta` 回调可以把增量文本实时打印到终端。

## Demo 3：结构化 JSON 输出

命令：

```powershell
make json
```

PowerShell 等价命令：

```powershell
go run ./cmd/assistant-cli --json
```

程序提示输入问题时，输入：

```text
MCP 和 tool calling 有什么关系？
```

预期结果：

```text
回答: <模型回答内容>
============================== 分界线 ==============================
整个 ans 结构: {Intent:tool_question Answer:... Confidence:...}
```

这个 Demo 验证：

- CLI 支持 JSON 输出模式。
- 模型输出受到 JSON Schema 约束。
- 程序可以把模型返回的 JSON 解析为 `IntentResult`。
- 程序会校验 `intent` 是否属于预定义类型、`confidence` 是否在 0 到 1 之间。

## Demo 4：通过命令行参数输入问题

命令：

```powershell
go run ./cmd/assistant-cli --stream "用 Go 怎么学习 AI Agent？"
```

预期结果：

```text
<模型回答会逐步显示>
```

这个 Demo 验证：

- 用户问题可以通过命令行参数传入。
- 当命令行中已经提供问题时，不需要再从 stdin 输入问题。

## Demo 5：自定义 system instruction

命令：

```powershell
go run ./cmd/assistant-cli --instruction "你是一个回答非常简洁的 Go 后端学习助手。" "解释 Go interface 的用途"
```

预期结果：

```text
输出
<更简洁的模型回答>
```

这个 Demo 验证：

- CLI 可以通过 `--instruction` 覆盖默认 system instruction。
- 可以用 system instruction 调整模型的回答风格。

## Demo 6：设置 timeout

命令：

```powershell
go run ./cmd/assistant-cli --timeout 30s "解释 streaming 和普通请求的区别"
```

预期结果：

```text
输出
<模型回答内容>
```

这个 Demo 验证：

- CLI 可以创建带 timeout 的 `context.Context`。
- LLM 调用会尊重请求上下文。

## Demo 7：运行测试

命令：

```powershell
make test
```

PowerShell 等价命令：

```powershell
go test ./...
```

预期结果：

```text
ok      ...
```

如果当前还没有测试文件，Go 可能会提示某些 package 没有 test files。只要命令本身能正常完成，就说明测试入口是可用的。

## Windows 编码说明

在 Windows 上，如果把中文问题直接写进 `makefile`，可能因为 GNU Make、`cmd.exe`、PowerShell 或当前代码页不一致导致乱码。

推荐使用以下方式之一：

- makefile 只负责选择运行模式，中文问题通过 stdin 输入。
- 在支持 UTF-8 的终端中直接运行 `go run ... "中文问题"`。
- makefile 中只放英文示例问题。

当前项目采用第一种方式：`make run`、`make stream`、`make json` 只负责启动对应模式，具体问题由运行时输入。

## 演示检查清单

在认为第 1 周 Demo 完成前，请确认：

- `make run` 可以运行。
- `make stream` 可以运行。
- `make json` 可以运行。
- `make test` 会执行 `go test ./...`。
- `.env.example` 已存在。
- `.env` 不会被提交。
- `README.md` 和 `README_zh-CN.md` 已说明项目如何运行。
- 普通输出、streaming 输出、JSON 输出都能稳定复现。

