package gopher_frag

import (
	"errors"
	"fmt"
	"github.com/JimHyw/gopher-notify"
	"os"
	"time"
)

// MonoGetTask 单线程下载任务对象
type MonoGetTask struct {
	// 继承基本任务对象
	baseTask
}

// NewMonoGetTask 构造函数，用于创建单线程下载任务对象
//
//   - url 下载地址
//   - filePath 下载文件的保存路径
//   - processFile 下载进度文件的保存位置，若传入空字符串""表示不记录为进度文件
func NewMonoGetTask(url, filePath, processFile string) *MonoGetTask {
	return &MonoGetTask{
		baseTask{
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
	}
}

// NewDefaultMonoGetTask 创建一个默认的单线程下载任务对象
// 设定进度保存文件为下载文件所在目录下
//
//   - url 下载地址
//   - filePath 下载文件的保存路径
func NewDefaultMonoGetTask(url, filePath string) *MonoGetTask {
	return NewMonoGetTask(url, filePath, fmt.Sprintf("%s.process.json", filePath))
}

// NewSimpleMonoGetTask 创建一个简单的单线程下载任务对象
// 不保存进度文件
//
//   - url 下载地址
//   - filePath 下载文件的保存路径
func NewSimpleMonoGetTask(url, filePath string) *MonoGetTask {
	return NewMonoGetTask(url, filePath, "")
}

// NewMonoGetTaskFromFile 从进度文件恢复单线程下载任务
//
//   - file 进度文件位置
func NewMonoGetTaskFromFile(file string) (*MonoGetTask, error) {
	// 加载任务
	task, e := loadTaskFromJson[MonoGetTask](file)
	if e != nil {
		return nil, e
	}
	// 设定对应字段
	task.processFile = file
	task.isRecover = true
	// 创建观察者主题
	task.statusSubject = gopher_notify.NewSubject[*TaskStatus](GlobalConfig.StatusNotifyDuration)
	logger.Info("从文件%s恢复单线程下载任务！\n", file)
	return &task, nil
}

// 单线程任务重试逻辑
//
//   - reason 重试原因
//   - e 实际发生的错误
//
// 若未达到最大重试次数，则返回可重试错误对象，否则返回实际错误对象
func (task *MonoGetTask) retry(reason string, e error) error {
	// 未到最大重试次数，返回重试错误
	if task.retryCount < GlobalConfig.Retry {
		task.retryCount++
		return createMonoRetryError(task, reason)
	}
	// 否则，中断并返回错误
	return e
}

// 获取下载文件大小
func (task *MonoGetTask) getLength() error {
	length, supportRange, e := getContentLength(task.Url)
	if e != nil {
		return e
	}
	// 不支持断点续传，则重设下载起始位置
	if !supportRange {
		task.DownloadSize = 0
		logger.Warn("下载任务：%s 不支持断点续传！\n", task.Url)
	}
	// 检查恢复的任务总大小是否和获取的一致
	if task.isRecover && task.TotalSize != length {
		return fmt.Errorf("恢复任务文件大小和请求大小不一致！(%d != %d)，请删除进度文件和已下载文件，重新创建任务！", task.TotalSize, length)
	}
	// 设定总大小
	task.TotalSize = length
	return nil
}

// 发送下载请求
func (task *MonoGetTask) fragFile() error {
	// 下载文件
	errorMessage, e := downloadFile(task.Url, task.FilePath, task.DownloadSize, -1, &task.DownloadSize, &task.taskDone,
		nil,
		func(addSize int64) {
			publishMonoTaskStatus(task, false)
		},
		func() {
			publishMonoTaskStatus(task, true)
		})
	// 出现错误视情况返回重试错误
	if e != nil {
		return task.retry(errorMessage, e)
	}
	logger.Info("文件%s下载完成！\n", task.FilePath)
	return nil
}

// Run 启动单线程下载任务
func (task *MonoGetTask) Run() error {
	// 获取文件大小
	e := task.getLength()
	if e != nil {
		return e
	}
	// 如果不是恢复的任务，则创建空白文件
	if !task.isRecover {
		e = task.createFile()
		if e != nil {
			return e
		}
	}
	// 在新的线程中定时保存进度
	if task.processFile != "" {
		go func() {
			for !task.taskDone {
				// 保存下载文件
				saveProcessError := saveTaskToJson[*MonoGetTask](task, task.processFile)
				if saveProcessError != nil {
					logger.ErrorLine("保存单线程任务进度文件出错！")
					logger.ErrorLine(saveProcessError.Error())
				}
				time.Sleep(350 * time.Millisecond)
			}
		}()
	}
	// 下载文件，失败视情况重试
	for {
		e = task.fragFile()
		if e != nil {
			// 如果是可重试错误则重试
			if errors.As(e, &retryErrorType) {
				logger.ErrorLine(e.Error())
				continue
			}
			// 否则返回错误
			return e
		}
		break
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
	return nil
}