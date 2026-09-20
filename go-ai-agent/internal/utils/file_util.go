package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// Exist 判断 path 路径是否存在
// 输入: path 路径
// 输出: 是否存在该 path
// 示例: Exist("/exist_path/file") --> true
func Exist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// IsDir 判断 path 路径是否存在且为目录
// 输入: path 路径
// 输出: 是否存在且为目录
// 示例: IsDir("/exist_path/file") --> false
func IsDir(path string) bool {
	fileStat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileStat.IsDir()
}

// ConvertToAbsPath 将 path 转换为绝对路径
// 输入: path 路径
// 输出: 绝对路径, 如果转换过程中出现问题, 则返回 error
// 示例: a/b/c.txt --> /abs/path/a/b/c.txt
func ConvertToAbsPath(path string) (string, error) {
	if Exist(path) == false {
		return path, fmt.Errorf("路径 %s 不存在, 无法转换为绝对路径", path)
	}
	if filepath.IsAbs(path) == true {
		return path, nil
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, err
	}
	return absPath, nil
}
