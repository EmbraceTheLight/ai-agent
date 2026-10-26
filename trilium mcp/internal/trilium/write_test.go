package trilium

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWriteEndpoints 验证新建请求携带固定 ID 和类型，HTML 更新以原始 text/plain 字节传输。
func TestWriteEndpoints(t *testing.T) {
	const content = "<p>中文 &amp; HTML</p>\n"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "test-token" {
			t.Error("missing write authentication")
		}
		if r.Method == http.MethodPost && r.URL.Path == "/etapi/create-note" {
			var body struct {
				CreateNoteInput
				Type, MIME string
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NoteID != "note123" || body.ParentNoteID != "temp" || body.Content != content || body.Type != "text" || body.MIME != "text/html" {
				t.Error("unexpected create payload")
			}
			w.WriteHeader(http.StatusCreated)
			return
		}
		if r.Method == http.MethodPut && r.URL.Path == "/etapi/notes/note123/content" {
			body, _ := io.ReadAll(r.Body)
			if string(body) != content || !strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") {
				t.Error("HTML was encoded incorrectly")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Error("unexpected write endpoint")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	client := newTestClient(t, server.URL)
	if err := client.CreateNote(context.Background(), CreateNoteInput{ParentNoteID: "temp", NoteID: "note123", Title: "new", Content: content}); err != nil {
		t.Fatal(err)
	}
	if err := client.SetNoteContent(context.Background(), "note123", content); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 write calls, got %d", calls)
	}
}

// TestWriteErrorsAreSanitizedAndNotRetried 验证失败状态不被自动重试，错误保留方法且不泄露服务器正文。
func TestWriteErrorsAreSanitizedAndNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, "PRIVATE test-token")
	}))
	defer server.Close()
	err := newTestClient(t, server.URL).SetNoteContent(context.Background(), "note123", "<p>x</p>")
	var status *HTTPError
	if !errors.As(err, &status) || status.Method != http.MethodPut || calls != 1 || strings.Contains(err.Error(), "PRIVATE") || strings.Contains(err.Error(), "test-token") {
		t.Fatalf("unexpected write failure: %v", err)
	}
}
