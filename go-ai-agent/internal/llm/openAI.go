package llm

import (
	"context"
	"fmt"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"go-ai-agent/internal/config"
	"strings"
)

type openAIClient struct {
	model  string
	client openai.Client
}

func (o *openAIClient) Generate(ctx context.Context, messages []Message) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, config.RequestTimeout)
	defer cancel()
	instruction, userMsg := handleMessages(messages)
	resp, err := o.client.Responses.New(
		ctx,
		responses.ResponseNewParams{
			Instructions: param.NewOpt[string](instruction),
			Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(userMsg)},
			Model:        o.model,
		},
		option.WithRequestTimeout(config.RetryTimeout),
	)
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
}

func (o *openAIClient) Stream(ctx context.Context, messages []Message, onDelta func(string)) error {
	instruction, userMsg := handleMessages(messages)
	stream := o.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Instructions: param.NewOpt[string](instruction),
		Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(userMsg)},
		Model:        o.model,
	})
	for stream.Next() {
		event := stream.Current()
		onDelta(event.Delta)
	}
	return stream.Err()
}

func handleMessages(messages []Message) (systemMessage, userMessage string) {
	var instructionsBuild strings.Builder
	var userMessageBuild strings.Builder

	for _, m := range messages {
		switch m.Role {
		case UserMessage:
			userMessageBuild.WriteString(m.Content)
		case SystemMessage:
			instructionsBuild.WriteString(m.Content)
		default:
			fmt.Printf("不支持的role: %d\n", m.Role)
			return "", ""
		}
	}
	return instructionsBuild.String(), userMessageBuild.String()
}

func NewOpenAIClient(apiKey, baseURL, model string) Client {
	opts := make([]option.RequestOption, 0)
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	client := openai.NewClient(opts...)
	return &openAIClient{
		model:  model,
		client: client,
	}
}
