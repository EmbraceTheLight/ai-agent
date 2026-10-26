package notes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"html"
	"strings"
	"unicode/utf8"

	"github.com/yuin/goldmark"
)

// ContentSHA256 返回原始正文的 SHA-256 十六进制摘要，用于写入前的内容版本检查。
// 输入：content 不进行格式转换或空白归一化；空正文也有确定的摘要。
// 输出：64 个小写十六进制字符，不包含正文。
func ContentSHA256(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

// renderContent 将输入内容转为 HTML，不隐式读取或归一化其他正文。
// 输入：content 为不超过 4 MiB 的非空 UTF-8 文本；format 为 markdown、text 或 html。
// 输出：返回不超过 8 MiB 的 HTML；格式非法、空白输入或超限时报错。
// Markdown 使用 Goldmark 默认安全策略，不启用原始 HTML；html 格式按字面值写入。
func renderContent(content, format string) (string, error) {
	if !utf8.ValidString(content) || strings.TrimSpace(content) == "" || len(content) > 4<<20 {
		return "", errors.New("input must be nonempty UTF-8 text of at most 4 MiB")
	}
	var result string
	switch format {
	case "markdown":
		var buffer bytes.Buffer
		if err := goldmark.Convert([]byte(content), &buffer); err != nil {
			return "", errors.New("cannot render Markdown")
		}
		result = buffer.String()
	case "text":
		result = "<p>" + strings.ReplaceAll(html.EscapeString(strings.ReplaceAll(content, "\r\n", "\n")), "\n", "<br>\n") + "</p>"
	case "html":
		result = content
	default:
		return "", errors.New("format must be markdown, text or html")
	}
	if len(result) > 8<<20 {
		return "", errors.New("rendered content exceeds 8 MiB")
	}
	return result, nil
}
