package notes

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"trilium-mcp/internal/trilium"
)

// gatedAPI 提供线程安全的模拟存储，并允许在 HTTP 写入位置暂停一个操作以稳定复现竞争。
type gatedAPI struct {
	mu                 sync.Mutex
	data               *memoryAPI
	onCreate, onDelete func(context.Context) error
}

// GetNote 在存储锁内读取元数据。
func (a *gatedAPI) GetNote(ctx context.Context, id string) (trilium.Note, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.GetNote(ctx, id)
}

// GetNoteContent 在存储锁内读取正文。
func (a *gatedAPI) GetNoteContent(ctx context.Context, id string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.GetNoteContent(ctx, id)
}

// CreateNote 在暂停点之后创建子笔记，同时更新父笔记的子节点元数据。
func (a *gatedAPI) CreateNote(ctx context.Context, input trilium.CreateNoteInput) error {
	if a.onCreate != nil {
		if err := a.onCreate(ctx); err != nil {
			return err
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.data.CreateNote(ctx, input); err != nil {
		return err
	}
	parent := a.data.notes[input.ParentNoteID]
	parent.Note.ChildNoteIDs = append(parent.Note.ChildNoteIDs, input.NoteID)
	a.data.notes[input.ParentNoteID] = parent
	return nil
}

// SetNoteContent 在存储锁内更新正文。
func (a *gatedAPI) SetNoteContent(ctx context.Context, id, body string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.SetNoteContent(ctx, id, body)
}

// DeleteNote 在暂停点之后删除目标笔记。
func (a *gatedAPI) DeleteNote(ctx context.Context, id string) error {
	if a.onDelete != nil {
		if err := a.onDelete(ctx); err != nil {
			return err
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.data.DeleteNote(ctx, id)
}

// TestCreateDeleteLeafCompetition 验证先创建则父笔记不再可删，先删除则不能向消失的父笔记创建子笔记。
func TestCreateDeleteLeafCompetition(t *testing.T) {
	for _, first := range []string{"create", "delete"} {
		t.Run(first, func(t *testing.T) {
			_, data := testService(t)
			api := &gatedAPI{data: data}
			s, err := NewService(api, "temp", t.TempDir(), "test")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			entered := make(chan struct{})
			release := make(chan struct{})
			var once sync.Once
			resume := func() { once.Do(func() { close(release) }) }
			defer resume()
			gate := func(ctx context.Context) error {
				close(entered)
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			create := CreateInput{OperationID: "create-child", ParentNoteID: "note123", Title: "child", Content: "x", Format: "text"}
			remove := DeleteInput{OperationID: "delete-parent", NoteID: "note123", ExpectedSHA256: ContentSHA256(data.notes["note123"].Content)}
			done := make(chan error, 1)
			if first == "create" {
				api.onCreate = gate
				go func() { _, err := s.Create(ctx, create); done <- err }()
			} else {
				api.onDelete = gate
				go func() { _, err := s.Delete(ctx, remove); done <- err }()
			}
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("first operation never reached write")
			}
			blocked, stop := context.WithTimeout(ctx, 80*time.Millisecond)
			if first == "create" {
				_, err = s.Delete(blocked, remove)
			} else {
				_, err = s.Create(blocked, create)
			}
			stop()
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("competing operation bypassed parent note lock: %v", err)
			}
			resume()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if first == "create" {
				_, err = s.Delete(ctx, remove)
				if err == nil || !strings.Contains(err.Error(), "leaf") {
					t.Fatal("nonleaf parent deletion accepted", err)
				}
				if data.creates != 1 || data.deletes != 0 {
					t.Fatal(data.creates, data.deletes)
				}
			} else {
				_, err = s.Create(ctx, create)
				if !isNotFound(err) {
					t.Fatal("created under missing parent", err)
				}
				if data.creates != 0 || data.deletes != 1 {
					t.Fatal(data.creates, data.deletes)
				}
			}
		})
	}
}
