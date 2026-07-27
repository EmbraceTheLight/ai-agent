package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/llm"
	"log"
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

	//err := client.Stream(context.TODO(), messages, func(delta string) {
	//	fmt.Print(delta)
	//})
	//if err != nil {
	//	panic(err)
	//}

	ret, err := client.GenerateWithJsonSchema(context.TODO(), messages)
	if err != nil {
		panic(err)
	}
	var ans llm.IntentResult
	err = json.Unmarshal([]byte(ret), &ans)
	if err != nil {
		panic(err)
	}
	if ans.Confidence > 1 || ans.Confidence < 0 {
		fmt.Println("置信度不在0-1之间, 当前置信度:", ans.Confidence)
	}
	if ans.Intent != config.AgentQuestion && ans.Intent != config.RagQuestion && ans.Intent != config.ToolQuestion && ans.Intent != config.GeneralQuestion {
		fmt.Println("意图不在指定范围内, 当前意图:", ans.Intent)
	}
	fmt.Println("回答:", ans.Answer)
	log.Printf("整个 ans 结构: %+v", ans)
}
