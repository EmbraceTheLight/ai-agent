# 第 4 周第 2 天：补全 Chunk Metadata 和测试

日期：2026-08-29

## 今日目标

今天的目标是让每个 chunk 不只是保存文本内容，还能稳定保存和传递来源信息。

第 3 周的 RAG Demo 已经可以完成：

```text
文档 -> chunk -> embedding -> 向量检索 -> prompt -> 回答
```

但如果检索结果只知道“这段文本相关”，而不知道它来自哪篇文档、标题是什么、位于第几个 chunk，就很难做可信引用，也不方便后续接入 Milvus 的 metadata filter。

因此今天重点是补全 chunk metadata，并用测试证明这些信息不会在 RAG 链路中丢失。

## 今日完成

- 补充 `Document.Title`，让加载后的文档携带标题。
- 补充 `Chunk.Title`，让 chunk 继承来源文档标题。
- 保留 `Chunk.SourceFile`、`Chunk.ChunkIndex` 和 `Chunk.Content`，用于定位和引用。
- 将偏移字段整理为 `RuneStartOffset` 和 `RuneEndOffset`，明确它们是 rune 级偏移，而不是字节偏移。
- 为 `Chunk` 增加 `CreatedAt` 和 `UpdatedAt` 时间戳。
- 在 `ChunkDocument` 中写入标题、来源路径、chunk index、rune offset 和时间戳。
- 在 prompt 构造中加入标题信息，让引用更清楚。
- 补充 loader、chunker、prompt、vector store 相关测试。
- 验证 RAG 相关测试通过。

## Metadata 字段

当前 `Document` 结构：

```go
type Document struct {
	Title      string
	SourcePath string
	Content    string
}
```

当前 `Chunk` 结构：

```go
type Chunk struct {
	SourceFile      string
	Title           string
	ChunkIndex      int
	Content         string
	CreatedAt       int64
	UpdatedAt       int64
	RuneStartOffset int
	RuneEndOffset   int
}
```

这些字段可以分成三类。

第一类是引用信息：

```text
Title
SourceFile
ChunkIndex
```

它们用于告诉用户答案来自哪篇文档、哪个片段。

第二类是文本和位置：

```text
Content
RuneStartOffset
RuneEndOffset
```

它们用于保存 chunk 正文和它在原始文档中的位置。

第三类是时间信息：

```text
CreatedAt
UpdatedAt
```

它们为后续持久化、重建索引、按时间过滤预留空间。

## Loader 到 Chunk 的传递

今天的一个重点是确认标题从文档加载阶段进入 chunk 阶段。

链路如下：

```text
TriliumDocumentLoader
-> Document.Title
-> ChunkDocument
-> Chunk.Title
```

对于 Trilium 导出的 Markdown 文档，标题解析规则是：

```text
优先读取第一行 Markdown 标题
如果没有标题，则回退到文件名
```

这样做的价值是：

```text
prompt 不只能展示文件路径
检索结果可以带更友好的文档标题
后续 Milvus 标量列可以保存 title 字段
```

## Rune Offset

原来的 `Start` / `End` 容易让人误解为字节偏移。

由于中文、英文和符号混合文本中，一个字符可能对应多个字节，所以今天将字段调整为：

```text
RuneStartOffset
RuneEndOffset
```

它表达的是 rune 级位置。

例如：

```text
你好x
```

其中 `x` 的 rune 偏移是 2，但字节偏移不是 2。

这个命名让字段语义更清楚，也避免后续在 Milvus schema 或测试中产生误解。

## Prompt 中的引用信息

现在 `BuildPrompt` 会把标题、来源文件、chunk index 和 score 都写入 prompt：

```text
[标题: xxx, 来源: path#chunk-0, 得分: 0.987600]
chunk content
```

这比只放正文更可靠。

模型在回答时能看到：

```text
这段内容来自哪篇文档
它的标题是什么
它是第几个 chunk
它和问题的相似度是多少
```

这为“基于资料回答”和“给出引用来源”提供了更好的上下文。

## 测试覆盖

今天的测试重点不是单纯提高覆盖率，而是证明 metadata 在链路中不丢失。

### Loader 测试

loader 测试覆盖：

```text
递归加载 .md / .txt
跳过不支持的扩展名
SourcePath 是绝对路径
Content 与源文件一致
Title 能从 Markdown 标题中解析
无标题时能回退到文件名或文本首行
```

这证明文档加载阶段能产出稳定的 `Document`。

### Chunker 测试

chunker 测试覆盖：

```text
普通文本按 size 切分
overlap 生效
中文内容按 rune 切分
短文本返回单个 chunk
overlap 为 0 时不重叠
非法参数返回错误
Document.Title 会传递到 Chunk.Title
SourceFile、ChunkIndex、Content、RuneStartOffset、RuneEndOffset 不丢失
```

这证明 chunk 阶段不会破坏来源和位置信息。

### VectorStore 测试

vector store 测试覆盖：

```text
Search 返回 topK
结果按 score 降序排列
SearchResult 能拿到 Chunk
Chunk 中保留 SourceFile 和 ChunkIndex
空向量、nil chunk、非法 topK、维度不一致、零向量会返回错误
```

这证明检索阶段不会只返回分数，而是能带回可引用的 chunk 信息。

### Prompt 测试

prompt 测试覆盖：

```text
prompt 包含标题
prompt 包含来源文件和 chunk index
prompt 包含 score
prompt 包含 chunk 正文
```

这证明最终交给模型的上下文包含可追踪来源。

## 和 Milvus 的关系

今天的 metadata 设计也对应后续 Milvus collection schema。

可以映射为：

```text
Chunk.SourceFile      -> source_file VARCHAR
Chunk.Title           -> title VARCHAR
Chunk.ChunkIndex      -> chunk_index INT32
Chunk.Content         -> content VARCHAR
Chunk.RuneStartOffset -> rune_start INT32
Chunk.RuneEndOffset   -> rune_end INT32
Chunk.CreatedAt       -> created_at INT64
Chunk.UpdatedAt       -> updated_at INT64
Vector                -> embedding FLOAT_VECTOR
```

后续接入 Milvus 时，向量列用于相似度检索，标量列用于：

```text
返回 SearchResult.Chunk
构造 prompt 引用
实现 metadata filter
排查检索结果
```

因此今天的工作不是单纯给结构体加字段，而是在为持久化向量检索铺好数据边界。

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

全量测试中还存在一个与本日 RAG 工作无关的失败：

```text
internal/llm/output_schema_test.go: declared and not used: testcases
```

这个文件当前是未跟踪文件，不影响 `internal/rag` 的完成判断。

## 今日核心理解

今天最重要的理解是：

```text
RAG 的引用可信度依赖 metadata 的稳定传递。
```

只保存 chunk 文本是不够的。

一个可展示、可调试、可继续生产化的 RAG 系统，至少需要知道：

```text
这段文本来自哪里
它在原始文档中的位置
它属于哪篇文档或标题
它为什么会进入 topK
```

metadata 不是附属信息，而是 RAG 工程化的基础。

## 当前不足

今天仍然有一些点暂时没有展开：

- 还没有实现真正的 metadata filter。
- `CreatedAt` 和 `UpdatedAt` 目前由 chunker 生成，还没有和源文件修改时间绑定。
- 还没有 `DocumentID` 或 `ContentHash`，后续如果要去重或重建索引会需要补充。
- `SourceFile` 当前保存绝对路径，后续展示引用时可以考虑转为相对路径。
- Milvus schema 还没有真正落地，字段长度和类型还需要在接入时确认。

这些不足不影响第二天目标，可以放到 Milvus schema、filter 和导入流程阶段继续处理。

## 今日记录

周二完成：

- 补全 `Document` 和 `Chunk` 的 metadata 字段。
- 让 loader 能解析并保留文档标题。
- 让 chunker 将标题、来源、序号、rune offset 和时间戳写入 chunk。
- 让 prompt 引用中包含标题、来源和分数。
- 补充多层测试，验证 metadata 不会在 RAG 链路中丢失。

metadata 包含字段：

```text
Title
SourceFile
ChunkIndex
Content
RuneStartOffset
RuneEndOffset
CreatedAt
UpdatedAt
```

暂时不做的字段：

```text
DocumentID
ContentHash
Author
Tags
真实文件更新时间
```

metadata 在 RAG 链路中的流动：

```text
文件
-> Document{Title, SourcePath, Content}
-> Chunk{Title, SourceFile, ChunkIndex, Content, RuneStartOffset, RuneEndOffset, CreatedAt, UpdatedAt}
-> VectorStore.Add(vector, chunk)
-> VectorStore.Search(queryVector, topK)
-> SearchResult{Chunk, Score}
-> BuildPrompt
-> LLM answer with citations
```

## 下一步准备

下一天要进入 Milvus collection schema 设计。

需要重点确认：

```text
Milvus 主键如何设计
哪些字段是 scalar field
embedding 维度从哪里读取或配置
Content 和 SourceFile 的 VARCHAR 长度如何设置
metric type 是否使用 COSINE
不同 embedding 模型是否需要不同 collection
```

今天补好的 metadata 会直接决定明天 Milvus schema 的字段设计。
