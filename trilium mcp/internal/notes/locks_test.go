package notes

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestNoteLocksAcrossProcesses 验证两个独立进程共享目录时，同笔记互斥而异笔记可继续取得锁。
func TestNoteLocksAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestNoteLockHelper$")
	cmd.Env = append(os.Environ(), "TRILIUM_TEST_LOCK_DIR="+dir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); cmd.Wait() }()
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || line != "locked\n" {
		t.Fatal(line, err)
	}
	unlock, err := acquireKeys(ctx, dir, []string{"note:sibling"})
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	blocked, stop := context.WithTimeout(ctx, 80*time.Millisecond)
	_, err = acquireKeys(blocked, dir, []string{"note:note123"})
	stop()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cross-process exclusion failed: %v", err)
	}
	if _, err := stdin.Write([]byte{1}); err != nil {
		t.Fatal(err)
	}
	unlock, err = acquireKeys(ctx, dir, []string{"note:note123"})
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}

// TestNoteLockHelper 作为测试子进程持锁，收到父进程信号或 EOF 后释放。
func TestNoteLockHelper(t *testing.T) {
	dir := os.Getenv("TRILIUM_TEST_LOCK_DIR")
	if dir == "" {
		return
	}
	unlock, err := acquireKeys(context.Background(), dir, []string{"note:note123"})
	if err != nil {
		os.Exit(2)
	}
	fmt.Fprintln(os.Stdout, "locked")
	io.ReadFull(os.Stdin, make([]byte, 1))
	unlock()
	os.Exit(0)
}
