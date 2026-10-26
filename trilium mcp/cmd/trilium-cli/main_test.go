package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelpWorksWithoutCredentials 验证顶层帮助无需读取认证配置即可显示命令说明。
func TestHelpWorksWithoutCredentials(t *testing.T) {
	var out, errOut bytes.Buffer
	err := run(context.Background(), []string{"--help"}, &out, &errOut, func(string) string {
		t.Fatal("help attempted to load credentials")
		return ""
	})
	if err != nil || !strings.Contains(out.String(), "TRILIUM_TOKEN") {
		t.Fatalf("help failed: output=%q, error=%v", out.String(), err)
	}
}

// TestReadPreservesHTML 验证 JSON 中的正文解码后与原文一致，raw 输出不改变内容或增加换行。
func TestReadPreservesHTML(t *testing.T) {
	const content = "<p>中文 &amp; Go</p>\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "test-token" {
			t.Errorf("unexpected request: %s", r.Method)
		}
		switch r.URL.Path {
		case "/etapi/notes/note123":
			io.WriteString(w, `{"noteId":"note123","title":"测试笔记","type":"text","mime":"text/html"}`)
		case "/etapi/notes/note123/content":
			io.WriteString(w, content)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	for _, args := range [][]string{{"read", "note123"}, {"read", "-raw", "note123"}} {
		var out, errOut bytes.Buffer
		if err := run(context.Background(), withEmptyEnvFile(t, args), &out, &errOut, testEnv(server.URL)); err != nil {
			t.Fatal(err)
		}
		if len(args) == 3 {
			if out.String() != content {
				t.Fatalf("raw body changed: %q", out.String())
			}
			continue
		}
		var result struct {
			Content string `json:"content"`
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Content != content {
			t.Fatalf("invalid JSON body: %q, %v", out.String(), err)
		}
	}
}

// TestReadRejectsUnsupportedNotesBeforeContent 验证图片及受保护笔记在下载正文前被拒绝，且不输出部分结果。
func TestReadRejectsUnsupportedNotesBeforeContent(t *testing.T) {
	for _, metadata := range []string{
		`{"noteId":"note123","type":"image","mime":"image/png"}`,
		`{"noteId":"note123","type":"text","isProtected":true}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/content") {
				t.Error("unsupported note body requested")
			}
			io.WriteString(w, metadata)
		}))
		var out, errOut bytes.Buffer
		err := run(context.Background(), withEmptyEnvFile(t, []string{"read", "note123"}), &out, &errOut, testEnv(server.URL))
		server.Close()
		if err == nil || out.Len() != 0 {
			t.Fatalf("expected failure without partial output, got %q, %v", out.String(), err)
		}
	}
}

// TestInfoAndSearchCommands 验证 info、search 调用对应接口，传递搜索条件并输出有效 JSON。
func TestInfoAndSearchCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/etapi/app-info":
			io.WriteString(w, `{"appVersion":"0.104.1"}`)
		case "/etapi/notes":
			if r.URL.Query().Get("search") != "Go" || r.URL.Query().Get("limit") != "3" {
				t.Errorf("unexpected query %s", r.URL.RawQuery)
			}
			io.WriteString(w, `{"results":[]}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	for _, args := range [][]string{{"info"}, {"search", "-limit", "3", "Go"}} {
		var out, errOut bytes.Buffer
		if err := run(context.Background(), withEmptyEnvFile(t, args), &out, &errOut, testEnv(server.URL)); err != nil || !json.Valid(out.Bytes()) {
			t.Fatalf("command %v: output=%q error=%v", args, out.String(), err)
		}
	}
}

// TestInvalidArgumentsFailBeforeConfig 验证未知命令或位置参数数量错误在加载认证配置前返回错误。
func TestInvalidArgumentsFailBeforeConfig(t *testing.T) {
	for _, args := range [][]string{{"unknown"}, {"info", "extra"}, {"read"}, {"read", "one", "two"}, {"search"}} {
		var out, errOut bytes.Buffer
		err := run(context.Background(), args, &out, &errOut, func(string) string {
			t.Error("invalid command loaded credentials")
			return ""
		})
		if err == nil {
			t.Errorf("accepted invalid arguments %v", args)
		}
	}
}

// testEnv 创建测试专用的环境变量读取函数，不访问真实环境。
// 输入：baseURL 为本地模拟服务地址。
// 输出：返回的函数提供指定地址及假 Token，其他变量返回空字符串。
func testEnv(baseURL string) func(string) string {
	return func(key string) string {
		switch key {
		case "TRILIUM_URL":
			return baseURL
		case "TRILIUM_TOKEN":
			return "test-token"
		default:
			return ""
		}
	}
}

// TestInfoLoadsCredentialsFromEnvFile 验证仅配置文件即可完成版本查询，且已有环境变量仍优先于文件。
func TestInfoLoadsCredentialsFromEnvFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/etapi/app-info" || r.Header.Get("Authorization") != "fake-token==" {
			t.Error("request did not use expected file credentials")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"appVersion":"0.104.1"}`)
	}))
	defer server.Close()
	for _, override := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), ".env")
		fileURL, fileToken := server.URL, "fake-token=="
		getenv := func(string) string { return "" }
		if override {
			fileURL, fileToken = "https://unused.invalid", "unused-token"
			getenv = func(key string) string {
				if key == "TRILIUM_URL" {
					return server.URL
				}
				if key == "TRILIUM_TOKEN" {
					return "fake-token=="
				}
				return ""
			}
		}
		if err := os.WriteFile(path, []byte("TRILIUM_URL="+fileURL+"\nTRILIUM_TOKEN="+fileToken+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
		var out, errOut bytes.Buffer
		if err := run(context.Background(), []string{"info", "-env-file", path}, &out, &errOut, getenv); err != nil || !strings.Contains(out.String(), "0.104.1") {
			t.Fatalf("env-file command failed: %v", err)
		}
	}
}

// TestExplicitMissingEnvFileFailsBeforeRequest 验证显式指定的其他文件不存在时报告错误，而非静默回退。
func TestExplicitMissingEnvFileFailsBeforeRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.env")
	var out, errOut bytes.Buffer
	err := run(context.Background(), []string{"info", "-env-file", path}, &out, &errOut, func(string) string {
		t.Error("missing file unexpectedly loaded environment")
		return ""
	})
	if err == nil || out.Len() != 0 {
		t.Fatal("expected missing-file error without output")
	}
}

// TestCommandHelpIgnoresMalformedEnvFile 验证帮助命令不解析配置，避免错误配置妨碍查看修复说明。
func TestCommandHelpIgnoresMalformedEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("TOKEN='SECRET_VALUE"), 0600); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if err := run(context.Background(), []string{"info", "-env-file", path, "-h"}, &out, &errOut, func(string) string {
		t.Error("help read credentials")
		return ""
	}); err != nil {
		t.Fatal(err)
	}
}

// withEmptyEnvFile 为既有 CLI 测试指定空配置，避免读取开发者真实的 .env。
// 输入：t 为当前测试；args 包含命令及其参数。
// 输出：返回插入 -env-file 的新参数切片；临时文件创建失败时终止测试。
func withEmptyEnvFile(t *testing.T, args []string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	return append([]string{args[0], "-env-file", path}, args[1:]...)
}
