package data

import (
	"context"
	"fmt"
	"go-ai-agent/app/rag-api/internal/biz"
	"go-ai-agent/internal/utils"

	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
)

/* Milvus type */

// MilvusCollectionField milvus collection 字段信息, 用于构建 schema 时使用。
type MilvusCollectionField struct {
	Id        int64 `json:"id"`
	Name      string
	DataType  string
	IsPrimary bool
	IsAutoID  bool
	Dim       int
	MaxLength int
}

// MilvusVS milvus vector store。
type MilvusVS struct {
	client     *milvusclient.Client
	collection string
	dim        int
}

// Add 向 Milvus 向量库添加一条 chunk 向量记录。
// 输入: `vector` 是 chunk embedding, `chunk` 是包含来源信息的 chunk。
// 输出: 成功返回 nil; 向量维度、chunk 或 Milvus 请求非法时返回错误。
// 示例: `store.Add(ctx, biz.Vector{1, 0}, chunk)`。
func (md *MilvusVS) Add(ctx context.Context, vector biz.Vector, chunk *biz.Chunk) error {
	if md == nil || md.client == nil {
		return fmt.Errorf("milvus client 未初始化")
	}
	if len(vector) != md.dim {
		return fmt.Errorf("插入的向量维度为 %d, 期望为: %d", len(vector), md.dim)
	}
	if chunk == nil {
		return fmt.Errorf("传入的 chunk 为 nil")
	}
	_, err := md.client.Insert(
		ctx,
		milvusclient.NewColumnBasedInsertOption(md.collection).
			WithInt64Column("id", []int64{utils.GetID()}).
			WithVarcharColumn("source_file_path", []string{chunk.SourceFile}).
			WithVarcharColumn("title", []string{chunk.Title}).
			WithInt32Column("chunk_index", []int32{int32(chunk.ChunkIndex)}).
			WithVarcharColumn("content", []string{chunk.Content}).
			WithInt64Column("created_at", []int64{chunk.CreatedAt}).
			WithInt64Column("updated_at", []int64{chunk.UpdatedAt}).
			WithInt32Column("rune_start_offset", []int32{int32(chunk.RuneStartOffset)}).
			WithInt32Column("rune_end_offset", []int32{int32(chunk.RuneEndOffset)}).
			WithFloatVectorColumn("chunk_vector", md.dim, [][]float32{vector}),
	)
	if err != nil {
		return err
	}
	return nil
}

// Search 在 Milvus 中检索与 queryVector 最相似的 topK 个 chunk。
// 输入: `queryVector` 是问题 embedding, `topK` 是返回数量。
// 输出: 返回按 Milvus 相似度降序排列的结果。
// 示例: `store.Search(ctx, biz.Vector{1, 0}, 3)`。
func (md *MilvusVS) Search(ctx context.Context, queryVector biz.Vector, topK int) ([]*biz.SearchResult, error) {
	if md == nil || md.client == nil {
		return nil, fmt.Errorf("milvus client 未初始化")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK 必须大于 0")
	}
	result, err := md.client.Search(
		ctx,
		milvusclient.NewSearchOption(md.collection, topK, []entity.Vector{entity.FloatVector(queryVector)}).
			WithOutputFields("source_file_path", "title", "content", "chunk_index", "created_at", "updated_at", "rune_start_offset", "rune_end_offset"))
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return []*biz.SearchResult{}, nil
	}

	// 只有一个 queryVector, result 只有一个结果, 即基于该 vector 得到的结果
	return parseSearchResToRagChunk(&result[0])
}

// ResetCollection 删除并重新创建 Milvus collection。
// 输入: `ctx` 用于控制删除、创建索引和加载 collection 的生命周期。
// 输出: collection 不存在或成功重建时返回 nil; 任一步骤失败时返回错误。
// 示例: `milvusVS.ResetCollection(ctx)` -> 使用相同名称创建一个空 collection。
func (md *MilvusVS) ResetCollection(ctx context.Context) error {
	if md == nil || md.client == nil {
		return fmt.Errorf("milvus client 未初始化")
	}
	exists, err := md.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(md.collection))
	if err != nil {
		return err
	}
	if exists {
		err = md.client.DropCollection(ctx, milvusclient.NewDropCollectionOption(md.collection))
		if err != nil {
			return err
		}
	}
	return md.InitCollections(ctx)
}

// InitCollections 检查、创建并加载 Milvus collection。
// 输入: `ctx` 用于控制 collection 检查、创建索引和加载的生命周期。
// 输出: collection 不存在时创建; collection 存在时直接加载; 任一步骤失败时返回错误。
// 示例: `milvusVS.InitCollections(ctx)`。
func (md *MilvusVS) InitCollections(ctx context.Context) error {
	if md == nil || md.client == nil {
		return fmt.Errorf("milvus client 未初始化")
	}
	if md.collection == "" {
		return fmt.Errorf("milvus collection 不能为空")
	}
	if md.dim <= 0 {
		return fmt.Errorf("embedding dim 必须大于 0")
	}

	exists, err := md.client.HasCollection(ctx, milvusclient.NewHasCollectionOption(md.collection))
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
		schema.WithField(entity.NewField().WithName("chunk_vector").WithDataType(entity.FieldTypeFloatVector).WithDim(int64(md.dim)))

		indexOptions := []milvusclient.CreateIndexOption{
			milvusclient.NewCreateIndexOption(md.collection, "chunk_vector", index.NewAutoIndex(entity.COSINE)),
		}
		err = md.client.CreateCollection(ctx, milvusclient.NewCreateCollectionOption(md.collection, schema).WithIndexOptions(indexOptions...))
		if err != nil {
			return err
		}
	}

	task, err := md.client.LoadCollection(ctx, milvusclient.NewLoadCollectionOption(md.collection))
	if err != nil {
		return err
	}
	return task.Await(ctx)
}

// parseSearchResToRagChunk 将 Milvus ResultSet 转换为 biz 检索结果。
// 输入: `searchRes` 是包含输出字段和相似度分数的 Milvus 检索结果。
// 输出: 返回领域 SearchResult 列表; 字段缺失或类型转换失败时返回错误。
// 示例: `parseSearchResToRagChunk(resultSet)`。
func parseSearchResToRagChunk(searchRes *milvusclient.ResultSet) ([]*biz.SearchResult, error) {
	if searchRes == nil {
		return nil, fmt.Errorf("Milvus 检索结果为空")
	}
	ret := make([]*biz.SearchResult, len(searchRes.Scores))
	var err error
	for i := 0; i < len(searchRes.Scores); i++ {
		ret[i] = &biz.SearchResult{Chunk: &biz.Chunk{}}
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

// formatColumnParseError 为 Milvus 字段转换错误补充字段名称。
// 输入: `column` 是转换失败的 Milvus 字段, `err` 是原始转换错误。
// 输出: 返回包含字段名称的格式化错误。
// 示例: `formatColumnParseError(column, err)`。
func formatColumnParseError(column column.Column, err error) error {
	return fmt.Errorf("获取字段 %s 出错: %v", column.Name(), err)
}
