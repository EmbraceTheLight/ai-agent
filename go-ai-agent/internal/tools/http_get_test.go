package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	urlpkg "net/url"
	"strings"
	"testing"
)

func TestHttpGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/large" {
			_, _ = w.Write([]byte("response body is too large"))
			return
		}

		if r.Method != http.MethodGet {
			t.Errorf("expected method GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("city"); got != "anyang" {
			t.Errorf("expected query city=anyang, got %q", got)
		}
		if got := r.URL.Query().Get("adcode"); got != "410502" {
			t.Errorf("expected query adcode=410502, got %q", got)
		}
		if got := r.URL.Query().Get("lang"); got != "zh" {
			t.Errorf("expected query lang=zh, got %q", got)
		}
		if got := r.Header.Get("X-Test-Header"); got != "tool-test" {
			t.Errorf("expected X-Test-Header=tool-test, got %q", got)
		}

		_, _ = w.Write([]byte("anyang weather ok"))
	}))
	defer server.Close()

	parsedURL, err := urlpkg.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL failed: %v", err)
	}

	tests := []struct {
		name      string
		tool      *httpGetTool
		req       *HttpGetReq
		want      string
		wantError bool
	}{
		{
			name: "允许访问 allowlist 内 URL 并带上 query 和 header",
			tool: &httpGetTool{
				allowURLList:    map[string]bool{parsedURL.Hostname(): true},
				allowMethodList: map[string]bool{http.MethodGet: true},
				respLimit:       1024,
			},
			req: &HttpGetReq{
				URL:    server.URL,
				Method: http.MethodGet,
				Header: map[string]string{
					"X-Test-Header": "tool-test",
				},
				Query: map[string]string{
					"city":   "anyang",
					"adcode": "410502",
					"lang":   "zh",
				},
			},
			want: "anyang weather ok",
		},
		{
			name: "拒绝访问 allowlist 外 URL",
			tool: &httpGetTool{
				allowURLList:    map[string]bool{"example.com": true},
				allowMethodList: map[string]bool{http.MethodGet: true},
				respLimit:       1024,
			},
			req: &HttpGetReq{
				URL:    server.URL,
				Method: http.MethodGet,
			},
			wantError: true,
		},
		{
			name: "拒绝不支持的 HTTP 方法",
			tool: &httpGetTool{
				allowURLList:    map[string]bool{parsedURL.Hostname(): true},
				allowMethodList: map[string]bool{http.MethodGet: true},
				respLimit:       1024,
			},
			req: &HttpGetReq{
				URL:    server.URL,
				Method: http.MethodPost,
			},
			wantError: true,
		},
		{
			name: "响应超过大小限制时返回错误",
			tool: &httpGetTool{
				allowURLList:    map[string]bool{parsedURL.Hostname(): true},
				allowMethodList: map[string]bool{http.MethodGet: true},
				respLimit:       5,
			},
			req: &HttpGetReq{
				URL:    server.URL + "/large",
				Method: http.MethodGet,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rawReq, err := json.Marshal(tt.req)
			if err != nil {
				t.Fatalf("marshal request failed: %v", err)
			}

			got, err := tt.tool.HttpGet(context.Background(), rawReq)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("expected nil error, got %v", err)
			}
			if !strings.Contains(got, tt.want) {
				t.Fatalf("expected response contains %q, got %q", tt.want, got)
			}
		})
	}
}
