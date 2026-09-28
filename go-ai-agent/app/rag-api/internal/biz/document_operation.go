package biz

// DocumentLoader 定义文档加载接口。
// 输入: 文件或目录路径。
// 输出: 解析后的文档列表。
// 示例: `loader.Load("testdata/documents")`。
type DocumentLoader interface {
	// Load 解析并返回 path 下所有符合条件的文档。
	// 输入: `path` 是待加载的文件或目录路径。
	// 输出: 返回解析后的文档列表; 读取、过滤或解析失败时返回错误。
	// 示例: `Load("testdata/documents")`。
	Load(path string) ([]*Document, error)
}

// Document 表示从本地文件加载得到的一篇文档。
// 输入: `SourcePath` 是源文件路径, `Content` 是完整文本内容。
// 输出: 供 `Split` 切分为多个 `Chunk`。
// 示例: `Document{SourcePath: "notes/rag.md", Content: "# RAG"}`。
type Document struct {
	Title      string
	SourcePath string
	Content    string
}

// ChunkConfig 保存文档切分所需的参数。
// 输入: `Size` 是单个 chunk 的最大 rune 数, `Overlap` 是相邻 chunk 重叠的 rune 数。
// 输出: 供 `Document.Split` 执行固定大小切分。
// 示例: `ChunkConfig{Size: 500, Overlap: 100}`。
type ChunkConfig struct {
	Size    int
	Overlap int
}
