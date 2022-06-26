/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : go_pool
@Description: go协程池
			统一开启协程入口；协程同步；
			统计当前go协程池中协程的数量以及当前进程主动开启协程数量，panic数量
*/
package easynet

import (
	"os"
	"sync"
	"sync/atomic"
)

const (
	StopTimeout       = 3000
	PoolSize    int32 = 80000
)

var (
	statis = &Statis{}

	goid        uint32
	gocount     int32 //goroutine数量
	poolGoCount int32
	poolChan    = make(chan func())

	stopChanForSys = make(chan os.Signal, 1)
	stopChanForGo  = make(chan struct{})
	stopChanForLog = make(chan struct{})

	waitAll         = &WaitGroup{} //等待所有goroutine
	waitAllForMysql sync.WaitGroup
	waitAllForRedis sync.WaitGroup
	waitAllForLog   sync.WaitGroup
)

func Go(fn func()) {
	pc := PoolSize + 1
	select {
	case poolChan <- fn:
		return
	default:
		pc = atomic.AddInt32(&poolGoCount, 1)
		if pc > PoolSize {
			atomic.AddInt32(&poolGoCount, -1)
		}
	}

	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}
	go func() {
		Try(fn, nil)
		for pc <= PoolSize {
			select {
			case <-stopChanForGo:
				pc = PoolSize + 1
			case nfn := <-poolChan:
				Try(nfn, nil)
			}
		}

		waitAll.Done()
		c = atomic.AddInt32(&gocount, -1)

		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func Go2(fn func(cstop chan struct{})) {
	Go(func() {
		fn(stopChanForGo)
	})
}

func GoArgs(fn func(...interface{}), args ...interface{}) {
	Go(func() {
		fn(args...)
	})
}

func goForMysql(fn func()) {
	waitAllForMysql.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("mysql goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}
	go func() {
		Try(fn, nil)
		waitAllForMysql.Done()
		c = atomic.AddInt32(&gocount, -1)

		if DefLog.Level() <= LogLevelDebug {
			LogDebug("mysql goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func goForRedis(fn func()) {
	waitAllForRedis.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("redis goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}
	go func() {
		Try(fn, nil)
		waitAllForRedis.Done()
		c = atomic.AddInt32(&gocount, -1)

		if DefLog.Level() <= LogLevelDebug {
			LogDebug("redis goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func goForLog(fn func(cstop chan struct{})) bool {
	if IsStop() {
		return false
	}
	waitAllForLog.Add(1)

	go func() {
		fn(stopChanForLog)
		waitAllForLog.Done()
	}()
	return true
}

func Try(fun func(), handler func(interface{})) {
	defer func() {
		if err := recover(); err != nil {
			if handler == nil {
				LogStack()
				LogError("error catch:%v", err)
			} else {
				handler(err)
			}
			atomic.AddInt32(&statis.PanicCount, 1)
			statis.LastPanic = Timestamp
		}
	}()
	fun()
}

func Try2(fun func(), handler func(interface{})) {
	defer func() {
		if err := recover(); err != nil {
			LogStack()
			LogError("error catch:%v", err)
			if handler != nil {
				handler(err)
			}
			atomic.AddInt32(&statis.PanicCount, 1)
			statis.LastPanic = Timestamp
		}
	}()
	fun()
}
