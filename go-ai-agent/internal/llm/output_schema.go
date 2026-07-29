package llm

import (
	"fmt"
	"go-ai-agent/internal/config"
)

type IntentResult struct {
	Intent     string  `json:"intent"`     // 用户意图
	Answer     string  `json:"answer"`     // 回答
	Confidence float64 `json:"confidence"` // 置信度
}

// IsValid 检查 json 格式输出内容是否正确
// 检查置信度是否在 0 -- 1 之间; 另外还会检查 Intent 类型是否是四种规定类型之一
func (res *IntentResult) IsValid() error {
	if res.Confidence > 1 || res.Confidence < 0 {
		return fmt.Errorf("置信度不在0-1之间, 当前置信度: %f", res.Confidence)
	}
	if res.Intent != config.AgentQuestion && res.Intent != config.RagQuestion && res.Intent != config.ToolQuestion && res.Intent != config.GeneralQuestion {
		return fmt.Errorf("意图不在指定范围内, 当前意图: %s", res.Intent)
	}
	return nil
}
