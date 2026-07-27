# 第 1 周第 4 天：结构化 JSON 输出

日期：2026-07-27

## 今日结论

周四的核心学习目标已经完成。

今天的目标是让模型按固定 JSON 结构返回，而不是只依赖自然语言输出。当前项目已经新增 `GenerateWithJsonSchema`，使用 OpenAI Responses API 的 JSON Schema 输出约束，让模型返回包含 `intent`、`answer`、`confidence` 的结构化结果，并在 CLI 中解析为 Go struct。

## 今日完成

- 定义了 `IntentResult` 结构体。
- 定义了四种意图类型常量：
  - `rag_question`
  - `tool_question`
  - `agent_question`
  - `general_question`
- 在 `Client` 接口中新增 `GenerateWithJsonSchema`。
- 在 `openAIClient` 中实现 JSON Schema 格式输出。
- 使用 `enum` 限制 `intent` 只能从四种意图中选择。
- 使用 `required` 要求模型必须返回 `intent`、`answer`、`confidence`。
- 使用 `additionalProperties: false` 限制模型不要返回额外字段。
- 给 `confidence` 增加 `minimum: 0` 和 `maximum: 1` 约束。
- 在 CLI 中使用 `encoding/json` 将模型输出解析为 `IntentResult`。
- 在解析后额外校验 `confidence` 范围和 `intent` 是否属于预定义类型。

## 输出结构

当前结构化输出目标是：

```json
{
  "intent": "agent_question",
  "answer": "这里是模型回答",
  "confidence": 0.86
}
```

对应 Go 结构体是：

```go
type IntentResult struct {
    Intent     string  `json:"intent"`
    Answer     string  `json:"answer"`
    Confidence float64 `json:"confidence"`
}
```

这一步的价值是：程序不再只拿到一段自然语言，而是可以稳定地拿到可解析、可校验、可分支处理的数据。

## JSON Schema 理解

今天重点理解了 `properties`、`required`、`enum`、`additionalProperties` 的作用。

`properties` 用来定义 JSON 对象中有哪些字段，以及每个字段的类型和约束。

例如：

```text
intent -> string，并且只能是四种意图之一
answer -> string
confidence -> number，并且范围在 0 到 1
```

`required` 用来说明哪些字段必须出现。它应该放在 schema object 的根层级，而不是放进某个字段内部，也不是放进 `properties` 内部。

正确的心智模型是：

```text
schema
├── type: object
├── properties
│   ├── intent
│   ├── answer
│   └── confidence
├── required: ["intent", "answer", "confidence"]
└── additionalProperties: false
```

`enum` 用来限制字段只能取固定值。这里用于限制 `intent` 只能是四种意图之一。

`additionalProperties: false` 用来限制模型不要返回 schema 未定义的额外字段，让输出更稳定。

## 遇到的问题

### required 层级放错

实现过程中曾遇到过错误：

```text
400 Bad Request
Invalid schema for response_format 'IntentResult':
['intent', 'answer', 'confidence'] is not of type 'object', 'boolean'
```

原因是 `required` 被放进了 `properties` 内部。

这样服务端会把 `required` 误认为是一个字段名，而它对应的值是字符串数组，不是合法的字段 schema object，于是报错。

修正后，将 `required` 移到和 `properties` 同一层：

```text
properties: ...
required: ["intent", "answer", "confidence"]
```

模型就可以正常返回结构化 JSON。

## 结构化输出和自然语言输出的区别

自然语言输出适合直接给人看，但程序很难稳定解析。

例如模型可能回答：

```text
这是一个 RAG 相关问题，答案是...
```

这种格式对人友好，但程序要判断意图、提取答案和置信度，就需要做不稳定的字符串解析。

结构化 JSON 输出则适合程序处理：

```json
{
  "intent": "rag_question",
  "answer": "...",
  "confidence": 0.91
}
```

这样 Go 程序可以直接 `json.Unmarshal` 到 struct，然后根据 `intent` 做后续分支逻辑。

## 今日收获

1. JSON Schema 可以约束模型输出结构。
2. `properties` 定义字段，`required` 定义必填字段，二者层级不能混。
3. `enum` 很适合限制 intent 这种固定分类字段。
4. `additionalProperties: false` 可以减少模型返回额外字段的可能。
5. Schema 约束格式，prompt/instruction 负责解释分类语义。
6. 即使 schema 已经限制了输出，Go 侧仍然应该做解析和业务校验。
7. 结构化输出是后续 tool calling、Agent Loop 和流程分发的重要基础。

## 后续可以改进

- 将 JSON Schema 构造逻辑从 `GenerateWithJsonSchema` 中提取出来，减少函数体复杂度。
- 给四种 intent 增加更清晰的分类说明，让模型分类更稳定。
- 当 `json.Unmarshal` 失败时，打印原始响应，方便排查。
- 后续可以让 `GenerateWithJsonSchema` 直接返回 `IntentResult`，但当前返回 string 再由 CLI 解析也适合作为学习阶段的过渡。
- 可以为 JSON 解析和 intent 校验补充基础测试。

## 明天要改进

下一步进入“周五：封装 LLMClient，整理工程结构”。

重点目标是让 `main` 更轻，只负责 CLI 输入输出；让 `internal/llm` 负责模型调用；让 `internal/config` 负责配置读取。这样后续扩展 streaming、JSON 输出、tool calling 和 mock 测试都会更顺。

