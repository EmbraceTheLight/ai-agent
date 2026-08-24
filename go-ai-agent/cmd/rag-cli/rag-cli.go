package main

import (
	"context"
	"flag"
	"fmt"
	"go-ai-agent/internal/config"
	"go-ai-agent/internal/llm"
	"go-ai-agent/internal/rag"
	"log"
	"strings"
	"time"
)

type ragCLIConfig struct {
	DocsPath    string
	ChunkSize   int
	Overlap     int
	EmbedURL    string
	EmbedModel  string
	LimitDocs   int
	LimitChunks int
	Timeout     time.Duration
	ShowPreview bool
	Question    string
	TopK        int
	Model       string
}

// main 启动 RAG CLI。
// 输入: 从命令行 flag 和环境变量读取配置。
// 输出: 成功时打印 load/chunk/embed 摘要; 失败时输出 fatal error 并退出。
// 示例: `go run ./cmd/rag-cli -docs "testdata/documents"` -> 执行 RAG 文档处理流程。
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

// parseFlags 解析 RAG CLI 命令行参数。
// 输入: 读取全局命令行参数, 并使用 `config.EmbeddingBaseURL`、`config.EmbeddingModel` 等作为默认值。
// 输出: 返回 `ragCLIConfig`, 包含文档路径、chunk 参数、embedding 配置、RAG 问题和输出控制项。
// 示例: `parseFlags()` -> 返回由 `-docs`、`-question`、`-topK` 等参数组成的配置。
func parseFlags() ragCLIConfig {
	cfg := ragCLIConfig{}

	flag.StringVar(&cfg.DocsPath, "docs", "", "文档目录或单个文档路径")
	flag.IntVar(&cfg.ChunkSize, "chunkSize", 500, "chunk 大小, 按 rune 数量计算")
	flag.IntVar(&cfg.Overlap, "overlap", 100, "相邻 chunk 的重叠大小, 按 rune 数量计算")
	flag.StringVar(&cfg.EmbedURL, "embedURL", config.EmbeddingBaseURL, "embedding 服务地址")
	flag.StringVar(&cfg.EmbedModel, "embedModel", config.EmbeddingModel, "embedding 模型名称")
	flag.IntVar(&cfg.LimitDocs, "limitDocs", 0, "最多处理多少篇文档, 0 表示不限制")
	flag.IntVar(&cfg.LimitChunks, "limitChunks", 0, "每篇文档最多 embed 多少个 chunk, 0 表示不限制")
	flag.DurationVar(&cfg.Timeout, "timeout", config.RequestTimeout, "整个 RAG CLI 流程的超时时间")
	flag.BoolVar(&cfg.ShowPreview, "showPreview", false, "是否打印 chunk 内容预览")
	flag.StringVar(&cfg.Question, "question", "", "需要基于文档回答的问题; 也可以放在命令末尾")
	flag.IntVar(&cfg.TopK, "topK", 3, "检索时返回的最相关 chunk 数")
	flag.StringVar(&cfg.Model, "model", config.OpenaiModel, "用于生成最终 RAG 回答的模型名称")
	flag.Parse()
	if cfg.Question == "" {
		cfg.Question = strings.TrimSpace(strings.Join(flag.Args(), " "))
	}

	return cfg
}

// run 执行 RAG CLI 的 load、chunk、embed、search 和 answer 流程。
// 输入: `ctx` 是请求上下文, `cfg` 是 RAG CLI 配置。
// 输出: 成功时打印文档索引摘要; 传入问题时继续打印检索结果和模型回答。
// 示例: `run(ctx, cfg)` -> 加载文档、检索相关 chunk 并生成带引用约束的回答。
func run(ctx context.Context, cfg ragCLIConfig) error {
	if cfg.DocsPath == "" {
		return fmt.Errorf("必须通过 -docs 指定文档目录或单个文档路径")
	}
	if cfg.EmbedURL == "" {
		return fmt.Errorf("embedding 服务地址不能为空, 请设置 -embedURL 或 EMBEDDING_BASE_URL")
	}
	if cfg.EmbedModel == "" {
		return fmt.Errorf("embedding 模型不能为空, 请设置 -embedModel 或 EMBEDDING_MODEL")
	}

	fmt.Println("RAG CLI 配置")
	fmt.Println("文档路径:", cfg.DocsPath)
	fmt.Println("embedding 服务:", cfg.EmbedURL)
	fmt.Println("embedding 模型:", cfg.EmbedModel)
	fmt.Printf("chunk size: %d, overlap: %d\n", cfg.ChunkSize, cfg.Overlap)
	fmt.Printf("limit docs: %d, limit chunks: %d\n", cfg.LimitDocs, cfg.LimitChunks)
	fmt.Println("topK:", cfg.TopK)
	fmt.Println()

	docs, err := rag.LoadRAGResources(cfg.DocsPath)
	if err != nil {
		return err
	}
	docs = limitDocuments(docs, cfg.LimitDocs)
	if len(docs) == 0 {
		return fmt.Errorf("没有可处理的文档")
	}
	fmt.Println("加载文档数:", len(docs))

	embedClient := rag.NewEmbeddingClient(cfg.EmbedURL, cfg.EmbedModel)
	vectorStore := rag.NewVectorStore()
	var totalChunks, totalEmbedded int
	var lastEmbeddingDimension int
	for i, doc := range docs {
		chunks, err := rag.ChunkDocument(doc, cfg.ChunkSize, cfg.Overlap)
		if err != nil {
			return fmt.Errorf("切分文档 %q 失败: %w", doc.SourcePath, err)
		}
		texts := chunkTexts(chunks, cfg.LimitChunks)
		if len(texts) == 0 {
			fmt.Printf("[%d] %s\n", i+1, doc.SourcePath)
			fmt.Println("chunk 数: 0")
			fmt.Println()
			continue
		}

		embeddings, err := embedClient.Embed(ctx, texts)
		if err != nil {
			return fmt.Errorf("生成文档 %q 的 embedding 失败: %w", doc.SourcePath, err)
		}
		if len(embeddings) != len(texts) {
			return fmt.Errorf("生成文档 %q 的 embedding 数量不匹配: 输入 %d, 输出 %d", doc.SourcePath, len(texts), len(embeddings))
		}
		indexedChunks := chunks
		if len(indexedChunks) > len(texts) {
			indexedChunks = indexedChunks[:len(texts)]
		}
		for j, embedding := range embeddings {
			if err := vectorStore.Add(rag.Vector(embedding), indexedChunks[j]); err != nil {
				return fmt.Errorf("写入向量库失败: %w", err)
			}
		}
		totalChunks += len(chunks)
		totalEmbedded += len(embeddings)
		lastEmbeddingDimension = embeddingDimension(embeddings)

		fmt.Printf("[%d] %s\n", i+1, doc.SourcePath)
		fmt.Println("chunk 数:", len(chunks))
		fmt.Println("本次 embedding chunk 数:", len(texts))
		fmt.Println("embedding 数:", len(embeddings))
		fmt.Println("embedding dimension:", lastEmbeddingDimension)
		if cfg.ShowPreview {
			fmt.Println("preview:", previewText(texts[0], 120))
		}
		fmt.Println()
	}

	fmt.Println("索引完成")
	fmt.Println("总 chunk 数:", totalChunks)
	fmt.Println("已写入向量库 chunk 数:", totalEmbedded)
	fmt.Println("embedding dimension:", lastEmbeddingDimension)
	fmt.Println()

	if cfg.Question == "" {
		fmt.Println("未提供问题, 已完成文档加载、切分和 embedding。可通过 -question 或命令行尾部参数继续测试 RAG 回答。")
		return nil
	}

	queryEmbeddings, err := embedClient.Embed(ctx, []string{cfg.Question})
	if err != nil {
		return fmt.Errorf("生成问题 embedding 失败: %w", err)
	}
	if len(queryEmbeddings) != 1 {
		return fmt.Errorf("问题 embedding 数量不匹配: 期望 1, 实际 %d", len(queryEmbeddings))
	}

	searchResults, err := vectorStore.Search(rag.Vector(queryEmbeddings[0]), cfg.TopK)
	if err != nil {
		return fmt.Errorf("检索相关 chunk 失败: %w", err)
	}

	fmt.Println("问题:", cfg.Question)
	fmt.Println("检索结果:")
	for i, result := range searchResults {
		chunk := result.Embed.Chunk
		fmt.Printf("[%d] score=%.4f source=%s#chunk-%d\n", i+1, result.Score, chunk.SourceFile, chunk.ChunkIndex)
		if cfg.ShowPreview {
			fmt.Println("preview:", previewText(chunk.Content, 120))
		}
	}
	fmt.Println()

	ragPrompt := rag.BuildPrompt(searchResults)
	client := llm.NewOpenAIClient(config.OpenaiApiKey, config.OpenaiBaseURL, cfg.Model)
	answer, err := client.Generate(ctx, []llm.Message{
		{Role: llm.SystemMessage, Content: ragPrompt},
		{Role: llm.UserMessage, Content: cfg.Question},
	})
	if err != nil {
		return fmt.Errorf("生成 RAG 回答失败: %w", err)
	}

	fmt.Println("回答:")
	fmt.Println(answer)
	return nil
}

// limitDocuments 按上限截取文档列表。
// 输入: `docs` 是文档列表, `limit` 是最大保留数量; `limit <= 0` 表示不限制。
// 输出: 返回被截取后的文档列表。
// 示例: `limitDocuments(docs, 2)` -> 返回最多 2 篇文档。
func limitDocuments(docs []*rag.Document, limit int) []*rag.Document {
	if limit <= 0 || limit >= len(docs) {
		return docs
	}
	return docs[:limit]
}

// chunkTexts 从 chunk 列表中提取文本内容。
// 输入: `chunks` 是 chunk 列表, `limit` 是最大提取数量; `limit <= 0` 表示不限制。
// 输出: 返回用于 embedding 的文本切片。
// 示例: `chunkTexts(chunks, 3)` -> 返回前 3 个 chunk 的 `Content`。
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

// embeddingDimension 获取 embedding 向量维度。
// 输入: `embeddings` 是 embedding 向量列表。
// 输出: 返回第一个 embedding 的维度; 列表为空时返回 0。
// 示例: `embeddingDimension([][]float64{{0.1, 0.2}})` -> 返回 `2`。
func embeddingDimension(embeddings [][]float64) int {
	if len(embeddings) == 0 {
		return 0
	}
	return len(embeddings[0])
}

// previewText 生成文本预览。
// 输入: `text` 是原始文本, `maxRunes` 是最多保留的 rune 数量。
// 输出: 文本未超长时返回原文; 超长时返回截断文本并追加省略号。
// 示例: `previewText("abcdef", 3)` -> 返回 `"abc..."`。
func previewText(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	return string(runes[:maxRunes]) + "..."
}
