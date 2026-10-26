package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReadEnvPreservesLiteralValues 验证 BOM、Windows 换行、引号和注释可被解析，Token 中的特殊字符不被展开。
func TestReadEnvPreservesLiteralValues(t *testing.T) {
	input := "\ufeff# 配置\r\n\r\nTRILIUM_URL = http://localhost:37740 # 本机\r\n" +
		"TRILIUM_TOKEN=abc+/==#literal$HOME\r\n" +
		"QUOTED=\"a # b=$HOME\" # comment\r\nSINGLE='a\\b'\r\nEMPTY= # empty\r\n"
	values, err := readEnv(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"TRILIUM_URL":   "http://localhost:37740",
		"TRILIUM_TOKEN": "abc+/==#literal$HOME",
		"QUOTED":        "a # b=$HOME",
		"SINGLE":        `a\b`,
		"EMPTY":         "",
	}
	for key, expected := range want {
		if actual, exists := values[key]; !exists || actual != expected {
			t.Errorf("unexpected parsed value for %s", key)
		}
	}
}

// TestReadEnvRejectsInvalidInputWithoutLeakingValues 验证非法赋值、重复键及引号错误返回行号，不输出配置内容。
func TestReadEnvRejectsInvalidInputWithoutLeakingValues(t *testing.T) {
	for _, input := range []string{
		"# line 1\nSECRET_VALUE_WITHOUT_EQUALS",
		"# line 1\n1BAD=SECRET_VALUE",
		"# line 1\nTOKEN=\"SECRET_VALUE",
		"# line 1\nTOKEN='SECRET_VALUE' trailing",
		"TOKEN=first\nTOKEN=SECRET_VALUE",
	} {
		values, err := readEnv(strings.NewReader(input))
		if err == nil || values != nil {
			t.Fatal("expected an error without partial configuration")
		}
		if !strings.Contains(err.Error(), "line 2") || strings.Contains(err.Error(), "SECRET_VALUE") {
			t.Fatalf("error lacks line number or exposes content: %v", err)
		}
	}
}

// TestReadEnvRejectsOversizedOrNonTextFiles 验证超出大小限制、非法 UTF-8 和 NUL 字节均被拒绝。
func TestReadEnvRejectsOversizedOrNonTextFiles(t *testing.T) {
	for _, input := range []string{strings.Repeat("x", maxEnvFileBytes+1), string([]byte{0xff}), "TOKEN=abc\x00def"} {
		if _, err := readEnv(strings.NewReader(input)); err == nil {
			t.Error("accepted oversized or non-text file")
		}
	}
}

// TestLoadEnvFileRespectsEnvironmentPriority 验证非空环境变量覆盖文件、空环境变量回退到文件，原查询函数不被修改。
func TestLoadEnvFileRespectsEnvironmentPriority(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("TRILIUM_URL=http://localhost:37740\nTRILIUM_TOKEN=file-token==\n"), 0600); err != nil {
		t.Fatal(err)
	}
	getenv := func(key string) string {
		if key == "TRILIUM_URL" {
			return "https://override.example.com"
		}
		return ""
	}
	lookup, err := LoadEnvFile(path, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if lookup("TRILIUM_URL") != "https://override.example.com" || lookup("TRILIUM_TOKEN") != "file-token==" || lookup("UNKNOWN") != "" {
		t.Fatal("configuration lookup did not follow documented precedence")
	}
	if getenv("TRILIUM_TOKEN") != "" {
		t.Fatal("original environment lookup was changed")
	}
}

// TestLoadEnvFilePreservesMissingFileError 验证文件不存在时保留错误类型，便于 CLI 区分默认文件缺失与其他失败。
func TestLoadEnvFilePreservesMissingFileError(t *testing.T) {
	_, err := LoadEnvFile(filepath.Join(t.TempDir(), "missing.env"), func(string) string { return "" })
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected missing-file error, got %v", err)
	}
}
