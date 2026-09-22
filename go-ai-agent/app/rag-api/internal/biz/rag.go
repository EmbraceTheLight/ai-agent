package biz

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
}

// RAGConfig 保存 RAG 用例所需的运行参数。
// 输入: 由配置文件转换而来。
// 输出: 为文档切分和 embedding 数量限制提供配置。
// 示例: `&RAGConfig{ChunkSize: 500, Overlap: 100}`。
type RAGConfig struct {
	ChunkSize   int
	Overlap     int
	LimitChunks int
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
	loader      DocumentLoader
	embedder    Embedder
	vectorStore VectorStore
	llm         LLM
	config      *RAGConfig
	log         *slog.Logger
}

// NewRAGUsecase 创建 RAG 用例。
// 输入: `loader`、`embedder`、`vectorStore` 和 `llm` 是 RAG 外部能力端口, `config` 是运行参数, `log` 是结构化日志对象。
// 输出: 返回可执行文档导入和问答流程的 RAG 用例。
// 示例: `NewRAGUsecase(loader, embedder, store, llm, cfg, slog.Default())`。
func NewRAGUsecase(loader DocumentLoader, embedder Embedder, vectorStore VectorStore, llm LLM, config *RAGConfig, log *slog.Logger) *RAGUsecase {
	if config == nil {
		config = &RAGConfig{ChunkSize: 500, Overlap: 100}
	}
	if log == nil {
		log = slog.Default()
	}
	return &RAGUsecase{
		loader:      loader,
		embedder:    embedder,
		vectorStore: vectorStore,
		llm:         llm,
		config:      config,
		log:         log,
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

	docs, err := usecase.loader.Load(path)
	if err != nil {
		return nil, fmt.Errorf("加载文档失败: %w", err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("没有可处理的文档")
	}

	result := &ImportResult{Documents: len(docs)}
	for _, doc := range docs {
		chunks, err := doc.Split(ChunkConfig{
			Size:    usecase.config.ChunkSize,
			Overlap: usecase.config.Overlap,
		})
		if err != nil {
			return nil, fmt.Errorf("切分文档失败: %w", err)
		}
		result.Chunks += len(chunks)

		texts := chunkTexts(chunks, usecase.config.LimitChunks)
		if len(texts) == 0 {
			continue
		}

		embeddings, err := usecase.embedder.Embed(ctx, texts)
		if err != nil {
			return nil, fmt.Errorf("生成文档 embedding 失败: %w", err)
		}
		if len(embeddings) != len(texts) {
			return nil, fmt.Errorf("生成文档 embedding 数量不匹配: 输入 %d, 输出 %d", len(texts), len(embeddings))
		}

		indexedChunks := chunks
		if len(indexedChunks) > len(texts) {
			indexedChunks = indexedChunks[:len(texts)]
		}
		for i, embedding := range embeddings {
			if err := usecase.vectorStore.Add(ctx, Vector(embedding), indexedChunks[i]); err != nil {
				return nil, fmt.Errorf("写入向量库失败: %w", err)
			}
		}
		result.Embedded += len(embeddings)
	}

	usecase.log.InfoContext(ctx, "RAG 文档导入完成",
		"path", path,
		"documents", result.Documents,
		"chunks", result.Chunks,
		"embedded", result.Embedded,
	)
	return result, nil
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
