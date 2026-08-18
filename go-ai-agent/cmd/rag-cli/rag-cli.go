package main

import (
	"fmt"
	"go-ai-agent/internal/rag"
)

func main() {
	//flags, args := config.GetFlagConf()

	// 测试 Tool Calling 使用 Groq, 其他场景还是使用标准 openai 中转站
	//client := llm.NewOpenAIClient(config.OpenaiApiKey, config.OpenaiBaseURL, flags.Model)
	//messages := make([]llm.Message, 0)
	//messages = append(messages, []llm.Message{
	//	//llm.GetSystemMessage(flags.Instruction),
	//	llm.GetUserMessage(args...),
	//}...,
	//)
	//ctx := context.Background()
	//var cancel func()
	//if flags.TimeOut != 0 {
	//	ctx, cancel = context.WithTimeout(ctx, flags.TimeOut)
	//	defer cancel()
	//}
	docs, err := rag.LoadRAGResources("D:\\Go\\WorkSpace\\src\\Go_Project\\ai-agent\\go-ai-agent\\testdata")
	if err != nil {
		panic(err)
	}
	for _, doc := range docs {
		fmt.Println("已找到:", doc.SourcePath)
	}
}
