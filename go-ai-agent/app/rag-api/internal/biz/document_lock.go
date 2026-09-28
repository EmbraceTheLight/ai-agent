package biz

import "context"

type DocumentLockRepo interface {
	LockDocument(ctx context.Context, docID string) bool
	UnlockDocument(ctx context.Context, docID string)
}
