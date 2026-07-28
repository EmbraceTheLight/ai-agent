package llm

import (
	"context"
	"fmt"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/tools"
)

type Client interface {
	Generate(ctx context.Context, messages []Message) (string, error)
	GenerateWithJsonSchema(ctx context.Context, messages []Message) (string, error)
	Stream(ctx context.Context, messages []Message, onDelta func(string)) error
}

func GetSystemMessage(systemMsg ...string) Message {
	var msgStr string
	if len(systemMsg) == 0 || len(systemMsg[0]) == 0 {
		msgStr = config.Instruction
	} else {
		msgStr = systemMsg[0]
	}
	msg, err := readMessage(SystemMessage, msgStr)
	if err != nil {
		fmt.Println("读取系统消息时发生错误:", err)
	}
	return msg
}

func GetUserMessage(userMsg ...string) Message {
	fmt.Println("请输入问题:")
	msg, err := readMessage(UserMessage, userMsg...)
	if err != nil {
		fmt.Println("读取用户问题时发生错误:", err)
	}
	return msg
}

// readMessage 默认读取命令行参数, 若未指定命令行参数, 则提示用户输入问题, 从标准输入中读取
func readMessage(messageType int, msg ...string) (Message, error) {
	var question string
	question, ok := getVararg(msg)
	if messageType == SystemMessage {
		fmt.Println("question:", question)
	}
	if ok == true {
		return Message{Role: messageType, Content: question}, nil
	}

	question, err := tools.ReadInputString()
	if err != nil {
		return Message{}, err
	}
	return Message{Role: messageType, Content: question}, nil

}

// getVararg 获取类型为 T 的不定长参数的参数值
// 当 varArg 的长度不为 0 时, 返回第一个参数 varArg[0]
// 若未指定命令行参数, 或 T 为字符串类型, 且 varArg[0] 为空, 则第二个返回值为 false
func getVararg[T any](varArg []T) (T, bool) {
	var zero T
	if len(varArg) == 0 {
		return zero, false
	}
	switch v := any(varArg[0]).(type) {
	case string:
		if v == "" {
			return zero, false
		}
		return varArg[0], true
	default:
		return varArg[0], true
	}
}
