/*
@Time       : 2022/1/26
@Author     : wuqiusheng
@File       : statis.go
@Description:
*/
package easynet

import (
	"sync/atomic"
	"time"
)

func GetStatis() *Statis {
	statis.GoCount = int(atomic.LoadInt32(&gocount))
	statis.PoolGoCount = atomic.LoadInt32(&poolGoCount)
	statis.MsgqueCount = len(msgqueMap)
	return statis
}

//性能统计单协程上报
type Statis struct {
	GoCount     int       //进程协程协程数
	PoolGoCount int32     //协程池协程数
	MsgqueCount int       //消息队列数量
	StartTime   time.Time //启动时间
	LastPanic   int64     //最近panic时间
	PanicCount  int32     //panic次数
	cpuPercent  float32   //cpu使用率
	memPercent  float32   //内存使用率
	diskPercent float32   // 磁盘使用率
}

//
//func GetCpuPercent() float32 {
//	percent, _:= cpu.Percent(time.Second, false)
//	return float32(percent[0])
//}
//
//func GetMemPercent()float32 {
//	memInfo, _ := mem.VirtualMemory()
//	return float32(memInfo.UsedPercent)
//}
//
//func GetDiskPercent() float32 {
//	parts, _ := disk.Partitions(true)
//	diskInfo, _ := disk.Usage(parts[0].Mountpoint)
//	return float32(diskInfo.UsedPercent)
//}
//
//func physicsStatis(){
//	Go(func() {
//		tick := time.NewTicker(time.Second * time.Duration(60))
//		for IsRuning() {
//			select {
//			case <-tick.C:
//				statis.cpuPercent = GetCpuPercent()
//				statis.memPercent = GetMemPercent()
//				statis.diskPercent = GetDiskPercent()
//				LogInfo("[statis]physics statis cpu:%v,mem:%v,disk:%v",statis.cpuPercent,statis.memPercent,statis.diskPercent)
//			}
//		}
//		tick.Stop()
//	})
//}
