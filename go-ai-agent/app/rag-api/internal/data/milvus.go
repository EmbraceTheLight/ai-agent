package data

import (
	"context"
	"fmt"
	"github.com/milvus-io/milvus/client/v2/column"
	"github.com/milvus-io/milvus/client/v2/entity"
	"github.com/milvus-io/milvus/client/v2/index"
	"github.com/milvus-io/milvus/client/v2/milvusclient"
	"go-ai-agent/app/rag-api/internal/biz"
	"io"
)

/* Milvus type */

// MilvusCollectionField milvusClient collection 字段信息, 用于构建 schema 时使用。
type MilvusCollectionField struct {
	Id        int64 `json:"id"`
	Name      string
	DataType  string
	IsPrimary bool
	IsAutoID  bool
	Dim       int
	MaxLength int
}

// MilvusVS milvusClient vector store。
type MilvusVS struct {
	client     *milvusclient.Client
	collection string
	batchSize  int
	dim        int
}

// VisitDocumentChunks 按主键顺序遍历指定文档的旧 chunk。
// 输入: `documentId` 是文档 ID, `handle` 逐批处理查询结果, `ctx` 控制读取。
// 输出: 读完返回 nil; 查询、解析或处理批次失败时返回错误。
// 示例: `store.VisitDocumentChunks(ctx, id, func(batch []*biz.Chunk) error { return nil })`。
func (md *MilvusVS) VisitDocumentChunks(ctx context.Context, documentId string, handle func([]*biz.Chunk) error) error {
	batchSize := md.batchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	iterator, err := md.client.QueryIterator(ctx, milvusclient.NewQueryIteratorOption(md.collection).
		WithFilter(fmt.Sprintf("document_id == %q", documentId)).
		WithOutputFields(getOutputFields()...).WithBatchSize(batchSize).
		WithConsistencyLevel(entity.ClStrong))
	if err != nil {
		return fmt.Errorf("创建 document %s iterator 失败: %w", documentId, err)
	}
	for {
		res, err := iterator.Next(ctx)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取 document %s chunk 失败: %w", documentId, err)
		}
		parsed, err := parseSearchResToRagChunk(&res)
		if err != nil {
			return err
		}
		batch := make([]*biz.Chunk, len(parsed))
		for i, row := range parsed {
			batch[i] = row.Chunk
		}
		if err := handle(batch); err != nil {
			return err
		}
	}
}

// Add 向 Milvus 向量库添加一条 chunk 向量记录。
// 输入: `vector` 是 chunk embedding, `chunk` 是包含来源信息的 chunk。
// 输出: 成功返回 nil; 向量维度、chunk 或 Milvus 请求非法时返回错误。
// 示例: `store.Add(ctx, biz.Vector{1, 0}, chunk)`。
func (md *MilvusVS) Add(ctx context.Context, vector biz.Vector, chunk *biz.Chunk) error {
	return md.UpsertBatch(ctx, []*biz.Embedding{{Chunk: chunk, Vector: vector}})
}

// UpsertBatch 使用 chunk 的物理主键更新或插入一批向量记录。
// 输入: `records` 包含非空 chunk、document ID 和符合 collection 维度的向量。
// 输出: 批量写入成功返回 nil; 参数或 Milvus 请求失败时返回错误。
// 示例: `store.UpsertBatch(ctx, []*biz.Embedding{{Chunk: chunk, Vector: vector}})`。
func (md *MilvusVS) UpsertBatch(ctx context.Context, records []*biz.Embedding) error {
	if md == nil || md.client == nil {
		return fmt.Errorf("milvusClient client 未初始化")
	}
	if len(records) == 0 {
		return nil
	}
	ids := make([]int64, len(records))
	documentIDs := make([]string, len(records))
	paths := make([]string, len(records))
	hashes := make([]string, len(records))
	indices := make([]int32, len(records))
	contents := make([]string, len(records))
	created := make([]int64, len(records))
	updated := make([]int64, len(records))
	starts := make([]int32, len(records))
	ends := make([]int32, len(records))
	vectors := make([][]float32, len(records))
	for i, record := range records {
		if record == nil || record.Chunk == nil || record.Chunk.DocumentId == "" {
			return fmt.Errorf("第 %d 条 chunk 或 document_id 为空", i)
		}
		if len(record.Vector) != md.dim {
			return fmt.Errorf("第 %d 条向量维度为 %d, 期望 %d", i, len(record.Vector), md.dim)
		}
		chunk := record.Chunk
		ids[i], documentIDs[i], paths[i] = chunk.Id, chunk.DocumentId, chunk.SourceFile
		hashes[i], indices[i], contents[i] = chunk.ChunkHash, int32(chunk.ChunkIndex), chunk.Content
		created[i], updated[i] = chunk.CreatedAt, chunk.UpdatedAt
		starts[i], ends[i] = int32(chunk.RuneStartOffset), int32(chunk.RuneEndOffset)
		vectors[i] = record.Vector
	}
	_, err := md.client.Upsert(ctx, milvusclient.NewColumnBasedInsertOption(md.collection).
		WithInt64Column("id", ids).
		WithVarcharColumn("document_id", documentIDs).
		WithVarcharColumn("source_file_path", paths).
		WithVarcharColumn("chunk_hash", hashes).
		WithInt32Column("chunk_index", indices).
		WithVarcharColumn("content", contents).
		WithInt64Column("created_at", created).
		WithInt64Column("updated_at", updated).
		WithInt32Column("rune_start_offset", starts).
		WithInt32Column("rune_end_offset", ends).
		WithFloatVectorColumn("chunk_vector", md.dim, vectors))
	return err
}

// DeleteChunksFromIndex 删除文档中序号不小于 from 的旧 chunk。
// 输入: `documentId` 是文档 ID, `from` 是本次保留的 chunk 数量。
// 输出: 删除成功返回 nil; Milvus 请求失败时返回错误。
// 示例: `store.DeleteChunksFromIndex(ctx, id, int64(len(chunks)))`。
func (md *MilvusVS) DeleteChunksFromIndex(ctx context.Context, documentId string, from int64) error {
	_, err := md.client.Delete(ctx, milvusclient.NewDeleteOption(md.collection).
		WithExpr(fmt.Sprintf("document_id == %q && chunk_index >= %d", documentId, from)))
	return err
}

// Search 在 Milvus 中检索与 queryVector 最相似的 topK 个 chunk。
// 输入: `queryVector` 是问题 embedding, `topK` 是返回数量。
// 输出: 返回按 Milvus 相似度降序排列的结果。
// 示例: `store.Search(ctx, biz.Vector{1, 0}, 3)`。
func (md *MilvusVS) Search(ctx context.Context, queryVector biz.Vector, topK int) ([]*biz.SearchResult, error) {
	if md == nil || md.client == nil {
		return nil, fmt.Errorf("milvusClient client 未初始化")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK 必须大于 0")
	}

	result, err := md.client.Search(ctx,
		milvusclient.NewSearchOption(md.collection, topK, []entity.Vector{entity.FloatVector(queryVector)}).
			WithOutputFields(getOutputFields()...))
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
		return fmt.Errorf("milvusClient client 未初始化")
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
		return fmt.Errorf("milvusClient client 未初始化")
	}
	if md.collection == "" {
		return fmt.Errorf("milvusClient collection 不能为空")
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
		schema.WithField(entity.NewField().WithName("document_id").WithDataType(entity.FieldTypeVarChar).WithMaxLength(128).WithIsAutoID(false))
		schema.WithField(entity.NewField().WithName("source_file_path").WithDataType(entity.FieldTypeVarChar).WithMaxLength(512))
		schema.WithField(entity.NewField().WithName("chunk_index").WithDataType(entity.FieldTypeInt32))
		schema.WithField(entity.NewField().WithName("chunk_hash").WithDataType(entity.FieldTypeVarChar).WithMaxLength(64))
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
	ret := make([]*biz.SearchResult, searchRes.Len())
	var err error
	for i := 0; i < searchRes.Len(); i++ {
		ret[i] = &biz.SearchResult{Chunk: &biz.Chunk{}}
		idColumn := searchRes.GetColumn("id")
		if idColumn == nil {
			idColumn = searchRes.IDs
		}
		documentIDColumn := searchRes.GetColumn("document_id")
		searchFilePathColumn := searchRes.GetColumn("source_file_path")
		chunkHashColumn := searchRes.GetColumn("chunk_hash")
		chunkIndexColumn := searchRes.GetColumn("chunk_index")
		contentColumn := searchRes.GetColumn("content")
		creationTimeColumn := searchRes.GetColumn("created_at")
		updatedTimeColumn := searchRes.GetColumn("updated_at")
		runeStartOffsetColumn := searchRes.GetColumn("rune_start_offset")
		runeEndOffsetColumn := searchRes.GetColumn("rune_end_offset")
		if idColumn == nil || documentIDColumn == nil {
			return nil, fmt.Errorf("字段 id 或 document_id 为 nil")
		}
		if searchFilePathColumn == nil {
			return nil, fmt.Errorf("字段 source_file_path 为 nil")
		}
		if chunkHashColumn == nil {
			return nil, fmt.Errorf("字段 chunk_hash 为 nil")
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
		ret[i].Chunk.Id, err = idColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(idColumn, err)
		}
		ret[i].Chunk.DocumentId, err = documentIDColumn.GetAsString(i)
		if err != nil {
			return nil, formatColumnParseError(documentIDColumn, err)
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

		ret[i].Chunk.ChunkHash, err = chunkHashColumn.GetAsString(i)
		if err != nil {
			return nil, formatColumnParseError(chunkHashColumn, err)
		}

		ret[i].Chunk.RuneStartOffset, err = runeStartOffsetColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(runeStartOffsetColumn, err)
		}

		ret[i].Chunk.RuneEndOffset, err = runeEndOffsetColumn.GetAsInt64(i)
		if err != nil {
			return nil, formatColumnParseError(runeEndOffsetColumn, err)
		}
		if len(searchRes.Scores) > i {
			ret[i].Score = float64(searchRes.Scores[i])
		}
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

// getOutputFields 列出解析 Milvus chunk 记录所需的标量字段。
// 输入: 无。
// 输出: 返回 Query iterator 和 Search 共用的输出列名, 不包含向量和分数。
// 示例: `option.WithOutputFields(getOutputFields()...)`。
func getOutputFields() []string {
	return []string{"id", "document_id", "source_file_path", "content", "chunk_index", "chunk_hash", "created_at", "updated_at", "rune_start_offset", "rune_end_offset"}
}
