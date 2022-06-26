/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : init.go
@Description: easynet包含数据库系统，日志系统，错误码，时间系统，消息队列，协程池与同步
*/
package easynet

import (
	"easyutil"
	"runtime"
	"sync"
)

var (
	msgqueId        uint32 //消息队列id
	msgqueMapSync   sync.Mutex
	msgqueMap       = map[uint32]IMsgQue{}
	MsgqueBroadcast = easyutil.NewBroadcast(255) //消息队列广播
)

var Config = struct {
	AutoEncrypt     bool
	AutoCompressLen uint32
	SSLCrtPath      string
	SSLKeyPath      string
	EnableWss       bool
	ReadDataBuffer  int
	TCPNoDelay      bool
}{ReadDataBuffer: 1 << 12, TCPNoDelay: true}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	DefLog = NewLog(false, 10000, &ConsoleLogger{})
	DefLog.SetLevel(LogLevelDebug)
	timerTick()
	//physicsStatis()
}
