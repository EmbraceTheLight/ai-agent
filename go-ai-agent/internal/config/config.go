package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var (
	OpenaiApiKey  string
	OpenaiBaseURL string
	OpenaiModel   string

	EmbeddingBaseURL string
	EmbeddingModel   string
	EmbeddingDim     int64

	MilvusAddr string
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
	OpenaiModel = firstNonEmpty(os.Getenv("OPENAI_MODEL"), "gpt-5.5")
	EmbeddingBaseURL = strings.TrimRight(strings.TrimSpace(os.Getenv("EMBEDDING_BASE_URL")), "/")
	EmbeddingModel = firstNonEmpty(os.Getenv("EMBEDDING_MODEL"), "qwen3-embedding:0.6b")
	EmbeddingDim = getIntTypeEnv("EMBEDDING_DIM")
	MilvusAddr = firstNonEmpty(os.Getenv("MILVUS_ADDR"), "localhost:19530")
}

// firstNonEmpty 获取 value, 若该值为空, 则返回 fallback
func firstNonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}

func getIntTypeEnv(key string) int64 {
	value, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil {
		panic(err)
	}
	return value
}
