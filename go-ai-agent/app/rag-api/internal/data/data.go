package data

import (
	"context"
	"fmt"
	"go-ai-agent/app/rag-api/internal/conf"
	"go-ai-agent/app/rag-api/internal/data/dal/query"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/utils"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/google/wire"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewEmbeddingClient,
	NewMilvusClient,
	NewMySQL,

	NewEmbedderRepo,
	NewDocumentMetadataRepo,
	NewVectorStore,
	NewRAGConfig,
	NewDocumentLoader,
	NewLLMRepo,
)

// Data holds the long-lived storage clients shared by repos.
type Data struct {
	mysqlClient    *gorm.DB
	milvusClient   *milvusclient.Client
	embedderClient *EmbedderClient
}

// EmbedderClient 是基于 HTTP 的 embedding 客户端实现。
// 输入: 保存 embedding 模型名称和通用 HTTP Client。
// 输出: 通过 `Embed` 方法调用外部 embedding 服务。
// 示例: `NewEmbeddingClient("http://localhost:11434", "qwen3-embedding:0.6b")`。
type EmbedderClient struct {
	model      string
	httpClient *utils.HttpClient
	dim        int
}

// NewData opens the database client and returns it with a cleanup function.
func NewData(
	mysqlClient *gorm.DB,
	milvusClient *milvusclient.Client,
	embeddingClient *EmbedderClient) (*Data, func(), error) {

	data := &Data{
		mysqlClient:    mysqlClient,
		embedderClient: embeddingClient,
		milvusClient:   milvusClient,
	}
	return data, func() {
		ctx, cancel := utils.GetContextWithTimeout(context.Background())
		defer cancelIfNotNil(cancel)
		_ = milvusClient.Close(ctx)
	}, nil
}

// NewEmbeddingClient 创建 embedding 客户端。
// 输入: `embedderConf` 是 embedding 服务配置。
// 输出: 返回 `EmbedderClient` 的客户端。
func NewEmbeddingClient(embedderConf *conf.Data) *EmbedderClient {
	return &EmbedderClient{
		model:      embedderConf.Embedder.Model,
		dim:        int(embedderConf.Embedder.Dim),
		httpClient: utils.NewHttpClient(embedderConf.Embedder.BaseUrl),
	}
}

// NewMilvusClient 创建长生命周期的 Milvus 客户端。
// 输入: `conf` 是 Milvus 地址和认证配置。
// 输出: 返回已连接的 Milvus client; 配置或连接失败时返回错误。
// 示例: `client, err := NewMilvusClient(confData.GetMilvus())`。
func NewMilvusClient(conf *conf.Data) (*milvusclient.Client, error) {
	if conf == nil {
		return nil, fmt.Errorf("milvusClient config 不能为空")
	}
	ctx, cancel := utils.GetContextWithTimeout(context.Background())
	defer cancelIfNotNil(cancel)
	milvusClient, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address:  conf.Milvus.Addr,
		Username: conf.Milvus.Username,
		Password: conf.Milvus.Password,
	})
	if err != nil {
		return nil, err
	}
	return milvusClient, nil
}

func NewMySQL(c *conf.Data) *gorm.DB {
	host, port, username, password, dbname := c.Mysql.Host, c.Mysql.Port, c.Mysql.User, c.Mysql.Password, c.Mysql.Database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", username, password, host, port, dbname)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(int(c.Mysql.MaxIdle))
	sqlDB.SetMaxOpenConns(int(c.Mysql.MaxOpen))
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}
	query.SetDefault(db)
	return db
}

// vectorStoreType 获取向量存储类型; 未配置时默认使用 Milvus。
// 输入: `c` 是应用数据配置。
// 输出: 返回规范化后的向量存储类型。
// 示例: `vectorStoreType(conf)` -> `"milvusClient"`。
func vectorStoreType(c *conf.Data) string {
	if c == nil || c.Rag == nil || c.Rag.VectorStore == "" {
		return config.Milvus
	}
	return strings.ToLower(strings.TrimSpace(c.Rag.VectorStore))
}

// cancelIfNotNil 结束可选的 context cancel 函数。
// 输入: `cancel` 可能为 nil 的取消函数。
// 输出: 当取消函数存在时执行取消操作。
// 示例: `cancelIfNotNil(cancel)`。
func cancelIfNotNil(cancel context.CancelFunc) {
	if cancel != nil {
		cancel()
	}
}
