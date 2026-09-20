package main

import (
	"context"
	"flag"
	"fmt"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/rag"
	"log"
	"time"
)

type ragEvalConfig struct {
	DocsPath    string
	CasesPath   string
	ChunkSize   int
	Overlap     int
	EmbedURL    string
	EmbedModel  string
	LimitDocs   int
	LimitChunks int
	Timeout     time.Duration
	TopK        int
	Store       string
}

// main 启动离线 RAG Eval。
// 输入: 从命令行 flag 和环境变量读取文档、case、embedding 与向量库配置。
// 输出: 逐条打印检索分类结果，并输出全部 case 的汇总统计。
// 示例: `go run ./cmd/rag-eval -store memory`。
func main() {
	cfg := parseFlags()

	ctx := context.Background()
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	if err := run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}

// parseFlags 解析离线 Eval 的命令行参数。
// 输入: 读取命令行 flag, 并使用配置包中的环境变量作为默认值。
// 输出: 返回 `ragEvalConfig`, 包含文档、case、embedding 和向量库配置。
// 示例: `parseFlags()` -> 返回默认读取 `testdata/eval_case.jsonl` 的配置。
func parseFlags() ragEvalConfig {
	cfg := ragEvalConfig{}

	flag.StringVar(&cfg.DocsPath, "docs", "testdata/documents/work_notes_May/五月/第四周", "文档目录或单个文档路径")
	flag.StringVar(&cfg.CasesPath, "cases", "testdata/eval_case.jsonl", "Eval case JSONL 文件路径")
	flag.IntVar(&cfg.ChunkSize, "chunkSize", 500, "chunk 大小, 按 rune 数量计算")
	flag.IntVar(&cfg.Overlap, "overlap", 100, "相邻 chunk 的重叠大小, 按 rune 数量计算")
	flag.StringVar(&cfg.EmbedURL, "embedURL", config.EmbeddingBaseURL, "embedding 服务地址")
	flag.StringVar(&cfg.EmbedModel, "embedModel", config.EmbeddingModel, "embedding 模型名称")
	flag.IntVar(&cfg.LimitDocs, "limitDocs", 0, "最多处理多少篇文档, 0 表示不限制")
	flag.IntVar(&cfg.LimitChunks, "limitChunks", 0, "每篇文档最多 embed 多少个 chunk, 0 表示不限制")
	flag.DurationVar(&cfg.Timeout, "timeout", config.RequestTimeout, "整个 Eval 流程的超时时间")
	flag.IntVar(&cfg.TopK, "topK", 3, "每个问题检索的 topK chunk 数")
	flag.StringVar(&cfg.Store, "store", "memory", "存储方式, 支持 milvus 和内存存储")
	flag.Parse()

	return cfg
}

// run 执行文档索引、逐题检索和 Eval 汇总。
// 输入: `ctx` 是请求上下文, `cfg` 是离线 Eval 配置。
// 输出: 成功时打印每个 case 的检索分类和总汇; 失败时返回错误。
// 示例: `run(ctx, ragEvalConfig{DocsPath: "testdata/documents", CasesPath: "testdata/eval_case.jsonl"})`。
func run(ctx context.Context, cfg ragEvalConfig) error {
	if cfg.DocsPath == "" {
		return fmt.Errorf("文档路径不能为空")
	}
	if cfg.CasesPath == "" {
		return fmt.Errorf("Eval case 路径不能为空")
	}
	if cfg.EmbedURL == "" {
		return fmt.Errorf("embedding 服务地址不能为空, 请设置 -embedURL 或 EMBEDDING_BASE_URL")
	}
	if cfg.EmbedModel == "" {
		return fmt.Errorf("embedding 模型不能为空, 请设置 -embedModel 或 EMBEDDING_MODEL")
	}
	if cfg.TopK <= 0 {
		return fmt.Errorf("topK 必须大于 0")
	}

	evalConfigs, err := rag.NewEvalCheckConfig(cfg.CasesPath)
	if err != nil {
		return fmt.Errorf("读取 Eval case 失败: %w", err)
	}
	if len(evalConfigs) == 0 {
		return fmt.Errorf("Eval case 为空: %s", cfg.CasesPath)
	}

	embedClient := rag.NewEmbeddingClient(cfg.EmbedURL, cfg.EmbedModel)
	vectorStore, cleanup, err := rag.NewVectorStore(
		config.NewMilvusConfig(cfg.Store, config.MilvusAddr, config.MilvusUser, config.MilvusPassword),
	)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := initMilvusCollection(ctx, vectorStore); err != nil {
		return err
	}

	loader := rag.NewTriliumDocumentLoader(map[string]bool{".md": true}, cfg.LimitDocs)
	docs, err := loader.Load(cfg.DocsPath)
	if err != nil {
		return fmt.Errorf("加载文档失败: %w", err)
	}
	if len(docs) == 0 {
		return fmt.Errorf("没有可处理的文档")
	}

	totalChunks, totalEmbedded, err := indexDocuments(ctx, embedClient, vectorStore, docs, cfg)
	if err != nil {
		return err
	}

	fmt.Println("RAG Eval 配置")
	fmt.Println("文档路径:", cfg.DocsPath)
	fmt.Println("Eval case:", cfg.CasesPath)
	fmt.Println("存储方式:", cfg.Store)
	fmt.Println("chunk size:", cfg.ChunkSize)
	fmt.Println("overlap:", cfg.Overlap)
	fmt.Println("topK:", cfg.TopK)
	fmt.Println("文档数:", len(docs))
	fmt.Println("总 chunk 数:", totalChunks)
	fmt.Println("已写入向量库 chunk 数:", totalEmbedded)
	fmt.Println()

	return evaluateCases(ctx, embedClient, vectorStore, evalConfigs, cfg.TopK)
}

// indexDocuments 将文档切分、向量化并写入指定向量库。
// 输入: `embedClient` 生成 embedding, `vectorStore` 保存向量, `docs` 是待处理文档, `cfg` 是切分配置。
// 输出: 返回原始 chunk 数和成功写入的 embedding 数; 任一步骤失败时返回错误。
// 示例: `indexDocuments(ctx, embedder, store, docs, cfg)`。
func indexDocuments(
	ctx context.Context,
	embedClient rag.Embedder,
	vectorStore rag.VectorStore,
	docs []*rag.Document,
	cfg ragEvalConfig,
) (int, int, error) {
	var totalChunks int
	var totalEmbedded int

	for _, doc := range docs {
		chunks, err := rag.ChunkDocument(doc, cfg.ChunkSize, cfg.Overlap)
		if err != nil {
			return 0, 0, fmt.Errorf("切分文档 %q 失败: %w", doc.SourcePath, err)
		}
		totalChunks += len(chunks)

		texts := chunkTexts(chunks, cfg.LimitChunks)
		if len(texts) == 0 {
			continue
		}

		embeddings, err := embedClient.Embed(ctx, texts)
		if err != nil {
			return 0, 0, fmt.Errorf("生成文档 %q 的 embedding 失败: %w", doc.SourcePath, err)
		}
		if len(embeddings) != len(texts) {
			return 0, 0, fmt.Errorf(
				"生成文档 %q 的 embedding 数量不匹配: 输入 %d, 输出 %d",
				doc.SourcePath,
				len(texts),
				len(embeddings),
			)
		}

		indexedChunks := chunks
		if len(indexedChunks) > len(texts) {
			indexedChunks = indexedChunks[:len(texts)]
		}
		for i, embedding := range embeddings {
			if err := vectorStore.Add(ctx, rag.Vector(embedding), indexedChunks[i]); err != nil {
				return 0, 0, fmt.Errorf("写入向量库失败: %w", err)
			}
		}
		totalEmbedded += len(embeddings)
	}

	return totalChunks, totalEmbedded, nil
}

// evaluateCases 对每个 Eval case 单独生成 query embedding、执行检索并打印检查结果。
// 输入: `embedClient` 生成问题向量, `vectorStore` 执行检索, `evalConfigs` 是固定 case 集, `topK` 是检索数量。
// 输出: 所有 case 完成且汇总通过时返回 nil; 任一 case 处理失败时返回错误。
// 示例: `evaluateCases(ctx, embedder, store, cases, 3)`。
func evaluateCases(
	ctx context.Context,
	embedClient rag.Embedder,
	vectorStore rag.VectorStore,
	evalConfigs []*rag.EvalCheckConfig,
	topK int,
) error {
	var (
		decisionMatches int
		retrievalHits   int
	)

	for i, evalConf := range evalConfigs {
		start := time.Now()
		queryEmbeddings, err := embedClient.Embed(ctx, []string{evalConf.QuestionContent})
		if err != nil {
			return fmt.Errorf("case %s 生成问题 embedding 失败: %w", evalConf.QuestionId, err)
		}
		if len(queryEmbeddings) != 1 {
			return fmt.Errorf(
				"case %s 的问题 embedding 数量不匹配: 期望 1, 实际 %d",
				evalConf.QuestionId,
				len(queryEmbeddings),
			)
		}

		searchResults, err := vectorStore.Search(ctx, rag.Vector(queryEmbeddings[0]), topK)
		if err != nil {
			return fmt.Errorf("case %s 检索失败: %w", evalConf.QuestionId, err)
		}

		result := evalConf.EvalCheck(searchResults)
		decisionMatch := result.EvalShouldAnswer == evalConf.ShouldAnswer
		if decisionMatch {
			decisionMatches++
		}
		if len(result.Legal) > 0 {
			retrievalHits++
		}

		fmt.Printf("===== Case %d/%d: %s =====\n", i+1, len(evalConfigs), evalConf.QuestionId)
		fmt.Println("问题:", evalConf.QuestionContent)
		fmt.Printf("期望回答: %t\n", evalConf.ShouldAnswer)
		printTopK(searchResults)
		result.SumFromCheckResult()
		fmt.Printf("决策是否匹配: %t\n", decisionMatch)
		fmt.Printf("耗时: %s\n\n", time.Since(start))
	}

	fmt.Println("===== Eval 汇总 =====")
	fmt.Printf("总 case 数: %d\n", len(evalConfigs))
	fmt.Printf("回答决策匹配: %d/%d\n", decisionMatches, len(evalConfigs))
	fmt.Printf("命中合法 source: %d/%d\n", retrievalHits, len(evalConfigs))
	return nil
}

// printTopK 打印一次检索返回的 topK 结果。
// 输入: `results` 是向量库按分数降序返回的检索结果。
// 输出: 将每个 chunk 的分数、来源和索引打印到标准输出。
// 示例: `printTopK(results)`。
func printTopK(results []*rag.SearchResult) {
	fmt.Println("topK 检索结果:")
	for i, result := range results {
		if result == nil || result.Chunk == nil {
			fmt.Printf("  [%d] result 为空\n", i+1)
			continue
		}
		fmt.Printf(
			"  [%d] score=%.4f source=%s#chunk-%d\n",
			i+1,
			result.Score,
			result.Chunk.SourceFile,
			result.Chunk.ChunkIndex,
		)
	}
}

// chunkTexts 从 chunk 列表中提取用于 embedding 的文本。
// 输入: `chunks` 是文档 chunk 列表, `limit` 是最多提取的数量; `limit <= 0` 表示不限制。
// 输出: 返回按原顺序排列的 chunk 文本。
// 示例: `chunkTexts(chunks, 3)` -> 返回最多 3 个 chunk 文本。
func chunkTexts(chunks []*rag.Chunk, limit int) []string {
	if limit <= 0 || limit > len(chunks) {
		limit = len(chunks)
	}

	texts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		texts = append(texts, chunks[i].Content)
	}
	return texts
}

// initMilvusCollection 初始化 Milvus collection; 内存向量库不需要初始化。
// 输入: `vectorStore` 是当前使用的向量库。
// 输出: Milvus collection 初始化成功或无需初始化时返回 nil。
// 示例: `initMilvusCollection(ctx, store)`。
func initMilvusCollection(ctx context.Context, vectorStore rag.VectorStore) error {
	if milvusVS, ok := vectorStore.(*rag.MilvusVS); ok {
		return milvusVS.InitCollections(ctx)
	}
	return nil
}
