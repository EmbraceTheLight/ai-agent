package llm

import (
	"context"
	"fmt"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/tools"
	"go-ai-agent/internal/utils"
	"strings"
)

type openAIClient struct {
	model  string
	client openai.Client
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

func (o *openAIClient) Generate(ctx context.Context, messages []Message) (string, error) {
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	instruction, userMsg := handleMessages(messages)
	resp, err := o.client.Responses.New(
		ctx,
		responses.ResponseNewParams{
			Instructions: openai.String(instruction),
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
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	instruction, userMsg := handleMessages(messages)
	stream := o.client.Responses.NewStreaming(ctx, responses.ResponseNewParams{
		Instructions: openai.String(instruction),
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
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	instruction, userMsg := handleMessages(messages)
	resp, err := o.client.Responses.New(
		ctx,
		responses.ResponseNewParams{
			Instructions: openai.String(instruction),
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

func (o *openAIClient) FunctionCall(ctx context.Context, messages []Message, e *tools.Executor) (string, error) {
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	instruction, userMsg := handleMessages(messages)
	resp, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
		Instructions:      openai.String(instruction),
		ParallelToolCalls: openai.Bool(false), // 不允许一次调用多个 Tool
		Input:             responses.ResponseNewParamsInputUnion{OfString: openai.String(userMsg)},
		Store:             openai.Bool(true),
		Tools:             toolsToToolParam(e.GetToolList()),
		Model:             o.model,
	})
	if err != nil {
		return "", err
	}

	var toolCallCount int
	var calledTools []string // 被调用的工具的名称 debug 使用
	for toolCallCount = 0; toolCallCount < e.MaxSteps; toolCallCount++ {
		toolCall, ok := functionCall(resp)
		if ok == false {
			break
		}
		toolResp, err := e.Execute(ctx, toolCall.Name, toolCall.Arguments)
		if err != nil {
			return "", err
		}
		calledTools = append(calledTools, toolCall.Name)
		fmt.Printf("tool %s Resp: %v\n", toolCall.Name, toolResp)
		resp, err = o.client.Responses.New(
			ctx,
			responses.ResponseNewParams{
				Instructions:       openai.String(instruction),
				ParallelToolCalls:  openai.Bool(false),     // 不允许一次调用多个 Tool
				PreviousResponseID: openai.String(resp.ID), // 上个请求的 ID
				Tools:              toolsToToolParam(e.GetToolList()),
				Store:              openai.Bool(true),
				Input: responses.ResponseNewParamsInputUnion{
					OfInputItemList: []responses.ResponseInputItemUnionParam{
						{
							OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
								CallID: toolCall.CallID,
								Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
									OfString: openai.String(toolResp),
								},
							},
						},
					},
				},
				Model: o.model,
			},
			option.WithRequestTimeout(config.RetryTimeout),
		)
		if err != nil {
			return "", errno.RequestFailed.WithError(err)
		}
	}

	fmt.Printf("工具被调用了 %d 次\n", toolCallCount)
	// 检测 tool 调用次数超过限制 ---- tool 调用已达 e.MaxSteps, 且下次还希望请求调用 Tool
	_, ok := functionCall(resp)
	if toolCallCount == e.MaxSteps && ok == true {
		return "", errno.ErrToolCallLimitExceed
	}
	if toolCallCount > 0 {
		fmt.Println("被调用的工具:")
		fmt.Println(calledTools)
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

// toolsToToolParam 将 tools 信息转换为 openai 的 ToolParam 格式
func toolsToToolParam(tools []*tools.Tool) []responses.ToolUnionParam {
	toolParams := make([]responses.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		toolParam := responses.ToolUnionParam{
			OfFunction: &responses.FunctionToolParam{
				Name:        t.Name,
				Strict:      openai.Bool(t.Strict),
				Description: openai.String(t.Description),
				Parameters:  t.Parameters,
			}}
		toolParams = append(toolParams, toolParam)
	}
	return toolParams
}

// functionCall 解析模型返回的 resp.Output 的类型. 如果是 function_call, 说明模型希望调用 Tool, 返回 Tool 的调用信息
// 入参: 模型回答 resp
// 返回值: Tool 的调用信息, 是否是 function_call 类型
func functionCall(resp *responses.Response) (functionCall responses.ResponseFunctionToolCall, ok bool) {
	for _, item := range resp.Output {
		if item.Type == "function_call" {
			return item.AsFunctionCall(), true
		}
	}
	return functionCall, false
}
