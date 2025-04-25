package gopher_frag

import "time"

// fragConfig 全局下载配置
type fragConfig struct {
	// 每个分片的最大重试次数
	Retry int
	// 请求头的UserAgent
	UserAgent string
	// 发送下载请求时，自定义的附加请求头
	Headers map[string]string
	// 监听分片任务下载状态时，每次监听的间隔时间
	StatusNotifyDuration time.Duration
}

// GlobalConfig 全局下载配置对象
var GlobalConfig = &fragConfig{
	Retry:                5,
	UserAgent:            "Gopherfrag/1.8.0",
	Headers:              make(map[string]string),
	StatusNotifyDuration: 300 * time.Millisecond,
}