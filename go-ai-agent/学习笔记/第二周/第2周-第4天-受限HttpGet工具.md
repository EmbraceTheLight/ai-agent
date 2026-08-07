# 第 2 周第 4 天：受限 HttpGet 工具

日期：2026-08-04

## 今日目标

今天的目标是实现一个受限的 `http_get` 工具，理解工具调用中的安全边界。重点不是写一个通用 HTTP 客户端，而是让模型只能在 Go 程序允许的范围内访问外部资源。

## 今日完成

- 实现了 `HttpGet` 工具函数。
- 定义了 `HttpGetReq` 请求结构。
- 为 `http_get` 定义了参数 JSON Schema。
- 将 allowlist 放入 `httpGetTool` 中，由工具自身管理访问范围。
- 默认只允许 `GET` 方法。
- 使用 `url.Parse` 解析 URL。
- 只允许 `http` / `https` 协议。
- 使用 `parsed.Hostname()` 做域名 allowlist 校验。
- 使用 `http.NewRequestWithContext`，让 Executor 传入的 timeout context 能真正作用到 HTTP 请求。
- 支持 query 参数和 header 参数。
- 使用 `io.LimitReader` 限制响应体大小。
- 使用 `httptest` 编写了稳定的本地测试。
- 执行 `go test ./internal/tools`，测试通过。

## 工具参数设计

`http_get` 的请求结构为：

```go
type HttpGetReq struct {
    URL    string            `json:"url"`
    Method string            `json:"method"`
    Header map[string]string `json:"header"`
    Query  map[string]string `json:"query"`
}
```

参数含义：

- `url`：请求目标 URL。
- `method`：请求方法，当前只允许 `GET`。
- `header`：可选请求头。
- `query`：可选 query 参数。

当前工具虽然保留了 `method` 字段，但 schema 和 Go 侧都只允许 `GET`。这样可以保持工具行为简单，也能避免模型把 `http_get` 用成通用写操作 HTTP 工具。

## JSON Schema 理解

`http_get` 的 JSON Schema 主要约束：

- 参数整体必须是 object。
- `url` 必须是 string。
- `method` 必须是 string，且只能是 `GET`。
- `header` 是 object，value 必须是 string。
- `query` 是 object，value 必须是 string。
- 不允许额外字段。
- `url` 和 `method` 必填。

这次进一步理解了 map 类型在 JSON Schema 中的表达：

```text
Go: map[string]string
JSON Schema: type object + additionalProperties: { type: string }
```

## 安全边界

`http_get` 不能开放任意 URL，因为模型可能被用户诱导访问：

- 内网地址。
- 本机敏感服务。
- 云厂商 metadata API。
- 任意大文件。
- 不可信外部网站。

因此当前工具设置了几个边界：

```text
只允许 http / https
只允许 allowlist 中的 hostname
默认只允许 GET
HTTP 请求使用 context timeout
限制响应体大小
```

allowlist 放在 `httpGetTool` 中是合理的，因为这是 HTTP 工具自己的安全配置，不属于 Executor 的通用职责。

## Context Timeout 理解

Executor 会创建带 timeout 的 context，并把它传给工具 handler。但 context 本身只是取消信号，不会自动中断所有操作。

HTTP 工具必须继续把 ctx 传给真正执行网络请求的 API：

```go
http.NewRequestWithContext(ctx, method, url, nil)
```

这样 `client.Do` 才能在 ctx 超时或取消时停止请求。

这个理解可以推广到其他阻塞操作：

```text
HTTP 请求：NewRequestWithContext
数据库查询：QueryContext / ExecContext
外部命令：exec.CommandContext
自定义等待：select 监听 ctx.Done()
```

## 响应大小限制

最开始的写法是在完整读取响应体后再判断大小。这样虽然能发现超限，但已经把超大响应读进了内存。

改进后使用：

```go
io.LimitReader(resp.Body, limit+1)
```

这里多读取 1 个字节，是为了区分：

```text
响应刚好等于 limit
响应超过 limit
```

如果读取到的字节数大于 limit，就返回错误，避免继续处理过大的响应。

## 测试整理

本次使用 `httptest` 搭建本地 HTTP 服务，而不是直接请求真实天气 API。

测试模拟了天气查询参数：

```text
city=anyang
adcode=410502
lang=zh
```

测试覆盖：

- allowlist 内 URL 可以访问。
- query 参数会正确拼接到请求 URL。
- header 会正确传入。
- allowlist 外 URL 会被拒绝。
- 非 `GET` 方法会被拒绝。
- 响应超过大小限制会返回错误。

使用本地 mock 服务的好处是：

```text
测试稳定
不依赖外网
不受真实 API 响应变化影响
可以精确验证请求参数和错误场景
```

## 今日核心理解

今天最重要的理解是：一旦工具可以访问外部资源，它就不再只是“功能问题”，而是“安全边界问题”。

模型只负责提出工具调用意图，Go 程序必须负责控制工具能做什么、不能做什么。

当前 `http_get` 的职责边界是：

```text
Executor：设置 timeout context，统一调用 handler。
httpGetTool：保存 allowlist、method allowlist、响应大小限制。
HttpGet：解析参数、校验 URL 和 method、执行 HTTP 请求。
测试：验证允许、拒绝和超限场景。
```

## 后续待完善

- 可以根据需要决定是否处理非 2xx HTTP 状态码。
- 可以把 `method` 从参数中移除，直接在 Go 侧固定为 `GET`。
- 可以继续补充 timeout 测试。
- 可以补充 URL path 级别的 allowlist。
- 后续接入 Agent Loop 后，需要把工具调用 trace 打印出来。
