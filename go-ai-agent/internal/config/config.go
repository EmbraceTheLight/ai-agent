package config

import (
	"os"
	"strings"
)

var (
	OpenaiApiKey  = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	OpenaiBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
	OpenaiModel   = firstNonEmpty(os.Getenv("OPENAI_MODEL"), "gpt-5.4-mini")
)

// firstNonEmpty 获取 value, 若该值为空, 则返回 fallback
func firstNonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
