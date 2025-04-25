package gopher_frag

import (
	"github.com/JimHyw/sclog"
	"github.com/fatih/color"
	"sync"
)

// 全局日志输出锁
var loggerLock = &sync.Mutex{}

// 全局日志对象
var logger = sclog.NewMutexLoggerShareLock(loggerLock)

// 全局实时日志对象
var realTimeLogger = sclog.NewMutexLoggerShareLock(loggerLock)

func init() {
	config := sclog.NewLineConfig()
	config.Time.Enabled = false
	config.Level.Enabled = false
	config.Message.Color = color.New(color.FgHiGreen)
	realTimeLogger.ConfigAll(config)
}

// ConfigEnableLogger 配置是否启用控制台日志输出
//
// enable 若为true则打开控制台日志输出，否则关闭日志
func ConfigEnableLogger(enable bool) {
	if enable {
		logger.Level = sclog.INFO
	} else {
		logger.Level = sclog.OFF
	}
}