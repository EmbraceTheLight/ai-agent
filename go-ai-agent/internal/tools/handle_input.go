package tools

import (
	"bufio"
	"fmt"
	"os"
)

// ReadInputString 从标准输入读取一行字符串
func ReadInputString() (string, error) {
	var str string
	scan := bufio.NewScanner(os.Stdin)
	if !scan.Scan() {
		return "", fmt.Errorf("读取输入时发生错误: %w", scan.Err())
	}
	str = scan.Text()
	return str, nil
}
