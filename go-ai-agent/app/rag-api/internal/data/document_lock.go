package data

import (
	"context"
	"sync"
)

type documentLockRepo struct {
	locks map[string]*sync.Mutex
}

func (d *documentLockRepo) LockDocument(ctx context.Context, docID string) bool {
	ok := d.locks[docID].TryLock()

}

func (d *documentLockRepo) UnlockDocument(ctx context.Context, docID string) {
	d.locks[docID].Unlock()
}
