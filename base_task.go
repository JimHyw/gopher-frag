package gopher_frag

import (
	"github.com/JimHyw/gopher-notify"
	"time"
)

// 基本的下载任务对象
type baseTask struct {
	// 配置性质属性
	// 文件的下载链接
	Url string `json:"url"`
	// 下载文件位置
	FilePath string `json:"filePath"`
	// 下载进度记录文件位置
	processFile string
	// 是否从进度文件恢复的任务
	isRecover bool
	// 状态性质属性
	// 下载文件的总大小（字节）
	TotalSize int64 `json:"totalSize"`
	// 已下载部分的大小（字节）
	DownloadSize int64 `json:"downloadSize"`
	// 任务是否下载完成
	taskDone bool
	// 任务重试次数
	retryCount int
	// 用户订阅进度变化的观察者主题
	statusSubject *gopher_notify.Subject[*TaskStatus]
}

// 创建空白文件，需要在获取长度后调用
func (task *baseTask) createFile() error {
	return createBlankFile(task.FilePath, task.TotalSize)
}

// CheckFile 检查文件摘要值，请在调用 Run 方法并下载完成后再调用该函数
//
//   - algorithm 摘要算法名称，支持： gopher_frag.ChecksumMd5 gopher_frag.ChecksumSha1 gopher_frag.ChecksumSha256
//   - excepted 期望的摘要值，16进制字符串，不区分大小写
//
// 当下载的文件摘要值和excepted相同时，返回true
func (task *baseTask) CheckFile(algorithm, excepted string) (bool, error) {
	return computeFileChecksum(task.FilePath, algorithm, excepted)
}

// SubscribeStatus 订阅该下载任务的实时下载状态
//
//   - lookup 观察者回调函数，当下载状态发生变化时，例如下载进度增加、实际并发数变化等，该函数就会被调用，其参数：
//     status 当前的下载状态对象
func (task *baseTask) SubscribeStatus(lookup func(status *TaskStatus)) {
	// 注册观察者
	task.statusSubject.Register(&taskObserver{
		subscribeFunction: lookup,
		lastSize:          task.DownloadSize,
		lastNotifyTime:    time.Now(),
	})
}