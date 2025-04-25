package gopher_frag

import "testing"

// 测试单线程下载运行
func TestMonoGetTask_Run(t *testing.T) {
	ConfigEnvironmentProxy()
	// 加入自定义请求头
	GlobalConfig.Headers["Origin"] = "github.com"
	// 创建一个分片下载任务
	url := "https://github.com/skeeto/w64devkit/releases/download/v2.0.0/w64devkit-x64-2.0.0.exe"
	task := NewDefaultMonoGetTask(url, "downloads/w64devkit-x64-2.0.0.exe")
	// 监听分片任务的下载状态
	// 使用默认的函数
	task.SubscribeStatus(DefaultProcessLookup)
	// 运行分片下载
	e := task.Run()
	if e != nil {
		t.Error(e)
		return
	}
	// 计算摘要
	result, e := task.CheckFile(ChecksumSha256, "cea23fc56a5e61457492113a8377c8ab0c42ed82303fcc454ccd1963a46f8ce1")
	if e != nil {
		t.Error(e)
		return
	}
	if result {
		logger.InfoLine("文件未损坏！")
	} else {
		logger.ErrorLine("文件损坏！")
		t.Error("文件下载损坏！")
		t.Fail()
	}
}

// 测试从文件恢复单线程任务运行
func TestMonoGetTask_Recover(t *testing.T) {
	ConfigEnvironmentProxy()
	// 从文件恢复并发任务
	task, e := NewMonoGetTaskFromFile("downloads/w64devkit-x64-2.0.0.exe.process.json")
	if e != nil {
		t.Error(e)
		return
	}
	// 监听分片任务的下载状态
	// 使用默认的函数
	task.SubscribeStatus(DefaultProcessLookup)
	e = task.Run()
	if e != nil {
		t.Error(e)
		return
	}
	// 计算摘要
	result, e := task.CheckFile(ChecksumSha256, "cea23fc56a5e61457492113a8377c8ab0c42ed82303fcc454ccd1963a46f8ce1")
	if e != nil {
		t.Error(e)
		return
	}
	if result {
		logger.InfoLine("文件未损坏！")
	} else {
		logger.ErrorLine("文件损坏！")
		t.Error("文件下载损坏！")
		t.Fail()
	}
}