package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-ai-agent/internal/errno"
	"io"
	"net/http"
	urlpkg "net/url"
)

type httpGetTool struct {
	ToolTemplate

	// 允许访问的域名列表
	allowURLList map[string]bool

	// 允许访问的 HTTP 方法列表
	allowMethodList map[string]bool

	// 响应大小限制 单位: Byte
	respLimit int64
}

func NewHttpGetTool(name, description string) ITool {
	return &httpGetTool{
		ToolTemplate:    NewTemplate(name, description),
		allowURLList:    defaultAllowURLList(),
		allowMethodList: defaultAllowMethodList(),
		respLimit:       1024 * 1024, // 1MB
	}
}

// GetToolParameterJSONSchema 获取 HttpGet Tool 的参数 json schema
func (hg *httpGetTool) GetToolParameterJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "请求 url",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "请求方法",
				"enum":        []string{"GET"},
			},
			"header": map[string]any{
				"type":        "object",
				"description": "请求头",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
			"query": map[string]any{
				"type":        "object",
				"description": "请求参数",
				"additionalProperties": map[string]any{
					"type": "string",
				},
			},
		},
		"additionalProperties": false,
		"required":             []string{"url", "method"},
	}
}

func (hg *httpGetTool) GetToolHandler() ToolFunc {
	return hg.HttpGet
}

func defaultAllowURLList() map[string]bool {
	return map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		//"uapis.cn/api/v1/misc/weather": true, // 请求天气信息 api
		"uapis.cn": true, // 请求天气信息 api
	}
}

// 默认只支持 get 方法
func defaultAllowMethodList() map[string]bool {
	return map[string]bool{
		"GET": true,
		//"POST":   true,
		//"PUT":    true,
		//"DELETE": true,
		//"PATCH":  true,
	}
}

type HttpGetReq struct {
	URL    string            `json:"url"`
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Query  map[string]string `json:"query"`
}

func (hg *httpGetTool) HttpGet(ctx context.Context, jsonReq json.RawMessage) (string, error) {
	var req HttpGetReq
	err := json.Unmarshal(jsonReq, &req)
	if err != nil {
		return "", err
	}
	err = hg.validateReq(req)
	if err != nil {
		return "", err
	}

	client := http.Client{}
	httpRequest, err := http.NewRequestWithContext(ctx, req.Method, req.URL, nil)
	if err != nil {
		return "", errno.ErrSendHttpRequestFailed.WithError(err)
	}
	for k, v := range req.Header {
		httpRequest.Header.Set(k, v)
	}
	queryParam := httpRequest.URL.Query()
	for k, v := range req.Query {
		queryParam.Add(k, v)
	}
	httpRequest.URL.RawQuery = queryParam.Encode()

	resp, err := client.Do(httpRequest)
	if err != nil {
		return "", errno.ErrSendHttpRequestFailed.WithError(err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	limitReader := io.LimitReader(resp.Body, hg.respLimit+1) // 多读取一个字节. 如果后续确实读到了 hg.respLimit+1 字节的内容, 说明响应大小超过了限制
	n, err := buf.ReadFrom(limitReader)
	if err != nil {
		return "", errno.ErrSendHttpRequestFailed.WithError(err)
	}
	if n > hg.respLimit {
		return "", errno.ErrSendHttpRequestFailed.WithError(errors.New("response size exceeds limit"))
	}
	return buf.String(), nil
}

func (hg *httpGetTool) validateReq(req HttpGetReq) error {
	err := hg.validateMethod(req.Method)
	if err != nil {
		return err
	}

	err = hg.validateURL(req.URL)
	if err != nil {
		return err
	}
	return nil
}

func (hg *httpGetTool) validateURL(url string) error {
	parsed, err := urlpkg.Parse(url)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errno.ErrURLNotAllow.WithMsgf("URL %s 不允许访问: 缺少 http/https 协议前缀", url)
	}
	if hg.allowURLList[parsed.Hostname()] {
		return nil
	}
	return errno.ErrURLNotAllow.WithMsgf("URL %s 不允许访问", url)
}
func (hg *httpGetTool) validateMethod(method string) error {
	if hg.allowMethodList[method] {
		return nil
	}
	return errno.ErrMethodNotSupport
}
