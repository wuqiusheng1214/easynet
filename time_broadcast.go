/*
@Time       : 2022/1/18
@Author     : wuqiusheng
@File       : time_broadcast.go
@Description: 基于channel,broadcast,高效并发偏差定时调度系统, 无轮询，较time wheel的性能提升 O(0)
			  偏差 [0,interval / 2*cell] (cell==50,50分钟定时误差0.5分钟内)
*/
package easynet

import (
	"easyutil"
	"sync/atomic"
)

//广播定时器
type BroadcastTimer struct {
	broadcastList []*easyutil.Broadcast
	interval      int //定时间隔 单位：ms
	cell          int //分片数 单位：ms
	offset        int //偏差
	startTime     int64
	close         int32
	closeC        chan struct{}
}

func (r *BroadcastTimer) run() {
	Go(func() {
		var ticker = NewTicker(r.offset)
		i := 0
		LogInfo("[BroadcastTimer]new broadcast timer interval:%v,cell:%v,offset:%v", r.interval, r.cell, r.offset)
		for IsRuning() {
			select {
			case <-ticker.C:
				r.broadcastList[i].Broadcast(i, nil)
				i = (i + 1) % r.cell
			case <-r.closeC:
				ticker.Stop()
				return
			}
		}
		ticker.Stop()
		r.Close()
	})
}

func (r *BroadcastTimer) isRunning() bool {
	return r.close == 0
}

//关闭广播定时器
func (r *BroadcastTimer) Close() {
	if atomic.CompareAndSwapInt32(&r.close, 0, 1) {
		close(r.closeC)
		LogInfo("[BroadcastTimer]close broadcast timer interval:%v", r.interval)
	}
}

//添加定时任务 偏差范围：[0,interval/2*cell]
func (r *BroadcastTimer) AddTask() *easyutil.Broadcast {
	if !r.isRunning() {
		return nil
	}
	curMs := UnixMs()
	curMs = ((curMs - r.startTime) % int64(r.interval))
	index := (int(curMs) - (r.offset / 2)) / (r.offset) //四舍五入
	index = index % r.offset
	return r.broadcastList[index]
}

/*
	创建广播定时器  interval越大，cell越大，越精准
 	interval 定时间隔，必须cell的倍数 单位:ms
 	cell 分片数
	偏差 [0,interval / 2*cell]
*/
func NewBroadcastTimer(interval, cell int) *BroadcastTimer {
	if interval < 1000 || cell < 10 || interval%cell != 0 {
		return nil
	}
	timer := &BroadcastTimer{
		broadcastList: make([]*easyutil.Broadcast, cell),
		interval:      interval,
		cell:          cell,
		offset:        interval / cell,
		startTime:     UnixMs(),
		closeC:        make(chan struct{}),
	}
	for i := 0; i < cell; i++ {
		timer.broadcastList[i] = easyutil.NewBroadcast(10)
	}
	timer.run()
	return timer
}
