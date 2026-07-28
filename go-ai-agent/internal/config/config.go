package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	OpenaiApiKey  string
	OpenaiBaseURL string
	OpenaiModel   string
)

func init() {
	initEnvVariable()
}
func initEnvVariable() {
	_, filename, _, _ := runtime.Caller(1)
	rootPath := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	fmt.Println(rootPath)
	err := godotenv.Load(filepath.Join(rootPath, ".env"))
	if err != nil {
		panic(err)
	}
	OpenaiApiKey = strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	OpenaiBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("OPENAI_BASE_URL")), "/")
	OpenaiModel = firstNonEmpty(os.Getenv("OPENAI_MODEL"), "gpt-5.4-mini")
}

// firstNonEmpty 获取 value, 若该值为空, 则返回 fallback
func firstNonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
