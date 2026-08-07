# 第 2 周第 5 天：基础 Tool Call Loop

日期：2026-08-06

## 今日目标

今天的目标是实现基础 Tool Call Loop，打通下面这条完整链路：

```text
用户提出问题
模型决定是否调用工具
Go 解析 function_call
Executor 查找并执行工具
Go 将 function_call_output 回传模型
模型继续调用工具或生成最终回答
```

在此基础上，还需要限制工具调用次数，避免模型反复请求工具形成无限循环。

## 今日完成

- 在 LLM `Client` 接口中接入 `FunctionCall` 方法。
- 将本地注册的 `Tool` 转换为 OpenAI Responses API 使用的工具参数。
- 将工具名称、描述、严格模式和 Parameters JSON Schema 传给模型。
- 从 `resp.Output` 中识别 `function_call` 类型的输出。
- 读取模型返回的工具名称、调用 ID 和 JSON 参数。
- 通过 `Executor.Execute` 查找并执行对应工具。
- 使用 `call_id` 将工具执行结果包装为 `function_call_output` 回传模型。
- 使用循环支持模型在工具执行后继续生成响应。
- 使用 `Executor.MaxSteps` 限制最大工具调用次数。
- 区分普通最终回答和工具调用次数超限两种退出情况。
- 设置 `ParallelToolCalls=false`，当前只处理每轮零个或一个工具调用。
- 在首次请求和后续请求中持续传入完整工具列表。
- 使用 `PreviousResponseID` 尝试衔接前后两次 Responses 请求。
- 显式设置 `Store=true`，验证中转站的响应状态保存能力。
- 增加工具调用次数和已调用工具名称的 trace 输出。
- 为 CLI 和 Makefile 接入 `function_call` 输出方式。

## Tool Call Loop 的工作流程

第一次请求包含用户问题和可用工具定义。模型可以直接返回文本，也可以返回 `function_call`。

当模型返回工具调用时，Go 程序执行以下流程：

```text
读取 resp.Output
找到 function_call
取得 name、arguments、call_id
交给 Executor.Execute
取得工具执行结果
构造 function_call_output
通过 call_id 关联原工具调用
再次请求模型
检查新响应是否还需要工具
```

如果新响应中不存在 `function_call`，说明模型已经结束工具调用，此时返回最新响应中的文本内容。

## functionCall 辅助函数

为了让主循环更容易理解，新增了 `functionCall` 辅助函数。它负责遍历 `resp.Output`，查找 `function_call` 类型的输出。

它的两个返回值分别表示：

- 工具调用的具体信息。
- 当前响应中是否存在工具调用。

主循环根据第二个返回值决定下一步：

```text
存在 function_call：执行工具并继续请求模型
不存在 function_call：结束循环并返回最终回答
```

这使“响应解析”和“工具调用编排”具有更清晰的职责边界。

## 最大调用次数

`MaxSteps` 限制的是一次用户请求中实际执行工具的最大次数。

正确的判断逻辑是：

```text
模型不再请求工具 -> 正常返回最终回答
已执行 MaxSteps 次，模型仍请求工具 -> 返回调用次数超限错误
第 MaxSteps 次调用后模型给出最终回答 -> 允许正常返回
```

这样既能支持多步任务，也能避免模型无限调用工具。

## 串行工具调用

当前实现设置了：

```text
ParallelToolCalls = false
```

它表示模型每轮最多返回一个工具调用，便于当前阶段使用单个 `functionCall` 结果进行处理。

需要注意，关闭并行调用不代表模型必须调用工具。默认工具选择仍由模型决定，因此模型可能：

- 不调用工具，直接回答。
- 调用一次工具后结束。
- 在获得结果后继续请求另一个工具。

简单数学问题可能被模型直接计算，不能仅根据模型是否主动选择 `calculator` 判断 Agent Loop 是否正确。

## 调试与验证过程

本次主要进行了以下验证：

- 查询当前时间时，模型能够返回 `function_call`。
- Go 能正确执行 `time` 工具。
- 工具结果能够通过 `function_call_output` 回传模型。
- 模型能够基于时间工具结果生成 `message` 类型的最终回答。
- 最终响应中包含 `reasoning` 和 `message` 两类 output。
- 当 `message.Phase` 为 `final_answer` 时，说明模型已经主动结束本轮任务。

在尝试连续工具调用时，发现了两个现象：

```text
查询上海和纽约时间时，模型只调用一次 time 并返回上海时间
第一次响应具有 ID，第二次响应没有返回 PreviousResponseID
```

即使两次请求都显式设置了 `Store=true`，结果仍然相同。

结合第二轮模型遗忘“还需要查询纽约”的表现，目前推测中转站可能没有完整实现 Responses API 的 `previous_response_id` 状态衔接。这个问题不代表本地循环无法再次执行工具，而是第二轮模型可能没有获得完整的原始任务上下文。

## 今日核心理解

Tool Call Loop 本质上是由 Go 程序负责控制的状态循环：

```text
模型负责决定下一步
Go 负责执行和约束工具
Executor 负责查找并调用工具
call_id 负责关联调用与结果
上下文负责让模型记住尚未完成的任务
MaxSteps 负责保证循环能够停止
```

同时，模型的工具选择和程序的循环能力是两个不同问题：

- 模型没有选择某个工具，不一定表示该工具无法执行。
- 程序支持多轮循环，不代表模型一定会发起多轮调用。
- 使用真实模型进行验证具有不确定性。
- 稳定验证循环边界需要模拟固定的响应序列。

## 可拓展项：由 Go 手动维护上下文

后续可以不依赖中转站的 `PreviousResponseID`，改为由 Go 程序维护完整的 Responses 输入历史。

计划维护的内容包括：

```text
原始用户输入
模型返回的全部 resp.Output
reasoning item
function_call item
对应的 function_call_output
后续模型输出
```

每次请求完成后，将模型输出和工具结果追加到上下文；下一次请求时重新发送完整上下文和工具定义。

需要特别注意：对于推理模型，手动管理上下文时不能只保存 `function_call`，还应保留模型返回的 reasoning item，否则可能破坏前后响应的推理连续性。

该扩展可以进一步封装为独立的会话或 Agent 状态结构，避免 `FunctionCall` 方法承担过多上下文管理职责。

## 后续待完善

- 使用 Go 手动维护完整 Responses 上下文，替代或兼容 `PreviousResponseID`。
- 为 Tool Call Loop 编写不依赖真实模型的模拟响应测试。
- 测试“普通回答、单次工具调用、连续工具调用、调用次数超限”四条路径。
- 根据需要支持一次响应中的多个工具调用。
- 设计工具执行失败后回传模型的错误结果，让模型生成更友好的最终回答。
- 将当前调试 trace 逐步整理为更统一的日志格式。

## 今日总结

今天完成了基础 Tool Call Loop：模型可以请求工具，Go 可以通过 Executor 执行工具，并将结果回传模型生成最终回答。同时加入了串行调用约束、最大调用次数和基础 trace。

本次排查还认识到，Agent Loop 不只是一个 `for` 循环，它还依赖可靠的上下文传递。当前中转站对 `PreviousResponseID` 的支持可能不完整，因此将“由 Go 手动维护完整上下文”记录为后续可拓展项。
