# 第 3 周第 5 天：拼接 Prompt 并生成 RAG 回答

日期：2026-08-24

## 今日目标

今天的目标是把前四天完成的能力串成 RAG 回答闭环：检索出相关 chunk 后，把这些资料拼进 prompt，让模型只能基于资料回答问题，并在资料不足时明确说不知道。

这一阶段重点不是追求检索效果最优，而是先确认完整链路能跑通：

```text
load -> chunk -> embed -> vector store -> query embed -> topK search -> prompt -> LLM answer
```

## 今日完成

- 完成 `BuildPrompt`，将 topK 检索结果拼接成模型可用的上下文。
- 在 prompt 中加入约束：
  - 只能基于给定资料回答。
  - 资料不足时说不知道。
  - 回答必须给出相关引用。
  - 禁止编造回答与来源。
- 调整向量库记录结构，让 `Embedding` 保存 `*Chunk`，避免检索后丢失来源信息。
- `Search` 结果现在能拿到：
  - `SourceFile`
  - `ChunkIndex`
  - `Content`
  - `Start`
  - `End`
  - `Score`
- 将 `vector` 调整为导出的 `Vector`，方便 `cmd/rag-cli` 直接使用 embedding 返回的 `[]float64`。
- 修改 `cmd/rag-cli`，支持完整 RAG 回答流程。
- 更新 `makefile`，新增 `make rag` 快捷命令。
- 实际验证了资料不足时拒答、资料充足时回答两个场景。

## 当前完整流程

当前 RAG CLI 的完整流程是：

```text
读取本地文档
  -> 切分 chunk
  -> 为 chunk 生成 embedding
  -> 写入内存 VectorStore
  -> 为用户问题生成 query embedding
  -> 使用 query embedding 检索 topK chunk
  -> 用 topK chunk 构造 RAG prompt
  -> 调用 LLM 生成最终回答
```

可以把今天完成的部分理解为：

```text
检索结果
  -> 带引用资料上下文
  -> 受约束的模型回答
```

## Prompt 设计

当前 `BuildPrompt` 的核心约束是：

```text
只能基于下面给出的资料回答，并给出相关引用。
当资料不足时要说不知道。
明确禁止编造回答与来源。
```

每个检索结果会被格式化为：

```text
[来源: file.md#chunk-3, 得分: 0.912345]
chunk content
```

这样做的好处是：

- 模型能看到资料来源。
- 回答时可以引用具体文件和 chunk。
- 如果资料不够，模型有明确指令拒答。
- 后续人工排查时，可以根据 source 和 chunk index 反查原文。

## 为什么需要保留 Chunk 元数据

之前向量库只保存 chunk 文本，检索之后会丢失：

```text
来自哪篇文档
是第几个 chunk
原文偏移是多少
```

这会影响 RAG 的可解释性和引用能力。

现在 `Embedding` 改为保存 `*Chunk`：

```go
type Embedding struct {
    Chunk  *Chunk
    Vector Vector
}
```

这样 `Search` 返回结果中可以继续访问：

```go
result.Embed.Chunk.SourceFile
result.Embed.Chunk.ChunkIndex
result.Embed.Chunk.Content
result.Score
```

这一步很关键，因为 RAG 不是只要“答出来”，还要能说明“依据来自哪里”。

## rag-cli 支持的能力

当前 `rag-cli` 支持两种模式。

### 只构建索引

不传问题时，只执行：

```text
load -> chunk -> embed -> vector store
```

用于验证文档读取、切分和 embedding 是否正常。

### 基于文档回答问题

传入 `-question` 或命令行尾部问题时，执行完整流程：

```text
load -> chunk -> embed -> vector store -> query embed -> search -> prompt -> answer
```

示例：

```powershell
go run ./cmd/rag-cli `
  -docs ".\testdata\documents" `
  -limitDocs 1 `
  -limitChunks 5 `
  -topK 3 `
  -showPreview=true `
  -question "请总结我第四周周五做了什么？"
```

也可以把问题放在命令尾部：

```powershell
go run ./cmd/rag-cli -docs ".\testdata\documents" -topK 3 "请总结我第四周周五做了什么？"
```

## make 快捷命令

当前新增：

```powershell
make rag DOCS_PATH=./testdata/documents QUESTION="请总结我第四周周五做了什么？"
```

可调参数：

```powershell
make rag `
  DOCS_PATH=./testdata/documents `
  QUESTION="请总结我第四周周五做了什么？" `
  TOPK=3 `
  LIMIT_DOCS=0 `
  LIMIT_CHUNKS=0 `
  SHOW_PREVIEW=true `
  RAG_MODEL=gpt-5.5
```

这些参数会影响检索范围和回答质量：

- `LIMIT_DOCS`：限制参与索引的文档数量。
- `LIMIT_CHUNKS`：限制每篇文档参与 embedding 的 chunk 数。
- `TOPK`：限制放进 prompt 的检索结果数量。
- `SHOW_PREVIEW`：打印命中 chunk 的预览，方便排查检索是否正确。

## 今日验证

今天验证了一个真实问题：

```text
请总结我第四周周五做了什么？
```

### 场景一：资料不足

第一次运行时存在文档或 chunk 限制，周五相关文档没有进入 embedding 和向量库。

模型回答：

```text
我不知道第四周周五做了什么。
目前给出的资料只包含第四周的周三笔记和 0525 周一笔记片段，没有提供“第四周周五”的工作记录，因此无法基于现有资料总结周五内容。
```

这个结果是符合预期的。

说明 prompt 中“资料不足时说不知道”的约束生效了，模型没有凭空编造周五内容。

### 场景二：资料充足

第二次放宽限制后，周五文档进入索引，检索结果包含周五相关 chunk。

模型回答能够总结：

- 上午继续处理或学习编译前端相关内容，关注 `aliasPath`、`targetPath`，以及“实例化组件”和“实例化元素”的关系。
- 下午继续理解平坦化流程，并使用 `FlatLearning.mo`、`frontend-flat-learning-todo`、`main.go` 等辅助文件。
- 晚上资料中没有记录具体事项，因此无法判断晚上做了什么。

这个结果也符合预期。

说明当检索资料足够时，模型能够基于资料回答；当资料只覆盖部分时间段时，模型仍然能对缺失部分保持克制。

## 今日核心理解

今天最重要的理解是：

```text
RAG 回答质量首先取决于检索结果，而不是模型本身。
```

如果正确 chunk 没有进入 prompt，模型就不应该回答。

如果正确 chunk 进入了 prompt，模型才有依据回答。

因此调试 RAG 时要先看：

```text
命中了哪些 chunk
这些 chunk 是否真的包含答案
topK 是否太小
limitDocs / limitChunks 是否挡住了关键文档
```

再看模型回答是否遵守 prompt 约束。

## 遇到的问题

### 1. 文档限制会导致误以为模型不会回答

第一次运行时，因为文档或 chunk 限制，周五内容没有进入向量库。

这不是模型问题，也不是 prompt 问题，而是检索资料覆盖范围不足。

因此排查 RAG 问题时，必须先看 CLI 打印的检索结果：

```text
source
chunk index
score
preview
```

### 2. 只保存 chunk 文本不够

如果检索结果只返回文本，后续无法给出可靠引用。

因此向量库记录必须保留 chunk 元数据，至少包含：

```text
SourceFile
ChunkIndex
Content
```

后续如果要做更精确引用，可以继续使用：

```text
Start
End
```

### 3. prompt 不是替代检索的魔法

prompt 可以约束模型不要胡编，但不能弥补检索不到资料的问题。

如果 topK 命中的 chunk 不相关，模型要么拒答，要么回答质量很差。

RAG 的关键顺序是：

```text
先检索对
再让模型答
```

## 今日记录

周五完成：

- 完成 RAG prompt 构造。
- 完成带来源格式的上下文拼接。
- 完成 `rag-cli` 的完整 RAG 回答流程。
- 完成 `make rag` 快捷入口。
- 验证资料不足时模型能拒答。
- 验证资料充足时模型能基于文档回答。

prompt 约束：

- 只能基于给定资料回答。
- 资料不足时说不知道。
- 回答必须给出引用。
- 不要编造来源。

能回答的问题：

```text
请总结我第四周周五做了什么？
```

在周五文档进入索引后，模型能基于资料总结上午和下午的工作内容。

不能回答的问题：

```text
请总结我第四周周五晚上做了什么？
```

当前资料中没有晚上记录时，模型应明确说资料不足。

引用格式：

```text
[来源: file.md#chunk-3, 得分: 0.912345]
```

## 下一步准备

下一步可以进入周六 RAG Demo v1。

建议准备固定 Demo 问题：

```text
能回答：第四周周五做了什么？
不能回答：第四周周五晚上做了什么？
容易混淆：第四周周四和周五分别做了什么？
```

同时记录每个问题的：

```text
命中的 chunk
score
回答是否正确
引用是否可信
是否应该拒答
```

第六天重点不是再堆新功能，而是把当前闭环做成稳定可演示的 Demo。
