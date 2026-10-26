// Command trilium-cli 通过 Trilium ETAPI 查询笔记，并在指定子树内创建、追加或更新正文。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"trilium-mcp/internal/config"
	"trilium-mcp/internal/notes"
	"trilium-mcp/internal/trilium"
)

const usage = `Usage:
  trilium-cli info [-env-file PATH]
  trilium-cli search [-env-file PATH] [-limit 20] [-ancestor NOTE_ID] "QUERY"
  trilium-cli read [-env-file PATH] [-raw] NOTE_ID
  trilium-cli create [-env-file PATH] [-parent NOTE_ID] -title TITLE -file PATH -operation ID [-format markdown]
  trilium-cli append [-env-file PATH] -file PATH -operation ID -expected-sha256 HASH [-format markdown] NOTE_ID
  trilium-cli update [-env-file PATH] -file PATH [-match-file PATH] -operation ID -expected-sha256 HASH [-format html] NOTE_ID

Configuration (non-empty environment values override the env file):
  TRILIUM_URL     Instance URL, optionally ending in /etapi (required).
  TRILIUM_TOKEN   ETAPI token (required; never passed on the command line).
  TRILIUM_WRITE_ROOT_ID   Writable subtree root (required for create/append/update).
  TRILIUM_STATE_DIR       Local snapshots and operation records (default: beside env file).

Loads .env from the current working directory, if present.
Use -env-file PATH to select a different file. Values are never expanded.
Writes are restricted to the configured subtree. The request timeout is 10 seconds.
Reuse the same operation ID, content and state directory after an uncertain result.
Rich-text bodies are returned as HTML; -raw outputs only the original body.
`

// main 启动命令行程序，收到 Ctrl+C 时取消进行中的请求。
// 命令执行失败时将错误写入标准错误，并以状态码 1 退出。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, os.Getenv); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// run 解析并执行单条命令，在参数检查后加载配置。
// 输入：ctx 控制请求生命周期；args 不含程序名；out 接收命令结果；errOut 接收 flag 帮助及解析诊断；getenv 读取环境变量。
// 输出：成功时返回 nil；参数、配置、HTTP 请求或结果输出失败时返回错误，由调用方报告。
// 顶层帮助无需认证；默认输出 JSON，read -raw 输出原始正文，选项必须位于位置参数之前。
// 默认加载工作目录的 .env（不存在时跳过），可用 -env-file 指定其他文件；非空环境变量优先。
func run(ctx context.Context, args []string, out, errOut io.Writer, getenv func(string) string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		_, err := io.WriteString(out, usage)
		return err
	}
	command := args[0]
	if command != "info" && command != "search" && command != "read" && command != "create" && command != "append" && command != "update" {
		return errors.New("unknown command; use --help")
	}
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(errOut)
	envFile := flags.String("env-file", ".env", "configuration file (default: .env in working directory)")
	var raw bool
	var write writeOptions
	search := trilium.SearchOptions{Limit: 20}
	if command == "read" {
		flags.BoolVar(&raw, "raw", false, "output only the original UTF-8 body")
	}
	if command == "search" {
		flags.IntVar(&search.Limit, "limit", 20, "maximum results (1–100)")
		flags.StringVar(&search.AncestorNoteID, "ancestor", "", "restrict to this subtree")
	}
	if command == "create" || command == "append" || command == "update" {
		defaultFormat := "markdown"
		if command == "update" {
			defaultFormat = "html"
			flags.StringVar(&write.matchFile, "match-file", "", "replace exactly one raw HTML fragment from this file")
		}
		flags.StringVar(&write.file, "file", "", "UTF-8 content file")
		flags.StringVar(&write.format, "format", defaultFormat, "markdown, text or html")
		flags.StringVar(&write.operation, "operation", "", "stable unique operation ID")
		if command == "create" {
			flags.StringVar(&write.parent, "parent", "", "parent note ID (default: write root)")
			flags.StringVar(&write.title, "title", "", "new note title")
		} else {
			flags.StringVar(&write.expected, "expected-sha256", "", "contentSHA256 returned by read")
		}
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if (command == "info" || command == "create") && flags.NArg() != 0 {
		return fmt.Errorf("%s accepts no positional arguments", command)
	}
	if command != "info" && command != "create" && flags.NArg() != 1 {
		return fmt.Errorf("%s requires exactly one argument; put flags before the argument", command)
	}
	lookup, err := config.LoadEnvFile(*envFile, getenv)
	if err != nil {
		if *envFile != ".env" || !errors.Is(err, os.ErrNotExist) {
			return err
		}
		lookup = getenv
	}
	client, err := trilium.NewClient(lookup("TRILIUM_URL"), lookup("TRILIUM_TOKEN"), 10*time.Second)
	if err != nil {
		return err
	}
	var result any
	switch command {
	case "create", "append", "update":
		result, err = executeWrite(ctx, client, lookup, *envFile, command, flags.Arg(0), write)
	case "info":
		result, err = client.GetAppInfo(ctx)
	case "search":
		search.Query = flags.Arg(0)
		result, err = client.SearchNotes(ctx, search)
	case "read":
		note, readErr := client.GetNote(ctx, flags.Arg(0))
		if readErr != nil {
			return readErr
		}
		if note.IsProtected {
			return errors.New("protected notes are not supported")
		}
		if note.Type != "text" && note.Type != "code" {
			return errors.New("read supports only text and code notes")
		}
		content, readErr := client.GetNoteContent(ctx, note.NoteID)
		if readErr != nil {
			return readErr
		}
		if raw {
			_, err = io.WriteString(out, content)
			return err
		}
		result = struct {
			Note          trilium.Note `json:"note"`
			Content       string       `json:"content"`
			ContentSHA256 string       `json:"contentSHA256"`
		}{Note: note, Content: content, ContentSHA256: notes.ContentSHA256(content)}
	}
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(result)
}
