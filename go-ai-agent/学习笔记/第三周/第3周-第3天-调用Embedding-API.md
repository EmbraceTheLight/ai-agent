# 第 3 周第 3 天：调用 Embedding API

日期：2026-08-21

## 今日目标

今天的目标是完成 RAG 最小闭环中的 embedding 阶段：把 chunk 文本转换为向量，为后续内存向量搜索做准备。

这一阶段不做向量库、不做 topK 排序，也不急着拼接 RAG prompt。重点是先理解 embedding API 的输入输出，并完成一个稳定的本地 embedding 调用封装。

## 今日完成

- 选择本地 Ollama 作为 embedding provider。
- 使用 `qwen3-embedding:0.6b` 作为当前 embedding 模型。
- 在 `.env.example` 中补充 embedding 配置。
- 在 `internal/config` 中读取 embedding base URL 和 model。
- 新增通用 HTTP Client，用于发送 JSON POST 请求。
- 在 `internal/rag` 中定义 `Embedder` 接口。
- 实现 `NewEmbeddingClient`，调用 Ollama `/api/embed`。
- 将 loader、chunker、embedding 串入 `cmd/rag-cli`。
- 为 `rag-cli` 增加参数化配置。
- 在 `makefile` 中增加 `make embed` 快捷命令。
- 补充 embedding 相关测试。

## 当前流程

当前第三周前三天已经形成了下面这条链路：

```text
本地 .md / .txt 文档
  -> LoadRAGResources
  -> Document{SourcePath, Content}
  -> ChunkDocument
  -> Chunk{SourceFile, ChunkIndex, Content, Start, End}
  -> Embed(ctx, []string{chunk.Content})
  -> [][]float64
```

今天完成的是最后一步：

```text
chunk 文本 -> embedding vector
```

## 为什么选择本地 Ollama

第三周的目标是跑通 RAG 最小闭环。embedding 阶段如果依赖在线 API，可能遇到这些干扰：

- 免费额度不稳定。
- 第三方中转站不一定支持 embedding。
- 网络问题会影响学习节奏。
- 不同 provider 的兼容字段可能不同。

因此当前选择本地 Ollama。这样可以把注意力放在 RAG 数据流本身：

```text
文本如何变成向量
向量如何表示语义
后续如何用相似度检索相关 chunk
```

## 当前配置

`.env.example` 中增加了：

```env
EMBEDDING_BASE_URL=http://localhost:11434
EMBEDDING_MODEL=qwen3-embedding:0.6b
```

字段含义：

- `EMBEDDING_BASE_URL`：embedding 服务地址。当前是本地 Ollama。
- `EMBEDDING_MODEL`：embedding 模型名称，必须与 `ollama list` 中显示的名称一致。

当前没有单独配置 `EMBEDDING_API_KEY`，因为本地 Ollama 默认不需要 API Key。这是第三周阶段的合理折中。

如果后续切换到 OpenAI-compatible embedding API，可以再补：

```env
EMBEDDING_API_KEY=xxx
```

## Ollama Embedding API

当前使用的接口是：

```text
POST /api/embed
```

请求体结构：

```json
{
  "model": "qwen3-embedding:0.6b",
  "input": [
    "第一段 chunk 内容",
    "第二段 chunk 内容"
  ]
}
```

响应体核心结构：

```json
{
  "embeddings": [
    [0.1, 0.2, 0.3],
    [0.4, 0.5, 0.6]
  ]
}
```

实际 embedding 向量会很长，不能在日志中完整打印。当前只关心：

```text
输入了多少段文本
返回了多少个 embedding
每个 embedding 的维度是多少
```

## 当前代码结构

### Embedder

`Embedder` 是 RAG 模块中的 embedding 抽象：

```go
type Embedder interface {
    Embed(ctx context.Context, chunks []string) ([][]float64, error)
}
```

它的输入是文本数组，输出是向量数组：

```text
[]string
  -> [][]float64
```

这样设计的原因是：

- Ollama `/api/embed` 支持批量输入。
- 后续 chunk embedding 和 query embedding 都可以复用同一个接口。
- 第四天实现向量搜索时，只需要关心 `[][]float64`，不用关心 HTTP 调用细节。

### EmbedReq / EmbedResp

当前请求和响应结构：

```go
type EmbedReq struct {
    Model string   `json:"model"`
    Input []string `json:"input"`
}

type EmbedResp struct {
    EmbeddingsData [][]float64 `json:"embeddings"`
}
```

其中 `EmbeddingsData` 对应 Ollama 响应里的 `embeddings` 字段。

### embedderClient

`embedderClient` 保存模型名和 HTTP Client：

```go
type embedderClient struct {
    model      string
    httpClient *utils.HttpClient
}
```

调用时会组装请求体：

```go
requestBody := map[string]any{
    "model": ec.model,
    "input": chunks,
}
```

然后请求：

```text
POST {EMBEDDING_BASE_URL}/api/embed
```

## rag-cli 验收方式

当前 `rag-cli` 已经串起完整的前三天流程：

```text
load -> chunk -> embed
```

可以通过命令运行：

```powershell
go run ./cmd/rag-cli -docs ".\testdata\documents" -limitDocs 1 -limitChunks 2 -showPreview=true
```

也可以通过 make 命令：

```powershell
make embed DOCS_PATH=./testdata/documents LIMIT_DOCS=1 LIMIT_CHUNKS=2
```

预期输出重点关注：

```text
加载文档数
chunk 数
本次 embedding chunk 数
embedding 数
embedding dimension
```

这里不打印完整向量，只打印维度。

## 测试覆盖

embedding 测试使用 `httptest` 模拟 Ollama 服务，不依赖真实模型和本地 Ollama 进程。

测试覆盖：

- 请求路径必须是 `/api/embed`。
- 请求方法必须是 `POST`。
- 请求头应包含 JSON content type。
- 请求体中的 `model` 应等于配置模型。
- 请求体中的 `input` 应等于 chunk 文本。
- 返回的 embedding 数量应与输入文本数量一致。
- 每个 embedding 应有固定维度。
- embedding 服务返回非 2xx 时，应向调用方返回错误。

当前验证命令：

```powershell
go test ./internal/rag
```

## 今日核心理解

embedding 不是让模型“回答问题”，而是把文本转换成向量。

可以把 embedding 理解为：

```text
一段文本在语义空间中的坐标
```

意思相近的文本，在向量空间里距离更近；意思不相关的文本，距离更远。

RAG 中需要两类 embedding：

```text
文档 chunk embedding
用户问题 embedding
```

这两类 embedding 必须使用同一个模型生成。不能让文档 chunk 用 `qwen3-embedding:0.6b`，用户问题却用另一个 embedding 模型，否则两个向量空间不一致，后续相似度计算没有意义。

## embedding API 和聊天 API 的区别

聊天 API 的目标是生成自然语言：

```text
messages -> answer text
```

embedding API 的目标是生成向量：

```text
texts -> vectors
```

所以 embedding API 的输出不是最终答案，而是后续检索的中间数据。

当前阶段要避免把这两件事混在一起：

- embedding 负责把 chunk 和问题变成向量。
- vector search 负责找出相关 chunk。
- LLM 负责基于相关 chunk 生成最终回答。

## 向量维度

向量维度由 embedding 模型决定。当前代码不硬编码维度，而是在运行时通过：

```go
len(embeddings[0])
```

获取并打印。

这样做的好处是：

- 后续更换 embedding 模型时，不需要修改业务逻辑。
- 测试可以使用较短的模拟向量。
- 真实运行时可以直接观察当前模型的输出维度。

记录维度时只需要写：

```text
当前模型输出维度：以本地 rag-cli 打印为准
```

不要打印完整 embedding 数组。

## 安全边界

当前 embedding 阶段需要注意：

- 不提交 `.env`。
- 不在日志中打印 API Key。
- 不在日志中打印完整 embedding 向量。
- 测试资料可以使用无敏感内容的个人工作文档、学习笔记或 mock 文档。
- 如果后续切换到在线 embedding provider，需要确认 provider 是否支持 embedding API，而不能只看它是否支持聊天 API。

## 遇到的问题

### 1. 是否应该直接使用 OpenAI embedding

当前阶段没有直接使用 OpenAI embedding，主要是为了降低第三周学习成本。

本地 Ollama 足够完成：

```text
chunk -> embedding -> vector search
```

如果未来要切到 OpenAI embedding，需要重新为所有 chunk 生成向量。不同 embedding 模型的向量不能混用。

### 2. 是否应该把 embedding 存入数据库

第三周暂时不做。

今天只需要拿到 `[][]float64`。第四天先实现内存向量搜索，第五天再把检索结果放进 prompt。PostgreSQL / pgvector 放到第 4 周。

### 3. 是否要打印完整向量排查问题

不建议。

embedding 向量很长，完整打印会污染日志，也不利于阅读。排查时只需要打印：

```text
chunk index
input text preview
embedding dimension
```

## 今日记录

周三完成：

- 完成 embedding 模型配置。
- 完成 Ollama embedding HTTP 调用封装。
- 完成 `Embedder` 接口。
- 完成 chunk 文本批量 embedding。
- 完成 rag-cli 参数化验收入口。
- 完成 embedding 测试。

embedding API 输入输出：

- 输入：模型名称 + 文本数组。
- 输出：与文本数组一一对应的向量数组。

向量维度：

- 当前代码运行时通过 `len(embeddings[0])` 获取。
- 真实维度以本地 Ollama 模型输出为准。
- 日志中只打印维度，不打印完整向量。

配置上的临时折中：

- 当前使用本地 Ollama，不需要 API Key。
- 当前默认模型是 `qwen3-embedding:0.6b`。
- 如果后续切换 provider，需要确认 embedding endpoint、请求结构和响应结构是否兼容。

今天仍然模糊的问题：

- 不同 chunk size 对 embedding 检索效果的影响还没有对比。
- topK 应该取多少还没有实验。
- 还没有实现 cosine similarity，所以暂时只能生成向量，不能检索。
- query embedding 和 chunk embedding 的复用方式需要在第四天实现时进一步确认。

## 下一步准备

下一天进入内存向量搜索。

需要准备：

- 定义 `VectorStore` 最小接口。
- 实现 `Add(chunk, vector)`。
- 实现 `Search(queryVector, topK)`。
- 实现 cosine similarity。
- 验证相似文本能排在更前面。
- 打印 `score`、`source`、`chunk index` 和 `content preview`。

第三天的产物可以总结为：

```text
文档已经能变成 chunk
chunk 已经能变成 embedding
下一步就是让 query embedding 和 chunk embedding 做相似度搜索
```
