# 第 3 周细则：RAG 文档问答最小闭环

适用时间：工作日每天 30-60 分钟，周末 2-4 小时。

本周目标：让模型能够基于本地私有文档回答问题，并在回答中给出引用来源。

本周最终交付：

- 能导入本地 markdown / txt 文档。
- 能把文档切分为 chunk。
- 能为 chunk 生成 embedding。
- 能用内存向量库完成 topK 相似度检索。
- 能把检索片段放入 prompt，让模型基于资料回答。
- 回答中包含引用来源。
- 当资料中没有答案时，模型应明确说不知道。
- 有固定 Demo、基础测试和周复盘。

## 本周核心理解

RAG 的核心不是“把文档塞给模型”，而是：

```text
先把文档切成可检索片段
再把用户问题转成向量
用向量相似度找出相关片段
把相关片段作为上下文交给模型
要求模型只基于上下文回答并给出引用
```

本周要建立三个意识：

- 检索质量决定回答上限。
- 引用来源决定回答可信度。
- 找不到答案时应拒答，而不是让模型凭空编造。

第 3 周优先跑通闭环，不急着上 PostgreSQL / pgvector。持久化向量存储放到第 4 周。

## 推荐目录

可以在现有项目结构上逐步增加：

```text
internal/
  rag/
    document.go
    loader.go
    chunker.go
    embedding.go
    vector_store.go
    retriever.go
    prompt.go
cmd/
  rag-cli/
testdata/
  documents/
docs/
  demo.md
学习笔记/
  第三周/
```

不必第一天就一次性建完。每天只补当天需要的最小文件。

## 周一：理解 RAG、Embedding、Chunk 和 TopK

时间预算：30-60 分钟。

### 今日目标

理解 RAG 文档问答的完整流程，以及 embedding、chunk、vector search、topK 分别解决什么问题。

### 今日步骤

1. 画出 RAG 流程：

```text
文档
-> chunk
-> embedding
-> vector store
-> 用户问题 embedding
-> topK 检索
-> prompt 拼接
-> LLM 回答
-> 引用来源
```

2. 写一份概念笔记，回答：

- 为什么不能把所有文档直接塞进 prompt？
- embedding 表示的是什么？
- chunk 太大或太小分别有什么问题？
- topK 太大或太小分别有什么问题？
- 为什么回答必须带引用？
- 资料中没有答案时为什么要拒答？

3. 初步设计核心数据结构草图：

```text
Document
Chunk
Embedding
SearchResult
```

### 今日验收

能用自己的话解释：

```text
RAG = 检索相关资料 + 让模型基于资料回答
```

### 今日记录

```text
周一完成：
我对 RAG 的理解：
chunk / embedding / topK 分别解决什么问题：
今天仍然模糊的问题：
```

## 周二：读取文档并切分 Chunk

时间预算：30-60 分钟。

### 今日目标

准备 5-10 篇 markdown / txt 文档，并用 Go 读取和切分为 chunk。

### 今日步骤

1. 在 `testdata/documents/` 准备少量本地文档。

文档可以来自项目 README、学习笔记、技术总结或手写 mock 文档。内容要适合问答，不要使用敏感信息。

2. 实现文档读取。

第一版只支持：

```text
.md
.txt
```

3. 实现简单 chunk 切分。

第一版可以使用字符数或 rune 数切分，例如：

```text
chunk size = 500-800 字
overlap = 50-100 字
```

4. 为 chunk 附加 metadata：

```text
source file
chunk index
start / end offset
title 可选
```

### 今日验收

运行一个小程序或测试，能打印：

```text
读取了多少文档
生成了多少 chunk
每个 chunk 的 source 和 index
```

### 今日记录

```text
周二完成：
chunk 切分策略：
metadata 包含哪些字段：
切分中遇到的问题：
```

## 周三：调用 Embedding API

时间预算：30-60 分钟。

### 今日目标

为每个 chunk 生成 embedding，并理解 embedding API 与普通聊天 API 的区别。

### 今日步骤

1. 在 `internal/llm` 或 `internal/rag` 中增加 embedding 调用封装。

2. 明确配置项：

```text
embedding api key
embedding base url
embedding model
```

如果暂时复用 OpenAI-compatible 配置，需要在笔记中记录这个折中。

3. 为每个 chunk 生成向量。

4. 记录向量维度，不要在日志中打印完整向量。

### 今日验收

能对一段文本生成 embedding，并打印：

```text
chunk id
embedding dimension
```

### 今日记录

```text
周三完成：
embedding API 输入输出：
向量维度：
配置上有哪些临时折中：
```

## 周四：内存向量搜索

时间预算：30-60 分钟。

### 今日目标

先用内存实现向量搜索，避免过早引入数据库和 pgvector。

### 今日步骤

1. 定义最小 `VectorStore` 能力：

```text
Add(chunk, vector)
Search(queryVector, topK)
```

2. 实现内存存储。

3. 实现相似度计算。

第一版可以选择 cosine similarity，并在笔记中写清楚公式含义。

4. 用固定文本验证检索结果。

### 今日验收

输入一个问题，能够检索出最相关的 topK chunk，并打印：

```text
score
source
chunk index
content preview
```

### 今日记录

```text
周四完成：
使用的相似度算法：
topK 检索结果是否符合预期：
检索不准的例子：
```

## 周五：拼接 Prompt 并生成 RAG 回答

时间预算：30-60 分钟。

### 今日目标

把检索结果塞进 prompt，让模型基于资料回答问题。

### 今日步骤

1. 设计 RAG prompt。

必须表达这些约束：

```text
只能基于给定资料回答
资料不足时说不知道
回答必须带引用
不要编造来源
```

2. 将 topK chunk 格式化为上下文。

建议格式：

```text
[来源: file.md#chunk-3]
chunk content
```

3. 调用 LLM 生成最终回答。

4. 测试一个能回答的问题和一个不能回答的问题。

### 今日验收

模型回答应满足：

```text
有答案时：基于资料回答，并带引用
无答案时：明确说资料中没有相关信息
```

### 今日记录

```text
周五完成：
prompt 约束：
能回答的问题：
不能回答的问题：
引用格式：
```

## 周六：做 RAG Demo v1

时间预算：2-4 小时。

### 今日目标

做一个固定可演示的 RAG CLI：

```text
导入文档
输入问题
检索相关 chunk
生成带引用的回答
```

### 今日步骤

1. 整理一个 `rag-cli` 或在现有 CLI 中增加最小 RAG 入口。

2. 准备固定 Demo 文档。

3. 准备固定问题：

```text
能从资料回答的问题
资料中没有答案的问题
相似但容易误检索的问题
```

4. 更新 `docs/demo.md`。

### 今日验收

能稳定演示：

```text
导入文档
检索 topK chunk
回答带引用
无答案时拒答
```

### 今日记录

```text
周六完成：
Demo 是否稳定：
哪些问题检索效果好：
哪些问题检索效果差：
```

## 周日：调参、对比和复盘

时间预算：2-4 小时。

### 今日目标

优化 chunk size、overlap、topK 和引用格式，并整理本周复盘。

### 今日步骤

1. 对比不同参数：

```text
chunk size
overlap
topK
```

2. 记录至少 5 个问题的效果：

```text
问题
命中的 chunk
回答是否正确
引用是否可信
是否应该拒答
```

3. 补基础测试：

```text
chunk 切分测试
cosine similarity 测试
topK 排序测试
```

4. 写周复盘。

### 今日验收

你应该能回答：

- RAG 的完整流程是什么？
- chunk size 为什么影响效果？
- topK 怎么调？
- 如何避免模型胡编？
- 引用来源为什么重要？
- 什么时候应该拒答？

### 今日记录

```text
周日完成：
本周完成：
本周最重要的理解：
效果最好的参数：
还不稳定的地方：
第 4 周 pgvector 前需要准备什么：
```

## 本周完成标准

最低完成标准：

- 能读取 markdown / txt 文档。
- 能切分 chunk。
- 能生成 embedding。
- 能用内存向量搜索 topK。
- 能把检索结果放入 prompt。
- 能生成带引用的回答。
- 无答案时能拒答。
- 有学习笔记。

优秀完成标准：

- 有固定 RAG Demo。
- 有 chunk / vector search 基础测试。
- 有 5-10 个手动 eval 问题。
- 能对比 chunk size、overlap、topK 的效果。
- 能解释 RAG、Tool Calling、上下文和安全边界之间的关系。

## 本周不要做什么

- 不要一开始就接 PostgreSQL / pgvector。
- 不要一开始就做复杂 HTTP API。
- 不要把所有文档直接塞进 prompt。
- 不要让模型在没有资料依据时硬答。
- 不要在日志中打印完整 API Key 或大段 embedding 向量。
- 不要用真实敏感文档做测试数据。

第 3 周的价值是建立 RAG 最小闭环：资料可导入、片段可检索、回答可引用、无资料能拒答。这个基础打稳后，第 4 周再做持久化、metadata filter、eval 和 HTTP API。
