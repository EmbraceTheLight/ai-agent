package biz

import (
	"context"
	"time"
)

// DocumentMetadataRepo 定义文档索引配置和标题的持久化能力。
// 输入: 文档 ID 或待保存的文档元数据。
// 输出: 返回已保存的元数据、标题映射, 或写入错误。
// 示例: `repo.GetDocumentMetadataList(ctx, []string{id})`。
type DocumentMetadataRepo interface {
	// GetDocumentMetadataList 按文档 ID 批量查询索引元数据。
	GetDocumentMetadataList(ctx context.Context, documentId []string) ([]*DocumentMetadataDO, error)
	// GetDocumentTitleByIdList 按文档 ID 批量查询标题。
	GetDocumentTitleByIdList(ctx context.Context, documentId []string) (map[string]string, error)
	// UpsertDocumentMetadata 插入或更新单篇文档的索引元数据。
	UpsertDocumentMetadata(ctx context.Context, document *DocumentMetadataDO) error
}

// DocumentMetadataDO 保存一篇文档最近一次成功索引时使用的配置。
// 输入: 来自文档、RAG 配置和已完成的索引时间。
// 输出: 供导入流程判断是否需要重新生成 chunk 向量。
// 示例: `DocumentMetadataDO{Id: id, EmbeddingModel: "qwen3-embedding:0.6b"}`。
type DocumentMetadataDO struct {
	Id              string
	SourcePath      string
	Title           string
	CollectionName  string
	EmbeddingModel  string
	EmbeddingMethod string
	EmbeddingDim    int
	ChunkSize       int
	ChunkOverlap    int
	LastIndexedAt   time.Time
}

// ToMapData 将文档元数据转换为按数据库列名索引的键值集合。
// 输入: 接收者包含待写入的索引配置和文档信息。
// 输出: 返回可用于数据库更新的字段映射。
// 示例: `metadata.ToMapData()["document_id"]`。
func (d *DocumentMetadataDO) ToMapData() map[string]any {
	return map[string]any{
		"document_id":      d.Id,
		"source_path":      d.SourcePath,
		"title":            d.Title,
		"collection_name":  d.CollectionName,
		"embedding_model":  d.EmbeddingModel,
		"embedding_method": d.EmbeddingMethod,
		"embedding_dim":    d.EmbeddingDim,
		"chunk_size":       d.ChunkSize,
		"chunk_overlap":    d.ChunkOverlap,
		"last_indexed_at":  d.LastIndexedAt,
	}
}
