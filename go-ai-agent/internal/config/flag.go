package config

import (
	"flag"
	"time"
)

// FlagConf 命令行参数
type FlagConf struct {
	Stream      bool
	JSON        bool
	Model       string
	Instruction string
	TimeOut     time.Duration
}

func GetFlagConf() (*FlagConf, []string) {
	var flagConf FlagConf
	flag.BoolVar(&flagConf.Stream, "stream", false, "是否流式输出")
	flag.BoolVar(&flagConf.JSON, "json", false, "是否以json格式输出")
	flag.StringVar(&flagConf.Model, "model", "gpt-5.5", "模型名称")
	flag.StringVar(&flagConf.Instruction, "instruction", "", "系统消息参数")
	flag.DurationVar(&flagConf.TimeOut, "timeout", 5*time.Minute, "请求超时时间")

	flag.Parse()
	return &flagConf, flag.Args()
}
