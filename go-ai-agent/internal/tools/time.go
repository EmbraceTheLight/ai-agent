package tools

import (
	"context"
	"encoding/json"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/utils"
	"time"
)

type timeTool struct {
	Func ToolFunc
}

func NewTimeTool() ITool {
	return &timeTool{
		Func: GetCurrentTime,
	}
}

// GetToolParameterJSONSchema 获取 GetCurrentTime Tool 的参数 json schema
func (time *timeTool) GetToolParameterJSONSchema() map[string]any {
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

func (time *timeTool) GetToolHandler() ToolFunc {
	return time.Func
}

type GetCurrentTimeReq struct {
	TimeZone string `json:"time_zone"`
}

// GetCurrentTime 获取当前时间.
// 支持指定时区.
func GetCurrentTime(ctx context.Context, jsonReq json.RawMessage) (string, error) {
	var req GetCurrentTimeReq
	err := json.Unmarshal(jsonReq, &req)
	if err != nil {
		return "", err
	}
	location, err := time.LoadLocation(req.TimeZone)
	if err != nil {
		return "", errno.ErrSetTimezoneFailed.WithError(err).WithMsgf("设置 timezone %s 失败", req.TimeZone)
	}
	ctx, cancel := utils.GetContextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	return time.Now().In(location).Format("2006-01-02 15:04:05"), nil
}
