package utils

import (
	"crypto/sha256"
	"encoding/hex"
)

// GetSHA256HexString 计算字符串内容的 SHA-256 十六进制摘要。
// 输入: `data` 是待计算摘要的原始字符串。
// 输出: 返回长度为 64 的小写十六进制字符串。
// 示例: `GetSHA256HexString("RAG")`。
func GetSHA256HexString(data string) string {
	res := sha256.Sum256([]byte(data))
	return hex.EncodeToString(res[:])
}
