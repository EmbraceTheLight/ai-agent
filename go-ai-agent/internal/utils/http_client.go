package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-ai-agent/internal/config"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// HttpClient 是项目内部通用 HTTP 客户端封装, 保存基础 URL 和底层 *http.Client。
type HttpClient struct {
	BaseURL string
	Client  *http.Client
}

// HttpRequestOpt 描述一次 HTTP 请求所需的 method、path、query、header 和 body。
type HttpRequestOpt struct {
	Method     string
	Path       string
	QueryParam map[string]string
	Header     map[string]string
	Body       io.Reader
}

// NewHttpRequestOpt 创建 HTTP 请求参数对象。
// 输入: `method` 是 HTTP 方法, `path` 是请求路径, `queryParam` 是 query 参数, `header` 是请求头, `body` 是请求体 reader。
// 输出: 返回组装后的 `*HttpRequestOpt`。
// 示例: `NewHttpRequestOpt(http.MethodPost, "/api/embed", nil, header, body)` -> 返回 POST 请求参数对象。
func NewHttpRequestOpt(method, path string, queryParam, header map[string]string, body io.Reader) *HttpRequestOpt {
	return &HttpRequestOpt{
		Path:       path,
		Method:     method,
		Header:     header,
		Body:       body,
		QueryParam: queryParam,
	}
}

// NewHttpClient 创建通用 HTTP 客户端。
// 输入: `baseURL` 是请求基础地址, 例如 `"http://localhost:11434"`。
// 输出: 返回使用 `config.RequestTimeout` 作为总超时时间的 `*HttpClient`。
// 示例: `NewHttpClient("http://localhost:11434")` -> 返回可访问本地 Ollama 的 HTTP 客户端。
func NewHttpClient(baseURL string) *HttpClient {
	return &HttpClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: config.RequestTimeout,
		},
	}
}

// buildRequest 根据请求参数构造 *http.Request。
// 输入: `ctx` 是请求上下文且可为 nil, `opt` 包含 method、path、query、header 和 body。
// 输出: 返回构造完成的 `*http.Request`; 请求构造失败时返回错误。
// 示例: `hc.buildRequest(ctx, NewHttpRequestOpt(http.MethodPost, "/api/embed", nil, header, body))` -> 返回 POST 请求。
func (hc *HttpClient) buildRequest(ctx context.Context, opt *HttpRequestOpt) (*http.Request, error) {
	requestURL := hc.buildURL(opt.Path, opt.QueryParam)
	if ctx == nil {
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, opt.Method, requestURL, opt.Body)
	if err != nil {
		return nil, err
	}
	for k, v := range opt.Header {
		req.Header.Set(k, v)
	}
	return req, nil
}

// buildURL 构建包含 query 参数的完整请求 URL。支持拼接带有 query 参数的 URL
// 输入: `path` 是请求路径, `params` 是 query 参数且可为 nil。
// 输出: 返回由 `BaseURL`、`path` 和 query 参数拼接出的完整 URL。
// 示例: `hc.buildURL("/api/embed", map[string]string{"debug": "true"})` -> 返回 `"http://host/api/embed?debug=true"`。
func (hc *HttpClient) buildURL(path string, params map[string]string) string {
	requestURL := normalizeURL(hc.BaseURL) + "/" + normalizeURL(path)
	if len(params) == 0 {
		return requestURL
	}

	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}
	return requestURL + "?" + query.Encode()
}

// HttpPostJSON 发送 JSON POST 请求并解析 JSON 响应。
// 输入: `ctx` 是请求上下文且可为 nil, `path` 是请求路径, `header` 是额外请求头, `body` 是可 JSON 序列化的请求体, `resp` 是响应体接收对象。
// 输出: 成功时将响应 JSON 解码到 `resp`; 请求构造、HTTP 调用、非 2xx 响应或 JSON 编解码失败时返回错误。
// 示例: `hc.HttpPostJSON(ctx, "/api/embed", nil, reqBody, &respBody)` -> 发送 JSON POST 并写入 `respBody`。
func (hc *HttpClient) HttpPostJSON(ctx context.Context, path string, header map[string]string, body any, resp any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	reader := bytes.NewReader(b)
	header = mergeMap(header, map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	})
	req, err := hc.buildRequest(ctx, NewHttpRequestOpt(http.MethodPost, path, nil, header, reader))
	if err != nil {
		return err
	}
	return hc.do(req, resp)
}

// do 执行 HTTP 请求并处理响应。
// 输入: `req` 是已构造完成的 HTTP 请求, `resp` 是响应体接收对象且可为 nil。
// 输出: 成功时关闭响应体并按需解析 JSON; 请求失败、非 2xx 响应或 JSON 解码失败时返回错误。
// 示例: `hc.do(req, &respBody)` -> 执行请求并将 JSON 响应写入 `respBody`。
func (hc *HttpClient) do(req *http.Request, resp any) error {
	response, err := hc.Client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		errMsg, err := io.ReadAll(response.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("http 请求响应返回非 2xx 状态码. 状态码: %d. 响应: %s", response.StatusCode, string(errMsg))
	}

	if resp == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	return parseResponse(response, resp)
}

// parseResponse 解析 JSON 响应体。
// 输入: `resp` 是 HTTP 响应对象, `v` 是 JSON 解码目标。
// 输出: 成功时将响应体解码到 `v`; JSON 解码失败时返回错误。
// 示例: `parseResponse(response, &respBody)` -> 将 response body 解码到 `respBody`。
func parseResponse(resp *http.Response, v any) error {
	return json.NewDecoder(resp.Body).Decode(v)
}

// mergeMap 将 addition 合并到 dest, 且不覆盖 dest 已有 key。
// 输入: `dest` 是目标 map 且可为 nil, `addition` 是待补充的 map。
// 输出: 返回合并后的 map; 当 `dest` 为 nil 时会创建新 map。
// 示例:	`mergeMap(header, map[string]string{"Accept": "application/json"})` -> 返回补充默认 Accept 的 header。
func mergeMap(dest, addition map[string]string) map[string]string {
	if dest == nil {
		dest = make(map[string]string)
	}
	for k, v := range addition {
		if _, ok := dest[k]; ok {
			continue
		}
		dest[k] = v
	}
	return dest
}

// normalizeURL 去掉 URL 片段首尾的 "/"。
// 输入: `rawUrl` 是原始 URL 或路径片段。
// 输出: 返回去掉首尾 "/" 后的字符串。
// 示例:	`normalizeURL("/api/embed/")` -> 返回 `"api/embed"`。
func normalizeURL(rawUrl string) string {
	return strings.Trim(rawUrl, "/")
}
