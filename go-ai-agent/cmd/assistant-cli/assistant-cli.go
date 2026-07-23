package main

import (
	"context"
	"fmt"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/llm"
)

func main() {
	client := llm.NewOpenAIClient(config.OpenaiApiKey, config.OpenaiBaseURL, config.OpenaiModel)
	messages := make([]llm.Message, 0)
	messages = append(messages, []llm.Message{
		llm.GetSystemMessage(),
		llm.GetUserMessage(),
	}...,
	)
	//output, err := client.Generate(context.TODO(), messages)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Println("输出")
	//fmt.Println(output)

	err := client.Stream(context.TODO(), messages, func(delta string) {
		fmt.Print(delta)
	})
	if err != nil {
		panic(err)
	}
}
