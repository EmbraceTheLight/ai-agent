package mcpserver

import (
	"context"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"trilium-mcp/internal/notes"
)

// TestMultiRootProtocol 验证协议保留嵌套目录，且多目录新建和搜索必须显式选择目标。
func TestMultiRootProtocol(t *testing.T) {
	api := fixture()
	cs := connectOptions(t, api, Options{WriteRootID: `["temp","note123","other"]`, StateDir: t.TempDir(), Instance: "test"})
	var info InfoOutput
	call(t, cs, "trilium_info", map[string]any{}, &info)
	if info.WriteRootID != "" || !reflect.DeepEqual(info.WriteRootIDs, []string{"note123", "other", "temp"}) || len(info.ConfiguredWriteRootIDs) != 3 {
		t.Fatal(info)
	}
	for _, tc := range []struct {
		name string
		args any
	}{
		{"trilium_create", map[string]any{"operationId": "ambiguous", "title": "x", "content": "x"}},
		{"trilium_search", map[string]any{"query": "x"}},
	} {
		result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: tc.name, Arguments: tc.args})
		if err != nil || !result.IsError {
			t.Fatal(result, err)
		}
	}
	var found NoteList
	call(t, cs, "trilium_search", map[string]any{"query": "x", "ancestorNoteId": "other"}, &found)
	if api.lastSearch.AncestorNoteID != "other" {
		t.Fatal(api.lastSearch)
	}
	var created notes.Result
	call(t, cs, "trilium_create", map[string]any{"operationId": "explicit", "parentNoteId": "other", "title": "x", "content": "x"}, &created)
	if api.notes[created.NoteID].ParentNoteIDs[0] != "other" {
		t.Fatal(created)
	}
	var read ReadOutput
	call(t, cs, "trilium_read", map[string]any{"noteId": created.NoteID}, &read)
	var deleted notes.DeleteResult
	call(t, cs, "trilium_delete", map[string]any{"operationId": "delete-explicit", "noteId": created.NoteID, "expectedSHA256": read.ContentSHA256}, &deleted)
	if !deleted.Deleted {
		t.Fatal(deleted)
	}

	// 单目录数组仍保留默认父级和旧单值字段，嵌套数组不会合并。
	one := connectOptions(t, api, Options{WriteRootID: `["temp","temp"]`, StateDir: t.TempDir(), Instance: "test"})
	call(t, one, "trilium_info", map[string]any{}, &info)
	if info.WriteRootID != "temp" || !reflect.DeepEqual(info.WriteRootIDs, []string{"temp"}) {
		t.Fatal(info)
	}
	call(t, one, "trilium_create", map[string]any{"operationId": "default", "title": "x", "content": "x"}, &created)
	if api.notes[created.NoteID].ParentNoteIDs[0] != "temp" {
		t.Fatal(created)
	}
}
