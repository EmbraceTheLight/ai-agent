# 第 4 周第 3 天：设计 Milvus Collection Schema

日期：2026-08-31

## 今日目标

今天的目标是为 RAG chunk 设计 Milvus collection schema。

第 4 周前两天已经完成了两件准备工作：

```text
VectorStore 返回 SearchResult{Chunk, Score}
Chunk 中补齐 Title、SourceFile、ChunkIndex、Content、Rune offset、CreatedAt、UpdatedAt
```

今天要把这些 Go 结构体字段映射到 Milvus 中，明确：

```text
哪个字段是主键
哪个字段是向量字段
哪些字段是 scalar field
embedding 维度从哪里来
为什么相似度使用 COSINE
```

## 今日完成

- 新增 `internal/data` 包，用于放置 Milvus 数据访问相关代码。
- 新增 `Data` 结构，保存 Milvus client。
- 新增 `MilvusOperation` 接口和 `MilvusUsecase`，为后续 Milvus 初始化和向量库实现预留边界。
- 新增 Milvus collection 初始化逻辑。
- 设计并实现第一版 RAG chunk collection schema。
- 新增 `MILVUS_ADDR` 配置。
- 新增 `EMBEDDING_DIM` 配置，并让 Milvus 向量字段维度使用该配置。
- 引入 Milvus Go SDK 依赖。
- 修正主键字段，只在向量字段上创建 COSINE 向量索引。
- 保留 `rag-cli` 中 Milvus 初始化注释，等待第四天正式接入时放开。
- 验证 RAG、data、rag-cli 相关包编译通过。

## Collection Schema

当前设计的 collection 名称：

```text
qwen3_embedding_chunk
```

当前字段设计：

```text
id                 VarChar PRIMARY KEY AUTO_ID
source_file_path   VarChar(512)
title              VarChar(256)
chunk_index        Int32
content            VarChar(4096)
created_at         Int64
updated_at         Int64
rune_start_offset  Int32
rune_end_offset    Int32
chunk_vector       FloatVector(dim = EMBEDDING_DIM)
```

其中：

- `id` 是 Milvus 主键，使用 auto id。
- `chunk_vector` 是向量字段。
- 其他字段都是 scalar field，用于返回 chunk metadata、构造引用和后续 filter。
- `chunk_vector` 的维度来自环境变量 `EMBEDDING_DIM`。
- 当前 qwen3 embedding 模型对应维度配置为 `1024`。

## 字段映射

Go 中的 `Chunk` 和 Milvus 字段的映射关系：

```text
Chunk.SourceFile      -> source_file_path
Chunk.Title           -> title
Chunk.ChunkIndex      -> chunk_index
Chunk.Content         -> content
Chunk.CreatedAt       -> created_at
Chunk.UpdatedAt       -> updated_at
Chunk.RuneStartOffset -> rune_start_offset
Chunk.RuneEndOffset   -> rune_end_offset
Vector                -> chunk_vector
```

这个映射让 Milvus 检索返回结果后，可以重新组装出：

```go
SearchResult{
	Chunk: chunk,
	Score: score,
}
```

也就是说，Milvus 不只是保存向量，还要保存能恢复引用信息的 metadata。

## 主键设计

当前主键设计为：

```text
id VarChar PRIMARY KEY AUTO_ID
```

选择 auto id 的原因：

- 第一版不需要自己生成唯一主键。
- 可以先把注意力放在 schema、向量字段和索引上。
- 后续如果需要删除、重建或去重，可以再补 `document_id`、`chunk_id` 或 `content_hash`。

今天明确了一点：

```text
主键不是业务定位字段。
```

真正用于引用和排查的字段仍然是：

```text
source_file_path
title
chunk_index
rune_start_offset
rune_end_offset
```

## 向量字段

当前向量字段：

```text
chunk_vector FloatVector(dim = EMBEDDING_DIM)
```

向量维度来自配置：

```env
EMBEDDING_DIM=1024
```

这样做比直接在代码中写死维度更清楚。

embedding 模型、向量维度和 collection 名称之间存在强绑定关系：

```text
同一个 collection 中的向量必须来自同一个 embedding 模型版本
query embedding 和 chunk embedding 必须处于同一个向量空间
更换 embedding 模型时，通常需要新建 collection 并重新 embedding
```

## 索引设计

当前只给向量字段创建索引：

```go
milvusclient.NewCreateIndexOption(
	collectionName,
	"chunk_vector",
	index.NewAutoIndex(entity.COSINE),
)
```

这里有两个关键点。

第一，`COSINE` 是向量相似度 metric，只应该用于向量字段。

第二，`id` 是主键，不应该用 `COSINE` 创建向量索引。

这次修正后，schema 更符合 RAG 向量检索的语义：

```text
主键负责唯一标识记录
向量字段负责相似度检索
标量字段负责 metadata 和 filter
```

## Scalar Field 的作用

Milvus 中除了向量字段，其他字段都是为 RAG 工程化服务的。

`source_file_path` 可以用于：

```text
按文件或目录过滤
展示引用来源
排查某个回答为什么引用了某篇文档
```

`title` 可以用于：

```text
让 prompt 中的来源更可读
让回答引用不只依赖路径
```

`chunk_index`、`rune_start_offset`、`rune_end_offset` 可以用于：

```text
定位原文片段
排查 chunk 切分是否合理
后续支持更精确的引用展示
```

`created_at`、`updated_at` 可以用于：

```text
记录索引时间
后续按时间过滤或重建索引
```

## 配置项

今天新增或使用的配置：

```env
EMBEDDING_DIM=1024
MILVUS_ADDR=http://localhost:19530
```

其中：

- `EMBEDDING_DIM` 用于创建向量字段。
- `MILVUS_ADDR` 用于后续初始化 Milvus client。

对于 `EMBEDDING_DIM`，当前策略是：

```text
如果配置不能解析为数字，程序直接 panic
```

这是可以接受的，因为 embedding 维度是 schema 的硬约束。配置错误时继续运行，反而可能导致更隐蔽的数据问题。

## 测试结果

本次验证命令：

```powershell
go test ./internal/rag ./internal/data ./cmd/rag-cli
```

结果：

```text
ok   go-ai-agent/internal/rag
?    go-ai-agent/internal/data [no test files]
?    go-ai-agent/cmd/rag-cli [no test files]
```

这说明当前 Milvus schema 相关代码已经能通过编译，且没有破坏现有 RAG 内存检索测试。

## 今日核心理解

今天最重要的理解是：

```text
Milvus collection schema 是 RAG 检索结果能否恢复业务上下文的基础。
```

向量数据库不是只存一个 embedding。

对 RAG 来说，一条向量记录至少应该回答：

```text
这条向量对应哪段文本
这段文本来自哪篇文档
它在文档中的位置是什么
它使用哪个 embedding 维度和模型生成
检索命中后如何构造引用
```

因此 schema 设计要同时考虑：

```text
向量检索
metadata filter
prompt 构造
引用展示
后续重建索引
```

## 当前不足

今天仍然有一些点暂时没有展开：

- 还没有真正启动 Milvus 验证 collection 创建。
- 还没有实现 chunk + vector 的插入。
- 还没有实现 Milvus search。
- 还没有把 `MilvusVectorStore` 接入 `VectorStore` 接口。
- 还没有实现 metadata filter。
- `rag-cli` 中 Milvus 初始化逻辑暂时保留为注释，等待第四天正式接入。
- collection 名称目前仍然写在代码中，后续可以配置化。
- 还没有处理 collection 已存在时的幂等初始化。

这些不足属于第四天“接入 Milvus VectorStore”的任务范围，不影响第三天 schema 设计目标。

## 今日记录

周三完成：

- 设计 Milvus collection schema。
- 明确 `chunk_vector` 是 vector 字段。
- 明确 source、title、index、content、offset、time 是 metadata 字段。
- 使用 `EMBEDDING_DIM` 配置控制向量维度。
- 使用 `COSINE` 作为向量检索 metric。
- 只对向量字段创建 COSINE 索引。
- 保持现有内存 RAG 流程可编译、可测试。

Milvus collection schema：

```text
qwen3_embedding_chunk
```

vector 字段：

```text
chunk_vector FloatVector(dim = EMBEDDING_DIM)
```

metadata 字段：

```text
source_file_path
title
chunk_index
content
created_at
updated_at
rune_start_offset
rune_end_offset
```

索引和相似度选择：

```text
index: AutoIndex
metric: COSINE
```

## 下一步准备

下一天要正式接入 Milvus VectorStore。

需要重点完成：

```text
启动 Milvus
collection 不存在时创建
collection 已存在时跳过或复用
将 chunk + vector 插入 Milvus
从 Milvus search topK
把 Milvus 返回字段组装为 SearchResult
让 rag-cli 支持 memory / milvus 两种 store
```

第四天的关键不是重新设计 schema，而是验证这个 schema 能真正支撑写入和检索。
