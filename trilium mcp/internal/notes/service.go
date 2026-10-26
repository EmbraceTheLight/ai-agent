// Package notes 在指定子树内执行带备份、版本检查和结果核对的笔记写入。
package notes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"trilium-mcp/internal/trilium"
)

var (
	idPattern        = regexp.MustCompile(`^[a-zA-Z0-9_]{4,32}$`)
	operationPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	hashPattern      = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

// API 是写入业务所需的 ETAPI 能力，便于使用模拟实现验证失败恢复。
type API interface {
	GetNote(context.Context, string) (trilium.Note, error)
	GetNoteContent(context.Context, string) (string, error)
	CreateNote(context.Context, trilium.CreateNoteInput) error
	SetNoteContent(context.Context, string, string) error
}

// Service 将写入限制在 rootID 下，使用 stateDir 保存操作记录和修改前快照。
// 同一实例必须使用同一状态目录，才能识别重试；该限制不替代 Trilium 本身的权限控制。
type Service struct {
	api                        API
	rootID, stateDir, instance string
}

// CreateInput 指定新笔记的父级、标题、内容格式及稳定的操作 ID。
type CreateInput struct {
	OperationID  string
	ParentNoteID string
	Title        string
	Content      string
	Format       string
}

// AppendInput 指定目标笔记、增量内容及调用方读取正文后得到的摘要。
type AppendInput struct {
	OperationID    string
	NoteID         string
	Content        string
	Format         string
	ExpectedSHA256 string
}

// Result 返回操作标识及已核对正文的摘要，不包含正文或 Token。
// AlreadyApplied 表示目标效果已存在或此操作先前已完成；结果描述核对时刻，不保证笔记之后未被编辑。
type Result struct {
	OperationID    string `json:"operationId"`
	NoteID         string `json:"noteId"`
	ContentSHA256  string `json:"contentSHA256"`
	AlreadyApplied bool   `json:"alreadyApplied"`
}

// NewService 创建具有明确写入边界的服务，不发起网络请求。
// 输入：api 为 ETAPI 实现；rootID 为允许的子树根，不能是 root；stateDir 为备份目录；instance 为实例标识。
// 输出：配置有效时返回服务，否则返回错误；只在实际写入时创建状态目录。
func NewService(api API, rootID, stateDir, instance string) (*Service, error) {
	if api == nil || !idPattern.MatchString(rootID) || rootID == "root" || strings.TrimSpace(stateDir) == "" || instance == "" {
		return nil, errors.New("writes require a non-root TRILIUM_WRITE_ROOT_ID, state directory and instance")
	}
	return &Service{api: api, rootID: rootID, stateDir: stateDir, instance: instance}, nil
}

// Create 在允许的子树中创建富文本笔记，预先记录固定笔记 ID 并读取核对结果。
// 输入：ctx 控制请求；input.OperationID 在同一状态目录内必须稳定且唯一。
// 输出：成功时返回 ID 和正文摘要；失败时保留操作记录，使用相同参数及操作 ID 重试可先查询已有结果。
func (s *Service) Create(ctx context.Context, input CreateInput) (Result, error) {
	if !idPattern.MatchString(input.ParentNoteID) || strings.TrimSpace(input.Title) == "" || !utf8.ValidString(input.Title) {
		return Result{}, errors.New("create requires a valid parent ID and nonempty title")
	}
	content, err := renderContent(input.Content, input.Format)
	if err != nil {
		return Result{}, err
	}
	unlock, err := acquireLock(s.stateDir)
	if err != nil {
		return Result{}, err
	}
	defer unlock()
	op, exists, previous, err := s.load(input.OperationID, "create", input)
	if err != nil {
		return Result{}, err
	}
	if previous != nil {
		return *previous, nil
	}
	if err := s.requireScope(ctx, input.ParentNoteID); err != nil {
		return Result{}, err
	}
	if !exists {
		var random [12]byte
		if _, err := rand.Read(random[:]); err != nil {
			return Result{}, errors.New("cannot allocate note ID")
		}
		op.NoteID, op.DesiredHash = hex.EncodeToString(random[:]), ContentSHA256(content)
		if err := saveRecord(s.recordPath(input.OperationID, ".json"), op); err != nil {
			return Result{}, err
		}
	}
	note, err := s.api.GetNote(ctx, op.NoteID)
	if isNotFound(err) {
		if err := s.api.CreateNote(ctx, trilium.CreateNoteInput{ParentNoteID: input.ParentNoteID, NoteID: op.NoteID, Title: input.Title, Content: content}); err != nil {
			return Result{}, fmt.Errorf("operation %s is pending; reuse its ID to verify: %w", input.OperationID, err)
		}
	} else if err != nil {
		return Result{}, err
	} else if note.Title != input.Title {
		return Result{}, errors.New("allocated note ID exists with different metadata; refusing to overwrite")
	}
	created, readErr := s.api.GetNote(ctx, op.NoteID)
	if readErr != nil {
		return Result{}, readErr
	}
	parentMatches := false
	for _, parent := range created.ParentNoteIDs {
		if parent == input.ParentNoteID {
			parentMatches = true
		}
	}
	if created.Title != input.Title || !parentMatches {
		return Result{}, errors.New("created note metadata does not match the requested parent and title")
	}
	return s.verifyAndFinish(ctx, input.OperationID, op, !isNotFound(err))
}

// Append 在原始 HTML 末尾追加新片段，并在写入前保存完整快照。
// 输入：input.ExpectedSHA256 必须与修改前正文一致；OperationID 用于识别重复请求及未知结果。
// 输出：成功时返回已核对的摘要；超出范围、版本冲突或写入失败时报错，不自动覆盖冲突内容。
// ETAPI 没有在此实现条件写入，因此最后一次检查与实际写入之间仍存在并发时间窗口。
func (s *Service) Append(ctx context.Context, input AppendInput) (Result, error) {
	fragment, err := renderContent(input.Content, input.Format)
	if err != nil {
		return Result{}, err
	}
	return s.changeContent(ctx, contentChange{
		Kind: "append", OperationID: input.OperationID, NoteID: input.NoteID,
		ExpectedSHA256: input.ExpectedSHA256, Input: input,
		Build: func(original string) (string, error) { return original + "\n" + fragment, nil },
	})
}

// requireScope 校验目标的所有父级路径均落在允许子树内，拒绝跨边界克隆、循环及受保护笔记。
// 输入：noteID 可以是子树根或其后代；最多读取 256 个不同节点的元数据。
// 输出：范围可确认时返回 nil；不能确认时拒绝写入，不下载正文。
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
		if id != s.rootID {
			if len(note.ParentNoteIDs) == 0 || id == "root" {
				return errors.New("note is outside the configured write subtree")
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

// readWritable 在范围检查后读取可修改的 HTML 笔记快照。
// 输入：noteID 为目标笔记；ctx 控制全部元数据及正文请求。
// 输出：返回元数据和原文；类型、MIME、保护状态或范围不满足时返回错误。
func (s *Service) readWritable(ctx context.Context, noteID string) (snapshot, error) {
	if err := s.requireScope(ctx, noteID); err != nil {
		return snapshot{}, err
	}
	note, err := s.api.GetNote(ctx, noteID)
	if err != nil {
		return snapshot{}, err
	}
	if note.IsProtected || note.Type != "text" || note.MIME != "text/html" {
		return snapshot{}, errors.New("writes support unprotected text/html notes only")
	}
	body, err := s.api.GetNoteContent(ctx, noteID)
	return snapshot{Note: note, Content: body}, err
}

// load 校验操作 ID 与参数指纹，并读取不可变意图或已完成结果。
// 输入：kind 区分新建、追加和更新；input、实例及范围共同参与指纹，不记录 Token。
// 输出：已完成时返回历史结果；同一 ID 被用于其他参数或记录损坏时拒绝继续。
func (s *Service) load(id, kind string, input any) (intent, bool, *Result, error) {
	if !operationPattern.MatchString(id) {
		return intent{}, false, nil, errors.New("operation ID must contain 1–64 ASCII letters, digits, underscores or hyphens")
	}
	encoded, err := json.Marshal([]any{s.instance, s.rootID, kind, input})
	if err != nil {
		return intent{}, false, nil, errors.New("cannot fingerprint operation")
	}
	fingerprint := ContentSHA256(string(encoded))
	var done completion
	if exists, err := readRecord(s.recordPath(id, ".done.json"), &done); err != nil {
		return intent{}, false, nil, err
	} else if exists {
		if done.Fingerprint != fingerprint {
			return intent{}, false, nil, errors.New("operation ID was already used with different parameters")
		}
		done.Result.AlreadyApplied = true
		return intent{}, true, &done.Result, nil
	}
	var op intent
	exists, err := readRecord(s.recordPath(id, ".json"), &op)
	if err != nil {
		return intent{}, false, nil, err
	}
	if exists && op.Fingerprint != fingerprint {
		return intent{}, false, nil, errors.New("operation ID was already used with different parameters")
	}
	if !exists {
		op.Fingerprint = fingerprint
	}
	return op, exists, nil, nil
}

// verifyAndFinish 读取写后正文进行逐字节摘要核对，成功后保存完成标记。
// 输入：op 为已持久化意图；replayed 表示恢复时发现操作已在服务器生效。
// 输出：核对及完成标记持久化成功时返回结果，否则保留待核对状态，不进行回滚或再次写入。
func (s *Service) verifyAndFinish(ctx context.Context, id string, op intent, replayed bool) (Result, error) {
	current, err := s.readWritable(ctx, op.NoteID)
	if err != nil {
		return Result{}, err
	}
	if ContentSHA256(current.Content) != op.DesiredHash {
		return Result{}, errors.New("write verification failed; operation remains pending, current content preserved")
	}
	result := Result{OperationID: id, NoteID: op.NoteID, ContentSHA256: op.DesiredHash, AlreadyApplied: replayed}
	if err := saveRecord(s.recordPath(id, ".done.json"), completion{Fingerprint: op.Fingerprint, Result: result}); err != nil {
		return Result{}, err
	}
	return result, nil
}

// recordPath 由校验后的操作 ID 生成状态文件路径，避免使用用户输入的任意路径。
func (s *Service) recordPath(id, suffix string) string { return filepath.Join(s.stateDir, id+suffix) }

// isNotFound 仅把明确的 HTTP 404 视为目标尚不存在，连接失败不能用于判断是否应重新创建。
func isNotFound(err error) bool {
	var status *trilium.HTTPError
	return errors.As(err, &status) && status.StatusCode == http.StatusNotFound
}
