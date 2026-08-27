package rag

import (
	"bufio"
	"fmt"
	"go-ai-agent/internal/utils"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type loader struct {
	allowExt map[string]bool // 允许的文件后缀. 若文件不是允许类型, 则跳过解析
	limit    int             //	解析文档数上限, 防止一次解析过多文档. limit <= 0 表示不限制
}

// NewTriliumDocumentLoader 创建 Trilium 导出文档加载器。
// 输入: `allowExt` 是允许加载的文件后缀集合, `limit` 是最多加载的文档数量。
// 输出: 返回实现 `DocumentLoader` 的 Trilium 文档加载器。
// 示例: `NewTriliumDocumentLoader(map[string]bool{".md": true}, 100)`。
func NewTriliumDocumentLoader(allowExt map[string]bool, limit int) DocumentLoader {
	if len(allowExt) == 0 {
		allowExt = map[string]bool{".md": true}
		log.Println("未设置文件过滤格式, 默认设置为 .md")
	}
	return &loader{
		allowExt: allowExt,
		limit:    limit,
	}
}

// Load 加载并解析指定路径下的 Trilium 导出文档。
// 输入: `path` 可以是单个文件路径, 也可以是包含允许后缀文件的目录路径。
// 输出: 返回解析后的文档列表; 文件收集、读取或解析失败时返回错误。
// 示例: `loader.Load("testdata/documents/work_notes")`。
func (l *loader) Load(path string) ([]*Document, error) {
	var documents []*Document
	filepathList, err := GetAllFile(path, l.allowExt)
	if err != nil {
		return nil, err
	}
	log.Printf("目录 %s, 文件后缀要求: %s\n收集到的文件共有 %d 个\n", path, strings.Join(GetAllAllowExt(l.allowExt), ","), len(filepathList))

	limit := len(filepathList)
	if l.limit > 0 && len(filepathList) >= l.limit {
		limit = l.limit
	}
	for i := 0; i < limit; i++ {
		filePath := filepathList[i]
		document, err := l.parseDocument(filePath)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, nil
}

// parseDocument 解析单个 Trilium 导出文档文件。
// 输入: `filepath` 是待解析的文件路径。
// 输出: 返回包含来源路径、标题和正文内容的 `Document`; 路径为目录或读取失败时返回错误。
// 示例: `l.parseDocument("notes/rag.md")`。
func (l *loader) parseDocument(filepath string) (*Document, error) {
	if utils.IsDir(filepath) == true {
		return nil, fmt.Errorf("%s 是目录, 不是文件", filepath)
	}
	doc := &Document{SourcePath: filepath}
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	doc.Content = string(fileContent)
	doc.Title = l.getDocTitle(filepath)
	return doc, nil
}

// getDocTitle 获取 Trilium 导出文档标题。
// 输入: `docPath` 是待读取标题的文档路径。
// 输出: 返回第一行 Markdown 标题; 若首行标题不存在则返回去掉后缀的文件名; 打开文件失败时返回空字符串。
// 示例: `l.getDocTitle("notes/rag.md")`。
func (l *loader) getDocTitle(docPath string) string {
	f, err := os.Open(docPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// 兜底: 移除后缀的文件名作为标题
	title := strings.TrimSuffix(filepath.Base(docPath), filepath.Ext(docPath))

	// 扫描第一行, trilium 导出的 md 文档第一行为标题
	scanner := bufio.NewScanner(f)
	if scanner.Scan() == true {
		firstLine := scanner.Text()
		tmp := strings.TrimLeft(strings.TrimSpace(firstLine), "#")
		title = strings.TrimSpace(tmp)
	}

	return title
}
