package rag

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func LoadRAGResources(path string) ([]*Document, error) {
	var documents []*Document
	allowExt := map[string]bool{
		".md":  true,
		".txt": true,
	}
	filepathList, err := getAllFile(path, allowExt)
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

// getAllFile 获取 path 下所有扩展名位于 allowExt 中的文件路径
// 输入: 要收集文件的路径, 一般为目录, 也可以是单个文件
// 输出: 文件路径数组, 符合条件的文件数量, 错误信息
func getAllFile(root string, allowExt map[string]bool) ([]string, error) {
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
