# Trilium MCP（Go 学习项目）

以 TriliumNext 0.104.1 为初始目标，通过官方 ETAPI 操作笔记，再把这些能力提供给 AI Agent。

当前已提供 ETAPI 客户端、CLI 和 stdio MCP 服务，支持指定子树内的新建、追加和正文更新。已与本机 TriliumNext 0.104.1 完成真实 MCP 读写验证；Codex 桌面端接入见 [MCP 使用说明](docs/mcp-guide.md)，功能边界及验证范围见 `docs/write-guide.md`、`docs/verification.md`。

## 当前能力

| 命令 | 功能 | ETAPI 请求 |
| --- | --- | --- |
| `info` | 读取实例版本 | `GET /etapi/app-info` |
| `search` | 按 Trilium 查询语法搜索，支持指定子树和数量 | `GET /etapi/notes` |
| `read` | 读取笔记元数据及正文 | `GET /etapi/notes/{id}` 和 `GET /etapi/notes/{id}/content` |
| `create` | 在指定子树中新建富文本笔记 | `POST /etapi/create-note`，随后读取核对 |
| `append` | 备份后追加内容并核对结果 | `PUT /etapi/notes/{id}/content`，随后读取核对 |
| `update` | 备份后替换整篇正文或唯一 HTML 片段 | `PUT /etapi/notes/{id}/content`，随后读取核对 |

- `info`、`search`、`read` 仅发送 GET；`create`、`append`、`update` 必须配置写入根，不能修改范围根本身的正文。当前不提供删除、重命名和移动。
- `read` 支持 UTF-8 的 `text`、`code` 笔记；富文本保留原始 HTML，不转换成 Markdown。
- 不读取受保护笔记、图片或文件笔记，不下载附件。
- 请求超时为 10 秒，单个响应上限为 8 MiB；超过上限会报错，不返回截断的正文。
- Token 从环境变量或 `.env` 读取；非空环境变量优先，错误信息不输出 Token 或服务器响应正文。
- 不跟随 HTTP 重定向；如果遇到 301/302，请配置最终实例地址。
- 未知 JSON 字段会被忽略，但必要字段缺失会报错；这只能容忍部分 API 变化，不能保证未来版本均兼容。
- `TRILIUM_WRITE_ROOT_ID` 限制写入范围；读取和搜索保持原有能力，`search -ancestor` 只是本次搜索条件。

## 环境

Go 1.25 或更新版本。固定使用官方 `github.com/modelcontextprotocol/go-sdk v1.7.0` 和 Markdown 库 `github.com/yuin/goldmark v1.7.13`，首次源码构建需下载依赖；使用已编译的 CLI 或 MCP 服务不需要 Go 或联网下载依赖。

在 PowerShell 中进入项目目录：

```powershell
Set-Location -LiteralPath 'D:\code\trilium mcp'
go version
```

如果终端找不到 `go`，或 `GOROOT` 指向旧目录，本机已发现可用安装，可以只在当前 PowerShell 会话中设置：

```powershell
$env:GOROOT = 'D:\Environments\golang\go1.26.8'
$env:Path = "$env:GOROOT\bin;$env:Path"
go version
```

上述操作不会永久修改系统环境变量。其他机器请替换为自己的 Go 安装位置。

## 配置 Trilium

在 Trilium 的选项中找到 ETAPI，生成一个 Token。项目根目录已提供空白 `.env`，填写实际地址和 Token 即可。`.env` 已被 `.gitignore` 忽略；可提交的 `.env.example` 只保留空值示例。

```dotenv
# 地址仅为本机桌面实例示例，端口以实际实例为准。
TRILIUM_URL=http://127.0.0.1:37840
TRILIUM_TOKEN=填入实际ETAPIToken
# 写入功能需要显式设置；本项目已通过 API 找到的 root/temp：
TRILIUM_WRITE_ROOT_ID=v2YidQTHkFIY
```

使用本地地址时，CLI 访问的是本地桌面实例，需要保持应用运行。本地同步完成的笔记可以直接读取；通过本地接口写入的更改由 Trilium 同步到远程服务器。本机测试不需要为了 ETAPI 开启局域网访问。

Token 由被访问实例校验，不绑定创建时的 URL。本地创建的 Token 可先配合本地地址使用；如改用远程地址，需要确认远端也具有对应 Token 数据，以认证结果为准。

加载规则：

- 默认读取**当前工作目录**下的 `.env`，不是自动读取可执行文件旁的文件。建议从项目根目录启动。
- 可用 `info -env-file 'D:\code\trilium mcp\.env'` 等命令指定路径，文件名也可以不同。
- 默认 `.env` 不存在时仍支持仅用环境变量运行；其他显式指定的文件不存在时会报错。
- 非空环境变量覆盖文件值。之前若设置过旧配置，可在当前 PowerShell 中清除后再使用文件：

```powershell
Remove-Item Env:\TRILIUM_URL -ErrorAction SilentlyContinue
Remove-Item Env:\TRILIUM_TOKEN -ErrorAction SilentlyContinue
```

文件请使用 UTF-8 保存（支持 BOM 和 Windows 换行）。格式为 `KEY=VALUE`，支持空行、井号注释、单行单引号或双引号；重复键会报错。未加引号时，空格或制表符之后的井号开始行尾注释。引号内的值及 Token 中间的 `=`、`#`、`$` 会按字面值保留。不支持 `export`、转义、多行值、变量替换或命令执行。

也可以继续仅使用环境变量：

```powershell
# 地址仅为示例，请改成自己的实例地址和端口。
$env:TRILIUM_URL = 'http://127.0.0.1:8080'

# 隐藏输入，不将 Token 明文写入命令历史。
$env:TRILIUM_TOKEN = [System.Net.NetworkCredential]::new('', (Read-Host 'ETAPI Token' -AsSecureString)).Password
```

`TRILIUM_URL` 支持实例地址、反向代理路径前缀，以及已经带 `/etapi` 的地址。例如 `https://notes.example.com/trilium` 或 `https://notes.example.com/trilium/etapi`。远程实例应使用 HTTPS。

程序不会使用 Trilium 登录密码获取 Token，也不会把文件配置写入系统环境变量。

## 运行

```powershell
# 离线查看帮助，无需 Token。
go run ./cmd/trilium-cli --help

# 先检查地址和 Token 是否可用。
go run ./cmd/trilium-cli info

# 从其他工作目录运行时，可以显式指定配置文件。
go run ./cmd/trilium-cli info -env-file 'D:\code\trilium mcp\.env'

# 找到目标笔记的 noteId；查询字符串作为一个参数传入。
go run ./cmd/trilium-cli search -limit 10 'Go'
go run ./cmd/trilium-cli search -ancestor root -limit 5 'Go #topic'

# 将 note123 替换为搜索结果中的实际 noteId。
go run ./cmd/trilium-cli read note123
go run ./cmd/trilium-cli read -raw note123
```

选项必须写在查询字符串或笔记 ID 前面。默认输出格式化 JSON；`-raw` 仅输出原始正文，不增加换行、不执行其中的 HTML。

`read` 的元数据和正文由两个请求获取，当前阶段不构成同一时刻的一致快照，也不能作为未来写入时的原子冲突检查。

`read` 的 JSON 结果包含 `contentSHA256`，用于追加和更新前确认正文版本。新建、追加、更新、备份和重试的完整示例见 [写入说明](docs/write-guide.md)。

常见错误：

| 现象 | 检查方向 |
| --- | --- |
| HTTP 401 / 403 | Token 是否有效、ETAPI 是否可用 |
| HTTP 404 | 实例路径、笔记 ID 和实际版本支持的接口 |
| HTTP 301 / 302 | 使用最终实例地址，检查反向代理路径 |
| invalid JSON / no appVersion | 地址可能指向登录页，或 API 返回结构发生变化 |
| connection failed | 实例是否运行、端口、TLS 证书和网络连接 |
| request timed out | 请求超过 10 秒 |

## 测试与构建

```powershell
go test ./... -count=1 -cover
go vet ./...
New-Item -ItemType Directory -Path bin -Force | Out-Null
go build -o ./bin/trilium-cli.exe ./cmd/trilium-cli
go build -o ./bin/trilium-mcp.exe ./cmd/trilium-mcp
```

自动化测试使用假 Token、临时配置和模拟笔记树，不读取真实 `.env`，不依赖真实笔记库。除只读和配置场景外，还覆盖越界与跨范围克隆拒绝、原文备份、并发编辑检测、片段匹配缺失或歧义、保留其他富文本内容、写后核对及响应丢失后的恢复。

编译后使用 `./bin/trilium-cli.exe`，不需要再次运行 `go run`。

## 代码阅读顺序

1. `cmd/trilium-cli/main.go` 的 `run`：命令参数如何变成普通 Go 方法调用。
2. `internal/trilium/client.go` 的 `NewClient`：如何固定 URL、认证与超时。
3. `GetAppInfo` → `getJSON` → `get`：一次 HTTP 请求、JSON 解码与错误返回的完整路径。
4. `GetNote` 和 `GetNoteContent`：为什么笔记元数据和正文需要分别读取。
5. 测试中的 `httptest.NewServer`：如何不依赖真实服务验证客户端行为。

后续开发顺序和需要掌握的内容见 `docs/learning-plan.md`。

## 参考

- [Trilium ETAPI 使用与认证](https://docs.triliumnotes.org/user-guide/advanced-usage/etapi)
- [Trilium ETAPI 接口参考](https://docs.triliumnotes.org/rest-api/etapi/)
- [桌面实例的本机与局域网访问](https://docs.triliumnotes.org/user-guide/setup/desktop/network-access)
- [Trilium 同步机制](https://docs.triliumnotes.org/user-guide/setup/synchronization)
- [MCP 官方 Go SDK](https://go.sdk.modelcontextprotocol.io/)
