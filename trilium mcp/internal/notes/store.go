package notes

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"trilium-mcp/internal/trilium"
)

// intent 是写入前保存的不可变操作记录；追加和更新记录同时保存原始正文和元数据。
type intent struct {
	Fingerprint string    `json:"fingerprint"`
	NoteID      string    `json:"noteId"`
	DesiredHash string    `json:"desiredHash"`
	Before      *snapshot `json:"before,omitempty"`
}

// snapshot 保留修改前的完整正文及元数据，用于冲突检测和人工恢复。
type snapshot struct {
	Note    trilium.Note `json:"note"`
	Content string       `json:"content"`
}

// completion 记录已读取核对成功的操作，重复调用时返回历史结果而不再次写入。
type completion struct {
	Fingerprint string `json:"fingerprint"`
	Result      Result `json:"result"`
}

// acquireLock 使用独占文件串行化同一状态目录的写操作。
// 输入：dir 为操作记录目录，必要时创建。
// 输出：成功时返回释放函数；已有锁或无法创建目录时返回错误；不自动移除崩溃遗留的锁。
func acquireLock(dir string) (func(), error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, errors.New("cannot create write-state directory")
	}
	path := filepath.Join(dir, ".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.New("write-state directory is locked or inaccessible; check for an active operation")
	}
	return func() { file.Close(); os.Remove(path) }, nil
}

// readRecord 读取有限大小的 JSON 操作记录，不将私有内容附加到错误信息。
// 输入：path 为操作 ID 派生的内部路径；target 为结果指针。
// 输出：文件不存在时返回 false、nil；存在且有效时返回 true、nil；损坏或读取失败时报错。
func readRecord(path string, target any) (bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, errors.New("cannot open operation record")
	}
	defer file.Close()
	body, err := io.ReadAll(io.LimitReader(file, (64<<20)+1))
	if err != nil || len(body) > 64<<20 || json.Unmarshal(body, target) != nil {
		return false, errors.New("operation record is unreadable or invalid; refusing to repeat write")
	}
	return true, nil
}

// saveRecord 独占创建并同步操作记录，不覆盖已有备份。
// 输入：path 为内部文件路径；value 为可编码的操作意图或完成结果。
// 输出：编码、写入、同步和关闭均成功时返回 nil；失败时保留文件供检查，调用方不得继续写入 API。
func saveRecord(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return errors.New("cannot encode operation record")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("cannot create operation record; existing files are preserved")
	}
	_, writeErr := file.Write(body)
	syncErr := file.Sync()
	closeErr := file.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		return errors.New("cannot persist operation record; API write stopped")
	}
	return nil
}
