// Command trilium-mcp 通过标准输入输出提供 MCP 工具，诊断信息仅写入标准错误。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"trilium-mcp/internal/config"
	"trilium-mcp/internal/mcpserver"
	"trilium-mcp/internal/trilium"
)

// main 在 stdio 上运行服务，收到 Ctrl+C 时取消工具请求。
// 客户端负责启动和结束该进程；stdout 必须完全留给 MCP 协议。
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	server, err := configure(os.Args[1:], os.Stderr, os.Getenv)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err == nil {
		err = server.Run(ctx, &mcp.StdioTransport{})
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// configure 解析配置并创建服务，不访问笔记库。
// 输入：args 支持 -env-file；errOut 接收帮助和参数诊断；getenv 的非空值覆盖文件。
// 输出：返回服务或配置错误；状态目录默认位于配置文件旁，与 CLI 共用持久化记录。
func configure(args []string, errOut io.Writer, getenv func(string) string) (*mcp.Server, error) {
	flags := flag.NewFlagSet("trilium-mcp", flag.ContinueOnError)
	flags.SetOutput(errOut)
	envFile := flags.String("env-file", ".env", "UTF-8 configuration file; use an absolute path in MCP clients")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if flags.NArg() != 0 {
		return nil, errors.New("unexpected arguments; use -h")
	}
	lookup, err := config.LoadEnvFile(*envFile, getenv)
	if err != nil {
		if *envFile != ".env" || !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		lookup = getenv
	}
	client, err := trilium.NewClient(lookup("TRILIUM_URL"), lookup("TRILIUM_TOKEN"), 10*time.Second)
	if err != nil {
		return nil, err
	}
	stateDir := lookup("TRILIUM_STATE_DIR")
	if stateDir == "" {
		path, err := filepath.Abs(*envFile)
		if err != nil {
			return nil, errors.New("cannot resolve write-state directory")
		}
		stateDir = filepath.Join(filepath.Dir(path), ".trilium-state")
	}
	return mcpserver.New(client, mcpserver.Options{WriteRootID: lookup("TRILIUM_WRITE_ROOT_ID"), StateDir: stateDir, Instance: lookup("TRILIUM_URL")})
}
