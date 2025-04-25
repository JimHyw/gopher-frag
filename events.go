package gopher_frag

import (
	"github.com/JimHyw/gopher-notify"
	"time"
)

// 事件主题常量
const (
	// 下载数据量增加
	sizeAdd = "size-add"
	// 分片任务启动
	shardStart = "shard-start"
	// 分片任务完成
	shardDone = "shard-done"
)

// TaskStatus 表示一个时刻的多线程下载任务的任务状态
type TaskStatus struct {
	// 下载文件的总大小（字节）
	TotalSize int64
	// 已下载大小（字节）
	DownloadSize int64
	// 当前实际并发数
	Concurrency int
	// 当前下载速度，单位：字节/毫秒
	Speed float64
	// 当前下载任务是否被终止或者结束
	IsShutdown bool
}

// 发布一个 ParallelGetTask 分片下载任务的当前状态，通知其所有的观察者
//
//   - task 对应的下载任务
//   - shutdown 该下载任务是否已经结束或者被中断
func publishParallelTaskStatus(task *ParallelGetTask, shutdown bool) {
	task.statusSubject.UpdateAndNotify(&TaskStatus{
		TotalSize:    task.TotalSize,
		DownloadSize: task.DownloadSize,
		Concurrency:  task.concurrentTaskCount,
		IsShutdown:   shutdown,
	}, false)
}

// 发布一个 MonoGetTask 单线程下载任务的当前状态，通知其所有观察者
//
//   - task 对应的单线程下载任务
//   - shutdown 该下载任务是否已经结束
func publishMonoTaskStatus(task *MonoGetTask, shutdown bool) {
	task.statusSubject.UpdateAndNotify(&TaskStatus{
		TotalSize:    task.TotalSize,
		DownloadSize: task.DownloadSize,
		Concurrency:  1,
		IsShutdown:   shutdown,
	}, false)
}

// 订阅下载数据量变化事件的订阅者
type sizeChangeSubscriber struct {
	// 对应的分片下载任务对象
	task *ParallelGetTask
}

// OnSubscribe 下载数据量新增时的自定义事件处理
func (subscriber *sizeChangeSubscriber) OnSubscribe(e *gopher_notify.Event[string, int64]) {
	// 改变多线程任务状态
	subscriber.task.DownloadSize += e.GetData()
	// 发布多线程任务状态
	publishParallelTaskStatus(subscriber.task, false)
}

// 订阅分片任务启动的订阅者
type shardStartSubscriber struct {
	// 对应的分片下载任务对象
	task *ParallelGetTask
}

// OnSubscribe 当有一个分片任务启动时的自定义时间处理
func (subscriber *shardStartSubscriber) OnSubscribe(e *gopher_notify.Event[string, int64]) {
	// 改变多线程任务状态
	subscriber.task.concurrentTaskCount++
	// 发布多线程任务状态
	publishParallelTaskStatus(subscriber.task, false)
}

// 订阅分片任务完成的订阅者
type shardDoneSubscriber struct {
	// 对应的分片下载任务对象
	task *ParallelGetTask
}

// OnSubscribe 当有一个分片任务完成时的自定义事件处理
func (subscriber *shardDoneSubscriber) OnSubscribe(e *gopher_notify.Event[string, int64]) {
	// 改变多线程任务状态
	subscriber.task.concurrentTaskCount--
	// 发布多线程任务状态
	publishParallelTaskStatus(subscriber.task, false)
}

// 观察下载任务状态变化的观察者
type taskObserver struct {
	// 上次下载大小
	lastSize int64
	// 上次通知时间
	lastNotifyTime time.Time
	// 用户传入的自定义接收进度变化的回调函数
	subscribeFunction func(status *TaskStatus)
}

// OnUpdate 当一个 ParallelGetTask 的下载状态发生变化时，该方法被调用
func (observer *taskObserver) OnUpdate(data *TaskStatus) {
	// 计算速度
	size := data.DownloadSize - observer.lastSize
	duration := time.Now().Sub(observer.lastNotifyTime).Milliseconds()
	data.Speed = float64(size) / float64(duration)
	// 记录本次状态
	observer.lastSize = data.DownloadSize
	observer.lastNotifyTime = time.Now()
	// 调用用户传入的自定义接收进度函数
	observer.subscribeFunction(data)
}