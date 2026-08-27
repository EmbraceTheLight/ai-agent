# 第 4 周第 1 天：整理 VectorStore 接口与 Chunk 元数据

日期：2026-08-27

## 今日目标

今天进入第 4 周 RAG 生产化阶段。

第 3 周已经完成了基于内存向量搜索的 RAG Demo，本周要逐步把它推进到更接近真实后端服务的形态。今天的重点不是直接接入 Milvus，而是先整理 RAG 核心抽象和文档元数据边界，为后续持久化向量库、metadata filter 和 HTTP API 做准备。

核心问题包括：

```text
RAG 流程应该依赖什么抽象
内存向量库的内部结构是否泄漏到了上层
后续 Milvus 返回结果能否自然适配当前接口
每个 chunk 是否具备稳定的来源、标题、位置和时间 metadata
不同来源导出的文档是否应该通过专属 loader 解析
```

## 今日完成

- 整理了 `VectorStore` 接口，继续保留最小能力边界。
- 将检索结果抽象为 `SearchResult`，返回 `Chunk + Score`。
- 将 topK 小根堆整理为 `SearchResultMinHeap`，并拆分到独立文件。
- 将 `VectorStore`、`Embedder`、`DocumentLoader` 等接口移动到 `interfaces.go`。
- 将 `type.go` 收敛为核心数据结构定义。
- 为 `Chunk` 增加 `Title`、`CreatedAt`、`UpdatedAt` 等 metadata 字段。
- 将 `Start`、`End` 重命名为语义更清楚的 `RuneStartOffset`、`RuneEndOffset`。
- 在 `ChunkDocument` 中将 `Document.Title` 传递到 `Chunk.Title`。
- 新增 `DocumentLoader` 接口。
- 新增 Trilium 专属文档加载器 `NewTriliumDocumentLoader`。
- 实现 Trilium Markdown 标题解析规则：优先读取第一行 `# <title>`，缺失时回退到文件名。
- 将原有文档加载工具拆分为 `loader_util.go`，保留 `LoadRAGResources` 作为旧入口。
- 新增 `utils.Exist` 和 `utils.IsDir` 文件工具函数。
- 补充 Trilium loader 测试，使用真实 `testdata` 验证标题解析和无标题回退。
- 修复 `internal/tools` 中时间工具测试，使其适配当前 JSON 返回格式。
- 验证全量测试通过。

## VectorStore 接口整理

整理后的 `VectorStore` 仍然只暴露两个核心能力：

```go
type VectorStore interface {
	Add(vector Vector, chunk *Chunk) error
	Search(queryVector Vector, topK int) ([]*SearchResult, error)
}
```

这个接口表达的是 RAG 流程真正需要的能力：

```text
写入一个 chunk 及其 embedding
根据 query embedding 检索 topK 个相关 chunk
```

上层 RAG 流程不需要知道底层是内存切片、Milvus collection，还是其他向量数据库。这样后续接入 Milvus 时，只要实现同一个接口即可替换底层存储。

## SearchResult 抽象

原来的检索结果更偏向内存实现，因为内存向量库内部保存的是：

```text
Embedding
  Chunk
  Vector
Score
```

但对上层 RAG 来说，真正需要的是：

```text
Chunk
Score
```

其中：

- `Chunk` 用于构造 prompt 和引用来源。
- `Score` 用于表示 query 和 chunk 的相似度。
- `Vector` 只是底层检索过程的数据，不应该暴露给 prompt 构造和业务调用方。

因此检索结果调整为：

```go
type SearchResult struct {
	Chunk *Chunk
	Score float64
}
```

这个调整让 `SearchResult` 更像业务层的检索结果，而不是某个具体存储实现的内部记录。

## 内存实现的保留

虽然 `SearchResult` 不再暴露 `Embedding`，但内存向量库内部仍然可以继续用 `Embedding` 保存数据：

```text
defaultVectorStore
  []*Embedding

Embedding
  Chunk
  Vector
```

这是合理的。接口层只负责暴露稳定抽象，具体实现内部如何保存数据可以不同。

内存实现中：

```text
Add 保存 Embedding
Search 计算 cosine similarity
Search 返回 SearchResult
```

后续 Milvus 实现中：

```text
Add 插入 Milvus collection
Search 调用 Milvus search
Search 将返回字段组装为 SearchResult
```

两者可以共享同一个 `VectorStore` 接口。

## Chunk 元数据整理

为了让 chunk 后续能够被稳定追踪、过滤和引用，今天扩展了 `Chunk` 元数据。

当前结构重点字段包括：

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

其中：

- `SourceFile` 表示来源文件路径。
- `Title` 表示来源文档标题。
- `ChunkIndex` 表示该 chunk 在文档中的序号。
- `Content` 表示 chunk 文本内容。
- `CreatedAt` 和 `UpdatedAt` 为后续入库、增量更新和 metadata filter 预留。
- `RuneStartOffset` 和 `RuneEndOffset` 表示基于 rune 的文本偏移。

将 `Start`、`End` 重命名为 `RuneStartOffset`、`RuneEndOffset` 后，字段语义更清楚：它们不是字节偏移，而是 rune 偏移。对于中文等多字节字符，这一点尤其重要。

## DocumentLoader 抽象

今天新增了 `DocumentLoader` 接口：

```go
type DocumentLoader interface {
	Load(path string) ([]*Document, error)
}
```

这个接口只描述调用方真正需要的能力：从某个路径加载文档列表。

至于 loader 内部如何遍历目录、过滤文件、读取内容、解析标题和构造 `Document`，都属于具体 loader 的实现细节，不应该暴露给 RAG 主流程。

这种设计避免了过早把 `parseDocument`、`getDocTitle` 等步骤放入公共接口。不同来源的文档可能有不同标题规则和 metadata 规则，公共接口应该保持稳定。

## Trilium 文档加载器

今天新增了 Trilium 专属文档加载器：

```go
NewTriliumDocumentLoader(allowExt map[string]bool, limit int) DocumentLoader
```

当前规则：

```text
默认只加载 .md 文件
limit <= 0 表示不限制文档数量
Trilium 导出的 Markdown 第一行为标题
如果没有标题，则回退为去掉后缀的文件名
```

标题解析逻辑示例：

```text
# 第三周
```

会解析为：

```text
第三周
```

如果文件为空，例如：

```text
周末.md
```

则标题回退为：

```text
周末
```

这里没有将 parser 继续拆成独立对象，因为当前 loader 本身就是 Trilium 专属实现。对于当前阶段来说，`loader -> parser` 会有些过度设计。更简单的方式是让 Trilium loader 内部包含私有方法：

```text
Load
parseDocument
getDocTitle
```

如果后续同一种来源内部出现多种复杂解析策略，再考虑单独抽出 parser。

## 文件结构调整

今天将部分类型和实现按职责进行了拆分：

```text
interfaces.go              接口定义
type.go                    核心数据结构
vector_store.go            内存 VectorStore 实现
search_result_heap.go      topK 小根堆实现
chunker.go                 文档切分逻辑
loader_util.go             通用文件收集与旧加载入口
trilium_doc_loader.go      Trilium 专属文档加载器
embedding.go               embedding client 实现
prompt.go                  RAG prompt 构造
```

这种拆分仍然保持在同一个 `rag` package 内，没有过早拆成子包。当前阶段这样更轻量，也避免了 internal 包之间的引用复杂度。等 Milvus 实现真正接入后，可以再评估是否拆分为：

```text
internal/rag/memory
internal/rag/milvus
internal/rag/source
```

## Milvus Schema 预研

今天仍然保留了对 Milvus collection schema 的初步设计方向：

```text
ID          INT64 PRIMARY KEY AUTO_ID
SourceFile  VARCHAR(512)
Title       VARCHAR(...)
ChunkIndex  INT32
RuneStart   INT32
RuneEnd     INT32
Content     VARCHAR(4096)
CreatedAt   INT64
UpdatedAt   INT64
Embedding   FLOAT_VECTOR(dim = 当前 embedding 维度)
```

其中：

- `ID` 是 Milvus 自动主键。
- `Embedding` 是向量字段。
- `SourceFile`、`Title`、`ChunkIndex`、`RuneStart`、`RuneEnd`、`Content`、`CreatedAt`、`UpdatedAt` 是标量字段。
- 标量字段后续可以用于结果返回、引用展示和 metadata filter。

今天只是 schema 预研，没有正式接入 Milvus SDK。

## Embedding 维度问题

今天继续明确了一个边界：

```text
向量维度是 collection schema 的一部分。
```

如果后续更换 embedding 模型，并且新模型输出维度和当前模型不同，不能把新旧向量混在同一个 vector field 中。

更稳妥的做法是：

```text
新建 collection
重新对原始文档生成 embedding
保证 query embedding 和 chunk embedding 来自同一个模型版本
```

即使两个模型输出维度相同，也不代表它们处于同一个语义空间。后续可以在配置中显式维护：

```text
EMBEDDING_MODEL
EMBEDDING_DIM
RAG_COLLECTION
```

这样更方便做模型切换和索引重建。

## 测试结果

今天补充和验证的测试包括：

```text
chunker 测试
vector store 测试
Trilium loader 标题解析测试
Trilium loader 无标题回退测试
time tool JSON 返回格式测试
```

其中 Trilium loader 使用真实 testdata：

```text
testdata/documents/work_notes_May/五月/第三周.md
```

验证标题可以解析为：

```text
第三周
```

以及：

```text
testdata/documents/work_notes_May/五月/第三周/周末.md
```

该文件为空，验证标题回退为：

```text
周末
```

全量测试通过：

```powershell
go test ./...
```

结果：

```text
?    go-ai-agent/cmd/assistant-cli [no test files]
?    go-ai-agent/cmd/rag-cli       [no test files]
?    go-ai-agent/internal/config   [no test files]
?    go-ai-agent/internal/errno    [no test files]
?    go-ai-agent/internal/llm      [no test files]
ok   go-ai-agent/internal/rag
ok   go-ai-agent/internal/tools
?    go-ai-agent/internal/utils    [no test files]
```

## 今日核心理解

今天最重要的理解有两个。

第一，接口返回值应该表达业务需要，而不是暴露某个实现的内部存储结构。

对 RAG 检索来说，上层真正需要的是：

```text
命中的 chunk
相似度 score
```

至于底层为了计算相似度保存了什么、Milvus 如何组织 collection、向量字段是否返回给 Go，这些都应该留在具体 `VectorStore` 实现内部。

第二，文档来源解析应该交给专属 loader，而不是让 `Document` 自己承担规则判断。

`Document` 应该是稳定的数据结构，loader 才负责处理不同来源的规则。例如 Trilium 导出的 Markdown 第一行是标题，其他来源可能完全不同。把规则放在具体 loader 内部，可以避免后续出现越来越大的 `switch doc_type`。

## 当前不足

今天仍然有一些内容没有展开：

- `rag-cli` 还没有切换到新的 `DocumentLoader`。
- 旧的 `LoadRAGResources` 仍然存在，后续需要决定保留兼容还是迁移到 loader。
- `Search` 还不支持 metadata filter。
- `Title` 暂时还没有进入 prompt 引用展示。
- `SearchResult` 还没有独立的 `Citation` 字段，目前引用信息仍然从 `Chunk` 中读取。
- Milvus collection schema 只是草案，还没有真正落地。
- 还没有处理重复导入、删除文档、增量更新和重建索引。
- `Vector` 当前是 `[]float64`，后续接入 Milvus 时可能需要在实现层转换为 `[]float32`。

这些问题不影响第 4 周第 1 天目标，可以放到后续 metadata、Milvus 接入和 HTTP API 阶段处理。

## 下一步准备

下一步可以继续推进文档加载链路和 metadata 的真正应用：

```text
让 rag-cli 支持选择 DocumentLoader
逐步替换或废弃 LoadRAGResources
在 prompt 引用中加入 Title
明确 SourceFile、Title、ChunkIndex、RuneStartOffset、RuneEndOffset 的引用格式
为后续 Milvus 入库确定 collection schema
准备 metadata filter 的接口设计
```

进入 Milvus 之前，需要确保每个 chunk 都能被稳定追踪、过滤和引用。

