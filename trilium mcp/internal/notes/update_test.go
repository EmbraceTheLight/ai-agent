package notes

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestUpdateWholeBodyAndReplay 验证整篇更新替换原文、保存完整快照，重复操作不重写且不改变标题。
func TestUpdateWholeBodyAndReplay(t *testing.T) {
	s, api := testService(t)
	original := api.notes["note123"]
	input := UpdateInput{OperationID: "update-full", NoteID: "note123", Content: "<h2>更新</h2>\n<p>新正文</p>", Format: "html", ExpectedSHA256: ContentSHA256(original.Content)}
	first, err := s.Update(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if api.notes["note123"].Content != input.Content || first.ContentSHA256 != ContentSHA256(input.Content) || api.notes["note123"].Note.Title != original.Note.Title {
		t.Fatal("whole-body update changed unexpected data")
	}
	body, err := os.ReadFile(s.recordPath(input.OperationID, ".json"))
	if err != nil {
		t.Fatal(err)
	}
	var op intent
	if err := json.Unmarshal(body, &op); err != nil || op.Before == nil || op.Before.Content != original.Content || op.Before.Note.Title != original.Note.Title {
		t.Fatal("update backup did not preserve the original")
	}
	again, err := s.Update(context.Background(), input)
	if err != nil || !again.AlreadyApplied || api.puts != 1 || again.ContentSHA256 != first.ContentSHA256 {
		t.Fatalf("update replay failed: %+v, %v", again, err)
	}
	input.Content += "changed"
	if _, err := s.Update(context.Background(), input); err == nil || api.puts != 1 {
		t.Fatal("operation ID accepted changed update parameters")
	}
}

// TestUpdateFragmentPreservesRichHTML 验证仅替换唯一原始 HTML 片段，内部链接、图片引用及列表逐字节保留。
func TestUpdateFragmentPreservesRichHTML(t *testing.T) {
	s, api := testService(t)
	const prefix = `<a href="#root/target">内部链接</a><img src="api/attachments/image123/image">` + "\r\n"
	const suffix = "\r\n" + `<ul><li>保留列表</li></ul>`
	note := api.notes["note123"]
	note.Content = prefix + "<p>旧段落</p>" + suffix
	api.notes["note123"] = note
	input := UpdateInput{OperationID: "update-fragment", NoteID: "note123", Match: "<p>旧段落</p>", Content: "**新段落**", Format: "markdown", ExpectedSHA256: ContentSHA256(note.Content)}
	if _, err := s.Update(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	want := prefix + "<p><strong>新段落</strong></p>\n" + suffix
	if api.notes["note123"].Content != want || api.puts != 1 {
		t.Fatal("fragment update altered surrounding HTML")
	}
}

// TestUpdateRejectsMissingOrAmbiguousMatch 验证缺失、多处及重叠匹配均拒绝写入，也不创建待执行记录。
func TestUpdateRejectsMissingOrAmbiguousMatch(t *testing.T) {
	for _, tc := range []struct{ body, match string }{
		{"<p>other</p>", "<p>missing</p>"},
		{"<p>same</p><p>same</p>", "<p>same</p>"},
		{"aaa", "aa"},
	} {
		s, api := testService(t)
		note := api.notes["note123"]
		note.Content = tc.body
		api.notes["note123"] = note
		input := UpdateInput{OperationID: "bad-match", NoteID: "note123", Match: tc.match, Content: "new", Format: "html", ExpectedSHA256: ContentSHA256(tc.body)}
		if _, err := s.Update(context.Background(), input); err == nil || api.puts != 0 || api.notes["note123"].Content != tc.body {
			t.Fatal("invalid match changed the body")
		}
		if _, err := os.Stat(s.recordPath(input.OperationID, ".json")); !os.IsNotExist(err) {
			t.Fatal("invalid match created a pending operation")
		}
	}
}

// TestUpdateRecoversLostResponseForBothModes 验证整篇和片段更新提交后响应丢失时，重试按目标摘要确认且不再次 PUT。
func TestUpdateRecoversLostResponseForBothModes(t *testing.T) {
	for _, fragment := range []bool{false, true} {
		s, api := testService(t)
		api.failPutAfterCommit = true
		input := UpdateInput{OperationID: "lost-update", NoteID: "note123", Content: "<p>new</p>", Format: "html", ExpectedSHA256: ContentSHA256(api.notes["note123"].Content)}
		if fragment {
			input.Match = "<p>原文</p>"
		}
		if _, err := s.Update(context.Background(), input); err == nil {
			t.Fatal("expected simulated response loss")
		}
		result, err := s.Update(context.Background(), input)
		if err != nil || !result.AlreadyApplied || api.puts != 1 {
			t.Fatalf("update recovery failed: %+v, %v", result, err)
		}
	}
}

// TestUpdateRetriesWhenOriginalIsStillPresent 验证请求未提交且服务器仍为原文时，相同操作可继续写入。
func TestUpdateRetriesWhenOriginalIsStillPresent(t *testing.T) {
	s, api := testService(t)
	original := api.notes["note123"].Content
	api.failPutBeforeCommit = true
	input := UpdateInput{OperationID: "update-not-committed", NoteID: "note123", Content: "<p>new</p>", Format: "html", ExpectedSHA256: ContentSHA256(original)}
	if _, err := s.Update(context.Background(), input); err == nil || api.notes["note123"].Content != original {
		t.Fatal("expected uncommitted update failure")
	}
	api.failPutBeforeCommit = false
	if _, err := s.Update(context.Background(), input); err != nil || api.notes["note123"].Content != input.Content || api.puts != 2 {
		t.Fatalf("pending update did not resume: %v", err)
	}
}

// TestUpdateRejectsStaleAndConcurrentEdits 验证旧摘要及备份期间的人工编辑不会被更新覆盖。
func TestUpdateRejectsStaleAndConcurrentEdits(t *testing.T) {
	for _, concurrent := range []bool{false, true} {
		s, api := testService(t)
		expected := ContentSHA256("stale")
		if concurrent {
			expected = ContentSHA256(api.notes["note123"].Content)
			api.beforeRead = func(count int) {
				if count == 2 {
					note := api.notes["note123"]
					note.Content = "manual edit"
					api.notes["note123"] = note
				}
			}
		}
		_, err := s.Update(context.Background(), UpdateInput{OperationID: "update-conflict", NoteID: "note123", Content: "<p>new</p>", Format: "html", ExpectedSHA256: expected})
		if err == nil || api.puts != 0 {
			t.Fatal("conflicting update was sent")
		}
		if concurrent && api.notes["note123"].Content != "manual edit" {
			t.Fatal("manual edit was overwritten")
		}
	}
}

// TestUpdateEnforcesScopeAndNoteType 验证越界、克隆、循环、保护、类型限制和范围根正文保护同样适用于更新。
func TestUpdateEnforcesScopeAndNoteType(t *testing.T) {
	for _, mode := range []string{"outside", "clone", "cycle", "protected", "code", "root"} {
		t.Run(mode, func(t *testing.T) {
			s, api := testService(t)
			id := "note123"
			note := api.notes[id]
			switch mode {
			case "outside":
				note.Note.ParentNoteIDs = []string{"root"}
			case "clone":
				note.Note.ParentNoteIDs = []string{"temp", "root"}
			case "cycle":
				note.Note.ParentNoteIDs = []string{id}
			case "protected":
				note.Note.IsProtected = true
			case "code":
				note.Note.Type = "code"
			case "root":
				id = "temp"
				note = api.notes[id]
			}
			api.notes[id] = note
			_, err := s.Update(context.Background(), UpdateInput{OperationID: mode, NoteID: id, Content: "<p>new</p>", Format: "html", ExpectedSHA256: ContentSHA256(note.Content)})
			if err == nil || api.puts != 0 || api.notes[id].Content != note.Content {
				t.Fatal("unsafe update was accepted")
			}
		})
	}
}

// TestUpdateNoOpAndInvalidInput 验证相同正文无需 PUT，空正文及非法匹配编码不会触发写入。
func TestUpdateNoOpAndInvalidInput(t *testing.T) {
	s, api := testService(t)
	original := api.notes["note123"].Content
	input := UpdateInput{OperationID: "no-change", NoteID: "note123", Content: original, Format: "html", ExpectedSHA256: ContentSHA256(original)}
	result, err := s.Update(context.Background(), input)
	if err != nil || !result.AlreadyApplied || api.puts != 0 {
		t.Fatalf("no-op update performed a write: %v", err)
	}
	input.OperationID = "empty-update"
	input.Content = ""
	if _, err := s.Update(context.Background(), input); err == nil {
		t.Fatal("empty replacement accepted")
	}
	input.Content = "new"
	input.Match = string([]byte{0xff})
	if _, err := s.Update(context.Background(), input); err == nil || api.puts != 0 {
		t.Fatal("invalid match encoding accepted")
	}
}

// TestPendingUpdatePreservesUnexpectedServerContent 验证写后内容不符及后续人工编辑时，重试不会强行再次替换。
func TestPendingUpdatePreservesUnexpectedServerContent(t *testing.T) {
	s, api := testService(t)
	api.alterPut = true
	input := UpdateInput{OperationID: "update-verify-failed", NoteID: "note123", Content: "<p>new</p>", Format: "html", ExpectedSHA256: ContentSHA256(api.notes["note123"].Content)}
	if _, err := s.Update(context.Background(), input); err == nil {
		t.Fatal("expected verification mismatch")
	}
	before := api.notes["note123"].Content
	if _, err := s.Update(context.Background(), input); err == nil || api.puts != 1 || api.notes["note123"].Content != before {
		t.Fatal("pending update overwrote unexpected content")
	}
}

// TestOperationIDCannotSwitchFromAppendToUpdate 验证已有追加操作 ID 不能被重新解释为更新。
func TestOperationIDCannotSwitchFromAppendToUpdate(t *testing.T) {
	s, api := testService(t)
	expected := ContentSHA256(api.notes["note123"].Content)
	if _, err := s.Append(context.Background(), AppendInput{OperationID: "shared-id", NoteID: "note123", Content: "x", Format: "html", ExpectedSHA256: expected}); err != nil {
		t.Fatal(err)
	}
	_, err := s.Update(context.Background(), UpdateInput{OperationID: "shared-id", NoteID: "note123", Content: "x", Format: "html", ExpectedSHA256: expected})
	if err == nil || !strings.Contains(err.Error(), "different parameters") || api.puts != 1 {
		t.Fatal("operation kind changed during replay")
	}
}
