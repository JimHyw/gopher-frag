package gopher_frag

import (
	"fmt"
	"math"
	"time"
)

// ComputeSpeed 计算网络速度
//
//   - size 一段时间内下载的数据大小，单位字节
//   - elapsed 下载 size 字节所经过的时间长度
//
// 返回计算得到的网速字符串形式，会自动换算单位
func ComputeSpeed(size float64, elapsed time.Duration) string {
	if elapsed < 0 {
		return ""
	}
	elapsedSecond := float64(elapsed) / float64(time.Second)
	bytePerSecond := size / elapsedSecond
	if 0 <= bytePerSecond && bytePerSecond <= 1024 {
		return fmt.Sprintf("%.2f Byte/s", bytePerSecond)
	}
	if bytePerSecond > 1024 && bytePerSecond <= math.Pow(1024, 2) {
		return fmt.Sprintf("%.2f KB/s", float64(bytePerSecond)/1024)
	}
	if bytePerSecond > 1024*1024 && bytePerSecond <= math.Pow(1024, 3) {
		return fmt.Sprintf("%.2f MB/s", bytePerSecond/math.Pow(1024, 2))
	}
	return fmt.Sprintf("%.2f GB/s", bytePerSecond/math.Pow(1024, 3))
}

// ComputeRemainTime 根据当前速度，计算剩余下载时间
//
//   - status 当前时刻的下载状态对象
//
// 返回剩余时间，单位：毫秒，若下载速度为0，则返回-1
func ComputeRemainTime(status *TaskStatus) float64 {
	if status.Speed <= 0 {
		return -1
	}
	return float64(status.TotalSize-status.DownloadSize) / status.Speed
}

// DefaultProcessLookup 默认的进度观察者回调函数，能够在控制台输出实时进度
//
//   - status 当前时刻的下载状态对象
func DefaultProcessLookup(status *TaskStatus) {
	realTimeLogger.Info("\033[2K当前下载进度：%.2f %% 实际并发数：%d 当前速度：%s 预计剩余时间：%.1f秒\n", float64(status.DownloadSize)/float64(status.TotalSize)*100, status.Concurrency, ComputeSpeed(status.Speed, time.Millisecond), ComputeRemainTime(status)/1000)
	if !status.IsShutdown {
		fmt.Print("\033[A")
	}
}