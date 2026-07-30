package tools

import (
	"context"
	"go-ai-agent/internal/errno"
	"go-ai-agent/internal/utils"
	"time"
)

type GetCurrentTimeReq struct {
	TimeZone string `json:"time_zone"`
}

// GetCurrentTime 获取当前时间.
// 支持指定时区, 默认为 UTC.
func GetCurrentTime(ctx context.Context, req *GetCurrentTimeReq) (string, error) {
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
