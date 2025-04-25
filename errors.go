package gopher_frag

import "fmt"

// 自定义可重试的错误类型
type retryError struct {
	// 出现错误的分片编号
	// 若设为0表示单线程下载任务
	order int
	// 下次重试的次数
	retryCount int
	// 错误原因
	reason string
}

// 表示可重试错误对象类型常量
var retryErrorType *retryError

// 实现error接口
func (e *retryError) Error() string {
	message := fmt.Sprintf("原因：%s，将进行第%d次重试...", e.reason, e.retryCount)
	if e.order > 0 {
		return fmt.Sprintf("分片%d出现错误！%s", e.order, message)
	}
	return fmt.Sprintf("单线程下载任务出现错误！%s", message)
}

// 创建一个用于分片任务的重试错误对象
//
//   - task 分片任务对象
//   - message 重试原因
func createShardRetryError(task *shardTask, message string) error {
	return &retryError{
		order:      task.Config.Order,
		retryCount: task.Status.retryCount,
		reason:     message,
	}
}

// 创建一个用于单线程任务的重试错误对象
//
//   - task 单线程任务对象
//   - message 重试原因
func createMonoRetryError(task *MonoGetTask, message string) error {
	return &retryError{
		order:      0,
		retryCount: task.retryCount,
		reason:     message,
	}
}