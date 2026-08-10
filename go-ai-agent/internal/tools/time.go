package tools

import (
	"context"
	"encoding/json"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/utils"
	"time"
)

type timeTool struct {
	ToolTemplate
}

func NewTimeTool(name, description string) ITool {
	return &timeTool{
		ToolTemplate: NewTemplate(name, description),
	}
}

// GetToolParameterJSONSchema 获取 GetCurrentTime Tool 的参数 json schema
func (t *timeTool) GetToolParameterJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"time_zone": map[string]any{
				"type":        "string",
				"description": "时区",
			},
		},
		"additionalProperties": false,
		"required":             []string{"time_zone"},
	}
}

func (t *timeTool) GetToolHandler() ToolFunc {
	return t.GetCurrentTime
}

type GetCurrentTimeReq struct {
	TimeZone string `json:"time_zone"`
}

// GetCurrentTime 获取当前时间.
// 支持指定时区.
func (t *timeTool) GetCurrentTime(ctx context.Context, jsonReq json.RawMessage) (string, error) {
	var req GetCurrentTimeReq
	err := json.Unmarshal(jsonReq, &req)
	if err != nil {
		return "", errno.RequestInvalid.WithError(err)
	}
	location, err := time.LoadLocation(req.TimeZone)
	if err != nil {
		return "", errno.ErrSetTimezoneFailed.WithError(err).WithMsgf("设置 timezone %s 失败", req.TimeZone)
	}
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	resp := map[string]string{
		"time_zone":    req.TimeZone,
		"current_time": time.Now().In(location).Format("2006-01-02 15:04:05"),
	}
	content, _ := json.Marshal(resp)
	return string(content), nil
}
