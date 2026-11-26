package notes

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// TestNestedRootsRemainConfigured 验证嵌套 ID 不合并，全部配置边界保持不可修改正文或删除。
func TestNestedRootsRemainConfigured(t *testing.T) {
	_, api := testService(t)
	addScopeNote(api, "leaf", "note123")
	s, err := NewService(api, `["temp","note123","temp"]`, t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s.RootIDs(), []string{"note123", "temp"}) {
		t.Fatal(s.RootIDs())
	}
	if _, err := s.Create(context.Background(), CreateInput{OperationID: "ambiguous", Title: "x", Content: "x", Format: "text"}); err == nil {
		t.Fatal("ambiguous parent accepted")
	}
	for _, id := range s.RootIDs() {
		if _, err := s.Update(context.Background(), UpdateInput{OperationID: "update-" + id, NoteID: id, Content: "x", Format: "text", ExpectedSHA256: ContentSHA256(api.notes[id].Content)}); err == nil {
			t.Fatal("configured boundary edited", id)
		}
		if _, err := s.Delete(context.Background(), DeleteInput{OperationID: "delete-" + id, NoteID: id, ExpectedSHA256: ContentSHA256(api.notes[id].Content)}); err == nil {
			t.Fatal("configured boundary deleted", id)
		}
	}
	if _, err := s.Append(context.Background(), AppendInput{OperationID: "leaf", NoteID: "leaf", Content: "x", Format: "text", ExpectedSHA256: ContentSHA256(api.notes["leaf"].Content)}); err != nil {
		t.Fatal(err)
	}
}

// TestNoteScopeAndClones 验证允许范围内的克隆通过检查，范围外路径、缺失、保护和循环仍被拒绝。
func TestNoteScopeAndClones(t *testing.T) {
	for _, mode := range []string{"allowed-clone", "outside-clone", "missing", "protected", "cycle"} {
		t.Run(mode, func(t *testing.T) {
			_, api := testService(t)
			addScopeNote(api, "side", "root")
			s, err := NewService(api, `["temp","side"]`, t.TempDir(), "test")
			if err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "allowed-clone":
				addScopeNote(api, "note123", "temp", "side")
			case "outside-clone":
				addScopeNote(api, "note123", "temp", "root")
			case "missing":
				delete(api.notes, "note123")
			case "protected":
				n := api.notes["note123"]
				n.Note.IsProtected = true
				api.notes["note123"] = n
			case "cycle":
				addScopeNote(api, "note123", "loop")
				addScopeNote(api, "loop", "note123")
			}
			err = s.requireScope(context.Background(), "note123")
			if (err == nil) != (mode == "allowed-clone") {
				t.Fatal(mode, err)
			}
		})
	}
}

// TestSiblingNotesAndCloneLock 验证同一目录内不同笔记可写，克隆位置不改变同一 noteId 的互斥关系。
func TestSiblingNotesAndCloneLock(t *testing.T) {
	s, api := testService(t)
	addScopeNote(api, "sibling", "temp")
	unlock, err := s.lockTarget(context.Background(), "note123")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := s.Append(context.Background(), AppendInput{OperationID: "sibling", NoteID: "sibling", Content: "x", Format: "text", ExpectedSHA256: ContentSHA256(api.notes["sibling"].Content)}); err != nil {
		t.Fatal(err)
	}
	addScopeNote(api, "side", "root")
	addScopeNote(api, "note123", "temp", "side")
	other, err := NewService(api, `["side","temp"]`, s.stateDir, "test")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_, err = other.lockTarget(ctx, "note123")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("clone bypassed note lock", err)
	}
	// 移到范围外后，取得的笔记锁不替代权限检查。
	addScopeNote(api, "note123", "root")
	if err := s.requireScope(context.Background(), "note123"); err == nil {
		t.Fatal("outside scope accepted under lock")
	}
}

// TestOperationLockAndStableScopeKey 验证操作 ID 独立互斥，原单 ID 及重排数组的记录仍可重放。
func TestOperationLockAndStableScopeKey(t *testing.T) {
	s, api := testService(t)
	input := CreateInput{OperationID: "old-single", ParentNoteID: "temp", Title: "x", Content: "x", Format: "text"}
	first, err := s.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	array, err := NewService(api, `["temp","temp"]`, s.stateDir, "test-instance")
	if err != nil {
		t.Fatal(err)
	}
	replay, err := array.Create(context.Background(), input)
	if err != nil || !replay.AlreadyApplied || replay.NoteID != first.NoteID {
		t.Fatal(replay, err)
	}
	addScopeNote(api, "side", "root")
	a, err := NewService(api, `["temp","side"]`, s.stateDir, "test-instance")
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewService(api, `["side","temp","side"]`, s.stateDir, "test-instance")
	if err != nil {
		t.Fatal(err)
	}
	input.OperationID = "ordered"
	if _, err := a.Create(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if got, err := b.Create(context.Background(), input); err != nil || !got.AlreadyApplied {
		t.Fatal(got, err)
	}
	unlock, err := a.lockOperation(context.Background(), "same-id")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	if _, err := b.lockOperation(ctx, "same-id"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}
