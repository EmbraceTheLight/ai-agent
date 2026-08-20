# 第 3 周第 2 天：读取文档并切分 Chunk

日期：2026-08-19

## 今日目标

今天的目标是完成 RAG 最小闭环中的第一段基础能力：从本地读取 markdown / txt 文档，并把文档内容切分成适合检索和 embedding 的 chunk。

这一阶段不追求复杂的语义切分，也不接入向量库和数据库。重点是先把文档读取、来源记录、chunk 边界、overlap 和基础测试跑通。

## 今日完成

- 在 `testdata/documents/` 准备了本地文档数据。
- 实现了 `LoadRAGResources`，支持读取单个文件或递归读取目录。
- 第一版 loader 只支持 `.md` 和 `.txt` 文件。
- loader 会跳过不支持的文件类型。
- loader 会保留文档的绝对路径和文本内容。
- 定义了 `Document` 数据结构，包含 `SourcePath` 和 `Content`。
- 定义了 `Chunk` 数据结构，包含来源文件、chunk 索引、内容、起止偏移量。
- 实现了 `ChunkDocument`，按 rune 数量切分文档。
- 支持 `overlap`，让相邻 chunk 之间保留一定重叠内容。
- 补充了 loader 和 chunker 的基础测试。
- 运行 `go test ./internal/rag` 通过。

## 当前数据结构

### Document

`Document` 表示从本地读取到的一篇文档。

```go
type Document struct {
    SourcePath string
    Content    string
}
```

字段含义：

- `SourcePath`：源文件路径，用于后续生成引用来源。
- `Content`：文档完整文本内容，用于后续 chunk 切分。

一开始曾考虑在 `Document` 中保存 `*os.File`，目的是担心文档很多时一次性读取内容会造成内存压力。后来调整为直接保存内容，因为当前第三周目标是跑通最小闭环，且后续 chunker 更容易基于字符串处理。

如果后续文档规模变大，可以再升级为流式处理，例如 `WalkDocuments(root, fn)`，每次只读取一个文件，切完 chunk 后立即进入 embedding 或 vector store。

### Chunk

`Chunk` 表示文档切分后的一个片段。

```go
type Chunk struct {
    SourceFile string
    ChunkIndex int
    Content    string
    Start      int
    End        int
}
```

字段含义：

- `SourceFile`：chunk 来自哪个源文件。
- `ChunkIndex`：该 chunk 在源文件中的序号。
- `Content`：chunk 文本内容。
- `Start`：chunk 起始 rune 偏移。
- `End`：chunk 结束 rune 偏移。

当前 `Start` / `End` 记录的是 rune 偏移，不是 byte 偏移。这样做的好处是对中文内容更自然，不会把多字节字符切坏。现阶段主要用于检索、展示和引用来源，rune 偏移已经够用。

如果后续需要原文高亮、按字节回切原始字符串或精确定位，可以再增加 `StartByte` / `EndByte`。

## Chunk 切分策略

当前采用固定长度 + overlap 的简单策略。

核心规则：

```text
step = size - overlap
chunk 1: start = 0
chunk 2: start = step
chunk 3: start = step * 2
```

例如：

```text
size = 10
overlap = 3
step = 7
```

切分结果类似：

```text
0-10
7-17
14-24
21-26
```

这样相邻 chunk 会有 3 个 rune 的重叠。overlap 的作用是减少重要信息刚好落在边界处时被切断的风险。

当前没有使用环形缓存。之前考虑过用 ring buffer 获取上一段 chunk 的后 overlap 个字符，但在当前实现中，文档已经读入内存，直接把 `Content` 转为 `[]rune` 后用下标切分更简单、更清楚，也更容易测试。

## 为什么按 rune 切分

Go 的字符串底层是 byte。中文字符通常占多个 byte，如果直接按 byte 下标切分，可能会把一个中文字符切坏，导致无效 UTF-8 或显示异常。

因此当前做法是：

```go
runes := []rune(doc.Content)
```

然后基于 rune 下标计算 `start` 和 `end`。这样 `size` 和 `overlap` 的含义都是“字符数量”，而不是“字节数量”。对于中文笔记、markdown 文档和普通文本问答来说，这个语义更符合直觉。

## metadata 包含哪些字段

当前 chunk metadata 包含：

- source file：通过 `SourceFile` 保存。
- chunk index：通过 `ChunkIndex` 保存。
- start offset：通过 `Start` 保存。
- end offset：通过 `End` 保存。
- title：暂时未实现。

title 暂时不做，是因为当前 chunker 只是固定长度切分，并没有解析 markdown 标题结构。后续如果要提升引用质量，可以在 loader 或 chunker 中解析 markdown heading，把最近的标题附加到 chunk metadata 上。

## 测试覆盖

loader 测试覆盖了：

- 递归加载目录中的 `.md` / `.txt` 文件。
- 跳过不支持的文件类型。
- 单文件加载。
- 没有支持文件时返回错误。
- `SourcePath` 是绝对路径。
- `Content` 与源文件内容一致。

chunker 测试覆盖了：

- 普通英文内容按 `size` 切分。
- 相邻 chunk 之间保留 `overlap`。
- 中文内容按 rune 切分，不按 byte 切坏字符。
- 文档长度不足一个 chunk 时返回单个 chunk。
- `overlap=0` 时不产生重叠。
- `nil doc`、负数 size、负数 overlap、`overlap >= size` 会返回错误。

当前验证命令：

```powershell
go test ./internal/rag
```

结果通过。

## 遇到的问题

### 1. Document 是否应该保存 `*os.File`

最初的想法是保存 `*os.File`，避免文件很多时一次性读入内存。

后来发现这个方案有两个问题：

- 如果一次性打开很多文件，可能造成文件句柄泄漏或超过系统限制。
- 后续 chunker 仍然需要读取文件内容，只是把读取时机往后推。

因此当前阶段选择直接保存 `Content`。这个方案更适合第三周最小闭环，也更容易写测试。

真正要解决大规模文档内存问题时，更好的方案不是保存 `*os.File`，而是流式处理：读取一个文件，切 chunk，写入后续处理，再释放原始文本。

### 2. 是否需要保存 byte offset

因为输入文本可能包含中文，所以需要区分 rune offset 和 byte offset。

当前 `Start` / `End` 是 rune offset。它适合表达第几个字符到第几个字符，也适合当前 chunk 切分逻辑。

byte offset 更适合：

- 从原始字符串精确回切。
- 做高亮定位。
- 生成更精确的引用。

当前阶段暂时不保存 byte offset，避免过早增加复杂度。等引用展示或高亮需求出现后，再补 `StartByte` / `EndByte`。

### 3. 最后一小段 chunk 是否应该保留

在 RAG 中，尾部短 chunk 也可能包含重要信息。如果直接丢弃，可能导致资料缺失。

当前策略是：只要还有未覆盖的内容，就生成最后一个 chunk。即使最后一块短于 `size`，也保留它。

## 今日核心理解

RAG 的文档处理阶段不是简单读取文件，而是要把原始资料变成“可检索、可引用、可追踪来源”的片段。

可以把今天完成的流程理解为：

```text
本地 .md / .txt 文件
  -> Document{SourcePath, Content}
  -> []rune(Content)
  -> 固定 size + overlap 切分
  -> Chunk{SourceFile, ChunkIndex, Content, Start, End}
```

其中最重要的点是：

```text
SourcePath 让回答能引用来源。
ChunkIndex 让来源能定位到片段。
overlap 减少边界信息丢失。
rune 切分避免中文字符被切坏。
```

## 今日记录

周二完成：

- 完成文档读取。
- 完成 chunk 切分。
- 完成基础 metadata。
- 完成 loader 和 chunker 测试。

chunk 切分策略：

- 固定 rune 数量切分。
- 使用 `step = size - overlap` 控制相邻 chunk 的重叠。
- 最后一块即使不足 `size` 也保留。

metadata 包含哪些字段：

- source file。
- chunk index。
- start rune offset。
- end rune offset。

切分中遇到的问题：

- 需要区分 rune offset 和 byte offset。
- 一开始考虑环形缓存，但当前阶段用 `[]rune` 下标切分更直接。
- 最后一小段 chunk 不应该轻易丢弃，否则可能丢失资料。
- 保存 `*os.File` 不能真正解决大规模文档处理问题，后续如有需要应改成流式处理。

## 下一步准备

下一天进入 embedding API。

需要准备：

- 明确 embedding API 的配置项。
- 选择 embedding model。
- 封装 chunk 文本到 embedding vector 的调用。
- 记录向量维度。
- 不在日志中打印完整向量和 API Key。
- 优先保证一个 chunk 可以稳定生成 embedding，再批量处理所有 chunk。
