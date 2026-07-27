package llm

type IntentResult struct {
	Intent     string  `json:"intent"`     // 用户意图
	Answer     string  `json:"answer"`     // 回答
	Confidence float64 `json:"confidence"` // 置信度
}
