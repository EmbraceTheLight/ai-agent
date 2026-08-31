# 第 4 周细则：RAG 生产化与 Milvus 持久化检索

适用时间：工作日每天 30-60 分钟，周末 2-4 小时。

本周目标：把第 3 周的 RAG Demo 从内存检索推进到更接近真实项目的后端服务，使用 Milvus 替代内存向量库，实现持久化向量存储、metadata filter、eval 和 HTTP API。

本周最终交付：

- 保持第 3 周内存 `VectorStore` 可用。
- 抽象出可替换的 `VectorStore` 接口，让内存实现和 Milvus 实现共享同一套检索流程。
- 为 chunk 补充更完整 metadata。
- 使用 Kubernetes 部署 Milvus。
- 设计 Milvus collection schema。
- 实现 Milvus 写入和向量检索。
- 支持至少一个 metadata filter，例如按 source file 或文档目录过滤。
- 有固定 eval 问题集，记录检索命中、回答质量、拒答表现和耗时。
- 提供简单 HTTP API：`/documents/import`、`/ask`、`/health`。
- 更新 Demo 文档和周复盘。

## 本周核心理解

第 3 周的 RAG Demo 已经证明：

```text
本地文档
-> chunk
-> embedding
-> 内存 topK 检索
-> prompt
-> LLM 回答
```

第 4 周要解决的是工程化问题：

```text
向量不能只存在内存里
chunk 需要可追踪 metadata
检索需要 filter
效果需要 eval
服务需要 HTTP API
```

本周选择 Milvus，而不是 pgvector，原因是：已有较多 MySQL / SQL 后端经验，本周更值得补齐“专业向量数据库”的能力，包括 collection schema、向量索引、search params、scalar field filter 和独立向量服务部署。

本周不要把目标变成“学完 Milvus 所有功能”。只需要把它作为 RAG 的持久化向量库接进当前项目。

## Milvus 部署约定

本周使用本地 Kubernetes 集群部署 Milvus。为了控制学习范围，第一版使用 Milvus Standalone，并通过 Helm 管理部署资源；后续需要高可用或横向扩展时，再评估 Milvus Distributed 或 Milvus Operator。

本地集群可以使用现有的 Kubernetes 环境，也可以使用 Minikube、Kind 等工具创建。应用在集群外运行时，通过 `kubectl port-forward` 暴露 Milvus Proxy；应用部署到同一集群后，则应使用 Kubernetes Service 的 DNS 地址连接。

建议将部署相关文件整理为：

```text
deploy/
  k8s/
    milvus-values.yaml
    README.md
```

`README.md` 至少记录：Kubernetes 前置条件、Helm 安装命令、命名空间、Pod/Service 检查命令、删除命令以及本地端口转发方式。不要把生成的集群状态或本地 kubeconfig 提交到仓库。

## 推荐目录

可以在现有项目结构上逐步增加：

```text
internal/
  rag/
    vector_store.go
    vector_store_memory.go        可选: 拆分内存实现
    vector_store_milvus.go
    metadata.go                   可选: metadata 结构
    retriever.go
    prompt.go
  eval/
    rag_eval.go
cmd/
  rag-cli/
  rag-api/
deploy/
  k8s/
    milvus-values.yaml
    README.md
testdata/
  eval/
    rag_cases.jsonl
docs/
  rag-demo-week03.md
  rag-demo-week04.md
学习笔记/
  第四周/
```

不必第一天一次性建完。每天只补当天需要的最小文件。

## 周一：重新整理 VectorStore 接口

时间预算：30-60 分钟。

### 今日目标

让内存向量库和 Milvus 向量库可以在同一套 RAG 流程里替换。

### 今日步骤

1. 回顾当前 `VectorStore` 接口：

```go
Add(vector, chunk)
Search(queryVector, topK)
```

2. 思考 Milvus 接入后需要哪些参数：

```text
collection name
embedding dimension
metric type
metadata filter
```

3. 设计更稳定的返回结构：

```text
SearchResult
  Chunk
  Score
```

4. 不急着接 Milvus，先保持内存实现测试通过。

### 今日验收

运行测试：

```powershell
go test ./internal/rag
```

内存向量库仍然能：

```text
Add chunk vector
Search topK
返回 score + source + chunk index + content
```

### 今日记录

```text
周一完成：
VectorStore 接口调整：
SearchResult 包含哪些字段：
为了 Milvus 预留了哪些能力：
```

## 周二：补全 Chunk Metadata 和测试

时间预算：30-60 分钟。

### 今日目标

让每个 chunk 不只是有文本，还能被稳定追踪、过滤和引用。

### 今日步骤

1. 检查当前 `Chunk` 字段：

```text
SourceFile
ChunkIndex
Content
Start
End
```

2. 设计是否需要新增：

```text
DocumentID
Title
SourceName
UpdatedAt
ContentHash
```

3. 第一版建议最小补充：

```text
DocumentID 可选
SourceFile
ChunkIndex
Content
Start
End
Title 可选
```

4. 补测试，确保 metadata 不会在 loader、chunker、vector store、prompt 之间丢失。

### 今日验收

测试能证明：

```text
chunk 来源文件不会丢
chunk index 不会丢
Search 结果能拿到 source file 和 chunk index
prompt 中能生成引用来源
```

### 今日记录

```text
周二完成：
metadata 包含哪些字段：
哪些字段暂时不做：
metadata 在 RAG 链路中如何流动：
```

## 周三：设计 Milvus Collection Schema

时间预算：30-60 分钟。

### 今日目标

设计 Milvus collection，用于保存 chunk、embedding 和 metadata。

### 今日步骤

1. 阅读 Milvus 官方文档中 collection、schema、index、search、filter 相关内容。

2. 设计第一版 collection：

```text
collection: rag_chunks

id: VarChar 或 Int64 主键
source_file: VarChar
chunk_index: Int64
content: VarChar
start_offset: Int64
end_offset: Int64
embedding: FloatVector(dim = 当前 embedding 维度)
created_at: Int64
```

3. 明确索引参数：

```text
metric type: COSINE
index type: 第一版可用 AUTOINDEX 或 Milvus 推荐的默认方式
```

4. 在笔记中记录：Milvus 的 collection schema 和关系型数据库表结构有什么不同。

### 今日验收

能写出一份清晰 schema 草图，并回答：

```text
哪一个字段是 vector
哪些字段用于 metadata filter
embedding dimension 从哪里来
为什么 metric type 选 cosine
```

### 今日记录

```text
周三完成：
Milvus collection schema：
vector 字段：
metadata 字段：
索引和相似度选择：
```

## 周四：接入 Milvus VectorStore

时间预算：30-60 分钟。

### 今日目标

实现 `MilvusVectorStore`，把内存向量搜索迁移到持久化向量数据库。

### 今日步骤

1. 准备 Kubernetes 环境并使用 Helm 部署 Milvus Standalone：

```powershell
kubectl create namespace milvus
helm repo add zilliz https://zilliztech.github.io/milvus-helm
helm repo update
helm install milvus zilliz/milvus --namespace milvus --set cluster.enabled=false
kubectl get pods,svc -n milvus
```

等待 Milvus 相关 Pod 进入 `Running` 或 `Ready` 状态。应用在本地运行时，可以转发 Milvus Proxy：

```powershell
kubectl port-forward svc/milvus 19530:19530 -n milvus
```

如果当前 Helm chart 生成的 Service 名称不同，以 `kubectl get svc -n milvus` 的实际名称为准，并将完整过程记录到 `deploy/k8s/README.md`。

2. 增加配置项：

```env
MILVUS_ADDR=localhost:19530
MILVUS_COLLECTION=rag_chunks
```

应用如果也部署在 Kubernetes 集群中，应将 `MILVUS_ADDR` 改为 Milvus Proxy Service 的集群内 DNS 地址，而不是使用 `localhost`。

3. 实现最小 Milvus client 封装：

```text
connect
create collection if not exists
insert chunk + vector
search queryVector topK
```

4. `rag-cli` 增加 store 参数：

```text
-store memory
-store milvus
```

5. 先跑通固定 Demo，不急着做复杂 filter。

### 今日验收

Milvus 部署完成且端口转发建立后，能用命令跑通：

```powershell
go run ./cmd/rag-cli -store milvus -docs "testdata/documents/..." -question "..."
```

输出中能看到：

```text
Milvus collection
inserted chunks
topK search results
source file
chunk index
score
answer with citation
```

同时能够使用以下命令确认 Kubernetes 资源状态：

```powershell
kubectl get pods,svc -n milvus
kubectl get pvc -n milvus
```

### 今日记录

```text
周四完成：
Milvus 如何通过 Kubernetes 启动：
Kubernetes 集群和命名空间：
Helm release 与 Service 名称：
本地端口转发方式：
插入了多少 chunk：
检索是否符合预期：
遇到的 SDK / schema / dimension 问题：
```

## 周五：Eval 脚本和效果记录

时间预算：30-60 分钟。

### 今日目标

不要只凭感觉判断 RAG 效果，开始用固定问题集记录效果。

### 今日步骤

1. 准备 eval case 文件：

```jsonl
{"id":"q1","question":"第四周周四做了什么","should_answer":true,"expected_sources":["周四.md"]}
{"id":"q2","question":"第四周有没有做 PostgreSQL 持久化","should_answer":false,"expected_sources":[]}
```

2. 实现一个简单 eval 脚本或命令：

```text
读取 eval cases
对每个问题执行检索
记录 topK source
可选: 调模型回答
输出结果表
```

3. 先统计最基础指标：

```text
retrieval hit
should answer / should abstain
latency
```

4. 不必第一版就做复杂自动打分。

### 今日验收

至少跑 5 个问题，并记录：

```text
问题
topK 命中文档
是否命中 expected source
是否应该拒答
回答备注
```

### 今日记录

```text
周五完成：
eval case 数量：
命中好的问题：
命中差的问题：
拒答是否符合预期：
```

## 周六：RAG HTTP API

时间预算：2-4 小时。

### 今日目标

把 RAG 从 CLI Demo 推进到后端服务形态。

### 今日步骤

1. 新增或整理 HTTP 入口：

```text
cmd/rag-api/
```

2. 实现最小接口：

```text
GET  /health
POST /documents/import
POST /ask
```

3. `/documents/import` 第一版可以接收本地路径：

```json
{"path":"testdata/documents/work_notes_May/五月/第四周"}
```

4. `/ask` 接收：

```json
{"question":"第四周周四做了什么","top_k":3}
```

5. 可选增加 metadata filter：

```json
{"source_file_contains":"周四"}
```

### 今日验收

能通过 HTTP 完成：

```text
导入文档
提问
返回 answer
返回 citations
返回 search results
```

### 今日记录

```text
周六完成：
HTTP API 有哪些：
请求/响应格式：
metadata filter 是否接入：
Demo 是否稳定：
```

## 周日：技术总结和复盘

时间预算：2-4 小时。

### 今日目标

整理第 4 周成果，让这个 RAG 项目更像可展示的后端工程项目。

### 今日步骤

1. 更新 README 或新增技术总结。

2. 画出流程：

```text
documents/import
-> loader
-> chunker
-> embedder
-> Milvus

ask
-> query embedding
-> Milvus search
-> prompt
-> LLM answer
```

3. 总结 Milvus 和内存检索的区别。

4. 总结 eval 结果。

5. 记录下周 Agent Loop 前还需要补的点。

### 今日验收

能用自己的话讲清楚：

```text
为什么用 Milvus
Milvus collection 如何设计
metadata filter 如何工作
RAG 怎么评估
HTTP API 怎么组织
当前还不是生产级的原因
```

### 今日记录

```text
周日完成：
本周完成：
Milvus 接入效果：
eval 结果：
还不稳定的地方：
可以写进简历的点：
```

## 本周完成标准

最低完成标准：

- `VectorStore` 接口可替换。
- 内存向量库测试仍通过。
- 有 Milvus collection schema。
- 有 Kubernetes 部署说明和配置文件。
- 能在 Kubernetes 中启动 Milvus。
- 应用能通过 Service 或 `kubectl port-forward` 连接 Milvus。
- 能插入 chunk + vector。
- 能从 Milvus topK 检索。
- 检索结果带 source file、chunk index、score。
- 有至少 5 个 eval 问题。
- 有第 4 周学习笔记。

优秀完成标准：

- `rag-cli` 支持 `-store memory|milvus`。
- 有 `POST /documents/import` 和 `POST /ask`。
- `/ask` 返回 answer + citations + search results。
- 支持至少一个 metadata filter。
- 有 10-20 个 eval 问题。
- 能对比 memory store 和 Milvus store 的差异。
- 能解释 Milvus schema、index、metric、filter。

## 本周不要做什么

- 不要一开始就做复杂 UI。
- 不要把 Milvus 接入写死在 CLI 里，要通过接口替换。
- 不要每次回答都重新导入全部文档，除非当前只是临时 Demo。
- 不要在日志中打印完整 embedding。
- 不要把 eval 变成模型主观打分，第一版先记录检索命中。
- 不要为了“生产化”一次性加入权限、异步任务、监控告警，这些后面再补。

第 4 周的价值是把 RAG 从“能跑的 Demo”推进到“有持久化、有 filter、有 eval、有 API 的后端服务雏形”。Milvus 只是实现这个目标的工具，核心仍然是检索质量、引用可信度和工程边界。
