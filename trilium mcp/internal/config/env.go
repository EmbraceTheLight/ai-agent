// Package config 读取本机配置文件，不执行其中的命令或修改进程环境。
package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const maxEnvFileBytes = 64 << 10 // 配置文件最多为 64 KiB，避免误读大型文件。

var envKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// LoadEnvFile 读取配置并返回环境变量查询函数，不修改 os.Environ。
// 输入：path 为 UTF-8 配置文件路径；getenv 用于查询已有环境变量。
// 输出：非空环境变量优先于文件值；文件无法读取或格式不合法时返回错误。
// 文件支持 KEY=VALUE、空行、注释及单行引号，不做变量替换、转义或命令执行；错误不包含配置值。
func LoadEnvFile(path string, getenv func(string) string) (func(string) string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open env file: %w", err)
	}
	defer file.Close()
	values, err := readEnv(file)
	if err != nil {
		return nil, err
	}
	return func(key string) string {
		if value := getenv(key); value != "" {
			return value
		}
		return values[key]
	}, nil
}

// readEnv 解析有限大小的 UTF-8 配置内容，允许 Windows 换行及文件开头的 BOM。
// 输入：reader 为配置内容来源。
// 输出：成功时返回键值映射；重复键、非法语法、编码或读取错误均报错，不返回部分配置。
func readEnv(reader io.Reader) (map[string]string, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxEnvFileBytes+1))
	if err != nil {
		return nil, errors.New("cannot read complete env file")
	}
	if len(body) > maxEnvFileBytes {
		return nil, errors.New("env file exceeds 64 KiB limit")
	}
	if !utf8.Valid(body) || strings.ContainsRune(string(body), '\x00') {
		return nil, errors.New("env file must be UTF-8 text without NUL bytes")
	}
	values := make(map[string]string)
	for index, line := range strings.Split(strings.TrimPrefix(string(body), "\ufeff"), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !envKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid env assignment at line %d; expected KEY=VALUE", index+1)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate env key at line %d", index+1)
		}
		parsed, err := parseEnvValue(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid env value at line %d; check quotes and comments", index+1)
		}
		values[key] = parsed
	}
	return values, nil
}

// parseEnvValue 读取单行配置值，保留 Token 中的等号、井号及美元符号。
// 输入：value 已去除首尾空白；引号内内容按字面值读取，行尾注释使用井号。
// 输出：成功时返回值；引号未闭合或闭合后含非注释内容时返回错误。
// 未加引号时，仅把行首或空格、制表符之后的井号视为注释；不支持多行值和转义语法。
func parseEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if value[0] == '\'' || value[0] == '"' {
		end := strings.IndexByte(value[1:], value[0])
		if end < 0 {
			return "", errors.New("unclosed quote")
		}
		end++
		tail := strings.TrimSpace(value[end+1:])
		if tail != "" && !strings.HasPrefix(tail, "#") {
			return "", errors.New("unexpected content after quote")
		}
		return value[1:end], nil
	}
	for index := 0; index < len(value); index++ {
		if value[index] == '#' && (index == 0 || value[index-1] == ' ' || value[index-1] == '\t') {
			return strings.TrimSpace(value[:index]), nil
		}
	}
	return value, nil
}
