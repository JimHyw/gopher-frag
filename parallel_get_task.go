package gopher_frag

import (
	"errors"
	"fmt"
	tp "github.com/JimHyw/concurrent-task-pool/v2"
	"github.com/JimHyw/gopher-notify"
	"os"
	"time"
)

// ParallelGetTask 多线程下载任务类
type ParallelGetTask struct {
	// 继承基本任务类型
	baseTask
	// 其它配置性质属性
	// 下载并发数
	Concurrent int `json:"concurrent"`
	// 分片请求时间间隔，若设为0则开始下载时所有分片同时开始请求
	ShardStartDelay time.Duration `json:"shardStartDelay"`
	// 其它状态性质属性
	// 当前实际并发任务数
	concurrentTaskCount int
	// 存放全部分片任务的列表
	ShardList []*shardTask `json:"shardList"`
	// 接收每个分片任务的下载事件变化的事件总线
	shardBroker *gopher_notify.Broker[string, int64]
}

// NewParallelGetTask 构造函数，用于创建一个全新的分片下载任务
//
//   - url 下载地址
//   - filePath 下载文件的保存路径
//   - processFile 下载进度文件的保存位置，若传入空字符串""表示不记录为进度文件
//   - shardRequestDelay 分片请求时间间隔，若设为0则开始下载时所有分片同时开始请求
//   - concurrent 多线程下载并发数
func NewParallelGetTask(url, filePath, processFile string, shardRequestDelay time.Duration, concurrent int) *ParallelGetTask {
	// 创建任务对象
	return &ParallelGetTask{
		baseTask: baseTask{
			Url:           url,
			FilePath:      filePath,
			processFile:   processFile,
			isRecover:     false,
			TotalSize:     0,
			DownloadSize:  0,
			taskDone:      false,
			retryCount:    0,
			statusSubject: gopher_notify.NewSubject[*TaskStatus](GlobalConfig.StatusNotifyDuration),
		},
		Concurrent:          concurrent,
		ShardStartDelay:     shardRequestDelay,
		concurrentTaskCount: 0,
		ShardList:           make([]*shardTask, 0),
		shardBroker:         gopher_notify.NewBroker[string, int64](concurrent * 3),
	}
}

// NewDefaultParallelGetTask 创建一个并发任务对象
// 设定进度文件保存至下载文件所在目录下
//
//   - url 下载地址
//   - filePath 下载文件的保存路径
//   - concurrent 多线程下载并发数
func NewDefaultParallelGetTask(url, filePath string, concurrent int) *ParallelGetTask {
	return NewParallelGetTask(url, filePath, fmt.Sprintf("%s.process.json", filePath), 0, concurrent)
}

// NewSimpleParallelGetTask 创建一个最简并发任务对象
// 不保存进度文件
//
//   - url 下载地址
//   - filePath 下载文件的保存路径
//   - concurrent 多线程下载并发数
func NewSimpleParallelGetTask(url, filePath string, concurrent int) *ParallelGetTask {
	return NewParallelGetTask(url, filePath, "", 0, concurrent)
}

// NewParallelGetTaskFromFile 从进度记录文件读取并恢复一个多线程下载任务对象
//
// file 进度文件位置
func NewParallelGetTaskFromFile(file string) (*ParallelGetTask, error) {
	// 加载任务
	task, e := loadTaskFromJson[ParallelGetTask](file)
	if e != nil {
		return nil, e
	}
	// 恢复对应配置
	task.processFile = file
	task.isRecover = true
	// 恢复对应状态
	task.DownloadSize = 0
	for _, shard := range task.ShardList {
		task.DownloadSize += shard.Status.DownloadSize
	}
	// 创建事件总线与主题对象
	task.shardBroker = gopher_notify.NewBroker[string, int64](task.Concurrent * 3)
	task.statusSubject = gopher_notify.NewSubject[*TaskStatus](GlobalConfig.StatusNotifyDuration)
	for _, shard := range task.ShardList {
		shard.statusPublisher = gopher_notify.NewBasePublisher[string, int64](task.shardBroker)
	}
	logger.Info("从文件%s恢复多线程下载任务！\n", file)
	return &task, nil
}

// 获取待下载文件大小
func (task *ParallelGetTask) getLength() error {
	length, supportRange, e := getContentLength(task.Url)
	if e != nil {
		return e
	}
	if !supportRange {
		return fmt.Errorf("该请求不支持部分获取，无法分片下载！")
	}
	// 获取成功则设定大小
	task.TotalSize = length
	return nil
}

// 获取文件大小并分配任务
func (task *ParallelGetTask) allocateTask() {
	// 检查并发数与大小
	if int64(task.Concurrent) > task.TotalSize {
		logger.Warn("并发数：%d大于总大小：%d，将调整并发数为：%d\n", task.Concurrent, task.TotalSize, task.TotalSize)
		task.Concurrent = int(task.TotalSize)
	}
	// 计算分片下载范围
	eachSize := task.TotalSize / int64(task.Concurrent)
	// 创建分片任务对象
	for i := 0; i < task.Concurrent; i++ {
		task.ShardList = append(task.ShardList, newShardTask(
			task.Url,
			i+1,
			task.FilePath,
			int64(i)*eachSize,
			int64(i+1)*eachSize-1,
			task.shardBroker,
		))
	}
	// 处理末尾部分
	if task.TotalSize%int64(task.Concurrent) != 0 {
		task.ShardList[task.Concurrent-1].Config.RangeEnd = task.TotalSize - 1
	}
	logger.Info("已完成分片计算！分片数：%d\n", task.Concurrent)
}

// 开始下载全部分片
func (task *ParallelGetTask) downloadShard() error {
	// 全局错误
	var totalError error
	// 创建并发任务池，下载分片数据
	taskPool := tp.NewTaskPool[*shardTask](task.Concurrent, task.ShardStartDelay, 0, task.ShardList,
		// 每个分片任务下载逻辑
		func(shardTask *shardTask, pool *tp.TaskPool[*shardTask]) {
			// 如果任务已下载完成，则直接退出
			if shardTask.Status.TaskDone {
				logger.Warn("分片任务%d已下载完成，无需继续下载！\n", shardTask.Config.Order)
				return
			}
			// 发送分片请求进行下载
			e := shardTask.getShard()
			if e != nil {
				// 判断是否是可重试错误，若是则执行重试逻辑
				if errors.As(e, &retryErrorType) {
					logger.WarnLine(e.Error())
					pool.Retry(shardTask)
					return
				}
				// 否则，中断整个任务
				totalError = e
				pool.Interrupt()
			}
		},
		// 接收到停机信号处理逻辑
		func(pool *tp.TaskPool[*shardTask]) {
			totalError = errors.New("任务被中断！")
		},
		// 下载时每隔一段时间保存状态
		func(pool *tp.TaskPool[*shardTask]) {
			e := saveTaskToJson(task, task.processFile)
			if e != nil {
				logger.ErrorLine("保存下载任务出错！")
				logger.ErrorLine(e.Error())
			}
			time.Sleep(350 * time.Millisecond)
		})
	// 创建订阅者，接收分片任务的下载变化事件
	task.shardBroker.Subscribe(sizeAdd, &sizeChangeSubscriber{task})
	task.shardBroker.Subscribe(shardStart, &shardStartSubscriber{task: task})
	task.shardBroker.Subscribe(shardDone, &shardDoneSubscriber{task})
	// 启动分片下载任务
	logger.InfoLine("开始执行分片下载...")
	// 启动分片下载
	taskPool.Start()
	// 完成下载，发布结束状态
	publishParallelTaskStatus(task, true)
	if !taskPool.IsInterrupt() {
		logger.Info("文件：%s下载完成！\n", task.FilePath)
	} else {
		logger.Warn("文件：%s未能完成下载！\n", task.FilePath)
	}
	return totalError
}

// Run 开始执行多线程分片下载任务
func (task *ParallelGetTask) Run() error {
	// 如果是新建的任务，则执行任务分配
	if !task.isRecover {
		// 获取文件长度
		e := task.getLength()
		if e != nil {
			return e
		}
		// 分配所有分片任务
		task.allocateTask()
		// 创建空白文件
		e = task.createFile()
		if e != nil {
			return e
		}
	}
	// 开始下载文件
	e := task.downloadShard()
	if e != nil {
		return e
	}
	// 删除进度文件
	if task.processFile != "" {
		e = os.Remove(task.processFile)
		if e != nil {
			logger.Warn("删除进度文件：%s失败！请稍后手动删除！\n", task.processFile)
			logger.ErrorLine(e.Error())
		}
	}
	// 释放部分资源
	task.statusSubject.RemoveAll()
	task.shardBroker.Close()
	return nil
}