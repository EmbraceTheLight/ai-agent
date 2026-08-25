# 第 3 周第 6 天：RAG Demo v1

日期：2026-08-25

## 今日目标

今天的目标是把前五天完成的 RAG 能力整理成一个可演示的 Demo v1。

重点不是继续增加新功能，而是验证当前最小闭环是否能稳定完成：

```text
导入文档
切分 chunk
生成 embedding
写入内存向量库
检索 topK chunk
构造 prompt
基于资料回答
资料不足时拒答
```

## 今日完成

- 编写第 3 周 RAG Demo 文档：`docs/rag-demo-week03.md`。
- 整理 Demo 前置准备，包括 `.env` 配置、Ollama 启动和 embedding 模型准备。
- 固定了 RAG CLI 的演示命令。
- 验证了只执行 load / chunk / embed 的索引流程。
- 验证了能从资料中直接回答的问题。
- 验证了待办事项类问题。
- 验证了资料中没有答案时的拒答表现。
- 验证了大范围问题在默认 `topK=3` 和扩大 `topK=20` 时的效果差异。
- 记录了 RAG Demo 排查时应该优先观察的指标。

## Demo 前置条件

当前 Demo 依赖两类模型能力：

- 本地 Ollama embedding 服务。
- OpenAI-compatible LLM 回答服务。

`.env` 需要包含：

```env
OPENAI_API_KEY=<your-key>
OPENAI_BASE_URL=<your-base-url>
OPENAI_MODEL=<your-model>

EMBEDDING_BASE_URL=http://localhost:11434
EMBEDDING_MODEL=qwen3-embedding:0.6b
```

Ollama 需要启动，并准备好 embedding 模型：

```powershell
docker compose up -d
docker exec -it ollama ollama pull qwen3-embedding:0.6b
```

## Demo 1：只验证 load / chunk / embed

命令：

```powershell
go run ./cmd/rag-cli -docs "testdata/documents/work_notes_May/五月/第四周" -limitDocs 2 -limitChunks 3 -showPreview=true
```

这个 Demo 的目标不是回答问题，而是确认索引流程正常：

```text
文档读取成功
chunk 切分成功
embedding 生成成功
embedding 维度可以打印
limitDocs / limitChunks 能限制索引范围
```

实际输出中能看到：

```text
文档路径
embedding 服务
embedding 模型
chunk size
overlap
limit docs
limit chunks
topK
```

这个结果说明 CLI 参数化已经能支撑基础演示。

## Demo 2：能从资料直接回答的问题

问题：

```text
第四周周四做了什么
```

检索结果：

```text
[1] 周三.md#chunk-0, score=0.4493
[2] 周四.md#chunk-0, score=0.4200
[3] 待办事项.md#chunk-0, score=0.3777
```

虽然第一名命中了周三，但第二名命中了真正相关的 `周四.md#chunk-0`，最终回答主要基于周四资料。

模型总结出周四内容：

- 下午继续处理编译前端的“未识别的符号”问题。
- 定位到 `reservoir.Medium.reference_X` 常量折叠失败。
- 原因与 `reservoir.Medium` 是别名、按模型实例查找失败有关。
- 学习和理解组件展开逻辑。
- 做了加密库授权测试，并补充操作手册。
- 晚上完成 LeetCode No.322 零钱兑换。

引用来源：

```text
周四.md#chunk-0
```

这个 Demo 说明：

```text
只要相关 chunk 进入 topK，模型可以基于资料生成较完整回答。
```

同时也暴露一个现象：

```text
top1 不一定总是最准确的文件。
```

因此调试时不能只看回答，还要看 topK 命中的来源。

## Demo 3：待办事项类问题

问题：

```text
第四周有哪些待办事项
```

检索结果：

```text
[1] 待办事项.md#chunk-0, score=0.5104
[2] 周三.md#chunk-0, score=0.4740
[3] 周四.md#chunk-0, score=0.4046
```

这次 top1 命中了 `待办事项.md#chunk-0`，检索结果符合预期。

模型回答中提到：

- 仿真部分已完成，但结果树没有结果，结果文件可下载。
- 编译前端 / 平坦化代码中有数组维度不匹配相关条目。
- 自动化测试 / 部署中有增加接口测试、修复接口测试用例。
- `yslab` 后端部署事项内容不完整，模型明确指出后续内容缺失。

这个 Demo 说明：

```text
当问题关键词和文档标题/内容高度匹配时，检索效果更稳定。
```

也说明 prompt 的约束有效：

```text
资料不完整时，模型能指出不清楚，而不是补全细节。
```

## Demo 4：资料中没有答案的问题

问题：

```text
第四周有没有做 PostgreSQL 持久化
```

检索结果：

```text
[1] 周三.md#chunk-0, score=0.4238
[2] 待办事项.md#chunk-1, score=0.3950
[3] 周四.md#chunk-0, score=0.3834
```

模型回答：

```text
没有明确记录做过 PostgreSQL 持久化。
```

回答中说明第四周资料里提到的相关后端事项主要是：

- 接口测试修复。
- `yslab` 后端部署。
- 重启服务时部分已加载模型丢失，后台添加日志记录。

但没有出现 PostgreSQL、数据库持久化或相关实现记录。

这个 Demo 很重要，因为它验证了 RAG 的安全边界：

```text
没有资料依据时，模型应拒绝下确定结论。
```

当前回答不是简单说“没有做”，而是更谨慎地说：

```text
没有证据表明第四周做了 PostgreSQL 持久化。
```

这比直接否定更符合 RAG 的回答方式。

## Demo 5：周级大范围问题

问题：

```text
第四周我做了什么
```

默认 `topK=3` 时，检索结果：

```text
[1] 周三.md#chunk-0, score=0.4377
[2] 周四.md#chunk-0, score=0.4071
[3] 待办事项.md#chunk-0, score=0.3902
```

模型回答能总结：

- 周三修复和更新测试用例。
- 排查编译前端 / 后端的“未识别的符号”问题。
- 整理 Dymola 加密与授权第三方库操作手册。
- 理解组件展开逻辑。
- 完成 LeetCode No.76 和 No.322。
- 待办事项只能作为待办，不能确认已经完成。

同时模型指出：

```text
只能看到周三、周四和待办事项资料；
周一、周二、周五及周末资料不足，不知道。
```

这个结果符合默认 topK 的局限。

当前 topK 只放入 3 个 chunk，大范围问题很难覆盖整周所有文档。

## Demo 5-2：扩大 topK 后的对比

命令：

```powershell
go run ./cmd/rag-cli -docs "testdata/documents/work_notes_May/五月/第四周" -question "第四周我做了什么" -topK 20
```

扩大 topK 后，检索结果中出现了更多文档：

```text
周三.md#chunk-0
周四.md#chunk-0
待办事项.md#chunk-0
周五.md#chunk-0
周四/加密 & 授权模型库.md#chunk-13
周五/frontend-flat-learning-todo.md#chunk-6
待办事项.md#chunk-1
周二.md#chunk-5
0525周一.md#chunk-2
```

这次回答补充出了更多内容：

- 前端平坦化学习整理。
- 示例模型 `FlatLearning.mo`。
- 学习计划 `frontend-flat-learning-todo`。
- 测试用 `main.go`。
- 名称补全、求值、变量合并、package projection 等调试阶段。
- 晚上完成多道算法题。
- 还有部分当周待办和遗留事项。

这个对比说明：

```text
topK 会直接影响大范围问题的覆盖率。
```

默认 `topK=3` 更适合单日、单主题问题。

扩大 `topK` 能覆盖更多文档，但也会带来新问题：

- prompt 上下文变长。
- 低相关 chunk 也可能进入 prompt。
- 回答可能更像周总结，但噪声也更多。
- 后续需要 eval 来判断 topK 的合适范围。

## Demo 结果总结

本次 Demo 验证结果：

```text
load / chunk / embed：通过
向量检索：通过
基于资料回答：通过
资料不足拒答：通过
引用来源：基本通过
大范围问题：可用，但受 topK 和文档覆盖影响明显
```

最稳定的问题类型：

```text
第四周周四做了什么
第四周有哪些待办事项
第四周有没有做 PostgreSQL 持久化
```

不太稳定的问题类型：

```text
第四周我做了什么
```

原因是这个问题范围太大，需要覆盖多个日期和多个主题。当前系统只是简单 topK 检索，不会自动做分阶段汇总。

## 今日核心理解

今天最重要的理解是：

```text
RAG Demo 要先验证检索，再验证回答。
```

如果回答不理想，排查顺序应该是：

```text
1. 文档有没有被加载
2. chunk 有没有被切出来
3. embedding 有没有生成
4. query embedding 是否成功
5. topK 命中的 chunk 是否相关
6. prompt 是否包含正确资料
7. 模型回答是否遵守约束
```

不要一开始就把问题归因到模型。

很多时候，回答不好是因为：

```text
关键文档没有进入索引
limitDocs / limitChunks 限制过小
topK 太小
问题范围太大
chunk 内容和问题语义距离不够近
```

## 参数影响

### limitDocs

`limitDocs` 会限制参与索引的文档数量。

如果设置太小，关键文档可能根本不会进入向量库。

### limitChunks

`limitChunks` 会限制每篇文档参与 embedding 的 chunk 数。

如果设置太小，文档后半部分可能无法被检索。

### topK

`topK` 会限制放入 prompt 的检索结果数量。

小 `topK` 的优点：

- prompt 更短。
- 噪声更少。
- 适合单点问题。

小 `topK` 的缺点：

- 可能漏掉关键资料。
- 不适合周总结这种大范围问题。

大 `topK` 的优点：

- 覆盖更多文档。
- 更适合跨日期、跨主题问题。

大 `topK` 的缺点：

- prompt 更长。
- 低相关 chunk 可能进入上下文。
- 回答可能混入更多待办或间接信息。

## 当前不足

当前 Demo v1 仍然有一些限制：

- 还没有持久化向量库，每次运行都要重新 embedding。
- 还没有 metadata filter，不能先限定日期、文件夹或文件名。
- 还没有固定 eval 表格记录每个问题的命中情况。
- 还没有自动区分“已完成事项”和“待办事项”。
- 周级总结类问题需要更好的检索策略，不能只依赖单次 topK。
- 引用格式已经可用，但还可以进一步精简为相对路径。

这些问题不影响第 3 周 Demo v1 的完成，但可以作为第 4 周 pgvector、metadata filter 和 eval 的准备方向。

## 今日记录

周六完成：

- 完成第 3 周 RAG Demo 文档。
- 完成 5 组 Demo 问题验证。
- 验证索引流程可用。
- 验证可回答问题能基于资料回答。
- 验证无答案问题能拒答。
- 验证 topK 对大范围问题影响明显。

Demo 是否稳定：

- 单日问题相对稳定。
- 待办事项类问题相对稳定。
- 无答案问题表现符合预期。
- 周级总结问题依赖 `topK`，默认值下覆盖不足。

哪些问题检索效果好：

- `第四周有哪些待办事项`
- `第四周有没有做 PostgreSQL 持久化`

哪些问题检索效果一般：

- `第四周周四做了什么`

原因是 top1 命中了周三，但 top2 命中了周四，最终回答仍然可用。

哪些问题检索效果受参数影响大：

- `第四周我做了什么`

默认 `topK=3` 只能覆盖周三、周四和待办事项；扩大到 `topK=20` 后能覆盖周一、周二、周五和更细的专题文档。

## 下一步准备

下一天进入调参、对比和复盘。

建议记录至少 5 个问题的效果：

```text
问题
topK
命中的 chunk
回答是否正确
引用是否可信
是否应该拒答
```

重点对比：

```text
chunk size
overlap
topK
limitDocs
limitChunks
```

第 7 天可以把本周经验总结为：

```text
RAG 不是一次性把所有资料塞给模型
RAG 的核心是检索质量、引用可信度和拒答边界
```
