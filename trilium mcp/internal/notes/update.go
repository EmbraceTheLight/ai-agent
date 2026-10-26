package notes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// UpdateInput 指定整篇正文替换或单个原始 HTML 片段替换，以及读取时取得的版本摘要。
// Match 为空时替换整篇正文；非空时必须恰好匹配一处，匹配内容不进行 Markdown 转换。
type UpdateInput struct {
	OperationID    string
	NoteID         string
	Content        string
	Format         string
	ExpectedSHA256 string
	Match          string
}

// contentChange 复用追加和更新的持久化、冲突检查与核对流程。
// Input 保持原操作的参数结构以兼容已有指纹；Build 始终以修改前快照生成完整目标正文。
type contentChange struct {
	Kind, OperationID, NoteID, ExpectedSHA256 string
	Input                                     any
	Build                                     func(string) (string, error)
}

// Update 在允许的子树中替换正文，保存修改前快照并读取核对结果。
// 输入：input 提供稳定的操作 ID、当前摘要及非空 UTF-8 新内容；Format 控制新内容的 HTML 转换。
// 输出：成功时返回笔记 ID 和核对摘要；越界、版本冲突或片段匹配不唯一时停止，不覆盖当前内容。
// 片段替换只改动原始 HTML 的唯一匹配区间；不提供清空正文、重命名或移动功能。
func (s *Service) Update(ctx context.Context, input UpdateInput) (Result, error) {
	if !utf8.ValidString(input.Match) || len(input.Match) > 4<<20 {
		return Result{}, errors.New("match must be UTF-8 text of at most 4 MiB")
	}
	replacement, err := renderContent(input.Content, input.Format)
	if err != nil {
		return Result{}, err
	}
	return s.changeContent(ctx, contentChange{
		Kind: "update", OperationID: input.OperationID, NoteID: input.NoteID,
		ExpectedSHA256: input.ExpectedSHA256, Input: input,
		Build: func(original string) (string, error) {
			if input.Match == "" {
				return replacement, nil
			}
			first := strings.Index(original, input.Match)
			if first < 0 {
				return "", errors.New("update match was not found; current content preserved")
			}
			if first != strings.LastIndex(original, input.Match) {
				return "", errors.New("update match is not unique; current content preserved")
			}
			return original[:first] + replacement + original[first+len(input.Match):], nil
		},
	})
}

// changeContent 执行追加与更新共有的备份、版本检查、写入及结果核对。
// 输入：change 包含操作身份、预期摘要和从原文生成目标正文的函数；ctx 控制 API 请求。
// 输出：已完成操作返回历史结果；待核对操作先检查当前内容，只有仍为原文时才尝试写入。
// 保持原有操作指纹结构和记录文件格式；检查与 PUT 之间仍有非原子的并发窗口。
func (s *Service) changeContent(ctx context.Context, change contentChange) (Result, error) {
	if !idPattern.MatchString(change.NoteID) || change.NoteID == s.rootID || !hashPattern.MatchString(change.ExpectedSHA256) {
		return Result{}, fmt.Errorf("%s requires a child note ID and its expected SHA-256; the scope root cannot be edited", change.Kind)
	}
	unlock, err := acquireLock(s.stateDir)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	op, exists, previous, err := s.load(change.OperationID, change.Kind, change.Input)
	if err != nil {
		return Result{}, err
	}
	if previous != nil {
		return *previous, nil
	}
	before, err := s.readWritable(ctx, change.NoteID)
	if err != nil {
		return Result{}, err
	}
	if !exists {
		if ContentSHA256(before.Content) != change.ExpectedSHA256 {
			return Result{}, errors.New("content conflict; read the note again before modifying it")
		}
		desired, err := change.Build(before.Content)
		if err != nil {
			return Result{}, err
		}
		if len(desired) > 8<<20 {
			return Result{}, errors.New("combined content exceeds 8 MiB")
		}
		op.NoteID, op.Before, op.DesiredHash = change.NoteID, &before, ContentSHA256(desired)
		if err := saveRecord(s.recordPath(change.OperationID, ".json"), op); err != nil {
			return Result{}, err
		}
	}
	if op.Before == nil || op.NoteID != change.NoteID || op.Before.Note.NoteID != change.NoteID || ContentSHA256(op.Before.Content) != change.ExpectedSHA256 {
		return Result{}, fmt.Errorf("%s operation record is inconsistent", change.Kind)
	}
	desired, err := change.Build(op.Before.Content)
	if err != nil {
		return Result{}, err
	}
	if len(desired) > 8<<20 || ContentSHA256(desired) != op.DesiredHash {
		return Result{}, errors.New("operation target does not match its saved record")
	}
	currentHash := ContentSHA256(before.Content)
	if currentHash == op.DesiredHash {
		return s.verifyAndFinish(ctx, change.OperationID, op, true)
	}
	if currentHash != change.ExpectedSHA256 {
		return Result{}, errors.New("content conflict; pending operation will not overwrite current content")
	}
	// 备份持久化后再次检查范围和正文，避免覆盖此期间发生的人工编辑。
	latest, err := s.readWritable(ctx, change.NoteID)
	if err != nil {
		return Result{}, err
	}
	if ContentSHA256(latest.Content) != change.ExpectedSHA256 {
		return Result{}, errors.New("content changed after backup; write stopped")
	}
	if err := s.api.SetNoteContent(ctx, change.NoteID, desired); err != nil {
		return Result{}, fmt.Errorf("operation %s is pending; reuse its ID to verify: %w", change.OperationID, err)
	}
	return s.verifyAndFinish(ctx, change.OperationID, op, false)
}
