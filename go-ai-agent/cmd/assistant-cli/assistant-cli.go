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
	flags, args := config.GetFlagConf()
	if flags.JSON == true && flags.Stream == true {
		panic("json 和 stream 不能同时为 true")
	}

	client := llm.NewOpenAIClient(config.OpenaiApiKey, config.OpenaiBaseURL, flags.Model)
	messages := make([]llm.Message, 0)
	messages = append(messages, []llm.Message{
		llm.GetSystemMessage(flags.Instruction),
		llm.GetUserMessage(args...),
	}...,
	)

	ctx := context.Background()
	var cancel func()
	if flags.TimeOut != 0 {
		ctx, cancel = context.WithTimeout(ctx, flags.TimeOut)
		defer cancel()
	}
	switch {
	case flags.JSON: // 要求以 json 格式输出
		ret, err := client.GenerateWithJsonSchema(ctx, messages)
		if err != nil {
			panic(err)
		}
		var ans llm.IntentResult
		err = json.Unmarshal([]byte(ret), &ans)
		if err != nil {
			panic(err)
		}

		err = ans.IsValid()
		if err != nil {
			fmt.Println(" json 结果检查发现错误:", err)
		}
		fmt.Println("回答:", ans.Answer)
		fmt.Println("============================== 分界线 ==============================")
		log.Printf("整个 ans 结构: %+v", ans)

	case flags.Stream: // 要求流式输出
		err := client.Stream(ctx, messages, func(delta string) {
			fmt.Print(delta)
		})
		if err != nil {
			panic(err)
		}
	default: // 要求普通输出
		output, err := client.Generate(ctx, messages)
		if err != nil {
			panic(err)
		}
		fmt.Println("输出")
		fmt.Println(output)
	}

}
