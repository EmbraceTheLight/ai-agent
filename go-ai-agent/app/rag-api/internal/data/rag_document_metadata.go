package data

import (
	"context"
	"errors"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/app/rag-api/internal/data/dal/model"
	"go-ai-agent/app/rag-api/internal/data/dal/query"
	"go-ai-agent/internal/utils"
	"gorm.io/gorm"
)

type documentMetadataRepo struct {
	data  *Data
	query query.IRagDocumentMetadatumDo
}

func (d *documentMetadataRepo) GetDocumentMetadataList(documentId []string) ([]*biz.DocumentMetadataDO, error) {
	ragDocumentMetadata := query.RagDocumentMetadatum
	if len(documentId) == 0 {
		return nil, errors.New("document id 为空")
	}
	res, err := d.query.Where(ragDocumentMetadata.DocumentID.In(documentId...)).Find()
	if err != nil {
		return nil, fmt.Errorf("document %s 查询出错: %w", documentId, err)
	}
	var documentMetadata []*biz.DocumentMetadataDO
	for _, v := range res {
		documentMetadata = append(documentMetadata, modelToDocumentMetadata(v))
	}
	return documentMetadata, nil
}

func (d *documentMetadataRepo) UpdateDocumentMetadataList(document []*biz.DocumentMetadataDO) error {
	if len(document) == 0 {
		return errors.New("待更新 document 为空")
	}
	err := d.data.mysqlClient.Transaction(func(tx *gorm.DB) error {
		m := query.Use(tx).RagDocumentMetadatum

		for _, doc := range document {
			if doc == nil || doc.Id == "" {
				return errors.New("document 或 document.Id 为空")
			}

			_, err := m.Where(m.DocumentID.Eq(doc.Id)).Updates(doc.ToMapData())
			if err != nil {
				return fmt.Errorf("document %s 更新出错: %w", doc.Id, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("document更新出错: %w", err)
	}
	return nil
}

func (d *documentMetadataRepo) GetDocumentTitleByIdList(documentIdList []string) (map[string]string, error) {
	ragDocumentMetadata := query.RagDocumentMetadatum
	if len(documentIdList) == 0 {
		return nil, errors.New("document id 为空")
	}
	idTitleMap := make(map[string]string)
	res, err := d.query.Where(ragDocumentMetadata.DocumentID.In(documentIdList...)).Find()
	if err != nil {
		return nil, fmt.Errorf("document %v 查询出错: %w", documentIdList, err)
	}
	for _, doc := range res {
		idTitleMap[doc.DocumentID] = doc.Title
	}
	return idTitleMap, nil
}

func NewDocumentMetadataRepo(data *Data) biz.DocumentMetadataRepo {
	ctx, _ := utils.GetContextWithTimeout(context.Background())
	return &documentMetadataRepo{
		data:  data,
		query: query.RagDocumentMetadatum.WithContext(ctx),
	}
}

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
