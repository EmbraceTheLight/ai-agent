package biz

import "context"

// DocumentLockRepo 定义单篇文档导入的互斥能力。
// 输入: 文档 ID 和请求上下文。
// 输出: 标识是否成功取得锁, 并允许在导入结束后释放锁。
// 示例: `if lock.LockDocument(ctx, id) { defer lock.UnlockDocument(ctx, id) }`。
type DocumentLockRepo interface {
	// LockDocument 尝试立即取得指定文档的锁, 不等待且不设置自动释放时间。
	LockDocument(ctx context.Context, docID string) bool
	// UnlockDocument 释放本次导入持有的文档锁。
	UnlockDocument(ctx context.Context, docID string)
}
