package data

import (
	"context"
	"errors"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/data/dal/model"
	"go-ai-agent/app/rag-api/internal/data/dal/query"
	"gorm.io/gorm/clause"
)

// documentMetadataRepo 通过 MySQL 保存文档的索引配置和标题。
type documentMetadataRepo struct {
	data *Data
}

// GetDocumentMetadataList 按文档 ID 查询已成功索引的配置。
// 输入: `ctx` 是查询上下文, `documentId` 是非空的文档 ID 列表。
// 输出: 返回找到的文档元数据; 查询失败时返回错误。
// 示例: `repo.GetDocumentMetadataList(ctx, []string{id})`。
func (d *documentMetadataRepo) GetDocumentMetadataList(ctx context.Context, documentId []string) ([]*biz.DocumentMetadataDO, error) {
	ragDocumentMetadata := query.RagDocumentMetadatum
	if len(documentId) == 0 {
		return nil, errors.New("document id 为空")
	}
	res, err := ragDocumentMetadata.WithContext(ctx).Where(ragDocumentMetadata.DocumentID.In(documentId...)).Find()
	if err != nil {
		return nil, fmt.Errorf("document %s 查询出错: %w", documentId, err)
	}
	var documentMetadata []*biz.DocumentMetadataDO
	for _, v := range res {
		documentMetadata = append(documentMetadata, modelToDocumentMetadata(v))
	}
	return documentMetadata, nil
}

// UpsertDocumentMetadata 在当前文档的向量变更完成后写入配置和索引时间。
// 输入: `ctx` 是写入上下文, `document` 是包含文档 ID 的完整元数据。
// 输出: 插入或更新成功返回 nil; 参数或数据库写入失败时返回错误。
// 示例: `repo.UpsertDocumentMetadata(ctx, metadata)`。
func (d *documentMetadataRepo) UpsertDocumentMetadata(ctx context.Context, document *biz.DocumentMetadataDO) error {
	if document == nil || document.Id == "" {
		return errors.New("document 或 document.Id 为空")
	}
	return d.data.mysqlClient.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "document_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"source_path", "title", "collection_name", "embedding_model", "embedding_method",
			"embedding_dim", "chunk_size", "chunk_overlap", "last_indexed_at",
		}),
	}).Create(documentMetadataToModel(document)).Error
}

// GetDocumentTitleByIdList 查询文档 ID 对应的标题, 供检索结果展示。
// 输入: `ctx` 是查询上下文, `documentIdList` 是非空的文档 ID 列表。
// 输出: 返回文档 ID 到标题的映射; 查询失败时返回错误。
// 示例: `repo.GetDocumentTitleByIdList(ctx, []string{id})`。
func (d *documentMetadataRepo) GetDocumentTitleByIdList(ctx context.Context, documentIdList []string) (map[string]string, error) {
	ragDocumentMetadata := query.RagDocumentMetadatum
	if len(documentIdList) == 0 {
		return nil, errors.New("document id 为空")
	}
	idTitleMap := make(map[string]string)
	res, err := ragDocumentMetadata.WithContext(ctx).Where(ragDocumentMetadata.DocumentID.In(documentIdList...)).Find()
	if err != nil {
		return nil, fmt.Errorf("document %v 查询出错: %w", documentIdList, err)
	}
	for _, doc := range res {
		idTitleMap[doc.DocumentID] = doc.Title
	}
	return idTitleMap, nil
}

// NewDocumentMetadataRepo 创建基于共享 MySQL 客户端的文档元数据仓库。
// 输入: `data` 保存已初始化的数据库连接。
// 输出: 返回实现 DocumentMetadataRepo 的仓库。
// 示例: `repo := NewDocumentMetadataRepo(data)`。
func NewDocumentMetadataRepo(data *Data) biz.DocumentMetadataRepo {
	return &documentMetadataRepo{
		data: data,
	}
}

// modelToDocumentMetadata 将数据库模型转换为领域元数据。
// 输入: `res` 是从 MySQL 查询到的文档记录。
// 输出: 返回不含数据库标签的 DocumentMetadataDO。
// 示例: `metadata := modelToDocumentMetadata(row)`。
func modelToDocumentMetadata(res *model.RagDocumentMetadatum) *biz.DocumentMetadataDO {
	return &biz.DocumentMetadataDO{
		Id:              res.DocumentID,
		SourcePath:      res.SourcePath,
		Title:           res.Title,
		CollectionName:  res.CollectionName,
		EmbeddingModel:  res.EmbeddingModel,
		EmbeddingMethod: res.EmbeddingMethod,
		EmbeddingDim:    int(res.EmbeddingDim),
		ChunkSize:       int(res.ChunkSize),
		ChunkOverlap:    int(res.ChunkOverlap),
		LastIndexedAt:   res.LastIndexedAt,
	}
}

// documentMetadataToModel 将领域元数据转换为数据库模型。
// 输入: `res` 是待保存的 DocumentMetadataDO。
// 输出: 返回可由 GORM 写入的文档记录。
// 示例: `row := documentMetadataToModel(metadata)`。
func documentMetadataToModel(res *biz.DocumentMetadataDO) *model.RagDocumentMetadatum {
	return &model.RagDocumentMetadatum{
		DocumentID:      res.Id,
		SourcePath:      res.SourcePath,
		Title:           res.Title,
		CollectionName:  res.CollectionName,
		EmbeddingModel:  res.EmbeddingModel,
		EmbeddingMethod: res.EmbeddingMethod,
		EmbeddingDim:    int64(res.EmbeddingDim),
		ChunkSize:       int64(res.ChunkSize),
		ChunkOverlap:    int64(res.ChunkOverlap),
		LastIndexedAt:   res.LastIndexedAt,
	}
}
