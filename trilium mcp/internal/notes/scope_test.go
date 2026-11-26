package notes

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"trilium-mcp/internal/trilium"
)

// TestParseWriteRoots 验证单 ID 兼容、数组去重排序及配置整体拒绝。
func TestParseWriteRoots(t *testing.T) {
	for _, value := range []string{"temp", `["temp"]`, `["temp","temp"]`} {
		roots, err := ParseWriteRoots(value)
		if err != nil || !reflect.DeepEqual(roots, []string{"temp"}) {
			t.Fatal(roots, err)
		}
	}
	roots, err := ParseWriteRoots(`["zzzz","aaaa"]`)
	if err != nil || !reflect.DeepEqual(roots, []string{"aaaa", "zzzz"}) {
		t.Fatal(roots, err)
	}
	if roots, err := ParseWriteRoots(""); err != nil || len(roots) != 0 {
		t.Fatal(roots, err)
	}
	for _, value := range []string{`[]`, `["temp","root"]`, `["temp",null]`, `["temp",3]`, `["temp",]`, `"temp"`, `["../bad"]`, `{"id":"temp"}`} {
		if _, err := ParseWriteRoots(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
	data, _ := json.Marshal(make([]string, 65))
	if _, err := ParseWriteRoots(string(data)); err == nil {
		t.Fatal("too many roots accepted")
	}
}

// addScopeNote 构造具有明确父路径的模拟富文本笔记。
func addScopeNote(api *memoryAPI, id string, parents ...string) {
	api.notes[id] = snapshot{Note: trilium.Note{NoteID: id, Title: id, Type: "text", MIME: "text/html", ParentNoteIDs: parents}, Content: "<p>body</p>"}
}

// TestDeleteRecoveryRequiresTarget404 验证待核对删除遇到祖先 404 时，不把仍存在的目标误记为已删除。
func TestDeleteRecoveryRequiresTarget404(t *testing.T) {
	s, api := testService(t)
	before := api.notes["note123"]
	input := DeleteInput{OperationID: "pending-missing-parent", NoteID: "note123", ExpectedSHA256: ContentSHA256(before.Content)}
	fingerprint, err := s.deleteFingerprint(input)
	if err != nil {
		t.Fatal(err)
	}
	op := deleteIntent{Fingerprint: fingerprint, NoteID: input.NoteID, ExpectedSHA256: input.ExpectedSHA256, Before: &before}
	if err := saveRecord(s.recordPath(input.OperationID, ".json"), op); err != nil {
		t.Fatal(err)
	}
	delete(api.notes, "temp")
	if _, err := s.Delete(context.Background(), input); err == nil {
		t.Fatal("ancestor 404 incorrectly completed deletion")
	}
	if _, err := os.Stat(s.recordPath(input.OperationID, ".done.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("incorrect completion marker", err)
	}
	if api.deletes != 0 {
		t.Fatal("delete request sent without a verified scope")
	}
}
