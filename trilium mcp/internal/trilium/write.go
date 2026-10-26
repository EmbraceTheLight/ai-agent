package trilium

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

// CreateNoteInput 指定新建富文本笔记所需的数据，NoteID 由业务层预先分配以便核对未知结果。
type CreateNoteInput struct {
	ParentNoteID string `json:"parentNoteId"`
	NoteID       string `json:"noteId"`
	Title        string `json:"title"`
	Content      string `json:"content"`
}

// CreateNote 创建指定 ID 的富文本笔记，不自动重试，也不直接提供操作范围保护。
// 输入：ctx 控制请求；input 提供父级、固定 ID、非空标题及 UTF-8 HTML 正文。
// 输出：服务器返回 201 时为 nil，否则报错；连接失败时写入结果可能未知，业务层须按 ID 查询核对。
func (c *Client) CreateNote(ctx context.Context, input CreateNoteInput) error {
	if err := validateNoteID(input.ParentNoteID); err != nil {
		return err
	}
	if err := validateNoteID(input.NoteID); err != nil {
		return err
	}
	if strings.TrimSpace(input.Title) == "" || !utf8.ValidString(input.Title) || !utf8.ValidString(input.Content) || len(input.Content) > maxResponseBytes {
		return errors.New("create requires a nonempty UTF-8 title and content of at most 8 MiB")
	}
	payload, err := json.Marshal(struct {
		CreateNoteInput
		Type string `json:"type"`
		MIME string `json:"mime"`
	}{CreateNoteInput: input, Type: "text", MIME: "text/html"})
	if err != nil {
		return errors.New("cannot encode create request")
	}
	return c.write(ctx, http.MethodPost, "/create-note", payload, "application/json", http.StatusCreated)
}

// SetNoteContent 替换笔记正文，不自动重试；范围、备份与冲突检查由业务层负责。
// 输入：ctx 控制请求；noteID 为合法 ID；content 为不超过 8 MiB 的 UTF-8 正文。
// 输出：服务器返回 204 时为 nil，否则报错；失败不保证服务器尚未写入。
// HTML 正文也按 ETAPI 要求使用 text/plain 传输，不进行 JSON 包装。
func (c *Client) SetNoteContent(ctx context.Context, noteID, content string) error {
	if err := validateNoteID(noteID); err != nil {
		return err
	}
	if !utf8.ValidString(content) || len(content) > maxResponseBytes {
		return errors.New("content must be UTF-8 text of at most 8 MiB")
	}
	return c.write(ctx, http.MethodPut, "/notes/"+noteID+"/content", []byte(content), "text/plain; charset=utf-8", http.StatusNoContent)
}

// write 执行一次写请求并校验状态，沿用客户端的认证、时限及禁止重定向策略。
// 输入：method、path 和 expected 由内部调用确定；body 为已编码正文；contentType 为接口要求的媒体类型。
// 输出：状态符合预期时返回 nil，否则返回不含 Token 或响应正文的错误；不自动重发请求。
func (c *Client) write(ctx context.Context, method, path string, body []byte, contentType string, expected int) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot construct ETAPI write request")
	}
	req.Header.Set("Authorization", c.token)
	req.Header.Set("Content-Type", contentType)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ETAPI %s %s: outcome unknown; verify with the same operation ID before retrying", method, path)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		return &HTTPError{StatusCode: resp.StatusCode, Path: path, Method: method}
	}
	return nil
}
