package notes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"trilium-mcp/internal/trilium"
)

// memoryAPI 模拟已提交但响应丢失、并发编辑及服务端改写，避免用真实笔记验证失败场景。
type memoryAPI struct {
	notes                                               map[string]snapshot
	creates, puts, deletes, contentReads                int
	failCreateAfterCommit, failPutAfterCommit, alterPut bool
	failDeleteAfterCommit                               bool
	failPutBeforeCommit                                 bool
	beforeRead                                          func(int)
}

// GetNote 返回模拟元数据，不存在时使用与真实客户端一致的 404 错误。
func (m *memoryAPI) GetNote(_ context.Context, id string) (trilium.Note, error) {
	if note, ok := m.notes[id]; ok {
		return note.Note, nil
	}
	return trilium.Note{}, &trilium.HTTPError{StatusCode: http.StatusNotFound}
}

// GetNoteContent 按读取次数触发可选的人工编辑，用于验证写前检查。
func (m *memoryAPI) GetNoteContent(_ context.Context, id string) (string, error) {
	m.contentReads++
	if m.beforeRead != nil {
		m.beforeRead(m.contentReads)
	}
	if note, ok := m.notes[id]; ok {
		return note.Content, nil
	}
	return "", &trilium.HTTPError{StatusCode: http.StatusNotFound}
}

// CreateNote 按固定 ID 新建模拟笔记，可在提交后返回错误以模拟响应丢失。
func (m *memoryAPI) CreateNote(_ context.Context, input trilium.CreateNoteInput) error {
	m.creates++
	if _, exists := m.notes[input.NoteID]; exists {
		return errors.New("duplicate note ID")
	}
	m.notes[input.NoteID] = snapshot{Note: trilium.Note{NoteID: input.NoteID, Title: input.Title, Type: "text", MIME: "text/html", ParentNoteIDs: []string{input.ParentNoteID}}, Content: input.Content}
	if m.failCreateAfterCommit {
		return errors.New("simulated lost create response")
	}
	return nil
}

// SetNoteContent 替换模拟正文，支持提交后报错或服务端改写两种失败注入。
func (m *memoryAPI) SetNoteContent(_ context.Context, id, content string) error {
	m.puts++
	if m.failPutBeforeCommit {
		return errors.New("simulated write not committed")
	}
	note := m.notes[id]
	if m.alterPut {
		content += "<!-- server changed content -->"
	}
	note.Content = content
	m.notes[id] = note
	if m.failPutAfterCommit {
		return errors.New("simulated lost append response")
	}
	return nil
}

// DeleteNote 模拟 ETAPI 软删除，并支持删除后响应丢失的恢复场景。
func (m *memoryAPI) DeleteNote(_ context.Context, id string) error {
	m.deletes++
	delete(m.notes, id)
	if m.failDeleteAfterCommit {
		return errors.New("simulated lost delete response")
	}
	return nil
}

// testService 创建独立状态目录和已知笔记树，正文含内部链接以检查原文保留。
func testService(t *testing.T) (*Service, *memoryAPI) {
	t.Helper()
	m := &memoryAPI{notes: map[string]snapshot{
		"root":    {Note: trilium.Note{NoteID: "root", Type: "text", MIME: "text/html"}},
		"temp":    {Note: trilium.Note{NoteID: "temp", Title: "temp", Type: "text", MIME: "text/html", ParentNoteIDs: []string{"root"}}, Content: "scope root unchanged"},
		"note123": {Note: trilium.Note{NoteID: "note123", Title: "existing", Type: "text", MIME: "text/html", ParentNoteIDs: []string{"temp"}}, Content: `<p>原文</p><a href="#root/target">内部链接</a>`},
	}}
	s, err := NewService(m, "temp", t.TempDir(), "test-instance")
	if err != nil {
		t.Fatal(err)
	}
	return s, m
}

// TestCreateAndReplay 验证 Markdown 转换、父级归属和同一操作重试不会重复创建。
func TestCreateAndReplay(t *testing.T) {
	s, api := testService(t)
	input := CreateInput{OperationID: "create-1", ParentNoteID: "temp", Title: "new note", Content: "# 新笔记\n\n正文", Format: "markdown"}
	first, err := s.Create(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	created := api.notes[first.NoteID]
	if !strings.Contains(created.Content, "<h1>") || created.Note.ParentNoteIDs[0] != "temp" || ContentSHA256(created.Content) != first.ContentSHA256 {
		t.Fatal("created note did not match the request")
	}
	replayed, err := s.Create(context.Background(), input)
	if err != nil || !replayed.AlreadyApplied || replayed.NoteID != first.NoteID || api.creates != 1 {
		t.Fatalf("create replay failed: %+v, %v", replayed, err)
	}
	if api.notes["temp"].Content != "scope root unchanged" {
		t.Fatal("scope root body was changed")
	}
	input.Title = "different"
	if _, err := s.Create(context.Background(), input); err == nil || api.creates != 1 {
		t.Fatal("operation ID reuse with changed parameters was accepted")
	}
}

// TestCreateRecoversLostResponse 验证新建提交后响应丢失时，重试先按固定 ID 查询，不再发送 POST。
func TestCreateRecoversLostResponse(t *testing.T) {
	s, api := testService(t)
	api.failCreateAfterCommit = true
	input := CreateInput{OperationID: "lost-create", ParentNoteID: "temp", Title: "lost response", Content: "内容", Format: "text"}
	if _, err := s.Create(context.Background(), input); err == nil {
		t.Fatal("expected simulated transport error")
	}
	result, err := s.Create(context.Background(), input)
	if err != nil || !result.AlreadyApplied || api.creates != 1 {
		t.Fatalf("create recovery failed: %+v, %v", result, err)
	}
}

// TestAppendPreservesOriginalAndBacksUp 验证原 HTML 保持不变、快照完整，并且同一操作只追加一次。
func TestAppendPreservesOriginalAndBacksUp(t *testing.T) {
	s, api := testService(t)
	original := api.notes["note123"].Content
	input := AppendInput{OperationID: "append-1", NoteID: "note123", Content: "**追加**", Format: "markdown", ExpectedSHA256: ContentSHA256(original)}
	first, err := s.Append(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(api.notes["note123"].Content, original+"\n") || !strings.Contains(api.notes["note123"].Content, "<strong>追加</strong>") {
		t.Fatal("append altered the existing body or failed to render Markdown")
	}
	body, err := os.ReadFile(s.recordPath(input.OperationID, ".json"))
	if err != nil {
		t.Fatal(err)
	}
	var op intent
	if err := json.Unmarshal(body, &op); err != nil || op.Before == nil || op.Before.Content != original || op.Before.Note.Title != "existing" {
		t.Fatal("original snapshot was not preserved")
	}
	again, err := s.Append(context.Background(), input)
	if err != nil || !again.AlreadyApplied || api.puts != 1 || first.ContentSHA256 != again.ContentSHA256 {
		t.Fatalf("append replay failed: %+v, %v", again, err)
	}
}

// TestDeleteLeafAndReplay 验证叶子删除前保存快照，重试返回已应用且不再次删除。
func TestDeleteLeafAndReplay(t *testing.T) {
	s, api := testService(t)
	before := api.notes["note123"]
	input := DeleteInput{OperationID: "delete-leaf", NoteID: "note123", ExpectedSHA256: ContentSHA256(before.Content)}
	first, err := s.Delete(context.Background(), input)
	if err != nil || !first.Deleted || first.AlreadyApplied || api.deletes != 1 {
		t.Fatalf("delete failed: %+v, %v", first, err)
	}
	if _, exists := api.notes[input.NoteID]; exists {
		t.Fatal("note still exists after delete")
	}
	body, err := os.ReadFile(s.recordPath(input.OperationID, ".json"))
	if err != nil {
		t.Fatal(err)
	}
	var op deleteIntent
	if err := json.Unmarshal(body, &op); err != nil || op.Before == nil || op.Before.Content != before.Content || op.Before.Note.Title != before.Note.Title {
		t.Fatal("delete intent did not preserve the original note snapshot")
	}
	again, err := s.Delete(context.Background(), input)
	if err != nil || !again.Deleted || !again.AlreadyApplied || api.deletes != 1 {
		t.Fatalf("delete replay failed: %+v, %v", again, err)
	}
}

// TestDeleteRecoversLostResponse 验证删除已提交但响应丢失时，重试确认缺失并完成记录。
func TestDeleteRecoversLostResponse(t *testing.T) {
	s, api := testService(t)
	api.failDeleteAfterCommit = true
	input := DeleteInput{OperationID: "lost-delete", NoteID: "note123", ExpectedSHA256: ContentSHA256(api.notes["note123"].Content)}
	if _, err := s.Delete(context.Background(), input); err == nil {
		t.Fatal("expected simulated response loss")
	}
	api.failDeleteAfterCommit = false
	result, err := s.Delete(context.Background(), input)
	if err != nil || !result.Deleted || !result.AlreadyApplied || api.deletes != 1 {
		t.Fatalf("delete recovery failed: %+v, %v", result, err)
	}
}

// TestDeleteRejectsUnsafeTargets 验证越界、克隆、非叶子、受保护和过期目标均不会删除。
func TestDeleteRejectsUnsafeTargets(t *testing.T) {
	for _, mode := range []string{"outside", "clone", "child", "protected", "unsupported", "stale", "scope-root"} {
		t.Run(mode, func(t *testing.T) {
			s, api := testService(t)
			note := api.notes["note123"]
			expected := ContentSHA256(note.Content)
			switch mode {
			case "outside":
				note.Note.ParentNoteIDs = []string{"root"}
				api.notes["note123"] = note
			case "clone":
				note.Note.ParentNoteIDs = []string{"temp", "temp"}
				api.notes["note123"] = note
			case "child":
				note.Note.ChildNoteIDs = []string{"child1"}
				api.notes["note123"] = note
			case "protected":
				note.Note.IsProtected = true
				api.notes["note123"] = note
			case "unsupported":
				note.Note.Type = "image"
				api.notes["note123"] = note
			case "stale":
				expected = ContentSHA256("older body")
			case "scope-root":
				noteID := "temp"
				expected = ContentSHA256(api.notes[noteID].Content)
			}
			noteID := "note123"
			if mode == "scope-root" {
				noteID = "temp"
			}
			_, err := s.Delete(context.Background(), DeleteInput{OperationID: mode, NoteID: noteID, ExpectedSHA256: expected})
			if err == nil || api.deletes != 0 {
				t.Fatal("unsafe delete target was accepted")
			}
		})
	}
}

// TestDeleteStopsWhenTargetChangesAfterSnapshot 验证持久化快照后发生编辑时不会继续删除。
func TestDeleteStopsWhenTargetChangesAfterSnapshot(t *testing.T) {
	s, api := testService(t)
	api.beforeRead = func(count int) {
		if count == 2 {
			note := api.notes["note123"]
			note.Content = "concurrent edit"
			api.notes["note123"] = note
		}
	}
	input := DeleteInput{OperationID: "delete-race", NoteID: "note123", ExpectedSHA256: ContentSHA256(api.notes["note123"].Content)}
	if _, err := s.Delete(context.Background(), input); err == nil || api.deletes != 0 {
		t.Fatal("delete ignored an edit made after its snapshot")
	}
}

// TestAppendRecoversLostResponse 验证追加已提交但响应丢失后，摘要核对可确认结果，不再次发送 PUT。
func TestAppendRecoversLostResponse(t *testing.T) {
	s, api := testService(t)
	api.failPutAfterCommit = true
	input := AppendInput{OperationID: "lost-append", NoteID: "note123", Content: "added", Format: "text", ExpectedSHA256: ContentSHA256(api.notes["note123"].Content)}
	if _, err := s.Append(context.Background(), input); err == nil {
		t.Fatal("expected simulated response loss")
	}
	result, err := s.Append(context.Background(), input)
	if err != nil || !result.AlreadyApplied || api.puts != 1 {
		t.Fatalf("append recovery failed: %+v, %v", result, err)
	}
}

// TestAppendRejectsStaleAndConcurrentEdits 验证旧摘要及备份期间发生的人工编辑均不会被覆盖。
func TestAppendRejectsStaleAndConcurrentEdits(t *testing.T) {
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
		_, err := s.Append(context.Background(), AppendInput{OperationID: "conflict", NoteID: "note123", Content: "added", Format: "text", ExpectedSHA256: expected})
		if err == nil || api.puts != 0 {
			t.Fatal("conflicting write was sent to API")
		}
		if concurrent && api.notes["note123"].Content != "manual edit" {
			t.Fatal("manual edit was overwritten")
		}
	}
}

// TestScopeRejectsOutsideClonesCyclesAndProtectedNotes 验证越界父级、跨边界克隆、循环及受保护笔记全部被拒绝。
func TestScopeRejectsOutsideClonesCyclesAndProtectedNotes(t *testing.T) {
	for _, mode := range []string{"outside", "clone", "cycle", "protected", "code"} {
		t.Run(mode, func(t *testing.T) {
			s, api := testService(t)
			note := api.notes["note123"]
			switch mode {
			case "outside":
				note.Note.ParentNoteIDs = []string{"root"}
			case "clone":
				note.Note.ParentNoteIDs = []string{"temp", "root"}
			case "cycle":
				note.Note.ParentNoteIDs = []string{"note123"}
			case "protected":
				note.Note.IsProtected = true
			case "code":
				note.Note.Type = "code"
			}
			api.notes["note123"] = note
			_, err := s.Append(context.Background(), AppendInput{OperationID: mode, NoteID: "note123", Content: "new", Format: "text", ExpectedSHA256: ContentSHA256(note.Content)})
			if err == nil || api.puts != 0 {
				t.Fatal("unsafe append was accepted")
			}
		})
	}
	s, api := testService(t)
	if _, err := s.Create(context.Background(), CreateInput{OperationID: "outside-create", ParentNoteID: "root", Title: "no", Content: "no", Format: "text"}); err == nil || api.creates != 0 {
		t.Fatal("outside create accepted")
	}
	if _, err := s.Append(context.Background(), AppendInput{OperationID: "root-append", NoteID: "temp", Content: "no", Format: "text", ExpectedSHA256: ContentSHA256(api.notes["temp"].Content)}); err == nil || api.puts != 0 {
		t.Fatal("scope root append accepted")
	}
}

// TestVerificationFailureNeverRepeatsAppend 验证服务端改写导致核对失败时，重试保留当前内容且不重复写入。
func TestVerificationFailureNeverRepeatsAppend(t *testing.T) {
	s, api := testService(t)
	api.alterPut = true
	input := AppendInput{OperationID: "verify-failed", NoteID: "note123", Content: "new", Format: "text", ExpectedSHA256: ContentSHA256(api.notes["note123"].Content)}
	if _, err := s.Append(context.Background(), input); err == nil {
		t.Fatal("expected verification mismatch")
	}
	if _, err := s.Append(context.Background(), input); err == nil || api.puts != 1 {
		t.Fatal("mismatched pending write was repeated")
	}
}

// TestStateFailuresStopWrites 验证备份目录不可用、锁冲突、记录损坏和非法操作 ID 时不发送写请求。
func TestStateFailuresStopWrites(t *testing.T) {
	s, api := testService(t)
	input := CreateInput{OperationID: "safe-op", ParentNoteID: "temp", Title: "new", Content: "new", Format: "text"}
	unlock, err := acquireLock(s.stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), input); err == nil {
		t.Fatal("lock conflict accepted")
	}
	unlock()
	if err := os.WriteFile(s.recordPath(input.OperationID, ".json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), input); err == nil {
		t.Fatal("corrupt record accepted")
	}
	input.OperationID = "../escape"
	if _, err := s.Create(context.Background(), input); err == nil {
		t.Fatal("unsafe operation ID accepted")
	}
	file := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	s.stateDir = file
	if _, err := s.Create(context.Background(), input); err == nil {
		t.Fatal("invalid state directory accepted")
	}
	if api.creates != 0 || api.puts != 0 {
		t.Fatal("API write occurred without a durable intent")
	}
}

// TestRenderFormatsAndBoundaries 验证普通文本转义、Markdown 安全默认值及非法格式和空输入边界。
func TestRenderFormatsAndBoundaries(t *testing.T) {
	text, err := renderContent("<tag>\r\n&", "text")
	if err != nil || text != "<p>&lt;tag&gt;<br>\n&amp;</p>" {
		t.Fatalf("unexpected plain-text rendering: %q, %v", text, err)
	}
	markdown, err := renderContent("<script>alert(1)</script>", "markdown")
	if err != nil || strings.Contains(markdown, "<script>") {
		t.Fatal("Markdown unexpectedly enabled raw HTML")
	}
	for _, input := range []struct{ content, format string }{{" ", "markdown"}, {"x", "unknown"}, {string([]byte{0xff}), "text"}} {
		if _, err := renderContent(input.content, input.format); err == nil {
			t.Error("invalid content accepted")
		}
	}
}
