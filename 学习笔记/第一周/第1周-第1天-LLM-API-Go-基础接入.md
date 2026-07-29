# 第 1 周第 1 天：LLM API 和 Go 基础接入

日期：2026-07-21

## 今日结论

第一天的核心学习目标已经完成。

根据 `learning-roadmap/week-01-llm-api-go-detail.md` 中“周一：跑通第一次 LLM 调用”的要求，当前项目已经完成：

- 安装并使用 `github.com/openai/openai-go/v3`。
- 从环境变量读取 `OPENAI_API_KEY`。
- 支持通过 `OPENAI_BASE_URL` 配置 OpenAI 兼容中转站地址。
- 支持通过 `OPENAI_MODEL` 配置模型名。
- 写死一个测试问题：`请用三句话解释什么是 AI Agent。`
- 使用 OpenAI Go SDK 发起一次模型请求。
- 在终端成功打印 AI 返回结果。

因此，“用 Go 发起一次最简单的 LLM 请求，并在终端打印回答”这一核心目标已经达成。

## 尚未完成或后续可补充

这些不是第一天最低要求，但属于路线中的加餐或本周交付内容：

- 还可以给请求加 `context.WithTimeout`，避免请求长时间阻塞。
- 可以补充 `.env.example`，列出 `OPENAI_API_KEY`、`OPENAI_BASE_URL`、`OPENAI_MODEL`。
- 后续可将入口目录调整为路线建议的 `cmd/assistant-cli/main.go`。
- 后续需要补 `README.md`、基础测试、`LLMClient` 接口、streaming 输出和结构化 JSON 输出。

## 使用的模型

当前模型从环境变量 `OPENAI_MODEL` 读取。

如果没有配置环境变量，代码默认使用：

```text
gpt-5.4-mini
```

实际可用模型取决于中转站支持的模型列表。

## 关键代码理解

当前 `main.go` 使用了 OpenAI Go SDK：

```go
client := openai.NewClient(opts...)
```

这里的 `opts` 至少包含：

```go
option.WithAPIKey(config.OpenaiApiKey)
```

如果配置了中转站地址，还会追加：

```go
option.WithBaseURL(config.OpenaiBaseURL)
```

这说明：

- `API Key` 负责认证。
- `BaseURL` 决定请求发送到哪里。
- 使用中转站时，必须把请求地址指向中转站的 OpenAI-compatible API 地址。

随后通过 Responses API 发起请求：

```go
resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
    Input: responses.ResponseNewParamsInputUnion{
        OfString: openai.String(question),
    },
    Model: config.OpenaiModel,
})
```

最后通过：

```go
fmt.Println(resp.OutputText())
```

打印模型输出文本。

## 遇到的问题

### 1. Incorrect API key provided

最初报错：

```text
Incorrect API key provided
```

原因不是 key 一定错误，而是中转站 key 被发到了 OpenAI 官方默认 API 地址。OpenAI 官方服务无法识别中转站签发的 key，所以返回认证失败。

解决方式：

```powershell
$env:OPENAI_BASE_URL="https://你的中转站-api域名/v1"
```

### 2. 返回 text/html 而不是 JSON

之后又遇到：

```text
expected destination type of 'string' or '[]byte' for responses with content-type 'text/html; charset=utf-8' that is not 'application/json'
```

这个错误说明 SDK 收到的是 HTML 页面，而不是 API JSON 响应。

最终原因是 `OPENAI_BASE_URL` 没有加 `/v1` 后缀，请求打到了错误地址。

解决后：

```text
OPENAI_BASE_URL = 中转站 API 地址 + /v1
```

程序可以正常返回 AI 输出。

## 今日收获

1. OpenAI Go SDK 默认请求 OpenAI 官方 API 地址。
2. 使用中转站或 OpenAI 兼容供应商时，必须显式配置 `BaseURL`。
3. `BaseURL` 通常应该以 `/v1` 结尾，不能填控制台首页、登录页或完整接口路径。
4. API 返回 `text/html` 时，通常说明请求没有进入真正的 JSON API。
5. 不应该把 API Key 写死在代码里，应通过环境变量读取。
6. 模型名也应该配置化，因为不同中转站支持的模型名称可能不同。

## 明天要改进

根据路线，下一步进入“周二：理解消息角色，做成命令行问答”：

- 把写死的问题改成命令行参数。
- 支持类似下面的运行方式：

```powershell
go run ./cmd/assistant-cli "请解释 Go interface 的用途"
```

- 增加固定 system prompt。
- 如果用户没有传入问题，打印 CLI 使用方式。

