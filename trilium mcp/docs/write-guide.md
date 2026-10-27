# 新建、追加、更新与删除

## 当前边界

已通过 API 定位到根笔记下唯一的 `temp`：`v2YidQTHkFIY`，父级为 `root`，未受保护。真实写入测试固定在该目录下的新建测试笔记中。没有定位到此目录或归属不符时，不执行真实写入。

当前支持新建富文本笔记，以及对后代 `text/html` 笔记追加、整篇替换或精确替换片段。删除只支持未受保护、单父级且没有子笔记的 `text` 或 `code` 笔记；不能删除范围根、克隆笔记或非叶子笔记。删除通过 ETAPI 移入 Trilium 的可恢复删除状态，不执行永久清除。

## 配置

在 `.env` 中设置：

```dotenv
TRILIUM_WRITE_ROOT_ID=v2YidQTHkFIY
# 可选；默认在所选 .env 文件旁创建 .trilium-state。
# TRILIUM_STATE_DIR=D:/code/trilium mcp/.trilium-state
```

写入范围由笔记 ID 确定，不根据每次调用时的标题猜测。所有父级路径必须落在该子树内；跨范围克隆、循环、受保护笔记及无法确认的范围会被拒绝。低层 ETAPI 客户端本身不执行范围限制，CLI 通过 `internal/notes` 执行这些检查。

## 新建

将待写内容保存为 UTF-8 的 `create.md`，再运行：

```powershell
Set-Location -LiteralPath 'D:\code\trilium mcp'
.\bin\trilium-cli.exe create -title '接口测试' -file '.\create.md' -operation 'my-create-001'
```

省略 `-parent` 时使用写入范围根，也可以指定范围内的其他父笔记。结果只输出 `operationId`、`noteId`、`contentSHA256` 和 `alreadyApplied`。

`-format` 可选 `markdown`（默认）、`text` 或 `html`。Markdown 使用 Goldmark 默认安全策略，不启用原始 HTML；需要原样写入 HTML 时显式使用 `-format html`。普通文本会转义 HTML 特殊字符。

## 追加

先读取目标笔记，取得当前摘要，再用独立的内容文件追加：

```powershell
$noteId = '替换为新建结果的noteId'
$current = .\bin\trilium-cli.exe read $noteId | ConvertFrom-Json
.\bin\trilium-cli.exe append -file '.\append.md' -operation 'my-append-001' -expected-sha256 $current.contentSHA256 $noteId
```

原有 HTML 保持原样，末尾插入一个换行及新片段。输入文件最多 4 MiB，组合后正文最多 8 MiB。

## 更新正文

更新命令默认 `-format html`，以便直接使用读取到的原始 HTML。必须先读取当前正文并取得摘要；标题、父级和笔记类型不属于此命令的修改范围。

整篇替换：将完整目标正文保存到 `new.html`，文件未包含的旧内容会被替换掉。

```powershell
$noteId = '替换为temp下的目标笔记ID'
$current = .\bin\trilium-cli.exe read $noteId | ConvertFrom-Json
.\bin\trilium-cli.exe update -file '.\new.html' -operation 'my-update-001' -expected-sha256 $current.contentSHA256 $noteId
```

需要用 Markdown 生成整篇新正文时，显式加入 `-format markdown`；也可用 `-format text` 写入转义后的普通文本。

仅更新一处段落：把要替换的原始 HTML 片段保存为 `old-fragment.html`，新片段放入 `new-fragment.html`。

```powershell
$current = .\bin\trilium-cli.exe read $noteId | ConvertFrom-Json
.\bin\trilium-cli.exe update -match-file '.\old-fragment.html' -file '.\new-fragment.html' -operation 'my-update-002' -expected-sha256 $current.contentSHA256 $noteId
```

- 匹配片段必须与原始 HTML 完全一致，包括空白和换行；匹配文件不经过 Markdown 转换。
- 只能匹配一处，缺失、多处或重叠匹配均停止；不使用正则、模糊匹配或整篇 HTML 自动格式化。
- 唯一匹配区间之外的原文逐字节保留，包括内部链接、图片引用和列表。
- 显式提供的空匹配文件会报错，不会回退到整篇替换。新内容不能为空，暂不支持用空文件清空或删除片段。
- 摘要以 `read` 返回的值为准；不要对正文归一化换行后重新计算。摘要不匹配时，不执行更新。
- 若新旧正文完全相同，不发送 PUT，结果中的 `alreadyApplied` 为 true。

更新与追加共用版本检查、备份、操作 ID 和写后核对流程；重试时仍使用原来的命令参数和摘要。

## 删除叶子笔记

删除前先读取目标笔记，确认目标 ID、父级和子笔记列表，并使用返回的 `contentSHA256`。删除按 note ID 生效；Trilium ETAPI 会删除该笔记的所有分支，因此本实现拒绝存在多个父级的克隆笔记。目标必须在写入范围内、未受保护、类型为 `text` 或 `code`，并且没有子笔记。

```powershell
$noteId = '替换为范围内的叶子笔记 ID'
$current = .\bin\trilium-cli.exe read $noteId | ConvertFrom-Json
.\bin\trilium-cli.exe delete -operation 'my-delete-001' -expected-sha256 $current.contentSHA256 $noteId
```

删除前会将笔记元数据和完整正文保存到状态目录，再检查摘要、范围和叶子状态后发送 `DELETE /etapi/notes/{noteId}`。成功后会再次读取确认目标已不可访问。ETAPI 将笔记软删除；需要恢复时可在 Trilium 中恢复，或调用 `POST /etapi/notes/{noteId}/undelete`，但该接口要求至少有一个未删除的父级。

若请求结果不确定，使用相同的操作 ID、笔记 ID 和摘要重试。目标已不存在时会完成操作记录；目标仍存在时，只有快照元数据和正文仍一致才会继续删除。若摘要或树结构已改变，服务停止并保留待处理记录。

## 备份与重试

- 状态目录中的 `<操作ID>.json` 在 API 写入前创建并同步到磁盘。追加和更新记录包含修改前元数据及完整原文，可用于人工恢复。
- `<操作ID>.done.json` 表示已读取核对成功。相同操作再次执行时返回历史结果，不再次写入；`alreadyApplied` 为 true。
- 删除操作同样先保存完整正文快照；重试会核对目标是否已删除，或其元数据和正文是否仍与快照相同。
- 新建操作预先分配固定笔记 ID。响应不明确时，重试先查询该 ID；不会因为连接错误直接换一个 ID 重新创建。
- 追加或更新结果不明确时，以记录的修改前和修改后摘要核对：已写入则确认完成，仍为原文才尝试写入，其他内容一律报冲突。片段更新恢复时，从保存的原文构造目标，不会因当前正文已没有旧片段而再次替换。
- 重试必须保持相同操作 ID、输入文件内容、实例 URL、写入范围及状态目录。新的操作使用新的 ID，不要为了消除报错盲目更换 ID。
- 已完成结果描述该操作被核对时的内容；笔记之后发生变化时，历史结果不代表当前正文。需要当前状态时重新执行 `read`。
- `.trilium-state` 已被 Git 忽略，其中包含私有原文；自定义位置也应妥善保存，不应提交到仓库。不要在结果未知时删除操作记录。
- 同一状态目录的写入用独占锁串行化。进程崩溃可能遗留 `.lock`；先确认没有写操作运行，再人工处理，不自动抢锁。

摘要检查、范围检查与 PUT/DELETE 并非一个服务器事务；本实现不能消除最后检查与请求之间的并发窗口。尤其是其他客户端若在检查后为目标增加子笔记，ETAPI 的笔记删除可能连同新增子树一起移入删除状态。发现冲突时保留当前内容并停止，不自动回滚。
