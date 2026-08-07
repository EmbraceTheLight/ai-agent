package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"codex-context-guard/internal/analyzer"
	"codex-context-guard/internal/guard"
	"codex-context-guard/internal/report"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "analyze":
		runAnalyze(os.Args[2:])
	case "guard":
		runGuard(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

// 作用：执行离线用量分析子命令，解析命令行参数并输出人类可读报告。
// 入参：args 为 analyze 子命令后面的参数，例如 --sessions、--since-days 和 --top。
// 返回值：无；分析失败时会通过 exitErr 输出错误并以非零状态退出。
// 示例：runAnalyze([]string{"--sessions", `C:\Users\me\.codex\sessions`, "--since-days", "7"})。
func runAnalyze(args []string) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	sessions := fs.String("sessions", defaultSessionsDir(), "Codex sessions directory")
	sinceDays := fs.Int("since-days", 7, "candidate event age in days")
	limit := fs.Int("top", 20, "number of high-usage rows to show")
	_ = fs.Parse(args)

	since := time.Now().AddDate(0, 0, -*sinceDays)
	result, err := analyzer.AnalyzeDirectory(*sessions, since)
	if err != nil {
		exitErr(err)
	}

	fmt.Print(report.Text(result, *limit))
}

// 作用：执行实时 Context Guard 子命令，读取 Hook 输入或显式 transcript 参数并按策略输出预警。
// 入参：args 为 guard 子命令后面的参数，例如 --transcript、--state、--warn 和 --high。
// 返回值：无；Hook 处理失败时会通过 exitErr 输出错误并以非零状态退出。
// 示例：runGuard([]string{"--transcript", `C:\Users\me\.codex\sessions\active.jsonl`, "--warn", "70"})。
func runGuard(args []string) {
	fs := flag.NewFlagSet("guard", flag.ExitOnError)
	transcript := fs.String("transcript", "", "transcript JSONL path; defaults to hook stdin")
	statePath := fs.String("state", "", "guard state path")
	warn := fs.Float64("warn", 70, "warning context utilization percent")
	high := fs.Float64("high", 85, "high context utilization percent")
	autoCompactLimit := fs.Int("auto-compact-limit", 3, "auto compacts in 24h before critical warning")
	_ = fs.Parse(args)

	policy := guard.Policy{
		WarnPercent:         *warn,
		HighPercent:         *high,
		AutoCompactsPerDay:  *autoCompactLimit,
		AutoCompactInterval: 24 * time.Hour,
	}

	if err := guard.Run(os.Stdin, os.Stdout, *transcript, defaultStatePath(*statePath), policy); err != nil {
		exitErr(err)
	}
}

// 作用：向 stderr 打印命令行帮助，说明 analyze 与 guard 两个子命令的参数形式。
// 入参：无。
// 返回值：无。
// 示例：usage() 会打印 codex-usage 的用法文本。
func usage() {
	fmt.Fprint(os.Stderr, `codex-usage

Usage:
  codex-usage analyze [--sessions PATH] [--since-days 7] [--top 20]
  codex-usage guard   [--transcript PATH] [--state PATH] [--warn 70] [--high 85]
`)
}

// 作用：推导默认 Codex sessions 目录，优先使用 CODEX_HOME，再回退到用户主目录。
// 入参：无。
// 返回值：sessions 目录路径；无法取得用户主目录时返回相对路径 "sessions"。
// 示例：当 CODEX_HOME=C:\Users\me\.codex 时，defaultSessionsDir() 返回 C:\Users\me\.codex\sessions。
func defaultSessionsDir() string {
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return filepath.Join(home, "sessions")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codex", "sessions")
	}
	return "sessions"
}

// 作用：推导 Context Guard 状态文件路径，显式参数优先，然后使用 CODEX_HOME 或用户主目录。
// 入参：explicit 为用户通过 --state 指定的路径；为空时自动推导。
// 返回值：状态文件路径；无法取得用户主目录时返回相对路径 "context-guard-state.json"。
// 示例：defaultStatePath("state.json") 直接返回 "state.json"。
func defaultStatePath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if home := os.Getenv("CODEX_HOME"); home != "" {
		return filepath.Join(home, "context-guard-state.json")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codex", "context-guard-state.json")
	}
	return "context-guard-state.json"
}

// 作用：统一处理命令执行错误，将错误写入 stderr 并退出进程。
// 入参：err 为需要展示给用户的错误。
// 返回值：无；该函数会调用 os.Exit(1)，正常情况下不会返回调用方。
// 示例：exitErr(fmt.Errorf("sessions directory not found")) 会打印 error: sessions directory not found 并退出。
func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
