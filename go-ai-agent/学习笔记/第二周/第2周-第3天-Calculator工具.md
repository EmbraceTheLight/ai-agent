# 第 2 周第 3 天：Calculator 工具

日期：2026-08-03

## 今日目标

今天的目标是实现第二个工具 `calculator`，让工具支持带参数的业务逻辑，并理解 JSON Schema 约束、Go 侧参数解析、业务校验之间的边界。

## 今日完成

- 实现了 `Calculate` 工具函数。
- 定义了 `CalculateReq` 请求结构。
- 支持四种基础运算：加、减、乘、除。
- 为 calculator 工具定义了参数 JSON Schema。
- 将 calculator 工具接入 `ITool` 结构，提供 schema 和 handler。
- 编写了 table-driven test。
- 测试覆盖了成功运算、除以 0、不支持的运算符和非法 JSON。
- 使用本地 Go 环境执行 `go test ./internal/tools`，测试通过。

## 参数设计

calculator 工具的请求结构为：

```go
type CalculateReq struct {
    Operator     string  `json:"operator"`
    LeftOperand  float64 `json:"left_operand"`
    RightOperand float64 `json:"right_operand"`
}
```

参数含义：

- `operator`：运算类型，支持 `add`、`subtract`、`multiply`、`divide`。
- `left_operand`：左操作数。
- `right_operand`：右操作数。

## JSON Schema 理解

calculator 的 JSON Schema 主要约束：

- 参数整体必须是 object。
- `operator` 必须是 string。
- `operator` 只能是枚举值：`add`、`subtract`、`multiply`、`divide`。
- `left_operand` 和 `right_operand` 必须是 number。
- 三个字段都必须存在。
- 不允许额外字段。

这次修正时重点确认了三处名称必须一致：

```text
schema properties
schema required
Go struct json tag
```

如果 schema 使用 `right_operand`，但 Go struct tag 写成 `right_operator`，模型即使按 schema 生成了正确参数，Go 侧也无法解析到对应字段。这类问题不会总是被业务代码立即发现，但会在 tool calling 接入模型后变成隐蔽 bug。

## Go 侧业务校验

JSON Schema 负责约束参数结构，但 Go 侧仍然需要做业务校验。

当前 calculator 在 handler 中处理了：

- JSON 参数解析失败。
- 不支持的运算符。
- 除以 0。

其中，除以 0 属于业务规则，不适合只依赖 JSON Schema 处理。即使后续 executor 引入完整 JSON Schema validator，`Calculate` 仍然应该保留这类业务校验。

## 测试整理

本次新增 `calculator_test.go`，使用 table-driven test 覆盖：

- `3 + 5 = 8`
- `10 - 4 = 6`
- `6 * 7 = 42`
- `8 / 2 = 4`
- `8 / 0` 返回错误
- 不支持的运算符返回错误
- 非法 JSON 返回错误

成功场景中，测试会把工具返回的字符串解析成 `float64` 再比较结果，避免直接依赖字符串格式。

执行命令：

```powershell
$env:GOROOT='D:\Environments\golang\go1.26.2'
$env:GOTOOLCHAIN='local'
& 'D:\Environments\golang\go1.26.2\bin\go.exe' test ./internal/tools
```

执行结果：

```text
ok go-ai-agent/internal/tools
```

## 今日核心理解

今天最重要的理解是：带参数工具需要同时维护三层一致性。

```text
模型看到的 JSON Schema
模型实际传入的 raw arguments
Go 侧解析使用的 Req struct
```

其中 JSON Schema 解决的是“参数形状是否正确”，Go handler 解决的是“参数在业务上是否可执行”。

因此 calculator 工具的职责边界可以理解为：

```text
Executor：统一工具调用流程。
JSON Schema：约束参数结构。
Calculate：解析参数并执行业务校验。
测试：覆盖正常路径和关键错误路径。
```

## 后续待完善

- 后续 executor 中实现完整 JSON Schema validator 后，可以在调用 handler 前先做结构校验。
- 当前工具返回值是字符串，之后可以考虑返回结构化结果。
- 可以继续补充 executor 层面的测试，例如工具不存在、handler 为空、请求无法 marshal。
- 后续接入模型时，需要把 `Tool` 定义转换成 OpenAI tools 参数。
