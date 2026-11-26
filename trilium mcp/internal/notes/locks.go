package notes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// acquireKeys 以固定顺序取得跨进程文件锁，竞争时等待 ctx，失败时释放已取得的锁。
// 输入：keys 是内部生成的锁身份；路径使用摘要避免非法字符和 Windows 大小写折叠。
// 输出：返回释放函数；不自动删除崩溃遗留锁，权限错误立即返回。
func acquireKeys(ctx context.Context, dir string, keys []string) (func(), error) {
	lockDir := filepath.Join(dir, ".locks")
	if err := os.MkdirAll(lockDir, 0700); err != nil {
		return nil, errors.New("cannot create write lock directory")
	}
	keys = slices.Clone(keys)
	slices.Sort(keys)
	keys = slices.Compact(keys)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		paths := []string{}
		release := func() {
			for i := len(paths) - 1; i >= 0; i-- {
				os.Remove(paths[i])
			}
		}
		busy := false
		for _, key := range keys {
			path := filepath.Join(lockDir, ContentSHA256(key)+".lock")
			file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				release()
				if !errors.Is(err, os.ErrExist) {
					return nil, errors.New("cannot acquire write lock")
				}
				busy = true
				break
			}
			file.Close()
			paths = append(paths, path)
		}
		if !busy {
			return release, nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// lockOperation 串行化同一操作 ID，并拒绝旧版本遗留的全局锁。
// 同一状态目录不能同时运行新旧锁协议的程序；升级时须先停止旧进程。
func (s *Service) lockOperation(ctx context.Context, id string) (func(), error) {
	if !operationPattern.MatchString(id) {
		return nil, errors.New("invalid operation ID")
	}
	if _, err := os.Stat(filepath.Join(s.stateDir, ".lock")); err == nil || !errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("legacy write-state directory is locked or inaccessible; stop old writers first")
	}
	return acquireKeys(ctx, s.stateDir, []string{"operation:" + id})
}

// lockTarget 按目标 noteId 取得跨进程锁，不读取目录关系。
// 新建时传入父笔记 ID，使创建子笔记与删除该父笔记竞争同一把锁。
// 目标已删除时仍可取得锁，随后由业务层核对持久化意图和 404。
func (s *Service) lockTarget(ctx context.Context, noteID string) (func(), error) {
	if !idPattern.MatchString(noteID) {
		return nil, errors.New("invalid lock target note ID")
	}
	return acquireKeys(ctx, s.stateDir, []string{"note:" + noteID})
}
