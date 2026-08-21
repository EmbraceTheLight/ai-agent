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
// 输出: 返回 `ragCLIConfig`, 包含文档路径、chunk 参数、embedding 配置和输出控制项。
// 示例: `parseFlags()` -> 返回由 `-docs`、`-chunkSize`、`-embedModel` 等参数组成的配置。
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
	flag.Parse()

	return cfg
}

// run 执行 RAG CLI 的 load、chunk、embed 流程。
// 输入: `ctx` 是请求上下文, `cfg` 是 RAG CLI 配置。
// 输出: 成功时打印每篇文档的 chunk 数、embedding 数和向量维度; 配置错误、加载失败、切分失败或 embedding 失败时返回错误。
// 示例: `run(ctx, cfg)` -> 加载 `cfg.DocsPath` 下的文档并生成 embedding 摘要。
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

		fmt.Printf("[%d] %s\n", i+1, doc.SourcePath)
		fmt.Println("chunk 数:", len(chunks))
		fmt.Println("本次 embedding chunk 数:", len(texts))
		fmt.Println("embedding 数:", len(embeddings))
		fmt.Println("embedding dimension:", embeddingDimension(embeddings))
		if cfg.ShowPreview {
			fmt.Println("preview:", previewText(texts[0], 120))
		}
		fmt.Println()
	}

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
