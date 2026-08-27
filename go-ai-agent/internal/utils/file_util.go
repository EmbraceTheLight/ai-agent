package utils

import "os"

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
