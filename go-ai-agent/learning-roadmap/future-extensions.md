# go-ai-agent 未来扩展项

更新时间：2026-08-10

## 文档定位

本文档集中记录当前学习阶段已经识别、但不阻塞近期主线验收的扩展项。

执行原则：

- 先完成每周主线，再处理扩展项。
- 每次只选择一个扩展点，避免同时重构多个模块。
- 扩展前先补对应测试，确保现有行为不会被破坏。
- 优先解决已经影响真实运行的问题，再做框架化和性能优化。

## 优先级 P1：由 Go 手动维护 Responses 上下文

### 背景

当前 Tool Call Loop 使用 `PreviousResponseID` 串联模型响应。在中转站环境中已经观察到：

```text
第一次响应具有 ID
后续请求传入 PreviousResponseID
第二次响应没有返回 PreviousResponseID
模型只记住第一次工具结果，遗忘原始任务中的后续要求
显式设置 Store=true 后行为不变
```

因此，中转站可能没有完整实现 Responses API 的服务端状态管理。

### 扩展目标

由 Go 程序维护一次 Agent Run 的完整上下文，不再依赖服务端保存前序响应。

需要保存：

- 原始用户输入。
- 模型返回的全部 output item。
- `reasoning` item。
- `function_call` item。
- Go 生成的 `function_call_output`。
- 后续模型返回的 message 或新工具调用。

### 设计方向

可以引入一个独立的 Run State 或 Session 结构，由它负责：

```text
保存当前输入历史
追加模型 output
追加工具执行结果
构造下一轮 Responses input
记录 step 和调用次数
```

`FunctionCall` 只负责流程编排，状态对象负责上下文数据，Executor 继续只负责工具执行。

### 验收标准

- 不使用 `PreviousResponseID` 也能完成单工具调用。
- 能连续调用同一个工具，例如分别查询上海和纽约时间。
- 能连续调用不同工具。
- 推理模型的 reasoning item 能完整传递。
- 达到 `MaxSteps` 后能够停止。

## 优先级 P1：可测试的 Agent Loop

### 背景

真实模型在默认 `tool_choice=auto` 时可以直接回答，也可以选择工具，因此不能依赖某个提示词稳定触发特定调用路径。

### 扩展目标

将 Responses 请求能力抽象为可替换依赖，在测试中提供固定响应序列。

建议覆盖：

```text
普通文本回答
一次 function_call 后返回最终回答
连续两次 function_call 后返回最终回答
工具不存在
工具参数非法
工具执行失败
达到 MaxSteps 后仍请求工具
context timeout
```

### 验收标准

- Agent Loop 测试不访问真实模型 API。
- 测试结果不受模型选择和网络状态影响。
- `go test ./...` 可以稳定通过。

## 优先级 P2：完整 JSON Schema Validator

### 背景

当前每个工具都会自行解析 `json.RawMessage` 并做业务校验，例如：

- time 校验时区。
- calculator 校验运算符和除零。
- http_get 校验 method、URL 和 allowlist。

Executor 中已经预留 `validateRequest`，但尚未根据 `Tool.Parameters` 做通用结构校验。

### 扩展目标

在 Executor 调用 Handler 前完成 JSON Schema 校验，Handler 继续负责业务规则校验。

职责划分：

```text
JSON Schema Validator：字段类型、必填字段、枚举、嵌套结构、额外字段
具体 Tool Handler：时区是否合法、除数是否为零、URL 是否允许访问
```

### 参考资料

- JSON Schema Specification：<https://json-schema.org/specification>
- JSON Schema Core：<https://json-schema.org/draft/2020-12/json-schema-core>
- JSON Schema Validation：<https://json-schema.org/draft/2020-12/json-schema-validation>

### 验收标准

- 缺少必填字段时在 Handler 执行前失败。
- 字段类型错误时返回明确错误。
- 非法 enum 和额外字段能够被拒绝。
- 结构校验错误与业务校验错误可以区分。

## 优先级 P2：结构化 Trace 与可观察性

### 扩展目标

将当前分散的 `fmt.Printf` 整理为统一 trace，至少包含：

```text
run ID
step
tool name
arguments
result
error
duration
final status
```

注意对 API Key、authorization header 和其他敏感字段脱敏。

### 验收标准

- 一次 Agent Run 的调用链可以按顺序查看。
- 工具成功、失败和超限都有明确记录。
- trace 不输出密钥或敏感 header。

## 优先级 P2：工具错误回传模型

### 背景

当前工具执行失败后，`FunctionCall` 直接返回 Go error。后续可以根据错误类型决定是否让模型获得安全、可解释的错误结果。

### 扩展方向

- 参数错误：返回简洁的可修正信息，让模型决定是否重新调用。
- 临时网络错误：在明确限制下重试。
- 安全拒绝：不重试，向模型返回不可绕过的拒绝原因。
- 内部错误：只向用户暴露通用信息，详细原因写入 trace。

必须保证模型不能通过重试绕过 allowlist、method 限制和最大调用次数。

## 优先级 P3：支持多个或并行工具调用

当前设置 `ParallelToolCalls=false`，每轮只处理零个或一个工具调用。

后续可以让解析函数返回工具调用列表，并按 `call_id` 收集所有结果，再统一回传模型。

需要考虑：

- 多个调用是否可以并发执行。
- 有副作用的工具能否并发。
- 多个错误如何汇总。
- `MaxSteps` 统计调用轮次还是实际工具数量。
- context 取消时如何停止所有并发任务。

## 优先级 P3：工具注册与配置增强

后续可以逐步补充：

- 注册重复名称时返回错误。
- 拒绝空名称、空 Handler 和非法 Schema。
- 将 HTTP allowlist、响应限制等配置从默认值中提取出来。
- 为只读工具和有副作用工具增加元数据。
- 为敏感工具增加用户确认机制。

## 优先级 P3：成熟 Go Agent 框架源码阅读

该任务不作为近期主线。完成自己的最小 Agent Loop、RAG 和 MCP 后，再对照成熟框架的设计。

建议关注：

- CloudWeGo Eino：<https://github.com/cloudwego/eino>
- Firebase Genkit Go：<https://github.com/firebase/genkit>
- LangChainGo：<https://github.com/tmc/langchaingo>
- openai-agents-go：<https://github.com/nlpodyssey/openai-agents-go>
- Microsoft Agent Framework：<https://github.com/microsoft/agent-framework>

阅读重点：Tool 定义、Schema、Registry、Executor、Agent Loop、状态管理、错误处理和安全边界。

## 建议执行顺序

```text
1. 完成第二周 Demo、测试和复盘
2. 抽象可测试的模型响应接口
3. 补 Agent Loop 固定响应测试
4. 实现 Go 手动上下文
5. 实现完整 JSON Schema Validator
6. 整理结构化 Trace 和错误回传策略
7. 再考虑并行工具调用与框架源码阅读
```
