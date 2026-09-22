package data

import (
	"context"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/conf"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/utils"
)

// NewRAGConfig 将 Kratos 配置转换为 biz 使用的 RAG 配置。
// 输入: `c` 是应用数据配置。
// 输出: 返回文档切分和 embedding 数量限制配置。
// 示例: `NewRAGConfig(confData)` -> `&biz.RAGConfig{ChunkSize: 500, Overlap: 100}`。
func NewRAGConfig(c *conf.Data) *biz.RAGConfig {
	result := &biz.RAGConfig{
		ChunkSize: 500,
		Overlap:   100,
	}
	if c == nil || c.Rag == nil {
		return result
	}
	if c.Rag.ChunkSize > 0 {
		result.ChunkSize = int(c.Rag.ChunkSize)
	}
	if c.Rag.Overlap >= 0 {
		result.Overlap = int(c.Rag.Overlap)
	}
	result.LimitChunks = int(c.Rag.LimitChunks)
	return result
}

// NewVectorStore 根据配置创建内存或 Milvus 向量库，并在 Milvus 模式下完成 collection 初始化。
// 输入: `data` 保存长生命周期客户端, `c` 指定向量库类型、collection 和 embedding 维度。
// 输出: 返回实现 `biz.VectorStore` 的向量库; 配置、连接或 collection 初始化失败时返回错误。
// 示例: `store, err := NewVectorStore(data, confData)`。
func NewVectorStore(data *Data, c *conf.Data) (biz.VectorStore, error) {
	if data == nil {
		return nil, fmt.Errorf("data 不能为空")
	}
	switch vectorStoreType(c) {
	case config.LocalVDB:
		return &memoryVectorStore{}, nil
	case config.Milvus:
		if data.milvus == nil {
			return nil, fmt.Errorf("milvus client 未初始化")
		}
		milvusConf := c.GetMilvus()
		embedderConf := c.GetEmbedder()
		if milvusConf == nil || milvusConf.Collection == "" {
			return nil, fmt.Errorf("milvus collection 不能为空")
		}
		if embedderConf == nil || embedderConf.Dim <= 0 {
			return nil, fmt.Errorf("embedding dim 必须大于 0")
		}
		store := &MilvusVS{
			client:     data.milvus,
			collection: milvusConf.Collection,
			dim:        int(embedderConf.Dim),
		}
		ctx, cancel := contextWithTimeout()
		defer cancelIfNotNil(cancel)
		if err := store.InitCollections(ctx); err != nil {
			return nil, err
		}
		return store, nil
	default:
		return nil, fmt.Errorf("不支持的 vector storage: %s", vectorStoreType(c))
	}
}

// contextWithTimeout 创建用于启动阶段外部资源初始化的超时上下文。
// 输入: 无。
// 输出: 返回带有项目默认超时和取消函数的 context。
// 示例: `ctx, cancel := contextWithTimeout()`。
func contextWithTimeout() (context.Context, context.CancelFunc) {
	return utils.GetContextWithTimeout(context.Background())
}
