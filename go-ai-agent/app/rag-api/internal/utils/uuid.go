package utils

import (
	"time"

	"github.com/yitter/idgenerator-go/idgen"
)

// init 配置项目使用的雪花 ID 生成器及其起始时间。
// 输入: 无。
// 输出: 设置生成器的全局配置, 后续可通过 GetID 获取 ID。
// 示例: 包加载完成后调用 `GetID()`。
func init() {
	tmp := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	baseTime := tmp / 1e6
	var options = idgen.NewIdGeneratorOptions(1)
	options.BaseTime = baseTime
	idgen.SetIdGenerator(options)
}

// GetID 生成新的雪花算法 ID。
// 输入: 无。
// 输出: 返回可作为 chunk 物理主键的 int64 ID。
// 示例: `id := GetID()`。
func GetID() int64 {
	return idgen.NextId()
}
