package config

import (
	"flag"
	"time"
)

// FlagConf 命令行参数
type FlagConf struct {
	OutputType  string        // 输出类型, 支持普通输出 "standard", 流式输出 "stream", JSON 结构化输出 "json" 和工具调用 "function_call"
	Model       string        // 模型名称
	Instruction string        // 系统消息
	TimeOut     time.Duration // 请求超时时间
}

func GetFlagConf() (*FlagConf, []string) {
	var flagConf FlagConf
	flag.StringVar(&flagConf.OutputType, "outputType", "standard", "是否流式输出")
	flag.StringVar(&flagConf.Model, "model", "gpt-5.5", "模型名称")
	flag.StringVar(&flagConf.Instruction, "instruction", "", "系统消息参数")
	flag.DurationVar(&flagConf.TimeOut, "timeout", 5*time.Minute, "请求超时时间")

	flag.Parse()
	return &flagConf, flag.Args()
}
