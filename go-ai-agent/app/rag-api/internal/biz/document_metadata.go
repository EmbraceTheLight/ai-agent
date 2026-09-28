package biz

import "time"

type DocumentMetadataRepo interface {
	GetDocumentMetadataList(documentId []string) ([]*DocumentMetadataDO, error)
	GetDocumentTitleByIdList(documentId []string) (map[string]string, error)
	UpdateDocumentMetadataList(document []*DocumentMetadataDO) error
}

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

func (d *DocumentMetadataDO) ToMapData() map[string]any {
	return map[string]any{
		"id":               d.Id,
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
