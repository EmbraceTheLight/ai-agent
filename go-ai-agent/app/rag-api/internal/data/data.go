package data

import (
	"context"
	"go-ai-agent/app/rag-api/internal/conf"
	"go-ai-agent/internal/utils"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/wire"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewEmbeddingClient,
	NewMilvusClient,

	NewEmbedderRepo,
)

// Data holds the long-lived storage clients shared by repos.
type Data struct {
	milvus       *milvusclient.Client
	embedderData *EmbedderData
}

// EmbedderData 是基于 HTTP 的 embedding 客户端实现。
// 输入: 保存 embedding 模型名称和通用 HTTP Client。
// 输出: 通过 `Embed` 方法调用外部 embedding 服务。
// 示例: `NewEmbeddingClient("http://localhost:11434", "qwen3-embedding:0.6b")`。
type EmbedderData struct {
	model      string
	httpClient *utils.HttpClient
	dim        int
}

// NewEmbeddingClient 创建 embedding 客户端。
// 输入: `embedderConf` 是 embedding 服务配置。
// 输出: 返回 `EmbedderClient` 的客户端。
func NewEmbeddingClient(embedderConf *conf.Data_Embedder) *EmbedderData {
	return &EmbedderData{
		model:      embedderConf.Model,
		dim:        int(embedderConf.Dim),
		httpClient: utils.NewHttpClient(embedderConf.BaseUrl),
	}
}

// NewData opens the database client and returns it with a cleanup function.
func NewData(c *conf.Data) (*Data, func(), error) {
	milvusClient, err := NewMilvusClient(c.Milvus)
	if err != nil {
		return nil, func() {}, err
	}
	ctx, _ := utils.GetContextWithTimeout(context.Background())
	embedderClient := NewEmbeddingClient(c.Embedder)
	return &Data{
			milvus:       milvusClient,
			embedderData: embedderClient,
		}, func() {
			_ = milvusClient.Close(ctx)
		}, nil
}

func NewMilvusClient(conf *conf.Data_Milvus) (*milvusclient.Client, error) {
	ctx, _ := utils.GetContextWithTimeout(context.Background())
	milvusClient, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address:  conf.Addr,
		Username: conf.Username,
		Password: conf.Password,
	})
	if err != nil {
		return nil, err
	}
	return milvusClient, nil
}
