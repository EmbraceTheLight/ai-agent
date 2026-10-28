package notes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// DeleteInput 指定叶子笔记及读取正文时取得的摘要。
type DeleteInput struct {
	OperationID    string
	NoteID         string
	ExpectedSHA256 string
}

// DeleteResult 表示笔记软删除已由 ETAPI 核对。
type DeleteResult struct {
	OperationID    string `json:"operationId"`
	NoteID         string `json:"noteId"`
	Deleted        bool   `json:"deleted"`
	AlreadyApplied bool   `json:"alreadyApplied"`
}

type deleteIntent struct {
	Fingerprint    string    `json:"fingerprint"`
	NoteID         string    `json:"noteId"`
	ExpectedSHA256 string    `json:"expectedSHA256"`
	Before         *snapshot `json:"before"`
}

type deleteCompletion struct {
	Fingerprint string       `json:"fingerprint"`
	Result      DeleteResult `json:"result"`
}

// Delete 在允许的子树中软删除一个未受保护的叶子 text 或 code 笔记。
// 输入：input.ExpectedSHA256 必须来自最新读取；OperationID 用于持久化意图及结果未知后的恢复。
// 输出：目标确认不存在时返回删除结果；越界、克隆、子笔记或摘要冲突时停止。
// 删除前保存元数据和正文快照；ETAPI 删除及范围检查不是服务器原子操作。
func (s *Service) Delete(ctx context.Context, input DeleteInput) (DeleteResult, error) {
	if !idPattern.MatchString(input.NoteID) || input.NoteID == "root" || input.NoteID == s.rootID || !hashPattern.MatchString(input.ExpectedSHA256) {
		return DeleteResult{}, errors.New("delete requires a writable leaf note ID and its expected SHA-256; the scope root cannot be deleted")
	}
	unlock, err := acquireLock(s.stateDir)
	if err != nil {
		return DeleteResult{}, err
	}
	defer unlock()

	fingerprint, err := s.deleteFingerprint(input)
	if err != nil {
		return DeleteResult{}, err
	}
	op, exists, previous, err := s.loadDelete(input.OperationID, fingerprint)
	if err != nil {
		return DeleteResult{}, err
	}
	if previous != nil {
		return *previous, nil
	}
	if !exists {
		before, err := s.readDeletable(ctx, input.NoteID)
		if err != nil {
			return DeleteResult{}, err
		}
		if ContentSHA256(before.Content) != input.ExpectedSHA256 {
			return DeleteResult{}, errors.New("content conflict; read the note again before deleting it")
		}
		op = deleteIntent{
			Fingerprint:    fingerprint,
			NoteID:         input.NoteID,
			ExpectedSHA256: input.ExpectedSHA256,
			Before:         &before,
		}
		if err := saveRecord(s.recordPath(input.OperationID, ".json"), op); err != nil {
			return DeleteResult{}, err
		}
	}
	if op.Fingerprint != fingerprint || op.NoteID != input.NoteID || op.ExpectedSHA256 != input.ExpectedSHA256 ||
		op.Before == nil || op.Before.Note.NoteID != input.NoteID || ContentSHA256(op.Before.Content) != input.ExpectedSHA256 {
		return DeleteResult{}, errors.New("delete operation record is inconsistent")
	}

	current, err := s.readDeletable(ctx, input.NoteID)
	if isNotFound(err) {
		return s.finishDelete(input.OperationID, op, true)
	}
	if err != nil {
		return DeleteResult{}, err
	}
	if !sameDeletionSnapshot(*op.Before, current) {
		return DeleteResult{}, errors.New("delete target changed after its snapshot; refusing to delete")
	}
	if err := s.api.DeleteNote(ctx, input.NoteID); err != nil {
		// ETAPI may report a missing target on retries; verify absence before treating it as complete.
		if isNotFound(err) {
			_, verifyErr := s.api.GetNote(ctx, input.NoteID)
			if isNotFound(verifyErr) {
				return s.finishDelete(input.OperationID, op, true)
			}
		}
		return DeleteResult{}, fmt.Errorf("operation %s is pending; reuse its ID to verify: %w", input.OperationID, err)
	}
	_, err = s.api.GetNote(ctx, input.NoteID)
	if isNotFound(err) {
		return s.finishDelete(input.OperationID, op, false)
	}
	if err != nil {
		return DeleteResult{}, fmt.Errorf("operation %s is pending; deletion verification failed: %w", input.OperationID, err)
	}
	return DeleteResult{}, errors.New("delete verification failed; operation remains pending because the note is still present")
}

func (s *Service) readDeletable(ctx context.Context, noteID string) (snapshot, error) {
	if err := s.requireScope(ctx, noteID); err != nil {
		return snapshot{}, err
	}
	note, err := s.api.GetNote(ctx, noteID)
	if err != nil {
		return snapshot{}, err
	}
	if note.IsProtected || (note.Type != "text" && note.Type != "code") {
		return snapshot{}, errors.New("delete supports unprotected text and code notes only")
	}
	if len(note.ParentNoteIDs) != 1 {
		return snapshot{}, errors.New("delete requires a note with exactly one parent; cloned notes are not supported")
	}
	if len(note.ChildNoteIDs) != 0 {
		return snapshot{}, errors.New("delete supports leaf notes only; notes with children are not supported")
	}
	body, err := s.api.GetNoteContent(ctx, noteID)
	return snapshot{Note: note, Content: body}, err
}

func sameDeletionSnapshot(before, current snapshot) bool {
	return before.Content == current.Content && reflect.DeepEqual(before.Note, current.Note)
}

func (s *Service) deleteFingerprint(input DeleteInput) (string, error) {
	if !operationPattern.MatchString(input.OperationID) {
		return "", errors.New("operation ID must contain 1–64 ASCII letters, digits, underscores or hyphens")
	}
	encoded, err := json.Marshal([]any{s.instance, s.rootID, "delete", input})
	if err != nil {
		return "", errors.New("cannot fingerprint operation")
	}
	return ContentSHA256(string(encoded)), nil
}

func (s *Service) loadDelete(id, fingerprint string) (deleteIntent, bool, *DeleteResult, error) {
	var done deleteCompletion
	if exists, err := readRecord(s.recordPath(id, ".done.json"), &done); err != nil {
		return deleteIntent{}, false, nil, err
	} else if exists {
		if done.Fingerprint != fingerprint {
			return deleteIntent{}, false, nil, errors.New("operation ID was already used with different parameters")
		}
		done.Result.AlreadyApplied = true
		return deleteIntent{}, true, &done.Result, nil
	}
	var op deleteIntent
	exists, err := readRecord(s.recordPath(id, ".json"), &op)
	if err != nil {
		return deleteIntent{}, false, nil, err
	}
	if exists && op.Fingerprint != fingerprint {
		return deleteIntent{}, false, nil, errors.New("operation ID was already used with different parameters")
	}
	return op, exists, nil, nil
}

func (s *Service) finishDelete(id string, op deleteIntent, replayed bool) (DeleteResult, error) {
	result := DeleteResult{OperationID: id, NoteID: op.NoteID, Deleted: true, AlreadyApplied: replayed}
	if err := saveRecord(s.recordPath(id, ".done.json"), deleteCompletion{Fingerprint: op.Fingerprint, Result: result}); err != nil {
		return DeleteResult{}, err
	}
	return result, nil
}
