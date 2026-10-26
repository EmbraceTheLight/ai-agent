package main

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"trilium-mcp/internal/notes"
	"trilium-mcp/internal/trilium"
)

// writeOptions 保存命令行写入参数；正文从文件读取，操作 ID 和摘要用于恢复及冲突检查。
type writeOptions struct {
	parent, title, file, format, operation, expected, matchFile string
}

// executeWrite 加载受控的正文输入并调用写入业务层。
// 输入：lookup 查询配置；command 为 create、append 或 update；noteID 用于追加和更新；options 来自已解析参数。
// 输出：成功时返回操作结果；范围未配置、文件无效或业务校验失败时返回错误。
// 默认状态目录位于配置文件旁，避免从其他目录启动时丢失重试记录；目录中保存私有快照。
func executeWrite(ctx context.Context, client *trilium.Client, lookup func(string) string, envFile, command, noteID string, options writeOptions) (notes.Result, error) {
	root := lookup("TRILIUM_WRITE_ROOT_ID")
	stateDir := lookup("TRILIUM_STATE_DIR")
	if stateDir == "" {
		configPath, err := filepath.Abs(envFile)
		if err != nil {
			return notes.Result{}, errors.New("cannot resolve write-state directory")
		}
		stateDir = filepath.Join(filepath.Dir(configPath), ".trilium-state")
	}
	service, err := notes.NewService(client, root, stateDir, lookup("TRILIUM_URL"))
	if err != nil {
		return notes.Result{}, err
	}
	if options.file == "" || options.operation == "" {
		return notes.Result{}, errors.New("write requires -file and a stable -operation ID")
	}
	body, err := readContentFile(options.file)
	if err != nil {
		return notes.Result{}, err
	}
	if command == "create" {
		parent := options.parent
		if parent == "" {
			parent = root
		}
		return service.Create(ctx, notes.CreateInput{OperationID: options.operation, ParentNoteID: parent, Title: options.title, Content: string(body), Format: options.format})
	}
	if command == "update" {
		var match string
		if options.matchFile != "" {
			old, err := readContentFile(options.matchFile)
			if err != nil {
				return notes.Result{}, err
			}
			if len(old) == 0 {
				return notes.Result{}, errors.New("match file must not be empty")
			}
			match = string(old)
		}
		return service.Update(ctx, notes.UpdateInput{OperationID: options.operation, NoteID: noteID, Content: string(body), Format: options.format, ExpectedSHA256: options.expected, Match: match})
	}
	return service.Append(ctx, notes.AppendInput{OperationID: options.operation, NoteID: noteID, Content: string(body), Format: options.format, ExpectedSHA256: options.expected})
}

// readContentFile 读取有大小限制的 UTF-8 正文或匹配片段文件，不输出文件内容。
// 输入：path 为调用方选择的文件路径。
// 输出：成功时返回完整字节；文件无法读取、超过 4 MiB 或编码非法时返回错误。
func readContentFile(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("cannot open content input file")
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, (4<<20)+1))
	if err != nil || len(body) > 4<<20 || !utf8.Valid(body) {
		return nil, errors.New("content input must be readable UTF-8 text of at most 4 MiB")
	}
	return body, nil
}
