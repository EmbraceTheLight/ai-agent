# 第 4 周第 4 天：接入 Milvus VectorStore

日期：2026-09-20

## 今日目标

今天的目标是实现 `MilvusVectorStore`，把第 3 周基于内存的向量检索迁移到可持久化的 Milvus，同时保持原有 RAG 流程不变。

需要跑通的完整链路是：

```text
加载文档
-> 切分 chunk
-> 生成 embedding
-> 写入 Milvus
-> 使用 query vector 执行 topK 检索
-> 从检索结果恢复 chunk metadata
-> 构造 prompt
-> 生成带来源引用的回答
```

## 今日完成

- 使用 Helm 在 Kubernetes 的 `infrastructure` namespace 中部署 Milvus Standalone。
- 了解 Standalone 模式仍然需要 etcd 和对象存储等依赖组件。
- 在 `rag` 包中实现 `MilvusVS`，并让它满足 `VectorStore` 接口。
- 实现 Milvus client 创建和资源释放。
- 实现 collection 不存在时创建、已存在时复用的初始化逻辑。
- 为 `chunk_vector` 创建 `AutoIndex + COSINE` 索引并加载 collection。
- 实现 chunk、metadata 和 vector 的写入。
- 实现 query vector 的 topK 搜索。
- 将 Milvus 返回的标量字段重新组装为 `SearchResult{Chunk, Score}`。
- 让 `rag-cli` 支持 `-store memory|milvus`，可以在两种向量库实现之间切换。
- 使用真实工作文档完成端到端验证。
- 在 Milvus 中确认 `qwen3_embedding_chunk` collection 已创建并包含 chunk 数据。

## Kubernetes 与 Helm 部署

本次使用 Milvus Standalone，而不是分布式集群。主要 Helm 参数如下：

```powershell
helm install milvus-standalone ./milvus `
  --namespace infrastructure `
  --set image.all.tag=v3.0.0 `
  --set cluster.enabled=false `
  --set pulsarv3.enabled=false `
  --set standalone.messageQueue=woodpecker `
  --set woodpecker.enabled=true `
  --set streaming.enabled=true
```

`cluster.enabled=false` 表示 Milvus 核心服务以 Standalone 形态运行，但不代表整个部署只有一个 Pod。

Helm chart 仍然会部署以下依赖：

```text
etcd：保存 Milvus metadata
MinIO：保存 segment、索引等对象数据
Woodpecker：作为本次配置中的消息队列
Milvus Standalone：提供向量写入和检索能力
```

因此，看到 etcd 和 MinIO 随 Milvus Standalone 一起启动是合理的。

部署后需要检查：

```powershell
kubectl get pods -n infrastructure
kubectl get svc -n infrastructure
kubectl get pvc -n infrastructure
```

应用运行在 Kubernetes 集群外时，可以通过 NodePort 或 `kubectl port-forward` 访问 Milvus；应用运行在集群内时，则应该使用 Service DNS 地址。

## VectorStore 的可替换实现

当前 `VectorStore` 接口保持不变：

```go
type VectorStore interface {
	Add(ctx context.Context, vector Vector, chunk *Chunk) error
	Search(ctx context.Context, queryVector Vector, topK int) ([]*SearchResult, error)
}
```

创建向量库时，根据配置选择具体实现：

```text
memory -> defaultVectorStore
milvus -> MilvusVS
```

因此，文档加载、chunk、embedding、prompt 和 LLM 回答流程不需要知道底层究竟是内存还是 Milvus。

这体现了接口抽象的价值：

```text
上层 RAG 流程依赖 VectorStore 行为
底层实现负责各自的存储和检索细节
```

## Milvus Client 生命周期

`NewMilvusVS` 根据配置创建 Milvus client：

```text
Address
Username
Password
```

同时返回 `cleanup` 函数，在 CLI 结束时关闭 client。

这样可以让 `NewVectorStore` 对 memory 和 Milvus 保持统一的返回形式：

```go
store, cleanup, err := NewVectorStore(config)
defer cleanup()
```

## Collection 初始化

`InitCollections` 的初始化流程是：

```text
HasCollection
-> 不存在时创建 schema 和 vector index
-> 已存在时直接复用
-> LoadCollection
-> Await collection 加载完成
```

当前 collection：

```text
qwen3_embedding_chunk
```

当前主要字段：

```text
id                 Int64 PRIMARY KEY
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

与第三天最初设想不同，当前实现没有使用 Milvus auto id，而是在应用侧通过 `utils.GetID()` 生成 `Int64` 主键并写入。这说明学习笔记和 schema 设计最终应以实际运行代码为准。

向量字段索引：

```text
index: AutoIndex
metric: COSINE
```

初始化具有基本幂等性：collection 已存在时不会重复创建，但每次仍会确保 collection 被加载后再执行写入和搜索。

## 写入 Milvus

`MilvusVS.Add` 在写入前进行两项检查：

```text
vector 维度必须等于 EMBEDDING_DIM
chunk 不能为 nil
```

随后将一条 chunk 转换为一条 Milvus 记录：

```text
应用生成的 id
chunk metadata
chunk content
embedding vector
```

写入 Milvus 后，vector 和 metadata 位于同一条记录中。这样检索命中向量时，可以同时拿到来源文件、标题、chunk 序号和正文。

## 从 Milvus 检索

`MilvusVS.Search` 使用 query embedding 搜索：

```text
collection: qwen3_embedding_chunk
vector field: chunk_vector
metric: COSINE
limit: topK
```

搜索时显式请求以下 output fields：

```text
source_file_path
title
content
chunk_index
created_at
updated_at
rune_start_offset
rune_end_offset
```

Milvus 返回结果经过解析后，恢复为：

```go
SearchResult{
	Chunk: &Chunk{
		SourceFile: ...,
		Title: ...,
		ChunkIndex: ...,
		Content: ...,
		CreatedAt: ...,
		UpdatedAt: ...,
		RuneStartOffset: ...,
		RuneEndOffset: ...,
	},
	Score: ...,
}
```

因此 Milvus 的引入没有改变后续 `BuildPrompt` 的输入结构。

## CLI 接入

`rag-cli` 新增：

```text
-store memory
-store milvus
```

使用 `milvus` 时，CLI 会：

```text
连接 Milvus
初始化并加载 collection
写入文档 chunks
搜索 topK
打印 score、source 和 chunk index
生成最终回答
```

本次验证命令：

```powershell
go run ./cmd/rag-cli `
  -store milvus `
  -docs "D:\Go\WorkSpace\src\Go_Project\ai-agent\go-ai-agent\testdata\documents\work_notes_May\五月\第四周\周五.md" `
  -question "请总结第四周周五完成了哪些工作？" `
  -topK 3 `
  -showPreview=true `
  -timeout 5m
```

## 实际验证结果

最终回答正确总结了第四周周五的内容：

```text
学习 aliasPath 与 targetPath
梳理组件实例化与元素实例化的关系
结合 FlatLearning.mo、frontend-flat-learning-todo 和 main.go 理解平坦化流程
```

回答中的引用为：

```text
周五.md#chunk-0
```

同时在 Milvus 中确认：

```text
collection: qwen3_embedding_chunk
collection 中已存在写入的 chunk 数据
```

这次验证证明以下链路已经打通：

```text
真实文档
-> qwen3 embedding
-> Milvus 持久化写入
-> COSINE topK 检索
-> metadata 恢复
-> prompt 构造
-> 带来源引用的回答
```

## 测试结果

代码级验证命令：

```powershell
go test ./internal/rag ./cmd/rag-cli
```

结果：

```text
ok   go-ai-agent/internal/rag
?    go-ai-agent/cmd/rag-cli [no test files]
```

真实环境验证则通过 `-store milvus` 完成。单元测试证明原有内存向量库和 RAG 逻辑未被破坏，CLI 验证证明 Milvus 的实际连接、写入和搜索能够工作。

## 今日核心理解

今天最重要的理解是：

```text
把内存检索替换为 Milvus，不只是更换一个 Search 函数，
而是同时处理部署、存储、schema、索引、client 生命周期和 metadata 恢复。
```

Milvus 在当前 RAG 中承担的是：

```text
持久化保存 embedding
使用向量索引执行 topK 检索
保存可恢复引用的 scalar metadata
让检索结果能够继续进入统一的 RAG 流程
```

接口设计则保证了：

```text
memory 和 Milvus 可以共享同一套上层流程
存储实现变化不会扩散到 chunk、prompt 和 LLM 代码
```

## 当前不足

- 当前 CLI 每次运行都会重新导入文档，重复执行可能产生重复 chunk。
- 还没有 `document_id`、`content_hash` 或 upsert 机制用于去重和增量更新。
- 当前按单条 chunk 调用 `Insert`，尚未实现批量写入。
- Milvus 集成依赖真实服务，目前还没有自动化 integration test。
- 还没有实现 metadata filter。
- 还没有记录写入、检索和回答各阶段耗时。
- 当前 Makefile 的 `rag` 目标还没有显式转发 `-store` 参数，Milvus 验证使用直接运行 CLI 的方式完成。
- collection 与 embedding 模型、维度之间的版本关系仍依赖人工维护。

这些内容不影响第四天目标。去重和导入流程可以在服务化阶段继续完善，metadata filter 和效果记录属于后续学习任务。

## 今日记录

周四完成：

```text
Milvus 运行方式：Kubernetes + Helm + Standalone
namespace：infrastructure
Helm release：milvus-standalone
向量库实现：MilvusVS
collection：qwen3_embedding_chunk
vector index：AutoIndex
metric：COSINE
embedding dimension：EMBEDDING_DIM
store 切换：-store memory|milvus
```

端到端验证：

```text
文档：周五.md
问题：请总结第四周周五完成了哪些工作？
命中来源：周五.md#chunk-0
Milvus collection：已创建
chunk 数据：已写入
最终回答：内容正确，并包含来源引用
```

遇到并理解的问题：

```text
Standalone 仍然会启动 etcd 和 MinIO 等依赖
Milvus 地址应使用 SDK 可连接的地址，而不是随意添加 HTTP 前缀
```

## 下一步准备

下一天进入 RAG Eval。

需要准备固定问题集，并至少记录：

```text
question
expected source
topK 命中结果
是否应该回答
是否正确拒答
检索耗时
回答备注
```

第四天已经解决“数据能否持久化并检索”的问题；第五天要开始回答“检索效果是否稳定、能否被重复评估”。
