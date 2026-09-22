package service

import (
	"context"
	"fmt"
	commonv1 "go-ai-agent/app/rag-api/api/common/v1"
	v1 "go-ai-agent/app/rag-api/api/rag/v1"
	"go-ai-agent/app/rag-api/internal/biz"
	"log/slog"
	"strings"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
)

// RAGService 是 RAG proto DTO 与 biz 用例之间的 transport adapter。
// 输入: 接收 HTTP/gRPC 请求并调用 RAGUsecase。
// 输出: 返回 proto 响应，不直接访问外部存储或模型。
// 示例: `v1.RegisterRagServiceHTTPServer(server, ragService)`。
type RAGService struct {
	v1.UnimplementedRagServiceServer

	ragUsecase *biz.RAGUsecase
	log        *slog.Logger
}

// NewRagService 创建 RAG transport service。
// 输入: `ragUsecase` 是 RAG 业务用例, `log` 是结构化日志对象。
// 输出: 返回可注册到 HTTP/gRPC server 的 RAG service。
// 示例: `NewRagService(ragUsecase, slog.Default())`。
func NewRagService(ragUsecase *biz.RAGUsecase, log *slog.Logger) *RAGService {
	if log == nil {
		log = slog.Default()
	}
	return &RAGService{
		ragUsecase: ragUsecase,
		log:        log,
	}
}

// Ask 接收问题并返回 RAG 模型回答、引用和检索结果。
// 输入: `request` 包含问题和 topK。
// 输出: 返回 proto AskResponse; 参数或下游调用失败时返回错误。
// 示例: `service.Ask(ctx, &v1.AskRequest{Question: "什么是 RAG?", TopK: 3})`。
func (r *RAGService) Ask(ctx context.Context, request *v1.AskRequest) (*v1.AskResponse, error) {
	if request == nil {
		return nil, kratosErrors.BadRequest("INVALID_ARGUMENT", "请求不能为空")
	}
	question := strings.TrimSpace(request.GetQuestion())
	if question == "" {
		return nil, kratosErrors.BadRequest("INVALID_ARGUMENT", "问题不能为空")
	}
	if request.GetTopK() <= 0 {
		return nil, kratosErrors.BadRequest("INVALID_ARGUMENT", "topK 必须大于 0")
	}
	if r.ragUsecase == nil {
		return nil, kratosErrors.InternalServer("RAG_USECASE_UNAVAILABLE", "RAG 用例未初始化")
	}

	result, err := r.ragUsecase.Ask(ctx, question, int(request.GetTopK()))
	if err != nil {
		r.log.ErrorContext(ctx, "RAG 问答失败", "question", question, "error", err)
		return nil, err
	}

	searchResults := make([]*v1.SearchResult, 0, len(result.SearchResults))
	citations := make([]*v1.Citation, 0, len(result.SearchResults))
	for _, searchResult := range result.SearchResults {
		if searchResult == nil || searchResult.Chunk == nil {
			continue
		}
		chunk := searchResult.Chunk
		searchResults = append(searchResults, &v1.SearchResult{
			SourceFile: chunk.SourceFile,
			Title:      chunk.Title,
			ChunkIndex: chunk.ChunkIndex,
			Score:      searchResult.Score,
			Content:    chunk.Content,
		})
		citations = append(citations, &v1.Citation{
			SourceFile: chunk.SourceFile,
			Title:      chunk.Title,
			ChunkIndex: chunk.ChunkIndex,
		})
	}

	return &v1.AskResponse{
		CommonResp:    successResponse("回答成功"),
		Answer:        result.Answer,
		Citations:     citations,
		SearchResults: searchResults,
	}, nil
}

// Health 返回 RAG 服务健康状态。
// 输入: `request` 是 proto 健康检查请求，目前没有业务字段。
// 输出: 返回成功的 CommonResp。
// 示例: `service.Health(ctx, &v1.HealthRequest{})`。
func (r *RAGService) Health(ctx context.Context, request *v1.HealthRequest) (*v1.HealthResponse, error) {
	return &v1.HealthResponse{CommonResp: successResponse("服务正常")}, nil
}

// ImportDocument 接收文档路径并执行 load、chunk、embed、store 流程。
// 输入: `request` 包含待导入的文件或目录路径。
// 输出: 返回导入成功响应; 参数或下游调用失败时返回错误。
// 示例: `service.ImportDocument(ctx, &v1.ImportDocumentRequest{Path: "testdata/documents"})`。
func (r *RAGService) ImportDocument(ctx context.Context, request *v1.ImportDocumentRequest) (*v1.ImportDocumentResponse, error) {
	if request == nil {
		return nil, kratosErrors.BadRequest("INVALID_ARGUMENT", "请求不能为空")
	}
	path := strings.TrimSpace(request.GetPath())
	if path == "" {
		return nil, kratosErrors.BadRequest("INVALID_ARGUMENT", "文档路径不能为空")
	}
	if r.ragUsecase == nil {
		return nil, kratosErrors.InternalServer("RAG_USECASE_UNAVAILABLE", "RAG 用例未初始化")
	}

	result, err := r.ragUsecase.ImportDocuments(ctx, path)
	if err != nil {
		r.log.ErrorContext(ctx, "RAG 文档导入失败", "path", path, "error", err)
		return nil, err
	}
	return &v1.ImportDocumentResponse{
		CommonResp: successResponse(fmt.Sprintf("文档导入成功: documents=%d chunks=%d embedded=%d", result.Documents, result.Chunks, result.Embedded)),
	}, nil
}

// successResponse 构造成功响应中的公共状态字段。
// 输入: `message` 是需要返回给调用方的成功消息。
// 输出: 返回状态码为 200 的 CommonResp。
// 示例: `successResponse("回答成功")`。
func successResponse(message string) *commonv1.CommonResp {
	return &commonv1.CommonResp{
		StatusCode: 200,
		Msg:        message,
	}
}
