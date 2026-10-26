package trilium

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestReadEndpoints 验证不同地址后缀均保留代理路径、携带认证头，并原样返回 HTML 正文。
func TestReadEndpoints(t *testing.T) {
	const body = "<p>你好 &amp; Go</p>\n<img src=\"api/attachments/image123/image\">"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "test-token" {
			t.Errorf("unexpected request: method=%s authorization=%q", r.Method, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/prefix/etapi/app-info":
			io.WriteString(w, `{"appVersion":"0.104.1","dbVersion":231,"futureField":true}`)
		case "/prefix/etapi/notes/note123":
			io.WriteString(w, `{"noteId":"note123","title":"测试","type":"text","mime":"text/html","futureField":123}`)
		case "/prefix/etapi/notes/note123/content":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, body)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	for _, suffix := range []string{"/prefix", "/prefix/", "/prefix/etapi", "/prefix/etapi/"} {
		t.Run(suffix, func(t *testing.T) {
			client := newTestClient(t, server.URL+suffix)
			info, err := client.GetAppInfo(context.Background())
			if err != nil || info.AppVersion != "0.104.1" {
				t.Fatalf("GetAppInfo = %+v, %v", info, err)
			}
			note, err := client.GetNote(context.Background(), "note123")
			if err != nil || note.Title != "测试" {
				t.Fatalf("GetNote = %+v, %v", note, err)
			}
			content, err := client.GetNoteContent(context.Background(), "note123")
			if err != nil || content != body {
				t.Fatalf("GetNoteContent = %q, %v", content, err)
			}
		})
	}
}

// TestSearchNotes 验证查询中的中文及特殊字符正确编码，子树和数量参数传入接口，无匹配时返回空切片。
func TestSearchNotes(t *testing.T) {
	const query = `"Go 学习" #topic & extra=1`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if r.URL.Path != "/etapi/notes" || q.Get("search") != query || q.Get("limit") != "7" || q.Get("ancestorNoteId") != "root" || len(q) != 3 {
			t.Errorf("unexpected search URL: %s", r.URL)
		}
		io.WriteString(w, `{"results":[],"futureField":"ok"}`)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	notes, err := client.SearchNotes(context.Background(), SearchOptions{Query: query, Limit: 7, AncestorNoteID: "root"})
	if err != nil || notes == nil || len(notes) != 0 {
		t.Fatalf("SearchNotes = %#v, %v", notes, err)
	}
}

// TestNewClientRejectsInvalidConfig 验证非法地址、Token 和超时配置在请求前被拒绝，错误不泄露配置内容。
func TestNewClientRejectsInvalidConfig(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		token   string
		timeout time.Duration
	}{
		{"missing URL", "", "token", time.Second},
		{"relative URL", "localhost:8080", "token", time.Second},
		{"wrong scheme", "file:///tmp/notes", "token", time.Second},
		{"URL credentials", "https://user:secret@example.com", "token", time.Second},
		{"URL query", "https://example.com/?token=secret", "token", time.Second},
		{"URL fragment", "https://example.com/#secret", "token", time.Second},
		{"missing token", "https://example.com", "", time.Second},
		{"header injection", "https://example.com", "token\r\nInjected: yes", time.Second},
		{"trailing space", "https://example.com", "token ", time.Second},
		{"no timeout", "https://example.com", "token", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewClient(tc.baseURL, tc.token, tc.timeout)
			if err == nil {
				t.Fatal("expected config error")
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "Injected") {
				t.Fatalf("error exposes configuration: %v", err)
			}
		})
	}
}

// TestInvalidNoteIDsAndQueriesMakeNoRequests 验证非法 ID、空查询及越界数量被拒绝，且不会发起 HTTP 请求。
func TestInvalidNoteIDsAndQueriesMakeNoRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("invalid input made an HTTP request: %s", r.URL)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	for _, id := range []string{"", "abc", "../root", "root/content", "root?secret", "笔记123", strings.Repeat("a", 33)} {
		if _, err := client.GetNote(context.Background(), id); err == nil {
			t.Errorf("GetNote accepted %q", id)
		}
		if _, err := client.GetNoteContent(context.Background(), id); err == nil {
			t.Errorf("GetNoteContent accepted %q", id)
		}
	}
	for _, options := range []SearchOptions{
		{Query: " ", Limit: 20},
		{Query: "Go", Limit: 0},
		{Query: "Go", Limit: 101},
		{Query: "Go", Limit: 20, AncestorNoteID: "../root"},
	} {
		if _, err := client.SearchNotes(context.Background(), options); err == nil {
			t.Errorf("SearchNotes accepted %+v", options)
		}
	}
}

// TestHTTPFailuresDoNotExposeResponseBodies 验证非 200 状态返回 HTTPError、不跟随重定向，错误不泄露响应正文。
func TestHTTPFailuresDoNotExposeResponseBodies(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError, http.StatusFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/etapi/app-info" {
					t.Error("client followed redirect")
				}
				w.Header().Set("Location", "/do-not-follow")
				w.WriteHeader(status)
				io.WriteString(w, `{"message":"test-token private-note"}`)
			}))
			defer server.Close()
			_, err := newTestClient(t, server.URL).GetAppInfo(context.Background())
			var apiErr *HTTPError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != status {
				t.Fatalf("expected HTTPError %d, got %v", status, err)
			}
			if strings.Contains(err.Error(), "test-token") || strings.Contains(err.Error(), "private-note") {
				t.Fatalf("error exposes response: %v", err)
			}
		})
	}
}

// TestMalformedResponsesRejectFalseSuccess 验证登录页、缺失版本信息和尾随额外 JSON 均被判为失败。
func TestMalformedResponsesRejectFalseSuccess(t *testing.T) {
	for _, response := range []string{"<html>login</html>", `{}`, `null`, `{"appVersion":"0.104.1"} {}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, response)
		}))
		_, err := newTestClient(t, server.URL).GetAppInfo(context.Background())
		server.Close()
		if err == nil {
			t.Errorf("accepted malformed response %q", response)
		}
	}
}

// TestResponseLimitAndUTF8 验证超过响应上限或含非法 UTF-8 字节的正文返回错误。
func TestResponseLimitAndUTF8(t *testing.T) {
	for _, body := range []string{strings.Repeat("a", maxResponseBytes+1), string([]byte{0xff, 0xfe})} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.WriteString(w, body)
		}))
		_, err := newTestClient(t, server.URL).GetNoteContent(context.Background(), "root")
		server.Close()
		if err == nil {
			t.Fatal("accepted oversized or binary body")
		}
	}
}

// TestCancellationAndTimeout 验证已取消的上下文保留取消错误，超过请求时限时返回超时错误。
func TestCancellationAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.GetAppInfo(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	client, err := NewClient(server.URL, "test-token", 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.GetAppInfo(context.Background()); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected timeout, got %v", err)
	}
}

// newTestClient 为本地 HTTP 测试创建带假 Token 和两秒超时的客户端。
// 输入：t 为当前测试；baseURL 为模拟服务地址。
// 输出：构造成功时返回客户端；失败时终止当前测试并报告错误。
func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := NewClient(baseURL, "test-token", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return client
}
