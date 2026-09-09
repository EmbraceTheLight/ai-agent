package rag

import (
	"context"
	"fmt"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/utils"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

/* Milvus type */

// MilvusCollectionField milvus collection 字段信息, 用于构建 schema 时使用
type MilvusCollectionField struct {
	Id        int64 `json:"id"`
	Name      string
	DataType  string
	IsPrimary bool
	IsAutoID  bool
	Dim       int
	MaxLength int
}

// MilvusVS milvus vector store
type MilvusVS struct {
	client *milvusclient.Client
}

func NewMilvusVS(conf *config.VectorDatabaseConfig) (milvusVs *MilvusVS, cleanup func(), err error) {
	ctx, _ := utils.GetContextWithTimeout(context.Background())
	milvusClient, err := milvusclient.New(ctx, &milvusclient.ClientConfig{
		Address:  conf.Addr,
		Username: conf.User,
		Password: conf.PassWord,
	})
	if err != nil {
		return nil, func() {}, err
	}
	return &MilvusVS{client: milvusClient}, func() { milvusClient.Close(ctx) }, nil
}
func (md *MilvusVS) Add(ctx context.Context, vector Vector, chunk *Chunk) error {
	if len(vector) != int(config.EmbeddingDim) {
		return fmt.Errorf("插入的向量维度为 %d, 期望为: %d", len(vector), config.EmbeddingDim)
	}
	if chunk == nil {
		return fmt.Errorf("传入的 chunk 为 nil")
	}
	_, err := md.client.Insert(
		ctx,
		milvusclient.NewColumnBasedInsertOption(config.MilvusCollection).
			WithInt64Column("id", []int64{utils.GetID()}).
			WithVarcharColumn("source_file_path", []string{chunk.SourceFile}).
			WithVarcharColumn("title", []string{chunk.Title}).
			WithInt32Column("chunk_index", []int32{int32(chunk.ChunkIndex)}).
			WithVarcharColumn("content", []string{chunk.Content}).
			WithInt64Column("created_at", []int64{chunk.CreatedAt}).
			WithInt64Column("updated_at", []int64{chunk.UpdatedAt}).
			WithInt32Column("rune_start_offset", []int32{int32(chunk.RuneStartOffset)}).
			WithInt32Column("rune_end_offset", []int32{int32(chunk.RuneEndOffset)}).
			WithFloatVectorColumn("chunk_vector", int(config.EmbeddingDim), [][]float32{vector}),
	)
	if err != nil {
		return err
	}
	return nil
}

func (md *MilvusVS) Search(ctx context.Context, queryVector Vector, topK int) ([]*SearchResult, error) {
	result, err := md.client.Search(
		ctx,
		milvusclient.NewSearchOption(config.MilvusCollection, topK, []entity.Vector{entity.FloatVector(queryVector)}).
			WithOutputFields("source_file_path", "title", "content", "chunk_index", "created_at", "updated_at", "rune_start_offset", "rune_end_offset"))
	if err != nil {
		return nil, err
	}

	// 只有一个 queryVector, result 只有一个结果, 即基于该 vector 得到的结果
	return parseSearchResToRagChunk(&result[0])
}

func (md *MilvusVS) InitCollections(ctx context.Context) error {
	exists, err := md.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(config.MilvusCollection))
	if err != nil {
		return err
	}
	if exists == false {
		schema := entity.NewSchema().WithDynamicFieldEnabled(true)
		schema.WithField(entity.NewField().WithName("id").WithDataType(entity.FieldTypeInt64).WithIsPrimaryKey(true))
		schema.WithField(entity.NewField().WithName("source_file_path").WithDataType(entity.FieldTypeVarChar).WithMaxLength(512))
		schema.WithField(entity.NewField().WithName("title").WithDataType(entity.FieldTypeVarChar).WithMaxLength(256))
		schema.WithField(entity.NewField().WithName("chunk_index").WithDataType(entity.FieldTypeInt32))
		schema.WithField(entity.NewField().WithName("content").WithDataType(entity.FieldTypeVarChar).WithMaxLength(4096))
		schema.WithField(entity.NewField().WithName("created_at").WithDataType(entity.FieldTypeInt64))
		schema.WithField(entity.NewField().WithName("updated_at").WithDataType(entity.FieldTypeInt64))
		schema.WithField(entity.NewField().WithName("rune_start_offset").WithDataType(entity.FieldTypeInt32))
		schema.WithField(entity.NewField().WithName("rune_end_offset").WithDataType(entity.FieldTypeInt32))
		schema.WithField(entity.NewField().WithName("chunk_vector").WithDataType(entity.FieldTypeFloatVector).WithDim(config.EmbeddingDim))

		collectionName := config.MilvusCollection
		indexOptions := []milvusclient.CreateIndexOption{
			milvusclient.NewCreateIndexOption(collectionName, "chunk_vector", index.NewAutoIndex(entity.COSINE)),
		}
		err = md.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(collectionName, schema).WithIndexOptions(indexOptions...))
		if err != nil {
			return err
		}
	}

	task, err := md.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(config.MilvusCollection))
	if err != nil {
		return err
	}
	return task.Await(ctx)
}

func parseSearchResToRagChunk(searchRes *milvusclient.ResultSet) ([]*SearchResult, error) {
	ret := make([]*SearchResult, len(searchRes.Scores))
	var err error
	for i := 0; i < len(searchRes.Scores); i++ {
		ret[i] = &SearchResult{Chunk: &Chunk{}}
		searchFilePathColumn := searchRes.GetColumn("source_file_path")
		titleColumn := searchRes.GetColumn("title")
		chunkIndexColumn := searchRes.GetColumn("chunk_index")
		contentColumn := searchRes.GetColumn("content")
		creationTimeColumn := searchRes.GetColumn("created_at")
		updatedTimeColumn := searchRes.GetColumn("updated_at")
		runeStartOffsetColumn := searchRes.GetColumn("rune_start_offset")
		runeEndOffsetColumn := searchRes.GetColumn("rune_end_offset")
		if searchFilePathColumn == nil {
			return nil, fmt.Errorf("字段 source_file_path 为 nil")
		}
		if titleColumn == nil {
			return nil, fmt.Errorf("字段 title 为 nil")
		}
		if chunkIndexColumn == nil {
			return nil, fmt.Errorf("字段 chunk_index 为 nil")
		}
		if contentColumn == nil {
			return nil, fmt.Errorf("字段 content 为 nil")
		}
		if creationTimeColumn == nil {
			return nil, fmt.Errorf("字段 created_at 为 nil")
		}
		if updatedTimeColumn == nil {
			return nil, fmt.Errorf("字段 updated_at 为 nil")
		}
		if runeStartOffsetColumn == nil {
			return nil, fmt.Errorf("字段 rune_start_offset 为 nil")
		}
		if runeEndOffsetColumn == nil {
			return nil, fmt.Errorf("字段 rune_end_offset 为 nil")
		}
		ret[i].Chunk.SourceFile, err = searchFilePathColumn.GetAsString(i)
		if err != nil {
			return nil, formatColumnParseError(searchFilePathColumn, err)
		}

		ret[i].Chunk.ChunkIndex, err = chunkIndexColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(chunkIndexColumn, err)
		}

		ret[i].Chunk.Content, err = contentColumn.GetAsString(i)
		if err != nil {
			return nil, formatColumnParseError(contentColumn, err)
		}

		ret[i].Chunk.CreatedAt, err = creationTimeColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(creationTimeColumn, err)
		}

		ret[i].Chunk.UpdatedAt, err = updatedTimeColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(updatedTimeColumn, err)
		}

		ret[i].Chunk.Title, err = titleColumn.GetAsString(i)
		if err != nil {
			return nil, formatColumnParseError(titleColumn, err)
		}

		ret[i].Chunk.RuneStartOffset, err = runeStartOffsetColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(runeStartOffsetColumn, err)
		}

		ret[i].Chunk.RuneEndOffset, err = runeEndOffsetColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(runeEndOffsetColumn, err)
		}
		ret[i].Score = float64(searchRes.Scores[i])
	}
	return ret, nil
}

func formatColumnParseError(column column.Column, err error) error {
	return fmt.Errorf("获取字段 %s 出错: %v", column.Name(), err)
}
