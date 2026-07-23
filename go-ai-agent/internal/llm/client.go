package llm

import (
	"context"
	"fmt"
	"go-ai-agent/internal/tools"
	"strings"
)

type Client interface {
	Generate(ctx context.Context, messages []Message) (string, error)
	Stream(ctx context.Context, messages []Message, onDelta func(string)) error
}

func GetSystemMessage() Message {
	fmt.Println("请输入系统 instruction:")
	msg, err := readMessage(SystemMessage)
	if err != nil {
		fmt.Println("读取系统消息时发生错误:", err)
	}
	return msg
}

func GetUserMessage() Message {
	fmt.Println("请输入问题:")
	msg, err := readMessage(UserMessage)
	if err != nil {
		fmt.Println("读取用户问题时发生错误:", err)
	}
	return msg
}

// readMessage 默认读取命令行参数, 若未指定命令行参数, 则提示用户输入问题, 从标准输入中读取
func readMessage(messageType int, msg ...string) (Message, error) {
	var question string
	if len(msg) > 0 && strings.TrimSpace(msg[0]) != "" {
		question = msg[0]
		return Message{Role: messageType, Content: question}, nil
	}
	question, err := tools.ReadInputString()
	if err != nil {
		return Message{}, err
	}
	return Message{Role: messageType, Content: question}, nil

}
