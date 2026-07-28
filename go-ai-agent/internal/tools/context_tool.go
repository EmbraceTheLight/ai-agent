package tools

import (
	"context"
	"go-ai-agent/internal/config"
)

// GetContextWithTimeout 获取带有超时的 context
// 若传入的 ctx 未设置超时时间, 则创建一个超时时间为 config.RequestTimeout 的 context 并返回
func GetContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	_, ok := ctx.Deadline()
	if ok == false {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.RequestTimeout)
		return ctx, cancel
	}
	return ctx, nil
}
