/*
@Time       : 2022/1/15
@Author     : wuqiusheng
@File       : time_wheel.go
@Description: 基于channel时间轮精准并发调度系统,插入删除的效率 O(1) 较系统最小堆O(nlogn)性能提升
*/
package easynet

import (
	"container/list"
	"easyutil"
	"errors"
	"sync/atomic"
)

//定时任务
type TimeTask struct {
	floor    int32
	pos      int32
	remain   int32
	interval int32
	owner    *TimeWheel
	C        chan bool
	element  *list.Element
}

func (r *TimeTask) Close() {
	if r.owner != nil && r.owner.isRunning() {
		r.owner.removeTask(r)
	} else {
		close(r.C)
	}
}

//时间层
type timeFloor struct {
	floor    int32
	pos      int32
	maxScale int32
	taskList []*list.List
}

//时间轮
type TimeWheel struct {
	addC   chan *TimeTask
	delC   chan *TimeTask
	closeC chan struct{}

	close       int32        //关闭标志
	taskList    []*timeFloor //任务列表
	minInterval int32        //最小间隔
	maxFloor    int32        //最大层数
	scale       int32        //刻度
}

func (r *TimeWheel) isRunning() bool {
	return r.close == 0
}

//插入删除的效率 O(1)
func (r *TimeWheel) tick() {
	for i := int32(0); i < r.maxFloor; {
		floor := r.taskList[i]
		floor.pos++
		if floor.pos >= r.scale {
			i++
			floor.pos = 0
			continue
		}
		l := r.taskList[i].taskList[floor.pos]
		for e := l.Front(); e != nil; {
			task := e.Value.(*TimeTask)
			f := e
			e = e.Next()
			l.Remove(f)
			if task.remain == 0 {
				select {
				case task.C <- true: //回调
				default:
					//LogInfo("[TimeWheel]failed callback taskinterval:%vms", task.interval*r.minInterval)
				}
				task.floor, task.pos, task.remain = r.getPos(task.interval)
				list := r.taskList[task.floor-1].taskList[task.pos]
				task.element = list.PushBack(task)
			} else {
				task.floor, task.pos, task.remain = r.getPos(task.remain)
				list := r.taskList[task.floor-1].taskList[task.pos]
				task.element = list.PushBack(task)
			}
		}
		break
	}
}

func (r *TimeWheel) run() {
	Go(func() {
		var ticker = NewTicker(int(r.minInterval))
		for IsRuning() {
			select {
			case <-ticker.C:
				r.tick()
			case task := <-r.addC:
				list := r.taskList[task.floor-1].taskList[task.pos]
				task.element = list.PushBack(task)
			case task := <-r.delC:
				list := r.taskList[task.floor-1].taskList[task.pos]
				list.Remove(task.element)
				close(task.C)
			case <-r.closeC:
				close(r.addC)
				close(r.delC)
				ticker.Stop()
				LogInfo("[timeWheel] end timewheel:%#v", r)
				return
			}
		}
		r.Close()
		close(r.addC)
		close(r.delC)
		ticker.Stop()
		LogInfo("[timeWheel] end timewheel:%#v", r)
	})
}

func (r *TimeWheel) resetPos(tick int32) (floor, pos, remain int32) {
	wheelFloor := r.taskList[r.maxFloor-1]
	tick -= wheelFloor.maxScale
	if tick >= wheelFloor.maxScale {
		LogError("[timeWheel]tick interval is too max tick:%v,maxScale:%v", tick, wheelFloor.maxScale)
		return 0, 0, 0
	}
	tick = easyutil.Ternary(tick == 0, int32(1), tick).(int32)
	for i := int32(0); i < r.maxFloor; i++ {
		wheelFloor := r.taskList[i]
		if tick < wheelFloor.maxScale {
			lastMaxScale := int32(1)
			if i > 0 {
				lastMaxScale = r.taskList[i-1].maxScale
			}
			floor, pos, remain = i+1, tick/lastMaxScale, tick%lastMaxScale
			LogInfo("interval:%v,floor:%v, pos:%v, remain:%v,ms:%v,curFloor:%v", tick, floor, pos, remain, UnixMs(), i+1)
			break
		}
	}
	return floor, pos, remain
}

//重定位获取时间轮刻度
func (r *TimeWheel) getPos(interval int32) (floor, pos, remain int32) {
	tick := interval
	for i := int32(0); i < r.maxFloor; i++ {
		wheelFloor := r.taskList[i]
		curTick := wheelFloor.pos
		if i > 0 {
			curTick = r.taskList[i-1].maxScale * wheelFloor.pos
		}
		tick += curTick
		if tick < wheelFloor.maxScale {
			lastMaxScale := int32(1)
			if i > 0 {
				lastMaxScale = r.taskList[i-1].maxScale
			}
			floor, pos, remain = i+1, tick/lastMaxScale, tick%lastMaxScale
			//LogInfo("interval:%vfloor:%v, pos:%v, remain:%v,ms:%v,curFloor:%v,curTick:%v",interval,floor, pos, remain,UnixMs(),i+1,curTick)
			break
		}
		//全部轮询完成 重置
		if i == r.maxFloor-1 {
			floor, pos, remain = r.resetPos(tick)
		}
	}
	return floor, pos, remain
}

func (r *TimeWheel) removeTask(task *TimeTask) {
	r.delC <- task
}

//时间轮添加定时器 interval: ms
func (r *TimeWheel) AddTask(interval int32) (*TimeTask, error) {
	if !r.isRunning() {
		return nil, errors.New("TimeWheel closed")
	}
	if interval < r.minInterval {
		return nil, errors.New("task interval is too small error")
	}
	interval = interval / r.minInterval
	if interval >= r.taskList[r.maxFloor-1].maxScale {
		return nil, errors.New("task interval is too big error")
	}
	floor, pos, remain := r.getPos(interval)
	if floor == 0 {
		return nil, errors.New("can't get interval wheel's floor pos")
	}
	task := &TimeTask{
		C:        make(chan bool, 2),
		interval: interval, //tick数
		remain:   remain,
		owner:    r,
		floor:    floor,
		pos:      pos,
	}
	r.addC <- task
	return task, nil
}

//关闭时间轮
func (r *TimeWheel) Close() {
	if atomic.CompareAndSwapInt32(&r.close, 0, 1) {
		close(r.closeC)
	}
}

/*
	创建时间轮定时器
	minInterval 最小间隔 50ms以上稳定
	maxFloor 时间轮层数
	scale 每层时间刻度
*/
func NewTimeWheel(minInterval, maxFloor, scale int32) *TimeWheel {
	if minInterval < 1 || maxFloor < 1 || scale < 5 {
		return nil
	}
	timeWheel := &TimeWheel{
		scale:       scale,
		minInterval: minInterval,
		maxFloor:    maxFloor,
		addC:        make(chan *TimeTask, 100),
		delC:        make(chan *TimeTask, 100),
		closeC:      make(chan struct{}),
		taskList:    make([]*timeFloor, maxFloor),
	}
	for i, floor := range timeWheel.taskList {
		floor = &timeFloor{
			maxScale: int32(easyutil.Pow(float64(scale), float64(i+1))),
			floor:    int32(i + 1),
			taskList: make([]*list.List, scale),
		}
		for i := int32(0); i < scale; i++ {
			floor.taskList[i] = list.New()
		}
		timeWheel.taskList[i] = floor
		LogInfo("[timewheel]new timewheel floor:%v,maxScale:%vms", floor.floor, floor.maxScale*minInterval)
	}
	timeWheel.run()
	LogInfo("[timeWheel]new timeWheel:%#v", timeWheel)
	return timeWheel
}
