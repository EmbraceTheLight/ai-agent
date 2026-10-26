# MCP 与 Codex 桌面端

`bin/trilium-mcp.exe` 是本机 stdio MCP 服务。Codex 启动它并通过标准输入输出调用工具，它再调用 Trilium ETAPI；不需要自己启动 HTTP 服务，也不需要额外的模型 API Key。

服务使用官方 Go MCP SDK v1.7.0。SDK 负责协议握手、工具发现、JSON Schema 参数校验、结构化结果及工具错误映射。业务层复用已有 `internal/notes`；CLI 仍可独立使用。

## 本机配置

现有 `.env` 可直接使用，无需复制 Token 到 Codex 配置。使用本地 URL 时保持 Trilium 桌面实例运行。当前已确认的写入范围为 `root/temp`，ID 为 `v2YidQTHkFIY`。

```dotenv
TRILIUM_URL=http://127.0.0.1:37840
TRILIUM_TOKEN=你的ETAPIToken
TRILIUM_WRITE_ROOT_ID=v2YidQTHkFIY
```

这些值只是格式示例；真实 Token 继续保留在被忽略的 `.env`。非空环境变量仍优先于文件；若 Codex 进程继承过旧的 `TRILIUM_*` 环境变量，应清理旧值后重启。

`TRILIUM_WRITE_ROOT_ID` **只限制写入**。MCP 读取能力与 CLI 相同，范围由 ETAPI Token 决定；`trilium_search` 默认以写入根为祖先，但可显式指定其他祖先，`trilium_read` 和 `trilium_list_children` 不限制到写入根。受保护正文、图片和文件下载不受支持。若未来需要子树级读取隔离，需要增加独立的读取检查。

不设置写入根时仅提供四个只读工具；配置了无效写入根时启动报错。更改 `.env` 后重启 MCP 进程才能生效。

## 注册到 Codex

在 PowerShell 中执行一次：

```powershell
codex mcp add trilium -- 'D:\code\trilium mcp\bin\trilium-mcp.exe' -env-file 'D:\code\trilium mcp\.env'
codex mcp get trilium
```

本机服务名为 `trilium`。已注册的机器无需重复添加；换机器时执行上述命令。路径作为单个参数传递，包含空格时也能启动。

对应配置位于 `%USERPROFILE%\.codex\config.toml`，只包含可执行文件和 `.env` 的绝对路径。可提交的独立片段见 `examples/codex-mcp.toml`；合并该片段时保留其他配置，不要覆盖整个文件，也不要重复创建同名表。

在 Codex 的 MCP 设置中查看 `trilium` 并重启该服务；若当前会话没有出现新工具，重启桌面应用后开启新会话。注册配置成功与当前会话实际加载工具是两项验证，服务不会自行重启 Codex。

官方设置说明：[Codex MCP](https://developers.openai.com/codex/mcp)。其他支持 stdio 的客户端可参考 `examples/mcp-stdio.json`，具体配置文件位置由客户端决定。

## 工具

| 工具 | 用途 | 关键参数 |
| --- | --- | --- |
| `trilium_info` | 实例版本、写入根和读取范围说明 | 无 |
| `trilium_search` | 原生语法搜索元数据 | `query`、可选 `ancestorNoteId`、`limit`（默认 20，最大 100） |
| `trilium_read` | 元数据、原始正文、`contentSHA256` | `noteId` |
| `trilium_list_children` | 分页列出直接子笔记 | `noteId`、`offset`、`limit`；`nextOffset=-1` 表示结束 |
| `trilium_create` | 在写入范围内新建富文本笔记 | `operationId`、`title`、`content`；父级默认写入根 |
| `trilium_append` | 备份后追加 | `operationId`、`noteId`、`content`、`expectedSHA256` |
| `trilium_update` | 整篇或唯一片段更新 | 追加同类参数；可选原始 HTML `match` |

新建和追加的 `format` 默认 `markdown`；更新默认 `html`，也支持显式指定 `markdown` 或 `text`。内容通过字符串参数传入，不提供让 AI 读取任意本机文件的工具。单次输入上限 4 MiB，读取正文上限 8 MiB；每个 HTTP 请求超时 10 秒，每次工具调用最长 60 秒。

列子笔记时按父笔记当次返回的 childNoteIds 分页，省略受保护子笔记；分页过程中树发生变化可能导致重复或遗漏，不能视为一致快照。搜索结果也会省略受保护笔记。

可以向 Codex 发送：

> 使用 trilium MCP，先确认写入根是根笔记下的 temp，再在 temp 下创建一篇「Go 学习记录」，内容是我今天学会了 MCP 工具的发现和调用。完成后读取核对。

随后：

> 读取刚才创建的 Go 学习记录，在末尾追加一段关于 MCP Server 和 ETAPI 的区别。保留原有内容，使用最新正文摘要，完成后读取核对。

## 写入约束与恢复

写入沿用子树检查、跨边界克隆拒绝、修改前完整快照、SHA-256 冲突检查和写后核对。不能编辑写入根自身正文，不提供删除、移动或重命名。

先 `trilium_read`，把返回的 `contentSHA256` 作为 `expectedSHA256`。每个新操作选择独立 `operationId`；若发生超时或连接中断，使用**相同操作 ID、相同参数、同一状态目录**重试。不要为同一次未知结果生成新 ID，否则无法识别重复意图。旧摘要冲突时重新读取，重新判断改动，并为新的意图使用新 ID。

CLI 和 MCP 默认共享 `.env` 所在目录下的 `.trilium-state`。该目录保存私有原文和操作记录，不要提交或随意删除。相同 ID 的已完成操作返回历史结果，`alreadyApplied=true`；它不代表笔记此后没有被其他人修改，需要当前内容时再次读取。

整篇更新会替换完整正文；局部修改优先提供唯一原始 HTML `match`，保留其他链接与附件引用。ETAPI 的检查和 PUT 不是原子操作，同时人工编辑仍可能产生竞争；错误与恢复细节见 `write-guide.md`。

## 排查与学习

- MCP 工具列表为空：检查配置路径，确认可执行文件存在，并重启服务或客户端。不要用 `go run` 作为日常启动命令。
- 启动后终端没有提示：stdio 服务正在等协议消息，属于正常现象。用客户端连接；单独运行只适合 `-h` 查看帮助。
- 工具返回连接失败：保持 Trilium 运行，使用 CLI `info -env-file` 检查 URL 与 Token；无需重新注册 MCP。
- 工具返回 `isError=true`：业务拒绝会保留 MCP 会话；读错误说明后处理冲突或配置，不要盲目重复写入。

阅读顺序：`cmd/trilium-mcp/main.go`（加载配置和 stdio）→ `internal/mcpserver/server.go`（类型、schema、工具注册）→ `internal/notes`（普通 Go 业务方法）。`server_test.go` 使用官方 SDK 客户端经内存传输调用工具；`cmd/trilium-mcp/main_test.go` 另用子进程验证 stdio 与模拟 ETAPI 的完整链路。

SDK 参考：[官方快速入门](https://go.sdk.modelcontextprotocol.io/quick_start/)、[固定版本 go.mod](https://github.com/modelcontextprotocol/go-sdk/blob/v1.7.0/go.mod)。
