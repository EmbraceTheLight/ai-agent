package rag

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// LoadRAGResources 从本地路径加载 RAG 文档资源。
// 输入: `path` 可以是单个 `.md`/`.txt` 文件, 也可以是包含这些文件的目录。
// 输出: 返回文档列表; 路径无效、没有支持文件或读取失败时返回错误。
// 示例: `LoadRAGResources("testdata/documents")` -> 返回可用于 chunk 切分的 `[]*Document`。
func LoadRAGResources(path string) ([]*Document, error) {
	var documents []*Document
	allowExt := map[string]bool{
		".md":  true,
		".txt": true,
	}
	filepathList, err := GetAllFile(path, allowExt)
	if err != nil {
		return nil, err
	}
	log.Printf("目录 %s, 收集到的 .md && .txt 文件共有 %d 个\n", path, len(filepathList))
	for _, filePath := range filepathList {
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		documents = append(documents, &Document{SourcePath: filePath, Content: string(fileContent)})
	}
	return documents, nil
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
		} else {
			return nil, errors.New("当前路径下未找到符合条件的文件")
		}
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
	for k, _ := range allowExt {
		exts[idx] = k
		idx++
	}
	return exts
}
