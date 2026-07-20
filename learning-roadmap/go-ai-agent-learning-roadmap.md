# Go 开发者 AI Agent 学习路线

适用对象：已有 Go 后端开发经验，工作日只有下班后约 30-60 分钟学习时间，周末可以集中投入 2-4 小时。

目标：用 8 周时间掌握 Go 方向 AI Agent 开发的核心能力，并完成两个可写进简历、可用于面试讲解的作品集项目。8 周后可以继续用 4 周做生产化补强，把 Demo 推近真实工作形态。

更新时间：2026-07-13

## 总体目标

8 周后，你应该具备以下能力：

- 使用 Go 调用主流 LLM API。
- 理解并实现 tool calling / function calling。
- 用 Go 实现一个基础 Agent Loop。
- 构建 RAG 文档问答系统。
- 理解 MCP，并能用 Go 写一个简单 MCP Server。
- 具备 Agent 生产化意识，包括权限、安全、成本、日志、评测、部署。
- 拥有两个可展示项目：
  - Go RAG Knowledge Agent
  - Go MCP Ops Agent

这条路线的重点不是成为算法工程师，而是成为能把 LLM/Agent 落到真实业务系统里的 Go 后端工程师。

## 成果定位

如果认真完成前 8 周，这两个项目的定位应该是：**工程化作品集项目**，不是简单玩具项目，也不要包装成生产级平台。

| 档位 | 特征 | 面试价值 |
|---|---|---|
| 玩具项目 | 只会调用 LLM API，缺少测试、权限、日志、README 和演示流程 | 只能证明试过 API |
| 作品集项目 | 有清晰模块、HTTP API、RAG 引用、tool calling、MCP、trace、权限、eval、演示脚本 | 能证明可以用 Go 落地 Agent 后端系统 |
| 生产级项目 | 接入真实业务系统、认证授权、多租户、审计、监控告警、稳定评测、成本治理、CI/CD | 能证明可以负责线上 Agent 服务 |

本路线前 8 周追求第二档：让面试官愿意围绕项目继续追问。第 9-12 周用于补齐生产化能力。

实际工作中的差距主要在：

- 真实业务数据更脏，文档权限、数据口径、历史版本会复杂很多。
- 评测不只是“能回答”，还要持续衡量召回、幻觉、拒答、成本和延迟。
- 工具一旦能读写内部系统，就必须有权限、审计、确认、脱敏和安全策略。
- 模型输出不稳定，工具会超时或失败，需要 fallback、重试、熔断、trace 和告警。
- 实际项目需要接入已有用户系统、知识库、工单系统、监控系统、CI/CD 和部署环境。

## 准备日：开始前先把环境铺平

建议在正式进入第 1 周之前，先用半小时到一小时完成准备。不要把宝贵的工作日学习时间消耗在环境问题上。

| 事项 | 目标 |
|---|---|
| 安装 Go 1.22+ | 确保可以运行 `go version`、`go test ./...` |
| 准备一个 LLM API Key | OpenAI、DeepSeek、阿里云百炼任选一个先跑通 |
| 新建 Git 仓库 | 每天学习结束都提交一次小 commit |
| 准备 `.env.example` | 只放变量名，不提交真实 API Key |
| 准备 `Makefile` | 至少包含 `make test`、`make run`、`make demo` |
| 准备 `README.md` | 一开始就记录目标、运行方式和学习日志 |
| 准备测试目录 | 后续 eval、mock 文档、失败样例都放进去 |

## 推荐项目结构

建议从第一天就按后端工程项目组织，不要写成散落脚本。

```text
cmd/
  rag-agent/
  ops-agent/
internal/
  llm/
  agent/
  tools/
  rag/
  mcp/
  eval/
  config/
  log/
docs/
  demo.md
  architecture.md
testdata/
  documents/
  eval_cases.jsonl
```

目录含义：

- `cmd/rag-agent`：企业知识库问答项目入口。
- `cmd/ops-agent`：运维排查 Agent 项目入口。
- `internal/llm`：LLM Client、streaming、structured output 封装。
- `internal/agent`：Agent Loop、状态机、memory、step limit。
- `internal/tools`：tool registry、参数校验、权限控制。
- `internal/rag`：chunk、embedding、retriever、vector store。
- `internal/mcp`：MCP Server / Client 相关代码。
- `internal/eval`：RAG 和 Agent 的评测脚本。
- `docs/demo.md`：固定演示脚本，面试前照着跑。
- `testdata`：测试文档、eval case、mock 数据。

## 官方文档索引

优先看官方文档，不要一开始就跟着二手教程堆框架。中文官方文档目前主要集中在国内模型供应商；OpenAI、MCP、Go、pgvector 等核心资料以英文官方文档为主。

### LLM API / Tool Calling / Agent

- OpenAI Responses API：<https://platform.openai.com/docs/guides/responses>
- OpenAI Function Calling：<https://platform.openai.com/docs/guides/function-calling>
- OpenAI Structured Outputs：<https://platform.openai.com/docs/guides/structured-outputs>
- OpenAI Agents 指南：<https://platform.openai.com/docs/guides/agents>
- OpenAI Go SDK：<https://github.com/openai/openai-go>
- Anthropic 有效构建 Agent 的工程文章：<https://www.anthropic.com/engineering/building-effective-agents>

### MCP

- MCP 官方介绍：<https://modelcontextprotocol.io/docs/getting-started/intro>
- MCP 官方规范：<https://modelcontextprotocol.io/specification>
- MCP Go SDK：<https://github.com/modelcontextprotocol/go-sdk>

### RAG / 向量检索 / 数据库

- pgvector 官方仓库：<https://github.com/pgvector/pgvector>
- PostgreSQL 官方文档：<https://www.postgresql.org/docs/>
- Qdrant 官方文档：<https://qdrant.tech/documentation/>

### Go 工程化

- Go 官方文档：<https://go.dev/doc/>
- Effective Go：<https://go.dev/doc/effective_go>
- Go Testing：<https://go.dev/doc/tutorial/add-a-test>
- Go Database Access：<https://go.dev/doc/tutorial/database-access>

### 部署和工程化

- Docker Compose 官方文档：<https://docs.docker.com/compose/>
- OpenTelemetry Go 官方文档：<https://opentelemetry.io/docs/languages/go/>

### 中文官方文档（可作为模型供应商替代入口）

- DeepSeek API 中文文档：<https://api-docs.deepseek.com/zh-cn/>
- 阿里云百炼中文文档：<https://help.aliyun.com/zh/model-studio/>

使用建议：

- 第 1-2 周优先看 OpenAI Responses API、Function Calling、Structured Outputs 和所选模型供应商的 API 文档。
- 第 3-4 周看 pgvector、PostgreSQL、Qdrant，不要过早纠结向量数据库选型。
- 第 5-6 周看 MCP 官方文档和 MCP Go SDK。
- 第 7-12 周补 Docker Compose、OpenTelemetry、Go testing、数据库和部署文档。

## 每天固定节奏

### 工作日

建议每天 30-60 分钟，按下面节奏执行：

1. 10 分钟：阅读文档或理解一个概念。
2. 30 分钟：写 Go 代码，哪怕只完成一个很小的函数。
3. 10 分钟：记录今天学到什么、遇到什么问题、明天继续做什么。

### 休息日

建议周末每次 2-4 小时：

- 周六：集中做项目，把工作日的小代码拼起来。
- 周日：补测试、整理 README、写复盘、准备面试表达。

## 第 1 周：LLM API 和 Go 基础接入

本周目标：跑通 Go 调用 LLM API，完成一个最小命令行 AI 助手。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 配置 API Key，跑通一次 Go SDK 普通对话请求 | `hello_llm.go` |
| 周二 | 理解 system/user/developer message，写一个命令行问答程序 | CLI 问答 Demo |
| 周三 | 加入 streaming 输出，让回答逐字返回 | 流式输出 Demo |
| 周四 | 学习 JSON / structured output，让模型返回固定结构 | `{intent, answer, confidence}` |
| 周五 | 封装 `LLMClient` 接口，方便以后替换模型 | `internal/llm/client.go` |
| 周六 | 做一个 Go CLI 助手：输入问题，流式回答，支持结构化输出 | 项目雏形 |
| 周日 | 写 README：项目目标、如何运行、核心代码说明 | README v1 |

本周重点：

- 不要急着学框架。
- 先把一次请求、一次响应、一次流式输出弄懂。
- 把模型当成外部服务，按后端工程方式封装它。

## 第 2 周：Tool Calling 和最小安全底座

本周目标：理解 Agent 的地基，让模型能决定调用 Go 写的工具，同时从一开始建立工具安全边界。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 理解 tool calling：模型不执行动作，只决定调用哪个工具 | 笔记 |
| 周二 | 写第一个工具：`get_current_time` | 时间工具 |
| 周三 | 写第二个工具：`calculator`，支持加减乘除 | 计算工具 |
| 周四 | 写第三个工具：`http_get`，只允许访问 allowlist 或本地 mock API | 受限 HTTP 工具 |
| 周五 | 实现 tool call loop：模型请求工具 -> Go 执行 -> 结果回传模型 | 基础 loop |
| 周六 | 做一个“工具型 Agent”：能算数、查时间、请求 HTTP API | Tool Agent Demo |
| 周日 | 加错误处理、安全限制：工具不存在、参数错误、超时、最大调用次数、URL allowlist | 稳定版 Demo |

本周重点：

- 工具参数必须做 JSON Schema 约束。
- Go 侧必须做参数校验，不能完全信任模型。
- Agent 必须设置最大调用次数，避免无限循环。
- 不要开放任意 URL 的 `http_get`，避免 SSRF、内网访问、无限请求等风险。
- 安全不是第 7 周才开始补，而是从第一个工具开始就要有边界。

## 第 3 周：RAG 文档问答

本周目标：让模型能基于你的私有文档回答问题。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 理解 embedding、chunk、vector search、topK | RAG 流程笔记 |
| 周二 | 准备 5-10 篇 markdown/txt 文档，写 Go 程序读取并切分 | 文档切分代码 |
| 周三 | 调 embedding API，把 chunk 转成向量 | embedding 代码 |
| 周四 | 先用本地内存向量搜索完成相似度搜索，减少环境和选型干扰 | 向量搜索 Demo |
| 周五 | 把检索结果塞进 prompt，让模型基于资料回答 | RAG 问答 Demo |
| 周六 | 做一个 Go RAG 问答 Demo：输入问题，返回答案和引用片段 | RAG Demo v1 |
| 周日 | 优化 chunk 大小、topK、引用格式；记录效果对比 | RAG 复盘 |

本周重点：

- 回答必须带引用来源。
- 如果资料里没有答案，模型应该说不知道。
- RAG 的难点不是 API，而是检索质量和引用可信度。
- 第 3 周优先跑通闭环，不急着上向量数据库；工程化存储放到第 4 周。

## 第 4 周：RAG 生产化

本周目标：把 RAG 从 Demo 做成接近真实项目的后端服务。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 设计 `VectorStore` 接口，先保持内存实现可用 | `VectorStore` 接口 |
| 周二 | 给内存向量搜索补测试，并给 chunk 加 metadata：文件名、标题、段落编号、更新时间 | metadata + tests |
| 周三 | 设计 PostgreSQL 表结构和迁移脚本，为 pgvector 接入做准备 | schema + migration |
| 周四 | 接入 PostgreSQL + pgvector，把内存向量搜索迁移到持久化存储 | pgvector 存储 |
| 周五 | 写一个简单 eval 脚本，统计 retrieval recall、groundedness、abstention、latency/cost | eval 脚本 |
| 周六 | 把 RAG 做成 HTTP API：`/ask`、`/documents/import`，并支持 metadata filter | RAG API |
| 周日 | 写技术总结：RAG 流程、问题、优化点、面试怎么讲 | README v2 |

本周重点：

- 面试时不要只说“用了向量数据库”。
- 要能讲清楚 chunk 策略、topK、metadata filter、无法回答策略。
- 要有一组测试问题证明系统效果。
- eval 至少覆盖 10-20 个问题，包括能回答、不能回答、引用错误、相似但无关的问题。
- 第 4 周不要一开始就硬切数据库；先抽象接口，再替换存储实现，这更符合 Go 后端工程习惯。

## 第 5 周：真正的 Agent Loop

本周目标：实现一个有状态、有工具、有停止条件的 Agent。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 理解 Agent = LLM + tools + memory/state + loop + stop condition | Agent 笔记 |
| 周二 | 给 Agent 加最大步数，例如最多调用 5 次工具 | step limit |
| 周三 | 加短期记忆：保存本轮对话历史 | memory |
| 周四 | 加任务状态：`planning/running/tool_call/final/error` | state machine |
| 周五 | 加日志 trace：记录每一步模型想做什么、调用了什么工具、结果是什么，并支持按 request id 回放 | trace log |
| 周六 | 做一个“运维排查 Agent”：能查日志、查服务状态、给出排查结论 | Ops Agent v1 |
| 周日 | 整理流程图和 README，把 Agent 执行过程展示出来 | README v3 |

本周重点：

- Agent 不是无限聊天，而是有目标的循环执行系统。
- 必须有停止条件、错误处理、超时控制。
- trace log 是调试 Agent 的生命线。
- Agent 的核心不是“更会聊天”，而是把不稳定的模型调用包进可观察、可控制的工程流程里。

## 第 6 周：MCP

本周目标：理解 MCP，并用 Go 暴露一组 Agent 可调用工具。

本周优先使用官方 Go SDK 跑通 stdio MCP Server；如果时间允许，再了解 Streamable HTTP。先完成“能被发现、能被调用、能返回结构化结果”的最小闭环，不要一开始陷入传输层细节。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 理解 MCP：Host / Client / Server，以及 Tools / Resources / Prompts 的区别 | MCP 协议笔记 |
| 周二 | 用 Go 写一个最小 stdio MCP Server，跑通 `tools/list` 和 `tools/call` | MCP Server Hello World |
| 周三 | 暴露工具 `search_docs` | 文档搜索工具 |
| 周四 | 暴露工具 `query_order`，模拟查询订单 | 订单查询工具 |
| 周五 | 暴露工具 `create_ticket`，模拟创建工单 | 工单创建工具 |
| 周六 | 把你的 Agent 接到 MCP Server，完成一次真实工具调用 | MCP Agent Demo |
| 周日 | 写总结：MCP stdio、Streamable HTTP 和普通 HTTP tool 的区别、适合什么场景 | MCP README |

本周重点：

- MCP 的价值在于标准化工具接入。
- Go 后端开发者很适合写 MCP Server。
- 公司内部 API、数据库、日志系统、工单系统都可以通过 MCP 暴露给 Agent。
- 不要只会说“MCP 是工具协议”，要能讲清普通 HTTP tool 和 MCP Server 在发现、调用、复用、接入生态上的区别。

## 第 7 周：安全、权限、成本、可观测性

本周目标：在前面最小安全底座之上，系统补齐生产环境 Agent 必须考虑的问题。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | API Key 管理，不把密钥写死在代码里 | env 配置 |
| 周二 | 给工具加权限控制，例如只读工具和写操作工具分开 | tool permission |
| 周三 | 给危险操作加人工确认，例如创建工单前必须确认 | human approval |
| 周四 | 加 token/cost 统计 | cost log |
| 周五 | 学 prompt injection，测试“忽略之前指令”这类攻击 | 攻击样例 |
| 周六 | 给 Agent 加安全层：工具白名单、参数校验、敏感信息脱敏 | safety layer |
| 周日 | 写一篇面试稿：生产环境 Agent 有哪些风险，怎么治理 | 面试稿 |

本周重点：

- Agent 调工具之前必须过权限和参数校验。
- 写操作要比读操作更谨慎。
- Prompt injection 是真实风险，不是概念题。
- 成本、延迟、日志都要可观测。
- 第 7 周不是从零开始做安全，而是把第 2 周开始的工具边界升级成完整治理。

## 第 8 周：作品集和面试准备

本周目标：把学习成果整理成可展示、可讲解的项目。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 整理项目一：Go RAG Knowledge Agent | 项目一 README |
| 周二 | 整理项目二：Go MCP Ops Agent | 项目二 README |
| 周三 | 补充单元测试和集成测试 | tests |
| 周四 | 准备固定演示脚本：导入文档、提问、查看引用、调用工具、触发人工确认、查看 trace | `demo.md` 或 `make demo` |
| 周五 | 准备常见面试题：RAG、tool calling、MCP、Agent loop、安全 | 面试题 |
| 周六 | 做一次完整演示录屏或截图，确保项目能跑 | 演示材料 |
| 周日 | 更新简历，把“AI Agent 能力”写成项目成果，而不是关键词 | 简历描述 |

本周重点：

- 简历不要写“熟悉 AI Agent”这种空话。
- 要写你实现了什么能力，解决了什么问题，用了什么技术。
- 面试时用项目流程图讲，比堆术语更有说服力。
- 演示脚本比临场发挥重要，最好能用一条命令复现核心流程。

## 第 9-12 周：生产化补强（可选但强烈建议）

如果目标是更接近真实工作，而不只是完成作品集，建议再追加 4 周。

### 第 9 周：RAG 质量优化

本周目标：从“能回答”升级到“知道为什么答得好或不好”。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 扩展 eval 数据集到 30-50 条问题 | eval cases |
| 周二 | 对比 chunk size、overlap、topK 对效果的影响 | 对比报告 |
| 周三 | 加 rerank 或 keyword + vector 混合检索的实验 | 检索优化 Demo |
| 周四 | 统计 retrieval recall 和引用命中率 | eval 指标 |
| 周五 | 加无法回答测试集，评估拒答能力 | abstention eval |
| 周六 | 输出 RAG 质量报告 | quality report |
| 周日 | 整理 README 的“效果评估”章节 | README 更新 |

### 第 10 周：安全攻防和权限治理

本周目标：让 Agent 不只是能调工具，而是能安全地调工具。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 设计用户角色和工具权限模型 | permission model |
| 周二 | 加工具级 RBAC：只读、写操作、管理员操作 | RBAC |
| 周三 | 加敏感字段脱敏，例如手机号、token、邮箱 | masking |
| 周四 | 写 prompt injection 攻击样例 | attack cases |
| 周五 | 给写操作加入二次确认和审计日志 | approval + audit |
| 周六 | 做一次安全回归测试 | safety test |
| 周日 | 写安全设计说明 | security README |

### 第 11 周：部署、异步任务、可观测性

本周目标：把项目做得更像真实后端服务。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 用 Docker Compose 启动 API、PostgreSQL、pgvector | docker compose |
| 周二 | 加健康检查和配置管理 | health + config |
| 周三 | 给文档导入做异步任务，避免大文件阻塞请求 | job queue |
| 周四 | 加 request id、trace id、结构化日志 | structured logging |
| 周五 | 加 token、耗时、错误率统计 | metrics |
| 周六 | 写部署说明和本地一键启动脚本 | deploy README |
| 周日 | 做一次从零启动演示 | deployment demo |

### 第 12 周：项目包装和真实业务化

本周目标：把项目包装成面试官愿意追问的工程项目。

| 时间 | 学习内容 | 交付物 |
|---|---|---|
| 周一 | 画系统架构图和请求链路图 | architecture diagram |
| 周二 | 整理核心模块：LLMClient、ToolRegistry、AgentLoop、Retriever、MCP Server | module doc |
| 周三 | 补端到端测试和失败场景测试 | e2e tests |
| 周四 | 准备 5 分钟讲解稿和 15 分钟深挖稿 | interview script |
| 周五 | 更新简历项目描述，突出工程能力而不是关键词 | resume |
| 周六 | 完整演示录屏 | demo video |
| 周日 | 总复盘：哪些地方还不是生产级，下一步怎么补 | final review |

## 项目验收标准

8 周结束时，至少满足：

- 一条命令可以启动项目或核心 Demo。
- README 包含运行方式、架构图、核心流程、配置说明和常见问题。
- RAG 回答带引用，资料没有答案时能够拒答。
- tool calling 有 JSON Schema、参数校验、超时、最大步数。
- 写操作有人工确认或明确的 dry-run 模式。
- Agent 每一步有 trace，可以看到模型决策、工具调用、工具结果和最终回答。
- 至少有 10-20 条 eval 问题，并记录基本效果。
- 有固定 `demo.md` 或 `make demo`，不用临场拼命解释。

12 周结束时，进一步满足：

- 使用 PostgreSQL + pgvector 做持久化向量存储。
- 有 Docker Compose 或等价的一键本地启动方式。
- 有权限模型、审计日志、脱敏和 prompt injection 测试样例。
- 有结构化日志、request id、token/cost/latency 统计。
- 有端到端测试和失败场景测试。

## 最终项目一：Go RAG Knowledge Agent

### 核心功能

- 导入 markdown/txt 文档。
- 自动切分 chunk。
- 调用 embedding API。
- 使用 PostgreSQL + pgvector 存储向量和 metadata。
- 根据问题检索相关片段。
- 基于引用回答问题。
- 提供 HTTP API。
- 提供 eval 脚本，评估召回、引用、拒答、延迟和成本。

### 推荐接口

```text
POST /documents/import
POST /ask
GET  /documents
GET  /health
```

### 面试讲法

可以这样描述：

> 我用 Go 实现了一个企业知识库 RAG Agent，支持文档导入、chunk 切分、embedding、pgvector 向量检索、metadata filter 和引用回答。为了减少幻觉，我加入了无法回答判断和引用来源展示，并用一组测试问题评估检索召回、引用质量、拒答能力、延迟和成本。

## 最终项目二：Go MCP Ops Agent

### 核心功能

- Go 实现 Agent Loop。
- 支持 tool calling。
- 接入 MCP Server。
- MCP Server 暴露日志查询、订单查询、工单创建等工具。
- 支持工具权限控制。
- 支持人工确认。
- 记录每一步执行 trace。
- 支持失败重试、超时、最大步数和审计日志。

### 推荐工具

```text
search_docs
query_order
search_logs
check_service_status
create_ticket
```

### 面试讲法

可以这样描述：

> 我用 Go 实现了一个运维排查 Agent，通过 MCP Server 把日志查询、服务状态检查和工单创建暴露为标准工具。Agent 会根据用户问题规划调用工具，记录每一步 trace，并对写操作加入人工确认、权限控制和审计日志。

## 简历表达模板

可以写成：

> 使用 Go 构建基于 LLM 的 Agent 系统，支持 tool calling、RAG 检索增强、MCP 工具接入、流式响应、权限控制、调用链日志和成本统计；实现企业知识库问答与运维排查 Agent Demo。

如果想更偏后端工程，可以写成：

> 基于 Go 设计并实现 AI Agent 后端服务，封装 LLM Client、Tool Registry、Agent Loop、RAG Retriever 和 MCP Server，支持多工具调用、错误重试、超时控制、权限校验、人工确认和可观测日志。

更准确的项目定位可以写成：

> 使用 Go 从零实现两个 AI Agent 工程化作品集项目，验证 tool calling、RAG、MCP、Agent Loop、安全控制和可观测性，并通过 eval 数据集和 trace 日志评估效果。

## 常见面试问题准备

### RAG

- RAG 的完整流程是什么？
- chunk 大小怎么选？
- topK 怎么调？
- 如何避免模型胡编？
- metadata filter 有什么作用？
- 如何评估 RAG 效果？

### Tool Calling

- tool calling 和普通 API 调用有什么区别？
- 工具参数为什么需要 JSON Schema？
- 模型传错参数怎么办？
- 工具调用失败怎么处理？
- 如何避免无限调用工具？

### Agent Loop

- Agent 和普通 Chatbot 的区别是什么？
- Agent 的停止条件有哪些？
- 如何设计 memory？
- 如何调试 Agent？
- 什么场景不应该用 Agent？

### MCP

- MCP 解决什么问题？
- MCP Server 和普通 HTTP API 有什么区别？
- 为什么 Go 适合写 MCP Server？
- 公司内部系统如何接入 MCP？

### 生产化

- 如何做权限控制？
- 如何处理 prompt injection？
- 如何控制成本？
- 如何做日志和 trace？
- 写操作为什么需要人工确认？

### 底层知识

- token、context window、temperature、top_p 分别影响什么？
- embedding 为什么可以用于语义检索？
- cosine similarity / dot product 的基本含义是什么？
- chunk size 和 overlap 为什么会影响 RAG 效果？
- rerank 解决什么问题？
- tool calling 为什么不是“模型真的执行了工具”？
- Agent Loop 为什么需要状态机、最大步数和超时？
- eval 为什么是 Agent 项目的生命线？

## 需要补的底层知识边界

这条路线需要补的是“AI 应用工程师需要的底层”，不是一开始就钻大模型训练。

优先学习：

- token、context window、temperature、top_p、stop sequence。
- embedding、向量相似度、cosine similarity、dot product。
- chunk、topK、rerank、metadata filter、hybrid search。
- structured output、JSON Schema、tool calling、Agent Loop。
- prompt injection、权限隔离、人工确认、审计日志。
- eval、trace、latency、token cost。

暂时不必深入：

- 从零训练 Transformer。
- 反向传播公式推导。
- CUDA / 分布式训练。
- LoRA / 微调细节。
- 大规模模型推理服务优化。

除非目标从 Go 后端 / AI 应用工程转向算法工程师或大模型基础设施岗位。

## 每周复盘模板

每个周日建议写一次复盘：

```text
本周完成：

本周最重要的概念：

本周写出的代码：

遇到的问题：

下周要解决：

可以写进简历的点：
```

## 学习原则

1. 每天必须留下代码，哪怕只有 30 行。
2. 不要先追框架，先理解 tool calling、RAG、Agent Loop。
3. 所有工具调用都要做参数校验。
4. 所有 Agent 都要有最大步数和超时控制。
5. 所有 RAG 回答都要尽量带引用。
6. 周末必须整理 README，因为面试时项目表达很重要。
7. 每个项目都要有固定演示脚本，确保面试前可以稳定复现。
8. 最终目标不是“学过 AI Agent”，而是“用 Go 实现过可解释、可验证、可演示的 AI Agent”。
