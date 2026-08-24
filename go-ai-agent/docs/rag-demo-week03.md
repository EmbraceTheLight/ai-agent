# 第 3 周 RAG Demo 文档

本文档记录第 3 周 RAG 最小闭环的演示方式。

当前阶段重点验收：

```text
文档读取
chunk 切分
embedding 生成
内存向量检索
基于检索结果生成回答
```

## 演示前准备

进入项目目录：

```powershell
cd D:\code\golang\projects\personal\ai-agent\go-ai-agent
```

确认本地 `.env` 已配置：

```env
OPENAI_API_KEY=<your-key>
OPENAI_BASE_URL=<your-base-url>
OPENAI_MODEL=<your-model>

EMBEDDING_BASE_URL=http://localhost:11434
EMBEDDING_MODEL=qwen3-embedding:0.6b
```

确认 Ollama 已启动，并已拉取 embedding 模型：

```powershell
docker compose up -d
docker exec -it ollama ollama pull qwen3-embedding:0.6b
```

## 演示命令

本周 RAG CLI 支持通过 flag 配置输入文档、chunk 参数和问答参数。

示例：

```powershell
go run ./cmd/rag-cli `
  -docs "testdata/documents/work_notes_May/五月/第四周" `
  -chunkSize 500 `
  -overlap 100 `
  -limitDocs 2 `
  -limitChunks 3 `
  -showPreview=true
```

如果要直接问答，可以继续补上：

```powershell
go run ./cmd/rag-cli `
  -docs "testdata/documents/work_notes_May/五月/第四周" `
  -chunkSize 500 `
  -overlap 100 `
  -question "第四周周四做了什么" `
  -topK 3
```

也可以通过命令末尾直接传问题：

```powershell
go run ./cmd/rag-cli `
  -docs "testdata/documents/work_notes_May/五月/第四周" `
  -question "第四周有哪些待办事项"
```

## Demo 1：只验证 load / chunk / embed

目的：确认本地文档可以被稳定读取、切分并生成 embedding。

```powershell
go run ./cmd/rag-cli -docs "testdata/documents/work_notes_May/五月/第四周" -limitDocs 2 -limitChunks 3 -showPreview=true
```

预期：

```text
加载文档数
每篇文档的 chunk 数
本次 embedding chunk 数
embedding 数
embedding dimension
```

这个 Demo 不要求模型回答问题，只验证索引流程。

## Demo 2：问一个能从资料直接回答的问题

建议优先选择和单篇笔记粒度一致的问题，例如：

```powershell
go run ./cmd/rag-cli -docs "testdata/documents/work_notes_May/五月/第四周" -question "第四周周四做了什么"
```

预期：

```text
检索结果能命中相关 chunk
回答中引用来源
回答内容与文档一致
```

## Demo 3：问一个更容易检索但仍需来源支撑的问题

```powershell
go run ./cmd/rag-cli -docs "testdata/documents/work_notes_May/五月/第四周" -question "第四周有哪些待办事项"
```

预期：

```text
模型先检索待办事项相关 chunk
回答包含来源
不会编造文档中没有的事项
```

## Demo 4：问一个资料里没有答案的问题

```powershell
go run ./cmd/rag-cli -docs "testdata/documents/work_notes_May/五月/第四周" -question "第四周有没有做 PostgreSQL 持久化"
```

预期：

```text
如果检索不到相关内容, 模型应明确说资料不足
不要硬编答案
```

## Demo 5：周级问题的注意事项

如果只提供每天的笔记，那么下面这种问题可能不稳定：

```text
第四周我做了什么
```

原因是：

```text
日记粒度太细
问题范围太大
模型需要先汇总多个 chunk, 但当前 Demo 更偏检索片段而不是自动总结
```

因此，本周 Demo 更推荐：

```text
第三周周二做了什么
第四周周四做了什么
第四周有哪些待办事项
```

如果后面补充了周总结文档，那么再把“第三周我做了什么”加入 Demo 会更稳。

## Demo 结果检查

每次演示时建议观察这些信息：

```text
是否成功加载文档
是否成功切分 chunk
是否成功生成 embedding
embedding 维度是否一致
检索到的 chunk 是否和问题相关
回答是否引用了来源
无法回答时是否明确拒答
```

## 本周演示重点

- 文档输入不写死在代码里。
- `rag-cli` 可以通过 flag 配置。
- RAG demo 先验证检索，再验证回答。
- 问题粒度尽量和文档粒度匹配。
- 无法从资料回答时，模型应明确说不知道。
