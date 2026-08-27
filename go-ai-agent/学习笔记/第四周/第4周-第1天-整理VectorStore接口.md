# 第 4 周第 1 天：整理 VectorStore 接口

日期：2026-08-27

## 今日目标

今天进入第 4 周 RAG 生产化阶段。

第 3 周已经完成了一个基于内存向量搜索的 RAG Demo，本周要逐步把它推进到更接近真实后端服务的形态。

今天的目标不是直接接入 Milvus，而是先整理 `VectorStore` 的接口边界，让后续内存向量库和 Milvus 向量库可以在同一套 RAG 流程里替换。

核心问题是：

```text
RAG 流程应该依赖什么抽象
内存实现的内部结构是否泄漏到了上层
后续 Milvus 返回结果能否自然适配当前接口
```

## 今日完成

- 回顾了当前 `VectorStore` 接口。
- 确认第一版接口继续保持简单：

```go
Add(vector Vector, chunk *Chunk) error
Search(queryVector Vector, topK int) ([]*SearchResult, error)
```

- 将原来的 `CosineSimilarity` 重命名为 `SearchResult`。
- 将检索结果从 `Embedding + Score` 调整为 `Chunk + Score`。
- 将 topK 小根堆从 `SmallRootCosineSimilarity` 重命名为 `SearchResultMinHeap`。
- 同步更新 `prompt` 构造逻辑。
- 同步更新 `rag-cli` 中打印检索结果的逻辑。
- 同步更新内存向量库测试。
- 初步设计了 Milvus collection schema。
- 思考了更换 embedding 模型后维度变化的处理方式。
- 验证 RAG 相关测试通过。

## 接口整理

整理后的 `VectorStore` 仍然只暴露两个核心能力：

```text
Add: 写入 chunk 及其 embedding
Search: 根据 query embedding 检索 topK chunk
```

当前接口：

```go
type VectorStore interface {
	Add(vector Vector, chunk *Chunk) error
	Search(queryVector Vector, topK int) ([]*SearchResult, error)
}
```

这个接口保持了第 3 周 Demo 的使用方式，也为第 4 周 Milvus 接入留下了空间。

上层 RAG 流程只需要知道：

```text
我要写入一个 chunk 的向量
我要根据问题向量取回相关 chunk
```

它不需要知道底层是内存切片、Milvus collection，还是其他向量数据库。

## SearchResult 抽象

原来的检索结果结构更偏向内存实现：

```text
Embedding
  Chunk
  Vector
Score
```

这对内存向量库很自然，因为内存里确实保存了完整的 `Embedding`。

但对 Milvus 来说，检索返回后真正需要交给上层的是：

```text
Chunk
Score
```

其中：

- `Chunk` 用于构造 prompt 和引用来源。
- `Score` 用于表示 query 和 chunk 的相似度。
- `Vector` 是底层检索过程中的数据，不应该出现在检索结果里。

因此将返回结构调整为：

```go
type SearchResult struct {
	Chunk *Chunk
	Score float64
}
```

这个调整让 `SearchResult` 更像业务层的检索结果，而不是某个具体存储实现的内部记录。

## 为什么不返回 Embedding

如果 `Search` 返回 `Embedding`，上层就会知道底层保存了：

```text
Chunk + Vector
```

这会带来两个问题。

第一，上层 prompt 构造并不需要 `Vector`。

第二，Milvus 检索时不一定需要把原始向量返回给 Go 代码。Milvus 可以只返回标量字段和 score，例如 source file、chunk index、content、start、end 等。

因此 `SearchResult` 只包含 `Chunk` 和 `Score` 更符合后续持久化向量库的返回方式。

## 内存实现的保留

虽然 `SearchResult` 不再暴露 `Embedding`，但内存向量库内部仍然可以继续用 `Embedding` 保存数据：

```text
defaultVectorStore
  []*Embedding

Embedding
  Chunk
  Vector
```

这是合理的。

接口层只负责暴露稳定抽象，具体实现内部如何存储可以不同。

内存实现中：

```text
Add 保存 Embedding
Search 计算 cosine similarity
Search 返回 SearchResult
```

后续 Milvus 实现中：

```text
Add 插入 Milvus collection
Search 调 Milvus search
Search 将返回字段组装为 SearchResult
```

两者可以共享同一个 `VectorStore` 接口。

## Milvus Schema 预研

今天初步设计的 Milvus collection schema：

```text
ID          INT64 PRIMARY KEY AUTO_ID
SourceFile  VARCHAR(512)
ChunkIndex  INT32
RuneStart   INT32
RuneEnd     INT32
Content     VARCHAR(4096)
CreateTime  INT64
Embedding   FLOAT_VECTOR(dim = 当前 embedding 维度)
```

其中：

- `ID` 是 Milvus 自动主键。
- `Embedding` 是向量列。
- `SourceFile`、`ChunkIndex`、`RuneStart`、`RuneEnd`、`Content`、`CreateTime` 是标量列。
- 标量列后续可以用于返回 `Chunk` 或做 metadata filter。

今天只是 schema 预研，不急着接入 Milvus SDK。

## Embedding 维度问题

今天还明确了一个重要边界：

```text
向量维度是 collection schema 的一部分。
```

如果后续更换 embedding 模型，并且新模型输出维度和当前模型不同，不能把新旧向量混在同一个 vector field 中。

更稳妥的做法是：

```text
新建 collection
重新对原始文档生成 embedding
将 query embedding 和 chunk embedding 保持在同一个模型版本
```

即使两个模型输出维度相同，也不代表它们处于同一个语义空间。后续可以在配置中显式维护：

```text
EMBEDDING_MODEL
EMBEDDING_DIM
RAG_COLLECTION
```

这样更方便做模型切换和索引重建。

## 测试结果

RAG 相关测试通过：

```powershell
go test ./internal/rag ./cmd/rag-cli
```

结果：

```text
ok   go-ai-agent/internal/rag
?    go-ai-agent/cmd/rag-cli [no test files]
```

全量测试中仍有其他包的失败，但不是今天 `SearchResult` 调整导致的：

```text
internal/llm/output_schema_test.go 中存在未使用变量
internal/tools 中 GetCurrentTime 测试期望值和当前 JSON 输出不一致
```

## 今日核心理解

今天最重要的理解是：

```text
接口返回值应该表达业务需要，而不是暴露某个实现的内部存储结构。
```

对 RAG 检索来说，上层真正需要的是：

```text
命中的 chunk
相似度 score
```

至于底层为了计算相似度保存了什么、Milvus 如何组织 collection、向量字段是否返回给 Go，这些都应该留在具体 `VectorStore` 实现内部。

这个调整让后续 Milvus 接入更自然。

## 当前不足

今天仍然有一些点暂时没有展开：

- `Search` 还不支持 metadata filter。
- `SearchResult` 还没有独立的 `Source` 或 `Citation` 字段，目前引用信息仍然从 `Chunk` 中读取。
- Milvus collection schema 只是草案，还没有真正落地。
- 还没有处理重复导入、删除文档、重建索引等问题。
- `Vector` 当前是 `[]float64`，后续接入 Milvus 时可能需要在实现层转换为 `[]float32`。

这些问题不影响第 4 周第 1 天目标，可以放到后续 metadata、Milvus 接入和 HTTP API 阶段处理。

## 今日记录

周一完成：

- 整理 `VectorStore` 接口。
- 将检索返回类型改为 `SearchResult`。
- 让 `SearchResult` 只包含 `Chunk` 和 `Score`。
- 清理旧的 `CosineSimilarity` 类型残留注释。
- 同步更新 prompt、rag-cli 和测试。
- 初步确认 Milvus collection schema 设计方向。

VectorStore 接口调整：

```text
Add(vector, chunk)
Search(queryVector, topK) -> SearchResult
```

SearchResult 包含字段：

```text
Chunk
Score
```

为了 Milvus 预留的能力：

```text
检索结果不再依赖内存 Embedding
Search 可以由 Milvus 返回字段组装结果
Chunk 字段可以映射到 Milvus 标量列
Vector 字段可以映射到 Milvus 向量列
后续可以在 Search 中扩展 metadata filter
```

## 下一步准备

下一天重点是补全 chunk metadata 和测试。

需要重点检查：

```text
SourceFile 是否稳定
ChunkIndex 是否稳定
Start / End 是否应该命名为 RuneStart / RuneEnd
Prompt 中引用来源是否准确
SearchResult 是否能保留完整引用信息
```

后续进入 Milvus 之前，要确保每个 chunk 都能被稳定追踪、过滤和引用。
