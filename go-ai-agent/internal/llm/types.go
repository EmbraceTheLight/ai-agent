package llm

const (
	SystemMessage = iota // 系统消息
	UserMessage          // 用户消息
)

type Message struct {
	Role    int
	Content string
}
