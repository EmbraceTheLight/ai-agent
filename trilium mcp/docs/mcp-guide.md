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

`TRILIUM_WRITE_ROOT_ID` **只限制写入**。MCP 读取能力与 CLI 相同，范围由 ETAPI Token 决定。配置只有一个目录时，`trilium_search` 默认以它为祖先；有多个目录时必须显式提供 `ancestorNoteId`，不因省略参数而扩大搜索范围。`trilium_read` 和 `trilium_list_children` 不限制到写入根。受保护正文、图片和文件下载不受支持。

不设置写入根时仅提供四个只读工具；配置了无效写入根时启动报错。配置有效写入根后还会注册创建、追加、更新和删除工具。更改 `.env` 后重启 MCP 进程才能生效。

## 多目录与笔记锁（0.3.0）

原有单 ID 写法继续有效，也可以使用 JSON 数组：

```dotenv
TRILIUM_WRITE_ROOT_ID='["directoryA","directoryB","directoryC"]'
```

程序只排序、去重，不合并嵌套关系。即使 B、C 位于 A 之下，三项仍全部保留。目录列表仅控制允许的写入范围，不用于选择锁；数组必须包含 1–64 个合法 noteId，仍禁止配置全库根 `root`。

`trilium_info` 返回 `writeRootIds`。兼容字段 `configuredWriteRootIds` 与它相同；只有去重后仅剩一个目录时，旧字段 `writeRootId` 才有值，否则为空。info 不再遍历目录祖先或验证配置 ID 是否存在；实际写入取得锁后，会检查目标的保护状态、类型和全部父级路径，写入前再次核对正文和权限。读取范围仍由 ETAPI Token 决定。

只有一个目录时，新建默认以它为父级；多个目录（包括嵌套目录）时，MCP 必须显式提供 `parentNoteId`，CLI 必须提供 `-parent`。多目录 MCP 搜索必须提供 `ancestorNoteId`。所有显式配置的根笔记自身都禁止修改正文和删除；若希望编辑 B 本身，可仅配置其祖先 A。

| 操作 | 笔记锁对象 |
| --- | --- |
| 更新、追加笔记 X | X 的 noteId |
| 删除叶子笔记 X | X 的 noteId |
| 在 P 下新建子笔记 | P 的 noteId |

每次操作先取得 operationId 锁，再取得上表对应的笔记锁；同一状态目录的操作 ID 必须全局唯一。不同笔记即使在同一目录也可以并行，同一笔记的竞争会等待；同一父笔记下的新建也会串行。克隆笔记共享同一个 noteId，因此自然互斥，无需锁多个目录。范围外克隆路径仍拒绝，删除仍拒绝克隆和非叶子笔记。

创建子笔记时锁父笔记，确保它与删除该父笔记竞争同一把锁：先创建成功则父笔记不再是叶子，删除被拒绝；先删除成功则后来的创建发现父笔记不存在，不会继续发送创建请求。锁等待可以取消，MCP 工具调用最长 60 秒，CLI 可按 Ctrl+C 取消。

文件锁位于状态目录的 `.locks` 下，支持跨进程互斥，崩溃遗留锁不自动清除。协作进程必须共享同一个状态目录；请保持同一套写入配置，避免操作记录指纹不一致。升级前先完成待核对操作并停止旧写入进程，**不要混用 0.1.0 全局锁、0.2.0 目录锁和 0.3.0 笔记锁的写入程序**。旧 `.lock` 文件会阻止新写入；旧目录锁与新笔记锁之间没有互斥保证。

单目录历史记录继续兼容；数组排序和重复项不影响指纹。0.2.0 曾将嵌套配置合并，因此保留同一嵌套数组升级到 0.3.0 后，指纹可能变化，已配置的后代边界也重新受到根节点保护。不要通过删除操作记录或盲目换操作 ID 绕过未知结果，先在旧版本及原配置下完成核对。

这些锁不提供会话权限隔离，也不能约束人工编辑、其他状态目录或不使用此协议的客户端。ETAPI 的检查与写入仍不是原子操作，其他客户端仍可能在最后检查后改动笔记。

本次另外构建 `bin/trilium-mcp-note-locks.exe` 和 `bin/trilium-cli-note-locks.exe`，避免替换运行中的程序。停止旧写入后，将 MCP 启动命令切换到新程序并重启；也可从更新后的源码重新构建原文件名。真实 `.env` 和 Codex 配置不会自动改写。

验证记录见 [笔记锁验证](note-lock-verification.md)。0.2.0 的记录保留在 `multi-root-verification.md`，其中的目录合并和目录锁描述属于历史行为。

## 注册到 Codex

在 PowerShell 中执行一次：

```powershell
codex mcp add trilium -- 'D:\code\ai-agent\trilium mcp\bin\trilium-mcp-note-locks.exe' -env-file 'D:\code\ai-agent\trilium mcp\.env'
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
| `trilium_delete` | 快照后软删除叶子笔记 | `operationId`、`noteId`、`expectedSHA256` |

新建和追加的 `format` 默认 `markdown`；更新默认 `html`，也支持显式指定 `markdown` 或 `text`。内容通过字符串参数传入，不提供让 AI 读取任意本机文件的工具。单次输入上限 4 MiB，读取正文上限 8 MiB；每个 HTTP 请求超时 10 秒，每次工具调用最长 60 秒。

列子笔记时按父笔记当次返回的 childNoteIds 分页，省略受保护子笔记；分页过程中树发生变化可能导致重复或遗漏，不能视为一致快照。搜索结果也会省略受保护笔记。

可以向 Codex 发送：

> 使用 trilium MCP，先确认写入根是根笔记下的 temp，再在 temp 下创建一篇「Go 学习记录」，内容是我今天学会了 MCP 工具的发现和调用。完成后读取核对。

随后：

> 读取刚才创建的 Go 学习记录，在末尾追加一段关于 MCP Server 和 ETAPI 的区别。保留原有内容，使用最新正文摘要，完成后读取核对。

## 写入约束与恢复

写入沿用子树检查、跨边界克隆拒绝、修改前完整快照、SHA-256 冲突检查和写后核对。不能编辑或删除写入根自身。删除仅支持未受保护、单父级、没有子笔记的 `text` 或 `code` 笔记；克隆笔记和非叶子笔记会被拒绝。

ETAPI 的 `DELETE /etapi/notes/{noteId}` 删除该 note ID 的所有分支；如果笔记有子分支，服务端可能连带将其子树移入删除状态。工具因此只开放叶子笔记。删除是可恢复软删除，不会清除删除状态或附件数据。

先 `trilium_read`，把返回的 `contentSHA256` 作为 `expectedSHA256`。每个新操作选择独立 `operationId`；若发生超时或连接中断，使用**相同操作 ID、相同参数、同一状态目录**重试。不要为同一次未知结果生成新 ID，否则无法识别重复意图。旧摘要冲突时重新读取，重新判断改动，并为新的意图使用新 ID。

删除同样需要最新的正文摘要。服务会保存元数据和正文快照，删除后确认目标不可读取；响应丢失后使用同一操作 ID 重试。快照用于留存和人工恢复参考，不是自动还原功能。Trilium 提供 `POST /etapi/notes/{noteId}/undelete` 恢复接口，恢复要求存在未删除的父级。

CLI 和 MCP 默认共享 `.env` 所在目录下的 `.trilium-state`。该目录保存私有原文和操作记录，不要提交或随意删除。相同 ID 的已完成操作返回历史结果，`alreadyApplied=true`；它不代表笔记此后没有被其他人修改，需要当前内容时再次读取。

整篇更新会替换完整正文；局部修改优先提供唯一原始 HTML `match`，保留其他链接与附件引用。ETAPI 的检查和 PUT/DELETE 不是原子操作，同时人工编辑或树结构变化仍可能产生竞争；错误与恢复细节见 `write-guide.md`。

## 排查与学习

- MCP 工具列表为空：检查配置路径，确认可执行文件存在，并重启服务或客户端。不要用 `go run` 作为日常启动命令。
- 启动后终端没有提示：stdio 服务正在等协议消息，属于正常现象。用客户端连接；单独运行只适合 `-h` 查看帮助。
- 工具返回连接失败：保持 Trilium 运行，使用 CLI `info -env-file` 检查 URL 与 Token；无需重新注册 MCP。
- 工具返回 `isError=true`：业务拒绝会保留 MCP 会话；读错误说明后处理冲突或配置，不要盲目重复写入。

阅读顺序：`cmd/trilium-mcp/main.go`（加载配置和 stdio）→ `internal/mcpserver/server.go`（类型、schema、工具注册）→ `internal/notes`（普通 Go 业务方法）。`server_test.go` 使用官方 SDK 客户端经内存传输调用工具；`cmd/trilium-mcp/main_test.go` 另用子进程验证 stdio 与模拟 ETAPI 的完整链路。

SDK 参考：[官方快速入门](https://go.sdk.modelcontextprotocol.io/quick_start/)、[固定版本 go.mod](https://github.com/modelcontextprotocol/go-sdk/blob/v1.7.0/go.mod)。
