package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestConfigure 验证帮助不需凭据、配置路径与非法参数处理，不访问真实 .env。
func TestConfigure(t *testing.T) {
	getenv := func(key string) string {
		return map[string]string{"TRILIUM_URL": "http://localhost:1", "TRILIUM_TOKEN": "fake"}[key]
	}
	var stderr bytes.Buffer
	if _, err := configure([]string{"-h"}, &stderr, getenv); !errors.Is(err, flag.ErrHelp) {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "env-file") {
		t.Fatal(stderr.String())
	}
	path := filepath.Join(t.TempDir(), "config.env")
	if err := os.WriteFile(path, []byte("TRILIUM_WRITE_ROOT_ID=temp\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := configure([]string{"-env-file", path}, &stderr, getenv); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"extra"}, {"-env-file", path + "missing"}, {"-unknown"}} {
		if _, err := configure(args, &stderr, getenv); err == nil {
			t.Fatal(args)
		}
	}
}

// TestStdioProcess 使用独立子进程验证握手、工具发现和 ETAPI 调用，防止 stdout 日志污染协议。
func TestStdioProcess(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "fake-test-token" {
			t.Error("wrong auth")
		}
		if r.URL.Path != "/etapi/app-info" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"appVersion": "fake-stdio"})
	}))
	defer api.Close()
	path := filepath.Join(t.TempDir(), "config.env")
	if err := os.WriteFile(path, []byte("TRILIUM_URL="+api.URL+"\nTRILIUM_TOKEN=fake-test-token\nTRILIUM_WRITE_ROOT_ID=temp\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.Command(os.Args[0], "-test.run=^TestMCPHelperProcess$")
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !strings.HasPrefix(strings.ToUpper(key), "TRILIUM_") && !strings.HasPrefix(key, "MCP_HELPER_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "MCP_HELPER_PROCESS=1", "MCP_HELPER_ENV="+path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	client := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v; stderr=%s", err, stderr.String())
	}
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 7 {
		t.Fatal(len(list.Tools))
	}
	result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "trilium_info", Arguments: map[string]any{}})
	if err != nil || result.IsError {
		t.Fatal(result, err)
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "fake-stdio") {
		t.Fatal(result)
	}
	if err := cs.Close(); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected diagnostics: %s", stderr.String())
	}
}

// TestMCPHelperProcess 将当前测试程序作为 stdio MCP 子进程，直接运行生产入口。
func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("MCP_HELPER_PROCESS") != "1" {
		return
	}
	os.Args = []string{os.Args[0], "-env-file", os.Getenv("MCP_HELPER_ENV")}
	main()
	os.Exit(0)
}
