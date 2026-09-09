package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	OpenaiApiKey  string
	OpenaiBaseURL string
	OpenaiModel   string

	EmbeddingBaseURL string
	EmbeddingModel   string
	EmbeddingDim     int64

	MilvusAddr       string
	MilvusUser       string
	MilvusPassword   string
	MilvusCollection string
)

type VectorDatabaseConfig struct {
	Type     string
	Addr     string
	User     string
	PassWord string
}

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

	MilvusAddr = os.Getenv("MILVUS_ADDR")
	MilvusUser = os.Getenv("MILVUS_USER")
	MilvusPassword = os.Getenv("MILVUS_PASSWORD")
	MilvusCollection = os.Getenv("MILVUS_COLLECTION")
}

func NewMilvusConfig(typ string, addr, username, password string) *VectorDatabaseConfig {
	return &VectorDatabaseConfig{Type: typ, Addr: addr, User: username, PassWord: password}
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
