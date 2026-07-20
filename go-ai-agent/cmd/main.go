package main

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"go-ai-agent/config"
)

func main() {
	ctx := context.Background()
	if config.OpenaiApiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}

	opts := []option.RequestOption{option.WithAPIKey(config.OpenaiApiKey)}
	if config.OpenaiBaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.OpenaiBaseURL))
	}
	client := openai.NewClient(opts...)

	question := "请用三句话解释什么是 AI Agent。"
	resp, err := client.Responses.New(
		ctx, responses.ResponseNewParams{
			Input: responses.ResponseNewParamsInputUnion{OfString: openai.String(question)},
			Model: config.OpenaiModel,
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.OutputText())
}
