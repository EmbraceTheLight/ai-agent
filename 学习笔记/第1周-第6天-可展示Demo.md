# 第 1 周第 6 天：做成本周可展示 Demo

日期：2026-07-28

## 今日结论

周六的核心任务已经完成。

今天的目标是把前 5 天的小功能整合成一个可展示的 CLI Demo。当前项目已经具备普通输出、streaming 输出、结构化 JSON 输出、`.env` 配置、makefile 快捷命令、README 文档和固定 demo 脚本。

测试用例计划放到周日收口时补充，这是合理安排。周六重点是让 Demo 能稳定展示本周学习成果。

## 今日完成

- 整理了 CLI 参数。
- 支持普通问答输出。
- 支持 `--stream` 流式输出。
- 支持 `--json` 结构化 JSON 输出。
- 支持 `--model` 指定模型。
- 支持 `--instruction` 指定 system instruction。
- 支持 `--timeout` 控制请求超时。
- 新增 `.env.example`。
- 接入 `.env` 本地配置，避免每次手动设置系统环境变量。
- 确认 `.env` 不应提交，只提交 `.env.example`。
- 新增 makefile，提供常用命令：
  - `make run`
  - `make stream`
  - `make json`
  - `make test`
- 新增英文 README。
- 新增中文 README。
- 新增中文 `docs/demo.md`，记录固定演示流程。

## 当前 Demo 能力

### 普通输出

通过：

```powershell
make run
```

或：

```powershell
go run ./cmd/assistant-cli
```

可以进入普通问答模式。模型会完整生成回答后一次性输出。

### 流式输出

通过：

```powershell
make stream
```

或：

```powershell
go run ./cmd/assistant-cli --stream
```

可以进入 streaming 模式。模型返回的增量文本会通过 `onDelta` 回调实时打印。

### 结构化 JSON 输出

通过：

```powershell
make json
```

或：

```powershell
go run ./cmd/assistant-cli --json
```

可以进入 JSON 输出模式。模型会按 JSON Schema 返回：

```json
{
  "intent": "agent_question",
  "answer": "...",
  "confidence": 0.86
}
```

程序会解析为 `IntentResult`，并校验 `intent` 与 `confidence`。

## 文档整理

今天补齐了项目展示所需的文档：

- `README.md`：英文说明，适合对外展示。
- `README_zh-CN.md`：中文说明，方便自己复盘和中文环境使用。
- `docs/demo.md`：中文 Demo 脚本，记录固定演示命令、输入示例和预期结果。

README 中记录了：

- 项目目标。
- 当前功能。
- 项目结构。
- 环境变量配置。
- 运行方式。
- make 命令。
- 第 1 周学习总结。
- 第 2 周 tool calling 方向。

Demo 文档中记录了：

- 普通输出演示。
- 流式输出演示。
- JSON 输出演示。
- 命令行问题输入演示。
- 自定义 system instruction 演示。
- timeout 演示。
- 测试入口。
- Windows 编码注意事项。

## 遇到的问题

### Windows 下 makefile 中文参数乱码

之前在 makefile 中直接写中文问题时，执行 `make stream` 可能出现乱码。

原因是 Windows 上 GNU Make、`cmd.exe`、PowerShell 和当前代码页之间可能存在编码不一致。makefile 是 UTF-8，但命令执行链路可能按 CP936 处理中文参数。

最终采用的方案是：

```text
makefile 只负责选择运行模式
中文问题通过 stdin 输入
```

这个方案更适合当前学习阶段，也能避免演示时被编码问题干扰。

## Demo 是否能稳定复现

当前 Demo 脚本已经固定下来，理论上可以稳定复现：

```powershell
make run
make stream
make json
make test
```

其中前三个命令需要 `.env` 中的 API 配置正确。

`make test` 已经指向：

```powershell
go test ./...
```

但当前还没有正式测试用例，测试计划放到周日完成。

## README 是否足够让别人跑起来

当前 README 已经包含运行项目所需的基本信息：

- Go 项目目标。
- 功能列表。
- `.env` 配置方式。
- 运行命令。
- 输出模式说明。
- make 命令。
- 第 1 周总结。
- 第 2 周计划方向。

对于学习项目而言，已经基本能让别人理解项目做了什么、如何配置、如何运行。

后续还可以继续补充：

- 示例输出截图或文本。
- 常见错误排查。
- 测试说明。
- 更清晰的周末 Demo 截图或录屏。

## 测试安排

测试用例计划放到周日任务中完成。

优先测试不依赖真实 LLM API 的部分：

- `GetContextWithTimeout`
- `IntentResult` JSON 解析
- intent 合法性校验
- confidence 范围校验
- 后续可加入 mock `llm.Client`

这样可以避免测试依赖网络、API Key 和模型输出稳定性。

## 今日收获

1. Demo 不只是代码能跑，还需要固定演示流程。
2. README 是项目对外表达能力的一部分。
3. `.env.example` 可以帮助别人理解需要哪些配置。
4. makefile 可以降低演示命令的记忆成本。
5. Windows 下中文命令行参数要注意编码链路。
6. 测试应该优先覆盖纯函数和稳定逻辑，不要一开始就测真实 LLM API。
7. 一个学习项目也应该逐步具备“可复现、可解释、可展示”的工程形态。

## 明天要改进

下一步进入“周日：复盘、补测试、准备下周 Tool Calling”。

重点任务：

- 跑 `go test ./...`。
- 补基础测试。
- 检查 README 是否还需要完善。
- 写第 1 周复盘。
- 预习第 2 周 tool calling。
- 思考工具参数为什么需要 JSON Schema，以及为什么 Go 侧必须做参数校验。

