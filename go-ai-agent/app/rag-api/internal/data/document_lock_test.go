package data

import (
	"context"
	"testing"
)

// TestDocumentLockRepo 验证同一文档不能并发导入，释放后可再次取得锁。
// 输入: 两个不同的文档 ID 和同一把进程内锁仓库。
// 输出: 断言同一文档互斥、不同文档独立且解锁后可重新加锁。
// 示例: `go test ./app/rag-api/internal/data -run TestDocumentLockRepo`。
func TestDocumentLockRepo(t *testing.T) {
	lock := NewDocumentLockRepo()
	ctx := context.Background()
	if !lock.LockDocument(ctx, "doc") || lock.LockDocument(ctx, "doc") {
		t.Fatal("same document lock must be exclusive")
	}
	if !lock.LockDocument(ctx, "other") {
		t.Fatal("different document should not be blocked")
	}
	lock.UnlockDocument(ctx, "doc")
	if !lock.LockDocument(ctx, "doc") {
		t.Fatal("lock should be reusable after release")
	}
	lock.UnlockDocument(ctx, "doc")
	lock.UnlockDocument(ctx, "other")
}
