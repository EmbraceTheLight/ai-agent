// Package trilium 提供 Trilium ETAPI 客户端，封装认证、超时与响应校验。
package trilium

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const maxResponseBytes = 8 << 20 // 单个响应上限为 8 MiB；超限报错，避免返回截断的笔记。

var noteIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_]{4,32}$`)

// Client 使用固定的实例地址和 Token 调用 ETAPI，可供多个 goroutine 并发使用。
// 必须通过 NewClient 构造；客户端不跟随重定向，避免认证信息被发送到其他路径。
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// AppInfo 保存实例及数据库的版本信息，不包含服务器数据目录等本机路径。
type AppInfo struct {
	AppVersion    string `json:"appVersion"`
	DBVersion     int    `json:"dbVersion"`
	SyncVersion   int    `json:"syncVersion"`
	BuildRevision string `json:"buildRevision,omitempty"`
}

// Attribute 表示笔记的标签或关系，Type 用于区分两者。
type Attribute struct {
	AttributeID   string `json:"attributeId"`
	NoteID        string `json:"noteId"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	Value         string `json:"value"`
	IsInheritable bool   `json:"isInheritable"`
}

// Note 保存笔记元数据；正文需通过 GetNoteContent 单独读取。
// 克隆笔记可以出现在多个位置，因此 ParentNoteIDs 可能包含多个父笔记 ID。
type Note struct {
	NoteID          string      `json:"noteId"`
	Title           string      `json:"title"`
	Type            string      `json:"type"`
	MIME            string      `json:"mime"`
	IsProtected     bool        `json:"isProtected"`
	ParentNoteIDs   []string    `json:"parentNoteIds"`
	ChildNoteIDs    []string    `json:"childNoteIds"`
	Attributes      []Attribute `json:"attributes,omitempty"`
	UTCDateModified string      `json:"utcDateModified"`
}

// SearchOptions 指定非空查询、结果数量及可选的搜索子树。
// Limit 必须为 1–100；AncestorNoteID 仅限制本次搜索，不构成全局访问权限控制。
type SearchOptions struct {
	Query          string
	AncestorNoteID string
	Limit          int
}

// HTTPError 保存 ETAPI 返回的非预期状态及接口路径，不包含响应正文或认证信息。
// 调用方可用 errors.As 获取此类型，通过 StatusCode 区分认证失败、笔记不存在等情况。
type HTTPError struct {
	StatusCode int
	Path       string
	Method     string
}

// Error 返回接口路径及 HTTP 状态说明，不输出 Token 或服务器响应正文。
func (e *HTTPError) Error() string {
	method := e.Method
	if method == "" {
		method = http.MethodGet
	}
	return fmt.Sprintf("ETAPI %s %s: HTTP %d %s", method, e.Path, e.StatusCode, http.StatusText(e.StatusCode))
}

// NewClient 校验配置并构造客户端，不立即发起网络请求。
// 输入：baseURL 为 HTTP(S) 实例地址，可含代理路径前缀或 /etapi 后缀；token 为 ETAPI Token；timeout 为单次请求的总时限。
// 输出：配置有效时返回客户端；地址含认证信息、查询或片段，Token 为空、首尾含空白或包含换行，以及时限非正时返回错误。
// timeout 覆盖连接及响应读取；客户端会补齐 /etapi 后缀，并禁止跟随重定向。
func NewClient(baseURL, token string, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, errors.New("invalid Trilium URL")
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.Opaque != "" {
		return nil, errors.New("Trilium URL must be an absolute http or https URL")
	}
	if u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, errors.New("Trilium URL must not contain credentials, a query or a fragment")
	}
	if token == "" || strings.TrimSpace(token) != token || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("ETAPI token is missing or contains whitespace at its boundaries")
	}
	if timeout <= 0 {
		return nil, errors.New("request timeout must be positive")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/etapi") {
		u.Path += "/etapi"
	}
	u.RawPath = ""
	return &Client{
		baseURL: u.String(),
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
			// 同源重定向也可能离开 ETAPI 路径，因此所有重定向均交由调用方处理。
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// GetAppInfo 通过 GET /etapi/app-info 查询运行中实例的版本。
// 输入：ctx 控制请求的取消与截止时间。
// 输出：成功时返回版本信息；请求失败、响应不是有效 JSON 或缺少 appVersion 时返回错误。
func (c *Client) GetAppInfo(ctx context.Context) (AppInfo, error) {
	var info AppInfo
	if err := c.getJSON(ctx, "/app-info", nil, &info); err != nil {
		return AppInfo{}, err
	}
	if info.AppVersion == "" {
		return AppInfo{}, errors.New("ETAPI app-info response has no appVersion")
	}
	return info, nil
}

// GetNote 按 ID 读取笔记元数据，不下载正文或附件。
// 输入：ctx 控制请求生命周期；noteID 必须为 4–32 个 ASCII 字母、数字或下划线。
// 输出：成功时返回元数据；ID 非法、请求失败、响应 ID 不符或缺少笔记类型时返回错误。
func (c *Client) GetNote(ctx context.Context, noteID string) (Note, error) {
	if err := validateNoteID(noteID); err != nil {
		return Note{}, err
	}
	var note Note
	if err := c.getJSON(ctx, "/notes/"+noteID, nil, &note); err != nil {
		return Note{}, err
	}
	if note.NoteID != noteID || note.Type == "" {
		return Note{}, errors.New("ETAPI note response has missing or inconsistent metadata")
	}
	return note, nil
}

// GetNoteContent 读取原始 UTF-8 正文，保留富文本的 HTML，不转换格式或下载附件。
// 输入：ctx 控制请求生命周期；noteID 必须符合 ETAPI 的笔记 ID 格式。
// 输出：成功时返回完整正文，允许为空；ID 非法、请求失败、响应超过 8 MiB 或不是 UTF-8 时返回错误。
// 此方法不检查笔记类型及保护状态，调用方应先根据元数据决定是否读取。
func (c *Client) GetNoteContent(ctx context.Context, noteID string) (string, error) {
	if err := validateNoteID(noteID); err != nil {
		return "", err
	}
	body, err := c.get(ctx, "/notes/"+noteID+"/content", nil)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(body) {
		return "", errors.New("note content is not UTF-8 text")
	}
	return string(body), nil
}

// SearchNotes 使用 Trilium 原生查询语法搜索笔记标题和正文。
// 输入：ctx 控制请求生命周期；options 指定非空查询、1–100 条结果上限及可选的祖先笔记 ID。
// 输出：成功时返回元数据切片，无匹配时为空切片；参数非法、请求失败或响应缺少结果数组时返回错误。
// 解码时忽略未知字段以兼容新增字段；不读取匹配笔记的正文。
func (c *Client) SearchNotes(ctx context.Context, options SearchOptions) ([]Note, error) {
	if strings.TrimSpace(options.Query) == "" {
		return nil, errors.New("search query must not be empty")
	}
	if options.Limit < 1 || options.Limit > 100 {
		return nil, errors.New("search limit must be between 1 and 100")
	}
	query := url.Values{
		"search": {options.Query},
		"limit":  {fmt.Sprint(options.Limit)},
	}
	if options.AncestorNoteID != "" {
		if err := validateNoteID(options.AncestorNoteID); err != nil {
			return nil, fmt.Errorf("ancestor: %w", err)
		}
		query.Set("ancestorNoteId", options.AncestorNoteID)
	}
	var result struct {
		Results []Note `json:"results"`
	}
	if err := c.getJSON(ctx, "/notes", query, &result); err != nil {
		return nil, err
	}
	if result.Results == nil {
		return nil, errors.New("ETAPI search response has no results array")
	}
	return result.Results, nil
}

// validateNoteID 检查 ID 是否由 4–32 个 ASCII 字母、数字或下划线组成。
// 输入：noteID 为待拼入接口路径的笔记 ID。
// 输出：符合格式时返回 nil，否则返回错误，阻止路径穿越或查询参数注入。
func validateNoteID(noteID string) error {
	if !noteIDPattern.MatchString(noteID) {
		return errors.New("note ID must contain 4–32 ASCII letters, digits or underscores")
	}
	return nil
}

// getJSON 发起 GET 请求，并将完整 JSON 响应解码到目标值。
// 输入：ctx 控制请求生命周期；path 为相对 ETAPI 的路径；query 为可选查询参数；target 为接收结果的指针。
// 输出：请求及解码成功时返回 nil，否则返回错误；未知字段会被忽略，但尾随的额外 JSON 会导致错误。
func (c *Client) getJSON(ctx context.Context, path string, query url.Values, target any) error {
	body, err := c.get(ctx, path, query)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		// 解码错误可能包含私有正文片段，因此仅报告接口路径和错误类别。
		return fmt.Errorf("ETAPI GET %s: invalid JSON response", path)
	}
	return nil
}

// get 执行一次带认证的 GET 请求，限制响应大小并关闭响应体。
// 输入：ctx 控制请求生命周期；path 为内部构造的相对 ETAPI 路径；query 经 URL 编码后附加到请求中。
// 输出：状态为 200 且正文完整、不超过 8 MiB 时返回原始字节，否则返回错误。
// 非 200 响应返回 HTTPError；不自动重试、不跟随重定向，错误信息不包含认证信息或响应正文。
func (c *Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("cannot construct ETAPI request")
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Accept", "application/json, text/*")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("ETAPI GET %s: %w", path, ctx.Err())
		}
		var timeoutError interface{ Timeout() bool }
		if errors.As(err, &timeoutError) && timeoutError.Timeout() {
			return nil, fmt.Errorf("ETAPI GET %s: request timed out", path)
		}
		return nil, fmt.Errorf("ETAPI GET %s: connection failed; check URL, TLS and network", path)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{StatusCode: resp.StatusCode, Path: path}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("ETAPI GET %s: cannot read complete response", path)
	}
	if len(body) > maxResponseBytes {
		return nil, fmt.Errorf("ETAPI GET %s: response exceeds 8 MiB limit", path)
	}
	return body, nil
}
