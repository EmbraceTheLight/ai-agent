package data

import (
	"bufio"
	"errors"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/conf"
	"go-ai-agent/internal/utils"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type loader struct {
	allowExt map[string]bool // 允许的文件后缀. 若文件不是允许类型, 则跳过解析
	limit    int             // 解析文档数上限, 防止一次解析过多文档. limit <= 0 表示不限制
}

// NewDocumentLoader 根据应用配置创建 Trilium 文档加载器。
// 输入: `c` 是应用数据配置, 包含允许的文件后缀和文档数量上限。
// 输出: 返回实现 `biz.DocumentLoader` 的文档加载器。
// 示例: `NewDocumentLoader(confData)`。
func NewDocumentLoader(c *conf.Data) biz.DocumentLoader {
	allowExt := map[string]bool{".md": true}
	limit := 0
	if c != nil && c.Rag != nil {
		if len(c.Rag.AllowExtensions) > 0 {
			allowExt = make(map[string]bool, len(c.Rag.AllowExtensions))
			for _, ext := range c.Rag.AllowExtensions {
				allowExt[strings.ToLower(strings.TrimSpace(ext))] = true
			}
		}
		limit = int(c.Rag.LimitDocs)
	}
	return NewTriliumDocumentLoader(allowExt, limit)
}

// NewTriliumDocumentLoader 创建 Trilium 导出文档加载器。
// 输入: `allowExt` 是允许加载的文件后缀集合, `limit` 是最多加载的文档数量。
// 输出: 返回实现 `biz.DocumentLoader` 的 Trilium 文档加载器。
// 示例: `NewTriliumDocumentLoader(map[string]bool{".md": true}, 100)`。
func NewTriliumDocumentLoader(allowExt map[string]bool, limit int) biz.DocumentLoader {
	if len(allowExt) == 0 {
		allowExt = map[string]bool{".md": true}
		slog.Default().Info("未设置文件过滤格式, 默认设置为 .md")
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
func (l *loader) Load(path string) ([]*biz.Document, error) {
	var documents []*biz.Document
	filepathList, err := GetAllFile(path, l.allowExt)
	if err != nil {
		return nil, err
	}
	slog.Default().Info("收集文档文件",
		"path", path,
		"extensions", strings.Join(GetAllAllowExt(l.allowExt), ","),
		"count", len(filepathList),
	)

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
// 输出: 返回包含来源路径、标题和正文内容的 `biz.Document`; 路径为目录或读取失败时返回错误。
// 示例: `l.parseDocument("notes/rag.md")`。
func (l *loader) parseDocument(filepath string) (*biz.Document, error) {
	if utils.IsDir(filepath) == true {
		return nil, fmt.Errorf("%s 是目录, 不是文件", filepath)
	}
	doc := &biz.Document{SourcePath: filepath}
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
	if err := scanner.Err(); err != nil {
		slog.Default().Warn("读取文档标题失败", "path", docPath, "error", err)
	}

	return title
}

// GetAllFile 获取 path 下所有扩展名位于 allowExt 中的文件路径。
// 输入: `root` 是待收集路径, 一般为目录, 也可以是单个文件; `allowExt` 是允许的扩展名集合。
// 输出: 返回符合条件的绝对文件路径列表; 路径无效或没有符合条件的文件时返回错误。
// 示例: `GetAllFile("docs", map[string]bool{".md": true})` -> 返回 docs 下所有 markdown 文件。
func GetAllFile(root string, allowExt map[string]bool) ([]string, error) {
	var (
		filePaths []string // 符合条件的文件路径列表
	)
	fileInfo, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if fileInfo.IsDir() == false {
		if allowExt[strings.ToLower(filepath.Ext(fileInfo.Name()))] == true {
			fPath, err := filepath.Abs(root)
			if err != nil {
				return nil, err
			}
			filePaths = append(filePaths, fPath)
			return filePaths, nil
		}
		return nil, errors.New("当前路径下未找到符合条件的文件")
	}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() == false {
			if allowExt[strings.ToLower(filepath.Ext(d.Name()))] == true {
				fPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				filePaths = append(filePaths, fPath)
			} else {
				log.Println("skip file:", path)
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(filePaths) == 0 {
		return nil, errors.New("当前路径下未找到符合条件的文件")
	}
	return filePaths, nil
}

// GetAllAllowExt 返回允许加载的文件后缀列表。
// 输入: `allowExt` 是后缀到是否允许的映射。
// 输出: 返回映射中所有后缀组成的切片。
// 示例: `GetAllAllowExt(map[string]bool{".md": true, ".txt": true})`。
func GetAllAllowExt(allowExt map[string]bool) []string {
	exts := make([]string, len(allowExt))
	idx := 0
	for k := range allowExt {
		exts[idx] = k
		idx++
	}
	return exts
}
