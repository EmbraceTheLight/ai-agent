package data

import (
	"context"
	"go-ai-agent/app/rag-api/internal/biz"
	"sync"
)

// documentLockRepo 在单个进程内为每个文档维护独立的互斥锁。
type documentLockRepo struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

// LockDocument 尝试立即取得文档锁, 不等待持锁者释放。
// 输入: `ctx` 用于检查请求是否已取消, `docID` 是文档标识。
// 输出: 取得锁返回 true; 文档 ID 为空、请求已取消或锁被占用时返回 false。
// 示例: `lock.LockDocument(ctx, documentID)`。
func (d *documentLockRepo) LockDocument(ctx context.Context, docID string) bool {
	if ctx.Err() != nil || docID == "" {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	lock, ok := d.locks[docID]
	if !ok {
		lock = &sync.Mutex{}
		d.locks[docID] = lock
	}
	return lock.TryLock()
}

// UnlockDocument 释放当前持有的文档锁并移除其进程内记录。
// 输入: `docID` 必须是此前成功加锁的文档 ID; `ctx` 不控制解锁。
// 输出: 无返回值, 该文档可再次被导入。
// 示例: `lock.UnlockDocument(ctx, documentID)`。
func (d *documentLockRepo) UnlockDocument(ctx context.Context, docID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	lock := d.locks[docID]
	lock.Unlock()
	delete(d.locks, docID)
}

// NewDocumentLockRepo 创建进程内按文档 ID 隔离的互斥锁仓库。
// 输入: 无。
// 输出: 返回实现 DocumentLockRepo 的新实例。
// 示例: `lock := NewDocumentLockRepo()`。
func NewDocumentLockRepo() biz.DocumentLockRepo {
	return &documentLockRepo{locks: make(map[string]*sync.Mutex)}
}
