package llm

import (
	"context"
	"fmt"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/tools"
	"strings"
)

type openAIClient struct {
	model  string
	client openai.Client
}

func (o *openAIClient) Generate(ctx context.Context, messages []Message) (string, error) {
	ctx, cancel := tools.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
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
	ctx, cancel := tools.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
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

func (o *openAIClient) GenerateWithJsonSchema(ctx context.Context, messages []Message) (string, error) {
	ctx, cancel := tools.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	instruction, userMsg := handleMessages(messages)
	resp, err := o.client.Responses.New(
		ctx,
		responses.ResponseNewParams{
			Instructions: param.NewOpt[string](instruction),
			Input:        responses.ResponseNewParamsInputUnion{OfString: openai.String(userMsg)},
			Model:        o.model,
			Text: responses.ResponseTextConfigParam{
				Verbosity: responses.ResponseTextConfigVerbosityMedium,
				Format: responses.ResponseFormatTextConfigUnionParam{
					OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
						Name:   "IntentResult",
						Strict: openai.Bool(true),
						Schema: map[string]any{
							"type": "object",
							"properties": map[string]any{
								"intent": map[string]any{
									"type":        "string",
									"description": "用户意图",
									"enum": []string{
										config.AgentQuestion,
										config.RagQuestion,
										config.ToolQuestion,
										config.GeneralQuestion,
									},
								},
								"answer": map[string]any{
									"type":        "string",
									"description": "回答",
								},
								"confidence": map[string]any{
									"type":        "number",
									"description": "置信度",
									"minimum":     0,
									"maximum":     1,
								},
							},
							"required":             []string{"intent", "answer", "confidence"},
							"additionalProperties": openai.Bool(false),
						},
						Description: openai.String("json 格式的回答, 包含 rag_question, tool_question, agent_question, general_question 这四种分类"),
					},
				},
			},
		},
		option.WithRequestTimeout(config.RetryTimeout),
	)
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
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
	if model == "" {
		model = config.OpenaiModel
	}
	client := openai.NewClient(opts...)
	return &openAIClient{
		model:  model,
		client: client,
	}
}
