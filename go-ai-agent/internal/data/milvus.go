package data

import (
	"context"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/rag"
)

type MilvusOperation interface {
}

/* Milvus type */

// MilvusCollectionField milvus collection 字段信息, 用于构建 schema 时使用
type MilvusCollectionField struct {
	Name      string
	DataType  string
	IsPrimary bool
	IsAutoID  bool
	Dim       int
	MaxLength int
}

type milvusData struct {
	data *Data
}

func NewMilvusData(data *Data) rag.MilvusOperation {
	return &milvusData{
		data: data,
	}
}

func (md *milvusData) InitCollections(ctx context.Context) error {
	schema := entity.NewSchema().WithDynamicFieldEnabled(true)
	schema.WithField(entity.NewField().WithName("id").WithIsAutoID(true).WithDataType(entity.FieldTypeVarChar).WithIsPrimaryKey(true))
	schema.WithField(entity.NewField().WithName("source_file_path").WithDataType(entity.FieldTypeVarChar).WithMaxLength(512))
	schema.WithField(entity.NewField().WithName("title").WithDataType(entity.FieldTypeVarChar).WithMaxLength(256))
	schema.WithField(entity.NewField().WithName("chunk_index").WithDataType(entity.FieldTypeInt32))
	schema.WithField(entity.NewField().WithName("content").WithDataType(entity.FieldTypeVarChar).WithMaxLength(4096))
	schema.WithField(entity.NewField().WithName("created_at").WithDataType(entity.FieldTypeInt64))
	schema.WithField(entity.NewField().WithName("updated_at").WithDataType(entity.FieldTypeInt64))
	schema.WithField(entity.NewField().WithName("rune_start_offset").WithDataType(entity.FieldTypeInt32))
	schema.WithField(entity.NewField().WithName("rune_end_offset").WithDataType(entity.FieldTypeInt32))
	schema.WithField(entity.NewField().WithName("chunk_vector").WithDataType(entity.FieldTypeFloatVector).WithDim(config.EmbeddingDim))

	collectionName := "qwen3_embedding_chunk"
	indexOptions := []milvusclient.CreateIndexOption{
		milvusclient.NewCreateIndexOption(collectionName, "chunk_vector", index.NewAutoIndex(entity.COSINE)),
	}
	return md.data.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(collectionName, schema).WithIndexOptions(indexOptions...))
}
