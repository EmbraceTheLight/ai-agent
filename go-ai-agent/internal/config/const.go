package config

import "time"

const (
	RequestTimeout = 5 * time.Minute
	RetryTimeout   = 30 * time.Second
)

const (
	Instruction = "你是一个帮助 Go 后端开发者学习 AI Agent 的技术助手。回答要准确、简洁、偏工程实践。"
)
