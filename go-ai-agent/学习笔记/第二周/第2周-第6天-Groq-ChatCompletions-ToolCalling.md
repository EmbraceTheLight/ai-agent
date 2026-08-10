# 第 2 周第 6 天：迁移 Groq Chat Completions Tool Calling

日期：2026-08-16

## 今日目标

今天的目标是解决第三方中转站对 OpenAI Responses API 上下文状态支持不稳定的问题，尝试改用 Groq 的 OpenAI-compatible Chat Completions 接口实现 Tool Calling。

重点不是重写所有 LLM 能力，而是先让 `function_call` 路径跑通：

```text
用户问题
模型返回 tool_calls
Go 执行本地工具
Go 将 tool 结果追加到 messages history
模型继续决定是否调用工具
最终生成自然语言回答
```

## 今日完成

- 确认第三方中转站的问题主要出在 Responses API 的上下文衔接上。
- 决定使用 Groq 作为 Tool Calling 的替代模型服务。
- 理解 Chat Completions 与 Responses API 的上下文格式差异。
- 使用 `groqClient` 重写 `FunctionCall` 方法，暂时不改普通生成、流式输出和 JSON Schema 输出。
- 复用现有 `tools.Executor` 和已注册工具。
- 将本地工具转换为 Chat Completions 使用的 `tools` 参数。
- 使用 `messages` 数组手动维护完整对话历史。
- 将模型返回的 assistant `tool_calls` 消息追加到 history。
- 将工具执行结果作为 `tool` 消息追加到 history，并通过 `tool_call_id` 关联原工具调用。
- 每轮请求继续传入 `Tools`，支持模型继续请求下一个工具。
- 使用 `MaxSteps` 限制最大工具调用次数。
- 验证无需工具的问题可以直接回答，不会触发工具调用。
- 验证连续工具调用与跨工具调用可以正常执行。

## Chat Completions 与 Responses API 的区别

之前使用 Responses API 时，上下文结构更细：

```text
user input
reasoning
function_call
function_call_output
message
```

其中 `function_call_output` 需要和模型返回的 `call_id` 对应。如果依赖 `PreviousResponseID`，还需要服务端正确保存上一轮响应状态。

Chat Completions 的结构更朴素，核心是 `messages` 数组：

```text
system
user
assistant(tool_calls)
tool(tool_call_id, result)
assistant(final answer)
```

也就是说，状态主要由 Go 程序维护。每次请求都把完整 history 发给模型，而不是依赖服务端保存上一轮 response。

## Groq 迁移策略

当前阶段没有把整个项目都迁移到 Groq，而是只迁移 Tool Calling 路径。

采用的方式是：

```text
groqClient 内嵌 openAIClient
重写 groqClient.FunctionCall
其他方法暂时沿用 openAIClient
```

这样可以降低改动范围。需要注意的是，`Generate`、`Stream`、`GenerateWithJsonSchema` 仍然是 Responses API 风格，当前不作为 Groq 的验收范围。

## Tool Calling History 顺序

今天最关键的理解是：Chat Completions 的工具调用历史必须按固定顺序追加。

正确顺序是：

```text
原始 system/user 消息
assistant 消息，包含 tool_calls
tool 消息，包含 tool_call_id 和工具结果
下一轮 assistant 消息
```

不能只把工具结果发给模型。模型需要同时看到：

- 原始用户任务是什么。
- 它上一轮请求了哪个工具。
- 当前工具结果对应哪个 `tool_call_id`。

因此每轮工具调用后都需要维护完整 history，而不是重建成只有最新消息的 input。

## 今日验证

测试问题：

```text
请告诉我现在广州和洛杉矶的时间, 并调用工具告诉我 4*5 的结果是多少
```

实际工具调用日志：

```text
第 1 次 tool 调用, tool 名称: time, 返回结果: {"current_time":"2026-08-16 22:21:39","time_zone":"Asia/Shanghai"}
第 2 次 tool 调用, tool 名称: time, 返回结果: {"current_time":"2026-08-16 07:21:39","time_zone":"America/Los_Angeles"}
第 3 次 tool 调用, tool 名称: calculator, 返回结果: 20.000000
```

这个测试覆盖了两类能力：

```text
连续调用同类工具：time -> time
跨工具调用：time -> calculator
```

广州使用 `Asia/Shanghai` 时区是合理的；洛杉矶使用 `America/Los_Angeles`。在当前日期下，洛杉矶处于夏令时，与中国时间相差 15 小时，日志中的时间差符合预期。

另外，询问不需要工具的问题时，模型可以直接回答，且没有触发工具调用。这说明 `ToolChoice=auto` 下模型可以在“直接回答”和“调用工具”之间做选择。

## 工具调用失败策略

今天讨论了工具调用失败时的处理方式。

当前阶段采用更简单、可控的策略：

```text
工具执行失败
FunctionCall 直接返回 error
用户侧看到通用错误
后台记录具体错误信息
```

这样便于调试，也避免把内部错误、请求细节、路径、上游响应等敏感信息暴露给用户。

后续可以增强为：

```text
工具执行失败
Go 将脱敏后的结构化错误作为 tool result 回传模型
模型生成友好的自然语言解释
```

例如：

```json
{
  "ok": false,
  "error_code": "divide_by_zero",
  "message": "除数不能为 0"
}
```

但不应把原始 stack trace、完整 HTTP error、header、API key、内部路径等内容直接回传给模型或用户。

## 今日核心理解

今天最大的收获是：Tool Calling 的稳定性不只取决于模型能否返回工具调用，还取决于上下文协议是否被完整维护。

Responses API 更细，能力更强，但第三方兼容可能不稳定；Chat Completions 更朴素，第三方兼容更好，也更适合当前学习阶段手动理解 Agent Loop。

当前 Groq Tool Calling 的核心链路是：

```text
messages history
-> assistant tool_calls
-> Executor.Execute
-> tool message
-> append history
-> next chat completion
-> final answer
```

这也进一步强化了 Agent 的职责边界：

```text
模型负责决策
Go 负责执行工具
Executor 负责工具查找、参数传递、超时和错误包装
messages history 负责保存任务上下文
MaxSteps 负责防止无限循环
```

## 后续待完善

- 为 Groq Tool Calling 增加不依赖真实模型的单元测试。
- 覆盖普通回答、单工具调用、连续工具调用、跨工具调用、工具失败、MaxSteps 超限等路径。
- 决定 CLI 是否增加 provider 参数，避免 Groq client 影响非 function_call 输出模式。
- 将工具调用日志整理成统一 trace 格式。
- 后续实现工具错误的安全结构化回传，让模型生成更友好的最终回答。
- 继续评估是否抽象通用 Chat Completions client，复用到 Groq、Gemini、OpenRouter 等兼容服务。

## 今日总结

今天完成了 Groq Chat Completions Tool Calling 的主链路验证。通过手动维护 `messages` history，模型能够连续调用多个工具，也能够跨工具调用，并在不需要工具时直接回答。

这说明当前 Tool Call Loop 已经从“单次工具调用”推进到“可连续决策的最小 Agent Loop”。第二周 Tool Calling 的核心能力已经基本跑通，后续重点可以转向测试、错误处理、安全回传和日志整理。
