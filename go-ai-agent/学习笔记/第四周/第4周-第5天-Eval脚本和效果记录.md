# 第 4 周第 5 天：Eval 脚本和效果记录

日期：2026-09-20

## 今日目标

不要只凭感觉判断 RAG 效果，而是使用固定问题集，对检索命中、回答决策和检索耗时进行重复验证。

本次 Eval 只做基础的离线检查，不引入复杂的模型自动评分。

## 今日完成

- 新增独立的 `cmd/rag-eval` 命令，不将离线 Eval 逻辑混入正常的 `rag-cli` 在线问答流程。
- 新增 `testdata/eval_case.jsonl`，共准备 15 个固定问题。
- 覆盖单文档问题、嵌套文档问题、多 source 问题和预期拒答问题。
- 为每个问题单独生成 query embedding，并执行 topK 检索。
- 输出每个 case 的问题、期望回答、topK source、score、chunk index 和耗时。
- 使用 `EvalCheck` 按 `expected_sources` 和 `min_score` 将结果分类为：
  - `NotFindInExpectSources`
  - `ScoreTooLow`
  - `Legal`
- 使用 `EvalShouldAnswer` 与 case 中预先定义的 `should_answer` 对比。
- 新增 q15，专门覆盖 `ScoreTooLow` 分支。
- 分别使用 memory 和 Milvus 完成完整 Eval。

## Eval 判断规则

离线 Eval 中的合法 chunk 必须同时满足：

```text
chunk 所在文件属于 expected_sources
chunk score >= min_score
```

当至少存在一个合法 chunk 时：

```text
EvalShouldAnswer = true
```

否则：

```text
EvalShouldAnswer = false
```

这里的 `expected_sources` 是离线评估用的 gold source，线上 RAG 流程不能依赖它决定是否调用 LLM。线上只能根据实际检索结果、score 和其他运行时规则做回答或拒答决策。

## Eval Case 设计

当前共有 15 个 case：

```text
q1 - q10：正常可回答问题
q11 - q13：资料中没有期望 source 的拒答问题
q14：待办事项问题
q15：人为设置高 min_score 的 ScoreTooLow 边界问题
```

q15 复制 q5 的问题和期望 source，但将 `min_score` 调整为 `0.95`。实际最高分为 `0.8602`，因此三个命中的 chunk 都会进入 `ScoreTooLow`，不会进入 `Legal`。

## Memory 验证

运行方式：

```powershell
go run ./cmd/rag-eval `
  -store memory `
  -docs "testdata/documents/work_notes_May/五月/第四周" `
  -cases "testdata/eval_case.jsonl" `
  -topK 3 `
  -chunkSize 500 `
  -overlap 100 `
  -timeout 10m
```

验证结果：

```text
文档数：9
总 chunk 数：46
已写入向量库 chunk 数：46
总 case 数：15
回答决策匹配：15/15
命中合法 source：11/15
```

q15 结果：

```text
不在期望 source：0
分数过低：3
合法 chunk：0
EvalShouldAnswer：false
决策是否匹配：true
```

## Milvus 验证

运行方式与 memory 版本相同，仅将存储方式改为：

```text
-store milvus
```

验证结果：

```text
存储方式：milvus
文档数：9
总 chunk 数：46
已写入向量库 chunk 数：46
总 case 数：15
回答决策匹配：15/15
命中合法 source：11/15
```

Milvus 版本同样完整执行了 q1～q15，没有出现 collection 初始化、写入或搜索错误。q15 的分类结果与 memory 版本一致：3 个 chunk 进入 `ScoreTooLow`，0 个 chunk 进入 `Legal`。

## 结果分析

本次 Eval 说明：

```text
固定问题集可以被重复执行
memory 和 Milvus 可以复用同一套 Eval 流程
正常可回答问题全部得到符合预期的回答决策
预期拒答问题全部正确拒答
ScoreTooLow 分支已被实际覆盖
每个 case 都记录了 topK source、score 和耗时
```

`命中合法 source: 11/15` 需要结合 case 类型理解：

```text
11 个正常可回答 case 命中合法 source
q11 - q13 是预期拒答，不应存在合法 source
q15 是人为设置高阈值的低分拒答，也不应存在合法 source
```

因此更直观的指标是：

```text
可回答问题的合法 source 命中：11/11
拒答决策正确：4/4
整体回答决策匹配：15/15
```

## 当前边界

- 当前 Eval 没有调用 LLM，符合第一版只做基础检索指标的计划。
- 当前没有自动评价最终答案文本的正确性、完整性或引用质量。
- `expected_sources` 只用于离线 gold source 检查，不能直接作为线上回答门控条件。
- q15 是为了覆盖 `ScoreTooLow` 的人工边界 case，不应作为普通检索质量样本解读。
- 当前每次运行仍会重新加载、切分、embedding 和写入文档；Milvus 的重复导入和去重问题仍待后续处理。

## 今日结论

第四周第五天完成。

本次完成了从“手工观察 RAG 结果”到“使用固定问题集重复验证 RAG 检索效果”的转变，并验证了 memory 和 Milvus 两种 VectorStore 实现可以共享同一套离线 Eval 流程。

下一步可以进入 HTTP API 或继续完善 metadata filter、批量写入、增量更新和更细的检索质量指标。

