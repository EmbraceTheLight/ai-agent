package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"trilium-mcp/internal/notes"
	"trilium-mcp/internal/trilium"
)

// memoryAPI 模拟 ETAPI 的笔记和正文，保留真实写入业务层用于协议集成测试。
type memoryAPI struct {
	mu                sync.Mutex
	notes             map[string]trilium.Note
	bodies            map[string]string
	writes, bodyReads int
	lastSearch        trilium.SearchOptions
}

// fixture 提供范围内笔记、越界笔记及受保护笔记。
func fixture() *memoryAPI {
	api := &memoryAPI{notes: map[string]trilium.Note{}, bodies: map[string]string{}}
	for id, parent := range map[string]string{"root": "", "temp": "root", "note123": "temp", "other": "root", "locked": "temp"} {
		note := trilium.Note{NoteID: id, Title: id, Type: "text", MIME: "text/html", ChildNoteIDs: []string{}, ParentNoteIDs: []string{}}
		if parent != "" {
			note.ParentNoteIDs = []string{parent}
		}
		note.IsProtected = id == "locked"
		api.notes[id], api.bodies[id] = note, "<p>原文</p>"
	}
	note := api.notes["temp"]
	note.ChildNoteIDs = []string{"note123", "locked"}
	api.notes["temp"] = note
	return api
}

// GetAppInfo 返回测试版本，无需真实服务。
func (a *memoryAPI) GetAppInfo(context.Context) (trilium.AppInfo, error) {
	return trilium.AppInfo{AppVersion: "test"}, nil
}

// GetNote 保持不存在笔记返回 404 的 ETAPI 语义。
func (a *memoryAPI) GetNote(_ context.Context, id string) (trilium.Note, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if note, ok := a.notes[id]; ok {
		return note, nil
	}
	return trilium.Note{}, &trilium.HTTPError{StatusCode: http.StatusNotFound}
}

// GetNoteContent 记录正文读取次数，用于确认受保护笔记没有被下载。
func (a *memoryAPI) GetNoteContent(_ context.Context, id string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.bodyReads++
	return a.bodies[id], nil
}

// SearchNotes 保存传入条件，验证 MCP 默认值确实传递至客户端。
func (a *memoryAPI) SearchNotes(_ context.Context, input trilium.SearchOptions) ([]trilium.Note, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastSearch = input
	return []trilium.Note{a.notes["note123"], a.notes["locked"]}, nil
}

// CreateNote 模拟一次新建，以计数核对重放没有重复写入。
func (a *memoryAPI) CreateNote(_ context.Context, input trilium.CreateNoteInput) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.writes++
	a.notes[input.NoteID] = trilium.Note{NoteID: input.NoteID, Title: input.Title, Type: "text", MIME: "text/html", ParentNoteIDs: []string{input.ParentNoteID}, ChildNoteIDs: []string{}}
	a.bodies[input.NoteID] = input.Content
	return nil
}

// SetNoteContent 模拟写入并保留原始字节。
func (a *memoryAPI) SetNoteContent(_ context.Context, id, body string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.writes++
	a.bodies[id] = body
	return nil
}

// connect 通过官方 SDK 完成真实的内存传输握手，而不是直接调用处理函数。
func connect(t *testing.T, api API, write bool) *mcp.ClientSession {
	t.Helper()
	options := Options{}
	if write {
		options = Options{WriteRootID: "temp", StateDir: t.TempDir(), Instance: "test-instance"}
	}
	server, err := New(api, options)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	return cs
}

// call 使用协议调用工具，检查成功状态后解码 SDK 自动生成的文本 JSON。
func call(t *testing.T, cs *mcp.ClientSession, name string, args any, out any) {
	t.Helper()
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("%s: %+v", name, result.Content)
	}
	if result.StructuredContent == nil {
		t.Fatal("missing structured output")
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content: %+v", result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("expected JSON text")
	}
	if err := json.Unmarshal([]byte(text.Text), out); err != nil {
		t.Fatal(err)
	}
}

// TestToolsAndWrites 验证工具发现、默认参数、真实业务写入及同 ID 重放。
func TestToolsAndWrites(t *testing.T) {
	api := fixture()
	cs := connect(t, api, true)
	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 7 {
		t.Fatalf("tools=%d", len(list.Tools))
	}
	for _, tool := range list.Tools {
		if tool.InputSchema == nil || tool.OutputSchema == nil || tool.Annotations == nil {
			t.Fatalf("missing schemas/annotations: %s", tool.Name)
		}
		if tool.Name == "trilium_update" && (tool.Annotations.ReadOnlyHint || !*tool.Annotations.DestructiveHint) {
			t.Fatal("update annotation")
		}
	}
	var info InfoOutput
	call(t, cs, "trilium_info", map[string]any{}, &info)
	if info.WriteRootID != "temp" || info.AppInfo.AppVersion != "test" {
		t.Fatal(info)
	}
	var found NoteList
	call(t, cs, "trilium_search", map[string]any{"query": "test"}, &found)
	if api.lastSearch.Limit != 20 || api.lastSearch.AncestorNoteID != "temp" || len(found.Notes) != 1 {
		t.Fatal(found, api.lastSearch)
	}
	var page ChildrenOutput
	call(t, cs, "trilium_list_children", map[string]any{"noteId": "temp", "limit": 1}, &page)
	if page.NextOffset != 1 || len(page.Notes) != 1 {
		t.Fatal(page)
	}
	call(t, cs, "trilium_list_children", map[string]any{"noteId": "temp", "offset": 1}, &page)
	if page.NextOffset != -1 || len(page.Notes) != 0 {
		t.Fatal(page)
	}
	var result notes.Result
	create := map[string]any{"operationId": "mcp-create", "title": "MCP", "content": "初始内容"}
	call(t, cs, "trilium_create", create, &result)
	id := result.NoteID
	call(t, cs, "trilium_create", create, &result)
	if !result.AlreadyApplied || api.writes != 1 {
		t.Fatal(result, api.writes)
	}
	var read ReadOutput
	call(t, cs, "trilium_read", map[string]any{"noteId": id}, &read)
	if read.Content != "<p>初始内容</p>\n" || notes.ContentSHA256(read.Content) != read.ContentSHA256 {
		t.Fatal(read)
	}
	appendArgs := map[string]any{"operationId": "mcp-append", "noteId": id, "content": "追加", "expectedSHA256": read.ContentSHA256}
	call(t, cs, "trilium_append", appendArgs, &result)
	call(t, cs, "trilium_append", appendArgs, &result)
	if !result.AlreadyApplied || api.writes != 2 {
		t.Fatal(result, api.writes)
	}
	call(t, cs, "trilium_read", map[string]any{"noteId": id}, &read)
	update := map[string]any{"operationId": "mcp-update", "noteId": id, "content": "<p>替换</p>", "match": "<p>追加</p>", "expectedSHA256": read.ContentSHA256}
	call(t, cs, "trilium_update", update, &result)
	call(t, cs, "trilium_update", update, &result)
	if !result.AlreadyApplied || api.writes != 3 {
		t.Fatal(result, api.writes)
	}
	call(t, cs, "trilium_read", map[string]any{"noteId": id}, &read)
	if !strings.Contains(read.Content, "初始内容") || !strings.Contains(read.Content, "<p>替换</p>") || strings.Contains(read.Content, "追加") {
		t.Fatal(read.Content)
	}
	update["operationId"] = "mcp-full"
	update["expectedSHA256"] = read.ContentSHA256
	delete(update, "match")
	call(t, cs, "trilium_update", update, &result)
	if api.bodies[id] != "<p>替换</p>" {
		t.Fatal(api.bodies[id])
	}
}

// TestToolFailures 验证 schema、越界和冲突错误不会执行写入，且受保护正文不会下载。
func TestToolFailures(t *testing.T) {
	api := fixture()
	cs := connect(t, api, true)
	cases := []struct {
		name string
		args any
	}{
		{"trilium_create", map[string]any{"operationId": "outside", "parentNoteId": "other", "title": "x", "content": "x"}},
		{"trilium_update", map[string]any{"operationId": "root-update", "noteId": "temp", "content": "x", "expectedSHA256": notes.ContentSHA256("<p>原文</p>")}},
		{"trilium_update", map[string]any{"operationId": "stale", "noteId": "note123", "content": "x", "expectedSHA256": strings.Repeat("0", 64)}},
		{"trilium_append", map[string]any{"operationId": "missing-hash", "noteId": "note123", "content": "x"}},
		{"trilium_read", map[string]any{"noteId": "locked"}},
		{"trilium_list_children", map[string]any{"noteId": "locked"}},
		{"trilium_list_children", map[string]any{"noteId": "temp", "offset": -1}},
		{"trilium_read", map[string]any{"noteId": 42}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := api.bodyReads
			result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
			if err == nil && !result.IsError {
				t.Fatal("expected tool or protocol validation error")
			}
			if tc.name == "trilium_read" && api.bodyReads != before {
				t.Fatal("downloaded forbidden body")
			}
		})
	}
	if api.writes != 0 {
		t.Fatalf("writes=%d", api.writes)
	}
}

// TestReadonlyAndInvalidConfiguration 验证只读模式没有写工具，错误范围不会降级启动。
func TestReadonlyAndInvalidConfiguration(t *testing.T) {
	cs := connect(t, fixture(), false)
	list, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 4 {
		t.Fatal(len(list.Tools))
	}
	if _, err := New(nil, Options{}); err == nil {
		t.Fatal("nil API accepted")
	}
	if _, err := New(fixture(), Options{WriteRootID: "root", StateDir: t.TempDir(), Instance: "test"}); err == nil {
		t.Fatal("root accepted")
	}
}

// failingAPI 用于确认 ETAPI 错误被转换成工具错误而不会断开 MCP 会话。
type failingAPI struct{ *memoryAPI }

// GetAppInfo 返回不包含服务端正文的故障。
func (a failingAPI) GetAppInfo(context.Context) (trilium.AppInfo, error) {
	return trilium.AppInfo{}, errors.New("upstream unavailable")
}

// TestUpstreamErrorKeepsSession 验证业务错误携带 isError，之后仍能发现工具。
func TestUpstreamErrorKeepsSession(t *testing.T) {
	cs := connect(t, failingAPI{fixture()}, false)
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "trilium_info", Arguments: map[string]any{}})
	if err != nil || !result.IsError {
		t.Fatal(result, err)
	}
	if _, err := cs.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
