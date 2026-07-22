package main

import (
	"context"
	"fmt"
	"github.com/openai/openai-go/v3/packages/param"
	"go-ai-agent/internal/tools"
	"log"
	"os"
	"strings"

	"go-ai-agent/internal/config"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

func main() {
	//fmt.Println(os.Args)
	if config.OpenaiApiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set")
	}
	opts := []option.RequestOption{option.WithAPIKey(config.OpenaiApiKey)}
	if config.OpenaiBaseURL != "" {
		opts = append(opts, option.WithBaseURL(config.OpenaiBaseURL))
	}
	client := openai.NewClient(opts...)
	question, err := readQuestion(os.Args)
	if err != nil {
		fmt.Println("未读取到有效的问题")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.RequestTimeout)
	defer cancel()
	resp, err := client.Responses.New(
		ctx,
		responses.ResponseNewParams{
			Instructions: param.NewOpt[string](config.Instruction),
			Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(question)},
			Model:        config.OpenaiModel,
		},
		option.WithRequestTimeout(config.RetryTimeout),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("输出")
	fmt.Println(resp.OutputText())

}

// readQuestion 默认读取命令行参数, 若未指定命令行参数, 则提示用户输入问题, 从标准输入中读取
func readQuestion(args []string) (string, error) {
	var question string
	if len(os.Args) > 1 && strings.TrimSpace(os.Args[1]) != "" {
		question = os.Args[1]
		return question, nil
	}
	fmt.Println("请输入问题:")
	return tools.ReadInputString()
}
