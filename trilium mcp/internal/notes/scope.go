package notes

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
)

// ParseWriteRoots 解析单个 ID 或 JSON 字符串数组，排序去重且拒绝全库根。
// 输入：value 来自 TRILIUM_WRITE_ROOT_ID；最多配置 64 项，空值表示只读。
// 输出：独立的有序 ID 列表；格式错误时拒绝整个配置，不接受部分有效值。
func ParseWriteRoots(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}, nil
	}
	roots := []string{value}
	if strings.HasPrefix(value, "[") {
		if json.Unmarshal([]byte(value), &roots) != nil || len(roots) == 0 || len(roots) > 64 {
			return nil, errors.New("TRILIUM_WRITE_ROOT_ID requires a note ID or a nonempty JSON array of at most 64 IDs")
		}
	}
	for _, id := range roots {
		if !idPattern.MatchString(id) || id == "root" {
			return nil, errors.New("TRILIUM_WRITE_ROOT_ID contains an invalid ID or root")
		}
	}
	slices.Sort(roots)
	return slices.Compact(roots), nil
}

// scopeKey 保留单目录的旧指纹格式，多目录使用规范化 JSON，数组顺序不影响重试。
func scopeKey(roots []string) string {
	if len(roots) == 1 {
		return roots[0]
	}
	data, _ := json.Marshal(roots)
	return string(data)
}

// RootIDs 返回排序去重后的配置副本，嵌套目录保持独立。
func (s *Service) RootIDs() []string { return slices.Clone(s.roots) }

// isScopeRoot 识别不可修改正文或删除的范围边界。
func (s *Service) isScopeRoot(id string) bool { return slices.Contains(s.roots, id) }

// requireScope 校验目标所有父级路径均位于允许范围内，不参与选择锁。
// 输入：noteID 可为范围根；最多读取 256 个节点。
// 输出：范围外路径、保护状态或循环均拒绝；应在取得笔记锁后及写入前调用。
func (s *Service) requireScope(ctx context.Context, noteID string) error {
	states := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		if states[id] == 1 {
			return errors.New("cycle detected while checking write scope")
		}
		if states[id] == 2 {
			return nil
		}
		if len(states) >= 256 {
			return errors.New("write scope graph exceeds traversal limit")
		}
		states[id] = 1
		note, err := s.api.GetNote(ctx, id)
		if err != nil {
			return err
		}
		if note.IsProtected {
			return errors.New("protected notes are outside the writable scope")
		}
		if !s.isScopeRoot(id) {
			if id == "root" || len(note.ParentNoteIDs) == 0 {
				return errors.New("note is outside the configured write subtrees")
			}
			for _, parent := range note.ParentNoteIDs {
				if err := visit(parent); err != nil {
					return err
				}
			}
		}
		states[id] = 2
		return nil
	}
	return visit(noteID)
}
