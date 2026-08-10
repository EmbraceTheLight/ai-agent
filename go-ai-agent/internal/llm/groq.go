package llm

import (
	"context"
	"fmt"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/tools"
	"go-ai-agent/internal/utils"
	"log"
)

type groqClient struct {
	*openAIClient
}

func NewGroqClient(apiKey, baseURL, model string) Client {
	iClient := NewOpenAIClient(apiKey, baseURL, model)
	openaiClient, ok := iClient.(*openAIClient)
	if !ok {
		panic("openai client is not openAIClient")
	}
	return &groqClient{
		openAIClient: openaiClient,
	}
}
func (gc *groqClient) FunctionCall(ctx context.Context, messages []Message, e *tools.Executor) (string, error) {
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	chatMessages := MessagesToChatCompletionParam(messages) // 向 ai 发送的消息, 包含历史数据
	// 第一步: 发送普通请求, 附带工具列表
	resp, err := gc.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: chatMessages,
		//ParallelToolCalls: openai.Bool(false), // 不允许一次调用多个 Tool
		Tools: toolsToChatCompletionToolParam(e.GetToolList()),
		ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String("auto"),
		},
		Model: gc.model,
	})

	if err != nil {
		return "", errno.RequestFailed.WithError(err)
	}
	// 第二步: 如果有工具调用, 则执行工具调用, 限制最大调用次数为 e.MaxSteps
	toolCallCount := 0
	for batchCount := 1; toolCallCount < e.MaxSteps; batchCount++ {
		if resp == nil || len(resp.Choices) == 0 {
			break
		}
		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			return getCompletionRespText(resp)
		}
		chatMessages = append(chatMessages, msg.ToParam())
		log.Printf("第 %d 批工具调用, 本批包含 %d 个工具调用\n", batchCount, len(msg.ToolCalls))
		for _, toolCall := range resp.Choices[0].Message.ToolCalls {
			toolCallCount++
			if toolCallCount > e.MaxSteps {
				return "", errno.ErrToolCallLimitExceed
			}
			toolResp, err := e.Execute(ctx, toolCall.Function.Name, toolCall.Function.Arguments)
			log.Printf("第 %d 次 tool 调用, tool 名称: %s, 返回结果: %s:\n", toolCallCount, toolCall.Function.Name, toolResp)
			if err != nil {
				return "", errno.ErrToolExecuteFailed.WithError(err)
			}
			chatMessages = append(chatMessages, openai.ToolMessage(toolResp, toolCall.ID)) // 附加 tool call 结果以及历史消息信息
		}
		resp, err = gc.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Messages: chatMessages,
			//ParallelToolCalls: openai.Bool(false), // 不允许一次调用多个 Tool
			Tools: toolsToChatCompletionToolParam(e.GetToolList()),
			ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String("auto"),
			},
			Model: gc.model,
		})
		if err != nil {
			return "", errno.RequestFailed.WithError(err)
		}
	}
	needToolCall := resp != nil && len(resp.Choices) != 0 && len(resp.Choices[0].Message.ToolCalls) > 0
	if toolCallCount == e.MaxSteps && needToolCall == true {
		return "", errno.ErrToolCallLimitExceed
	}
	return getCompletionRespText(resp)
}

func MessagesToChatCompletionParam(messages []Message) []openai.ChatCompletionMessageParamUnion {
	var res = make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for _, msg := range messages {
		if msg.Role == SystemMessage {
			res = append(res, openai.SystemMessage(msg.Content))
		} else if msg.Role == UserMessage {
			res = append(res, openai.UserMessage(msg.Content))
		}
	}
	return res
}
func toolsToChatCompletionToolParam(tools []*tools.Tool) []openai.ChatCompletionToolUnionParam {
	toolParams := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		toolParam := openai.ChatCompletionToolUnionParam{
			OfFunction: &openai.ChatCompletionFunctionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        t.Name,
					Description: openai.String(t.Description),
					Parameters:  t.Parameters,
					Strict:      openai.Bool(t.Strict),
				},
			}}
		toolParams = append(toolParams, toolParam)
	}
	return toolParams
}
func getCompletionRespText(resp *openai.ChatCompletion) (string, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("ai 回答为空")
	}
	return resp.Choices[0].Message.Content, nil
}
