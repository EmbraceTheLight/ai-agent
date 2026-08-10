package llm

import "go-ai-agent/internal/tools"

const (
	SystemMessage = iota // 系统消息
	UserMessage          // 用户消息
)

type Message struct {
	Role    int
	Content string
}

type ChatCompletionOpts struct {
	OutputType   string
	ToolExecutor *tools.Executor
}
