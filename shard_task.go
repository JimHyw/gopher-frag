package gopher_frag

import (
	"github.com/JimHyw/gopher-notify"
)

// 一个分片下载任务的配置性质属性
type shardTaskConfig struct {
	// 下载链接
	Url string `json:"url"`
	// 分片序号，从1开始
	Order int `json:"order"`
	// 下载文件路径
	FilePath string `json:"filePath"`
	// 分片的起始范围（字节，包含）
	RangeStart int64 `json:"rangeStart"`
	// 分片的结束范围（字节，包含）
	RangeEnd int64 `json:"rangeEnd"`
}

// 一个分片下载任务的状态性质属性
type shardTaskStatus struct {
	// 已下载的部分（字节）
	DownloadSize int64 `json:"downloadSize"`
	// 该任务是否完成
	TaskDone bool `json:"taskDone"`
	// 当前分片重试次数
	retryCount int
}

// shardTask 单个分片下载任务对象
type shardTask struct {
	// 分片任务配置
	Config shardTaskConfig `json:"config"`
	// 分片任务执行状态
	Status shardTaskStatus `json:"status"`
	// 用于实时发布下载状态变化的发布者
	statusPublisher *gopher_notify.BasePublisher[string, int64]
}

// newShardTask 分片任务对象构造函数
func newShardTask(url string, order int, filePath string, rangeStart int64, rangeEnd int64, broker *gopher_notify.Broker[string, int64]) *shardTask {
	return &shardTask{
		Config: shardTaskConfig{
			Url:        url,
			Order:      order,
			FilePath:   filePath,
			RangeStart: rangeStart,
			RangeEnd:   rangeEnd,
		},
		Status: shardTaskStatus{
			DownloadSize: 0,
			TaskDone:     false,
			retryCount:   0,
		},
		statusPublisher: gopher_notify.NewBasePublisher[string, int64](broker),
	}
}

// 分片重试逻辑
//
//   - reason 重试原因
//   - e 实际发生的错误
//
// 若未达到最大重试次数，则返回可重试错误对象，否则返回实际错误对象
func (task *shardTask) retry(reason string, e error) error {
	// 未到最大重试次数，返回重试错误
	if task.Status.retryCount < GlobalConfig.Retry {
		task.Status.retryCount++
		return createShardRetryError(task, reason)
	}
	// 否则，中断并返回错误
	return e
}

// 下载对应分片，该方法在并发任务池中作为一个异步任务并发调用
func (task *shardTask) getShard() error {
	// 进行下载
	errorMessage, e := downloadFile(task.Config.Url, task.Config.FilePath, task.Config.RangeStart+task.Status.DownloadSize, task.Config.RangeEnd, &task.Status.DownloadSize, &task.Status.TaskDone,
		func() {
			// 发布分片启动事件
			task.statusPublisher.Publish(gopher_notify.NewEvent(shardStart, int64(0)), false)
		},
		func(addSize int64) {
			// 发布下载大小变化事件
			task.statusPublisher.Publish(gopher_notify.NewEvent(sizeAdd, addSize), false)
		},
		func() {
			// 发布分片任务完成事件
			task.statusPublisher.Publish(gopher_notify.NewEvent(shardDone, int64(0)), false)
		})
	// 视情况重试
	if e != nil {
		return task.retry(errorMessage, e)
	}
	return nil
}