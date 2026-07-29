# 第 1 周第 5 天：封装 LLMClient，整理工程结构

日期：2026-07-28

## 今日结论

周五的核心学习目标已经完成。

今天的目标是把散落在 `main` 中的 LLM 调用整理到 `internal/llm`，让 CLI 主入口只负责参数解析、消息组装和输出模式分发。当前项目已经有了 `Client` 接口、OpenAI client 实现、配置读取、命令行 flag 和 makefile 入口，为后续 tool calling、Agent Loop 和工具注册表打下了更清晰的工程基础。

## 今日完成

- 将 LLM 能力抽象为 `Client` 接口。
- 在接口中保留普通输出、流式输出和 JSON Schema 输出三种能力。
- 使用 `openAIClient` 实现具体 OpenAI Responses API 调用。
- 将模型调用逻辑从 CLI 主流程中抽离到 `internal/llm`。
- 将环境变量读取集中到 `internal/config`。
- 新增命令行 flag 配置，用于控制输出模式。
- 支持通过 `--stream` 选择流式输出。
- 支持通过 `--json` 选择结构化 JSON 输出。
- 普通模式下调用 `Generate`。
- 流式模式下调用 `Stream`，并通过 `onDelta` 实时打印增量文本。
- JSON 模式下调用 `GenerateWithJsonSchema`，再解析为 `IntentResult`。
- 新增 `makefile`，提供 `run`、`stream`、`json`、`test` 等入口。

## 当前项目结构理解

当前代码已经开始形成清晰边界：

```text
cmd/assistant-cli
  负责 CLI 入口、参数解析后的分发、输出展示

internal/config
  负责环境变量、默认配置、命令行 flag、常量

internal/llm
  负责 Message、Client 接口、OpenAI 实现、结构化输出类型

internal/tools
  负责通用输入读取等辅助能力
```

这个结构比把所有逻辑都写在 `main.go` 里更容易扩展。后续第 2 周做 tool calling 时，可以继续新增 `internal/tools` 或 `internal/agent`，而不用让 CLI 主入口变得越来越重。

## LLMClient 接口理解

当前 `Client` 接口表达的是“一个 LLM 客户端应该具备哪些能力”：

```go
type Client interface {
    Generate(ctx context.Context, messages []Message) (string, error)
    GenerateWithJsonSchema(ctx context.Context, messages []Message) (string, error)
    Stream(ctx context.Context, messages []Message, onDelta func(string)) error
}
```

它的价值是让调用方不直接依赖 OpenAI SDK 的细节。

CLI 不需要知道：

- Responses API 的参数结构。
- JSON Schema 如何传给 SDK。
- streaming 事件如何循环读取。
- OpenAI client 如何初始化。

CLI 只需要根据输出模式调用对应方法即可。

这就是接口封装的意义：把“怎么调用模型”的复杂度藏到 `internal/llm`，把“用户想怎么使用 CLI”的逻辑留在 `cmd/assistant-cli`。

## 输出模式分发

当前 CLI 已经可以根据命令行参数决定输出方式：

```text
默认模式 -> Generate
--stream -> Stream
--json   -> GenerateWithJsonSchema
```

这个分发方式让前三天和第四天的成果整合到同一个 CLI 中：

- 周一、周二的普通问答能力。
- 周三的 streaming 输出能力。
- 周四的结构化 JSON 输出能力。

这一步很重要，因为它把“零散练习代码”变成了一个可以逐步演进的小工具。

## makefile 理解

新增 makefile 后，可以用更短的命令运行常见模式：

```powershell
make run
make stream
make json
make test
```

这让后续演示和自测更方便，也为周六 Demo 做准备。

今天还遇到过 Windows 下 makefile 中文参数乱码的问题。原因是 makefile 本身是 UTF-8，但 Windows shell 或 make 执行链路可能使用 CP936，导致中文命令行参数在进入 Go 程序前已经乱码。

因此当前选择让 makefile 只指定运行模式，具体问题通过 stdin 输入，是更稳妥的方式。

## 目前还可以改进的地方

这些不影响周五核心目标完成，但后续可以继续收口：

- `make test` 当前应确认是否实际执行的是 `go test ./...`。
- `--model` 已经在 flag 中定义，但主流程当前仍主要使用环境变量中的模型配置。
- `--timeout` 已经在 flag 中定义，但模型请求当前仍主要使用配置中的默认超时。
- CLI 中 JSON 输出后的校验逻辑后续可以封装到更靠近 `llm` 或独立 validation 的位置。
- `main` 仍然承担了一部分 JSON 解析和业务校验，后续可以继续变薄。
- 后续可以增加 mock client，为 `Client` 接口写基础测试。

## 哪些代码还不够干净

目前比较明显的是：CLI 主流程仍然知道一些 JSON 输出细节，例如：

- 如何 `json.Unmarshal`。
- 如何校验 `confidence`。
- 如何判断 `intent` 是否在枚举范围内。

这在学习阶段完全可以接受，因为它能帮助理解结构化输出的完整链路。

但如果继续工程化，可以思考：

```text
GenerateWithJsonSchema 是否应该直接返回 IntentResult？
intent 校验是否应该放到 IntentResult 自己的方法里？
输出展示是否应该从 main 中拆出去？
```

这些问题可以留到周末 Demo 或后续重构时处理，不必今天一次性完成。

## 今日收获

1. `main` 应该尽量薄，主要负责输入输出和流程分发。
2. LLM API 调用细节应该封装在 `internal/llm`。
3. `Client` 接口让后续替换模型供应商或写 mock 更容易。
4. CLI 参数可以把普通输出、streaming、JSON 输出整合到一个程序中。
5. makefile 能固定常用命令，降低演示成本。
6. Windows 下 makefile 里直接写中文参数容易遇到编码问题。
7. 工程结构不是为了好看，而是为了后续 tool calling 和 Agent Loop 不失控。

## 明天要改进

下一步进入“周六：做成本周可展示 Demo”。

明天重点是把第 1 周成果整理成可以稳定演示的 CLI：

- 固定几条演示命令。
- 确认普通输出、streaming、JSON 输出都能运行。
- 补 README 或 demo 文档。
- 记录环境变量配置方式。
- 尽量让别人能在几分钟内理解如何运行项目。

