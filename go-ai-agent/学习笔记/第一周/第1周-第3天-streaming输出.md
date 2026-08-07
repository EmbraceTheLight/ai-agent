# 第 1 周第 3 天：加入 streaming 输出

日期：2026-07-23

## 今日结论

周三的核心学习目标已经完成。

今天的目标是让模型回答逐步输出，而不是等完整响应返回后一次性打印。当前程序已经实现了 `Stream` 方法，并通过 `onDelta` 回调把模型返回的增量内容实时打印到终端。

虽然学习路线中示例使用 `--stream "问题"` 这种命令行参数形式，但当前实现选择通过 stdin 输入 system instruction 和用户问题。这是一个合理的阶段性取舍：输入方式和路线示例不同，但 streaming 能力本身已经跑通。

## 今日完成

- 定义了 `Client` 接口，包含普通生成和流式生成两个能力。
- 定义了基础 `Message` 类型，用于表达 system/user 消息。
- 实现了 `openAIClient.Generate`，用于普通非流式请求。
- 实现了 `openAIClient.Stream`，用于流式请求。
- 在 `Stream` 中使用 OpenAI Go SDK 的 streaming 调用。
- 通过 `onDelta func(string)` 把每次返回的增量文本交给调用方处理。
- 在 CLI 中使用 `fmt.Print(delta)` 连续打印增量内容，避免每个 delta 单独换行。
- 使用 stdin 输入 system instruction 和 user question，让测试不同上下文更灵活。
- 将 API Key、Base URL、Model 统一从 `config` 中读取。
- `Generate` 中已经使用外部传入的 `ctx` 派生超时上下文，并将错误返回给调用方。

## Client 接口理解

当前接口是：

```go
type Client interface {
    Generate(ctx context.Context, messages []Message) (string, error)
    Stream(ctx context.Context, messages []Message, onDelta func(string)) error
}
```

它的含义是：把“调用大模型”抽象成统一能力。

`Generate` 表示普通请求：

- 输入一组消息。
- 等模型完整生成。
- 一次性返回完整文本。

`Stream` 表示流式请求：

- 输入一组消息。
- 模型每生成一小段内容，就触发一次回调。
- 调用方可以决定如何处理这些增量内容。

这个接口让 CLI 不需要关心底层到底是 OpenAI、中转站，还是未来别的模型供应商。后续如果要替换模型供应商，只要新的 client 实现同样的接口即可。

## onDelta 的作用

`onDelta` 是流式输出中的回调函数。

它的作用是：每当模型返回一小段增量文本时，`Stream` 方法就把这段文本交给 `onDelta`。

例如在 CLI 中可以这样处理：

```go
func(delta string) {
    fmt.Print(delta)
}
```

这样终端会逐步显示模型回答，体验接近 ChatGPT 的实时输出。

今天遇到过一个现象：模型输出被切成很多小块，每个字或词单独占一行。原因不是 streaming 错了，而是打印时使用了类似 `fmt.Println(delta)` 的方式。`Println` 会给每个 delta 自动加换行，所以看起来像被拆碎了。

改成 `fmt.Print(delta)` 后，增量文本会连续拼接，输出效果自然很多。

## streaming 和普通请求的区别

普通请求：

```text
发送请求 -> 等完整回答生成完 -> 一次性拿到完整文本
```

优点是调用简单，返回值就是完整字符串。缺点是用户必须等待整个回答完成，长回答时会显得卡顿。

streaming 请求：

```text
发送请求 -> 模型生成一段就返回一段 -> 程序边接收边显示
```

优点是响应更快，用户能马上看到模型开始回答。缺点是代码需要处理增量事件，输出逻辑也要交给回调函数。

## 输入方式选择

学习路线中建议使用：

```powershell
go run ./cmd/assistant-cli --stream "解释一下 Agent Loop"
```

当前实现暂时没有加入 `--stream` 参数，而是直接通过 stdin 输入内容。

这个选择的优点是：

- 可以更方便地分别输入 system instruction 和用户问题。
- 不需要在命令行中处理较长文本的引号和转义。
- 在学习阶段更适合反复试验不同提示词。

后续如果要做成更标准的 CLI，可以再加入：

```text
--stream
```

让普通请求和流式请求通过参数切换。

## 封装时遇到的问题

### 1. messages 应该放在哪里

一开始考虑过把 messages 放进 `openAIClient` 内部，并通过 `Add` 方法添加消息。

后来理解到：如果 messages 只是一次请求的输入，更适合作为参数传给 `Generate` 或 `Stream`。这样 `openAIClient` 保持相对无状态，更容易测试，也不容易出现不同请求之间消息互相污染的问题。

如果后续要做多轮对话，可以再引入 Conversation 或 Session 这类对象来维护历史消息。

### 2. ctx 应该如何传递

`Generate` 和 `Stream` 都接收 `context.Context`。这意味着调用方可以控制请求生命周期，例如超时或取消。

封装时要注意，不要直接丢弃调用方传入的 `ctx`。如果需要加超时，可以基于传入的 `ctx` 派生新的 timeout context。

### 3. 错误应该返回而不是直接退出

接口已经设计成返回 `error`，所以底层实现遇到错误时更适合返回给调用方，而不是直接 `log.Fatal`。

这样 CLI 可以决定是打印错误、重试，还是退出程序。这个边界更清晰。

## 今日收获

1. streaming 的核心不是一次性返回完整文本，而是持续返回增量文本。
2. `onDelta` 让流式方法不关心输出方式，只负责把增量交给调用方。
3. `fmt.Print` 比 `fmt.Println` 更适合直接打印 streaming delta。
4. `Client` 接口让 CLI 与具体模型供应商解耦。
5. messages 作为请求参数传入，比直接放进 client 内部更清晰。
6. `context.Context` 是控制请求超时和取消的重要入口。
7. stdin 输入在学习阶段很灵活，后续可以再补标准 CLI 参数。

## 明天要改进

下一步进入“周四：结构化 JSON 输出”。

重点目标是让模型按固定 JSON 结构返回，并在 Go 中解析成 struct。

明天可以优先思考：

- 为什么不能依赖自然语言随便解析？
- 如何用 prompt 要求模型输出 JSON？
- Go 中如何用 `encoding/json` 把字符串解析成结构体？
- 如果模型返回的 JSON 不合法，程序应该如何提示原始响应和错误？

