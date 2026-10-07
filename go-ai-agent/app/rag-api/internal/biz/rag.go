package biz

import (
	"context"
	"fmt"
	"go-ai-agent/app/rag-api/internal/utils"
	"log/slog"
	"strings"
	"time"
)

// VectorStore 定义 RAG 阶段最小向量库能力。
// 输入: chunk embedding 和 query embedding。
// 输出: 支持写入向量记录并按相似度检索 topK。
// 示例: `store.Add(ctx, vec, chunk); store.Search(ctx, queryVec, 3)`。
type VectorStore interface {
	// Add 保存一个 chunk 及其 embedding 向量。
	// 输入: `Vector` 是 chunk embedding, `chunk` 是包含来源信息的 chunk。
	// 输出: 成功返回 nil; 向量非法或 chunk 为空时返回错误。
	// 示例: `Add(ctx, Vector{1, 0}, chunk)`。
	Add(ctx context.Context, vector Vector, chunk *Chunk) error

	// Search 检索与 queryVector 最相似的 topK 个 chunk。
	// 输入: `queryVector` 是问题 embedding, `topK` 是返回数量。
	// 输出: 返回按相似度降序排列的结果。
	// 示例: `Search(ctx, Vector{1, 0}, 2)`。
	Search(ctx context.Context, queryVector Vector, topK int) ([]*SearchResult, error)

	// VisitDocumentChunks 遍历一篇文档的旧 chunk; handle 每次只接收一批。
	VisitDocumentChunks(ctx context.Context, documentId string, handle func([]*Chunk) error) error
	// UpsertBatch 使用 chunk 的物理主键写入一批向量。
	UpsertBatch(ctx context.Context, records []*Embedding) error
	// DeleteChunksFromIndex 删除此文档中序号不小于 from 的旧 chunk。
	DeleteChunksFromIndex(ctx context.Context, documentId string, from int64) error
}

// RAGConfig 保存 RAG 用例所需的运行参数。
// 输入: 由配置文件转换而来。
// 输出: 为文档切分和 embedding 数量限制提供配置。
// 示例: `&RAGConfig{ChunkSize: 500, Overlap: 100}`。
type RAGConfig struct {
	ChunkSize       int
	Overlap         int
	EmbeddingModel  string
	EmbeddingMethod string
	EmbeddingDim    int
	LimitChunks     int
	CollectionName  string
}

// ImportResult 描述一次文档导入的统计结果。
// 输入: 由 ImportDocuments 根据实际处理结果生成。
// 输出: 保存文档数、chunk 数和写入向量数。
// 示例: `ImportResult{Documents: 1, Chunks: 3, Embedded: 3}`。
type ImportResult struct {
	Documents int
	Chunks    int
	Embedded  int
}

// AskResult 描述一次 RAG 问答的结果。
// 输入: 由 Ask 根据检索结果和模型回答生成。
// 输出: 保存回答文本和原始检索结果。
// 示例: `AskResult{Answer: "...", SearchResults: results}`。
type AskResult struct {
	Answer        string
	SearchResults []*SearchResult
}

// RAGUsecase 编排文档导入和 RAG 问答流程。
// 输入: 依赖文档加载、embedding、向量存储和 LLM 等领域端口。
// 输出: 提供可被 service 调用的完整 RAG 用例。
// 示例: `usecase.ImportDocuments(ctx, "testdata/documents")`。
type RAGUsecase struct {
	loader           DocumentLoader
	embedder         Embedder
	documentMetadata DocumentMetadataRepo
	documentLock     DocumentLockRepo
	vectorStore      VectorStore
	llm              LLM
	config           *RAGConfig
	log              *slog.Logger
}

// NewRAGUsecase 创建 RAG 用例。
// 输入: 加载、embedding、向量库、LLM、元数据和文档锁为外部能力端口, `config` 是运行参数。
// 输出: 返回可执行文档导入和问答流程的 RAG 用例。
// 示例: `NewRAGUsecase(loader, embedder, store, llm, metadata, lock, cfg, slog.Default())`。
func NewRAGUsecase(loader DocumentLoader, embedder Embedder, vectorStore VectorStore, llm LLM, docMetadata DocumentMetadataRepo, docLock DocumentLockRepo, config *RAGConfig, log *slog.Logger) *RAGUsecase {
	if log == nil {
		log = slog.Default()
	}
	return &RAGUsecase{
		loader:           loader,
		embedder:         embedder,
		documentMetadata: docMetadata,
		documentLock:     docLock,
		vectorStore:      vectorStore,
		llm:              llm,
		config:           config,
		log:              log,
	}
}

// ImportDocuments 执行文档加载、chunk 切分、embedding 和向量写入流程。
// 输入: `ctx` 是请求上下文, `path` 是文档文件或目录路径。
// 输出: 返回本次导入的统计结果; 任一步骤失败时返回错误。
// 示例: `usecase.ImportDocuments(ctx, "testdata/documents")`。
func (usecase *RAGUsecase) ImportDocuments(ctx context.Context, path string) (*ImportResult, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("文档路径不能为空")
	}
	if usecase.loader == nil {
		return nil, fmt.Errorf("文档加载器不能为空")
	}
	if usecase.embedder == nil {
		return nil, fmt.Errorf("embedding provider 不能为空")
	}
	if usecase.vectorStore == nil {
		return nil, fmt.Errorf("vector store 不能为空")
	}
	if usecase.documentMetadata == nil || usecase.documentLock == nil || usecase.config == nil {
		return nil, fmt.Errorf("文档元数据、锁或 RAG 配置未初始化")
	}

	docs, err := usecase.loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("加载文档失败: %w", err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("没有可处理的文档")
	}

	result := &ImportResult{}
	for _, doc := range docs {
		if doc == nil {
			return result, fmt.Errorf("文档不能为空")
		}
		absPath, err := utils.GetAbsPath(doc.SourcePath)
		if err != nil {
			return result, fmt.Errorf("获取文档 %q 绝对路径失败: %w", doc.SourcePath, err)
		}
		doc.DocumentId = utils.GetSHA256HexString(absPath)
		if !usecase.documentLock.LockDocument(ctx, doc.DocumentId) {
			return result, fmt.Errorf("文档 %s 加锁失败或请求已取消", doc.SourcePath)
		}
		err = usecase.importDocument(ctx, doc, result)
		usecase.documentLock.UnlockDocument(ctx, doc.DocumentId)
		if err != nil {
			return result, fmt.Errorf("导入文档 %s 失败: %w", doc.SourcePath, err)
		}
		result.Documents++
	}

	usecase.log.InfoContext(ctx, "RAG 文档导入完成",
		"path", path,
		"documents", result.Documents,
		"chunks", result.Chunks,
		"embedded", result.Embedded,
	)
	return result, nil
}

// importDocument 对单篇文档比对旧 chunk, 分批写入向量并提交索引元数据。
// 输入: `ctx` 是请求上下文, `doc` 是已计算 ID 的文档, `result` 收集导入统计。
// 输出: 成功返回 nil; 批次失败不推进 MySQL 元数据, 重试可补齐未完成的写入。
// 示例: `usecase.importDocument(ctx, doc, result)`。
func (usecase *RAGUsecase) importDocument(ctx context.Context, doc *Document, result *ImportResult) error {
	metaList, err := usecase.documentMetadata.GetDocumentMetadataList(ctx, []string{doc.DocumentId})
	if err != nil {
		return fmt.Errorf("查询文档元数据失败: %w", err)
	}
	var oldMeta *DocumentMetadataDO
	if len(metaList) > 0 {
		oldMeta = metaList[0]
	}
	chunks, err := doc.Split(ChunkConfig{Size: usecase.config.ChunkSize, Overlap: usecase.config.Overlap, DocumentId: doc.DocumentId})
	if err != nil {
		return fmt.Errorf("切分文档失败: %w", err)
	}
	if usecase.config.LimitChunks > 0 && len(chunks) > usecase.config.LimitChunks {
		chunks = chunks[:usecase.config.LimitChunks]
	}
	result.Chunks += len(chunks)

	// 只保留当前文档的新 chunk 和变更标记；旧 chunk 按迭代批次释放。
	changed := make([]bool, len(chunks)) // 记录文档所有 chunk 中有哪些 chunk 是更改过的，需要更新
	hasSurplus := false                  // 记录 milvus 中存储的该文档的 chunk 中是否有多余部分，需要删除
	for i := range changed {
		changed[i] = true
	}
	err = usecase.vectorStore.VisitDocumentChunks(ctx, doc.DocumentId, func(batch []*Chunk) error {
		for _, old := range batch {
			if old.ChunkIndex < 0 || old.ChunkIndex >= int64(len(chunks)) {
				hasSurplus = true
				continue
			}
			i := int(old.ChunkIndex)
			current := chunks[i]
			current.Id = old.Id
			current.CreatedAt = old.CreatedAt
			// 检查当前 chunk 是否有变更, 需要更新
			changed[i] = usecase.isChunkNeedUpsert(current, oldMeta, old)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("遍历旧 chunk 失败: %w", err)
	}

	const batchSize = 32
	for start := 0; start < len(chunks); {
		pending := make([]*Chunk, 0, batchSize) // pending 保存待更新的 chunk 列表
		for start < len(chunks) && len(pending) < batchSize {
			if changed[start] {
				pending = append(pending, chunks[start])
			}
			start++
		}
		if len(pending) == 0 {
			continue
		}
		vectors, err := usecase.embedder.Embed(ctx, chunkTexts(pending, 0))
		if err != nil {
			return fmt.Errorf("生成文档 embedding 失败: %w", err)
		}
		if len(vectors) != len(pending) {
			return fmt.Errorf("生成文档 embedding 数量不匹配: 输入 %d, 输出 %d", len(pending), len(vectors))
		}
		records := make([]*Embedding, len(pending))
		for i := range pending {
			records[i] = &Embedding{Chunk: pending[i], Vector: Vector(vectors[i])}
		}
		if err := usecase.vectorStore.UpsertBatch(ctx, records); err != nil {
			return fmt.Errorf("批量写入向量库失败: %w", err)
		}
		result.Embedded += len(records)
	}

	// 文档存在旧的多余的 chunk, 需要删除
	if hasSurplus {
		if err := usecase.vectorStore.DeleteChunksFromIndex(ctx, doc.DocumentId, int64(len(chunks))); err != nil {
			return fmt.Errorf("删除多余 chunk 失败: %w", err)
		}
	}

	// 对比文档元数据是否有改变. 如果没有, 且文档 chunk 内容没有改变, 则直接返回 nil. 否则更新该文档元数据
	if oldMeta != nil && usecase.configMatches(oldMeta) && oldMeta.Title == doc.Title && oldMeta.SourcePath == doc.SourcePath {
		anyChanged := false
		for _, v := range changed {
			anyChanged = anyChanged || v
		}
		if !anyChanged && !hasSurplus {
			return nil
		}
	}

	// 更新 mysql 对应 doc 文档元数据记录
	return usecase.documentMetadata.UpsertDocumentMetadata(ctx, &DocumentMetadataDO{
		Id: doc.DocumentId, SourcePath: doc.SourcePath, Title: doc.Title,
		CollectionName: usecase.config.CollectionName, EmbeddingModel: usecase.config.EmbeddingModel,
		EmbeddingMethod: usecase.config.EmbeddingMethod, EmbeddingDim: usecase.config.EmbeddingDim,
		ChunkSize: usecase.config.ChunkSize, ChunkOverlap: usecase.config.Overlap, LastIndexedAt: time.Now(),
	})
}

// Ask 执行问题 embedding、向量检索、prompt 构建和 LLM 回答流程。
// 输入: `ctx` 是请求上下文, `question` 是用户问题, `topK` 是检索数量。
// 输出: 返回模型回答和原始检索结果; 任一步骤失败时返回错误。
// 示例: `usecase.Ask(ctx, "什么是 RAG?", 3)`。
func (usecase *RAGUsecase) Ask(ctx context.Context, question string, topK int) (*AskResult, error) {
	question = strings.TrimSpace(question)
	if question == "" {
		return nil, fmt.Errorf("问题不能为空")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("topK 必须大于 0")
	}
	if usecase.embedder == nil {
		return nil, fmt.Errorf("embedding provider 不能为空")
	}
	if usecase.vectorStore == nil {
		return nil, fmt.Errorf("vector store 不能为空")
	}
	if usecase.llm == nil {
		return nil, fmt.Errorf("LLM provider 不能为空")
	}

	queryEmbeddings, err := usecase.embedder.Embed(ctx, []string{question})
	if err != nil {
		return nil, fmt.Errorf("生成问题 embedding 失败: %w", err)
	}
	if len(queryEmbeddings) != 1 {
		return nil, fmt.Errorf("问题 embedding 数量不匹配: 期望 1, 实际 %d", len(queryEmbeddings))
	}

	searchResults, err := usecase.vectorStore.Search(ctx, Vector(queryEmbeddings[0]), topK)
	if err != nil {
		return nil, fmt.Errorf("检索相关 chunk 失败: %w", err)
	}

	// 获取 chunk 标题
	// 1. 统计 chunk 列表中包含的 document_id
	seen := make(map[string]struct{})
	documentIdList := make([]string, 0)
	for _, res := range searchResults {
		if _, ok := seen[res.Chunk.DocumentId]; ok {
			continue
		}
		seen[res.Chunk.DocumentId] = struct{}{}
		documentIdList = append(documentIdList, res.Chunk.DocumentId)
	}

	// 2. 查询 document_metadata 获取 document_id 对应的标题
	idTitleMap := make(map[string]string)
	if len(documentIdList) > 0 {
		idTitleMap, err = usecase.documentMetadata.GetDocumentTitleByIdList(ctx, documentIdList)
		if err != nil {
			return nil, fmt.Errorf("查询 document_metadata 失败: %w", err)
		}
	}
	// 2.1 为每个 chunk 的 Title 字段赋值
	for i := range searchResults {
		searchResults[i].Chunk.Title = idTitleMap[searchResults[i].Chunk.DocumentId]
	}
	// 3. 将检索结果构造 prompt 信息发送给 llm, 并获取回答
	answer, err := usecase.llm.Generate(ctx, BuildPrompt(searchResults), question)
	if err != nil {
		return nil, fmt.Errorf("生成 RAG 回答失败: %w", err)
	}
	return &AskResult{Answer: answer, SearchResults: searchResults}, nil
}

// chunkTexts 从 chunk 列表中提取用于 embedding 的文本。
// 输入: `chunks` 是文档 chunk 列表, `limit` 是最多提取的数量; `limit <= 0` 表示不限制。
// 输出: 返回按原顺序排列的 chunk 文本。
// 示例: `chunkTexts(chunks, 3)` -> 返回最多 3 个 chunk 文本。
func chunkTexts(chunks []*Chunk, limit int) []string {
	if limit <= 0 || limit > len(chunks) {
		limit = len(chunks)
	}

	texts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		texts = append(texts, chunks[i].Content)
	}
	return texts
}

// isChunkNeedUpsert 比较索引配置、内容和位置, 判断现有 chunk 是否需要更新。
// 输入: `srcChunk` 是当前切分结果, `docMeta` 是上次索引配置, `targetChunk` 是已存记录。
// 输出: 配置或 chunk 信息不一致时返回 true。
// 示例: `usecase.isChunkNeedUpsert(current, metadata, stored)`。
func (usecase *RAGUsecase) isChunkNeedUpsert(srcChunk *Chunk, docMeta *DocumentMetadataDO, targetChunk *Chunk) bool {
	if !usecase.configMatches(docMeta) {
		return true
	}
	return srcChunk.ChunkHash != targetChunk.ChunkHash ||
		srcChunk.RuneStartOffset != targetChunk.RuneStartOffset ||
		srcChunk.RuneEndOffset != targetChunk.RuneEndOffset ||
		srcChunk.SourceFile != targetChunk.SourceFile
}

// configMatches 比较当前 RAG 配置与文档上次成功索引时的配置。
// 输入: `meta` 是数据库中的文档元数据, 可以为 nil。
// 输出: 全部参与索引的配置一致时返回 true。
// 示例: `usecase.configMatches(metadata)`。
func (usecase *RAGUsecase) configMatches(meta *DocumentMetadataDO) bool {
	return meta != nil && usecase.config.CollectionName == meta.CollectionName &&
		usecase.config.EmbeddingModel == meta.EmbeddingModel &&
		usecase.config.EmbeddingMethod == meta.EmbeddingMethod &&
		usecase.config.EmbeddingDim == meta.EmbeddingDim &&
		usecase.config.ChunkSize == meta.ChunkSize &&
		usecase.config.Overlap == meta.ChunkOverlap
}
