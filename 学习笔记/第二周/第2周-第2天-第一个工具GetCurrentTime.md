# 第 2 周第 2 天：第一个工具 GetCurrentTime

日期：2026-08-03

## 今日目标

今天的目标是实现第一个低风险工具 `get_current_time`，理解一个 tool 不只是一个普通函数，还需要包含给模型看的定义信息、参数 JSON Schema，以及 Go 侧真正执行的 handler。

## 今日完成

- 实现了 `GetCurrentTime` 工具函数。
- 将工具函数签名调整为统一的 `ToolFunc` 形式。
- 为 `get_current_time` 定义了参数请求结构 `GetCurrentTimeReq`。
- 为时间工具定义了参数 JSON Schema。
- 新增了 `ITool` 接口，用于约束工具需要提供参数 schema 和 handler。
- 定义了 `Tool` 结构，包含 `Name`、`Description`、`Strict`、`Parameters` 和 `Handler`。
- 定义了 `Executor` 雏形，用于注册工具、查找工具、统一执行工具。
- 修正了 `Executor` 的 map 初始化问题，新增 `NewExecutor`。
- 使用 table-driven test 重写了 `GetCurrentTime` 测试。
- 本地执行 `go test ./internal/tools` 后测试通过。

## 工具结构理解

当前一个完整工具可以拆成两部分：

```text
给模型看的部分：
Name
Description
Parameters
Strict

给 Go 程序执行的部分：
Handler
```

`Name` 用于在模型返回 tool call 后查找工具。

`Description` 用于帮助模型判断什么时候应该调用这个工具。

`Parameters` 是工具参数的 JSON Schema，用于约束模型生成的 arguments 结构。

`Handler` 是 Go 侧真正执行工具逻辑的函数。

## 时间工具实现

`GetCurrentTime` 现在接收 `json.RawMessage`，内部再解析成自己的请求结构：

```go
type GetCurrentTimeReq struct {
    TimeZone string `json:"time_zone"`
}
```

这样可以让所有工具都遵循统一的 handler 签名：

```go
type ToolFunc func(ctx context.Context, req json.RawMessage) (string, error)
```

Executor 不需要知道每个工具具体的请求结构。它只负责把请求转换成 `json.RawMessage`，再交给对应工具的 handler。具体工具再负责解析和校验自己的业务参数。

## 参数 JSON Schema

时间工具的参数 schema 描述了模型调用该工具时应该传入的参数结构：

```text
type: object
properties:
  time_zone:
    type: string
required:
  time_zone
additionalProperties: false
```

这里 `time_zone` 被设计为必填字段，因此代码注释中不再保留“默认 UTC”的描述，避免 schema 和业务语义不一致。

这个设计让我进一步理解了三种数据的区别：

```text
Parameters JSON Schema：描述参数应该长什么样。
raw arguments：模型实际传来的 JSON 参数。
Req struct：Go 内部使用的参数结构。
```

它们的关系是：

```text
Parameters JSON Schema 约束 raw arguments。
raw arguments 解析成 Req struct。
Req struct 交给具体工具执行逻辑使用。
```

## Executor 理解

Executor 不负责某个工具的业务参数校验，而是负责工具调用的通用流程：

```text
接收 tool name 和 arguments
将请求转换成 json.RawMessage
根据 tool name 查找工具
检查工具是否存在
设置 timeout context
调用工具 handler
统一返回结果或错误
```

具体工具仍然负责自己的业务规则：

```text
time_zone 是否能被 time.LoadLocation 加载
calculator 是否除以 0
http_get 是否命中 allowlist
```

因此当前分层可以理解为：

```text
Executor 校验工具调用外层流程。
Tool Handler 校验业务参数并执行具体逻辑。
```

## 测试整理

`GetCurrentTime` 测试已经改成 table-driven test，覆盖了：

- 有效时区 `Asia/Shanghai`。
- 非法时区 `invalid`。

测试中会先将请求结构体 marshal 成 JSON，再传入 `GetCurrentTime`，从而匹配实际 tool calling 中 handler 接收 `json.RawMessage` 的方式。

成功场景会校验：

- 返回字符串能按照固定格式解析。
- 返回时间与当前时间足够接近。

失败场景会校验：

- 非法时区会返回 error。

## 今日核心理解

今天最重要的理解是：tool 不是普通业务函数，而是“模型可见定义”和“Go 可执行函数”的组合。

```text
模型需要看到 name、description、parameters。
Go 需要根据 name 找到 handler。
Executor 负责统一调用流程。
具体工具负责自己的业务解析和校验。
```

这为后续新增 `calculator` 和 `http_get` 打好了基础。之后新增工具时，应该尽量只新增工具自己的 schema、handler 和测试，而不是反复修改 executor 的核心流程。

## 后续待完善

- `validateRequest` 目前仍是 TODO，后续需要引入完整 JSON Schema validator。
- `Execute` 的注释可以和实际返回值保持一致。
- `RegisterTool` 后续可以补充 nil tool、空 name、重复注册等防御。
- `GetCurrentTime` 中 timeout context 已经传入，但本地时间函数很快，实际效果不明显；后续 `http_get` 需要真正把 ctx 传入 HTTP 请求。
