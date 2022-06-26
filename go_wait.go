/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : go_wait
@Description: 进程同步
  			要求stopCheckMap所有协程全部关闭，
			没有及时关闭的说明确定死锁WaitForSystemExit 按顺序关闭服务
*/
package easynet

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	stop int32 //停止标志

	atexitMap = struct {
		atexitId uint32
		sync.Mutex
		M map[uint32]func()
	}{M: map[uint32]func(){}}

	stopCheckIndex uint64
	stopCheckMap = sync.Map{}
)

type WaitGroup struct {
	count int64
}

func (r *WaitGroup) Add(delta int) {
	atomic.AddInt64(&r.count, int64(delta))
}

func (r *WaitGroup) Done() {
	atomic.AddInt64(&r.count, -1)
}

func (r *WaitGroup) Wait() {
	for atomic.LoadInt64(&r.count) > 0 {
		time.Sleep(time.Millisecond * time.Duration(1))
	}
}

func (r *WaitGroup) TryWait() bool {
	return atomic.LoadInt64(&r.count) == 0
}

//停机检查，要求协程安全释放
func AddStopCheck(cs string) uint64 {
	id := atomic.AddUint64(&stopCheckIndex, 1)
	if id == 0 {
		id = atomic.AddUint64(&stopCheckIndex, 1)
	}
	stopCheckMap.Store(id, cs)
	return id
}

func RemoveStopCheck(id uint64) {
	stopCheckMap.Delete(id)
}

func AtExit(fun func()) {
	id := atomic.AddUint32(&atexitMap.atexitId, 1)
	if id == 0 {
		id = atomic.AddUint32(&atexitMap.atexitId, 1)
	}

	atexitMap.Lock()
	atexitMap.M[id] = fun
	atexitMap.Unlock()
}

func Stop() {
	if !atomic.CompareAndSwapInt32(&stop, 0, 1) {
		return
	}

	close(stopChanForGo)
	for sc := 0; !waitAll.TryWait(); sc++ {
		time.Sleep(time.Millisecond * time.Duration(1))
		if sc >= StopTimeout {
			LogError("Server Stop Timeout")
			stopCheckMap.Range(func(key, v interface{}) bool {
				LogError("Server Stop Timeout deadlock info:%v", v)
				return true
			})
			break
		}
	}

	LogInfo("Server Stop")
	close(stopChanForSys)
}

//系统是否已关闭
func IsStop() bool {
	return stop == 1
}

//系统是否还在执行
func IsRuning() bool {
	return stop == 0
}

func WaitForSystemExit(exit ...func()) {
	statis.StartTime = time.Now()
	signal.Notify(stopChanForSys, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case <-stopChanForSys:
		Stop()
	}
	//自定义关服操作
	for _, v := range exit {
		if v != nil {
			v()
		}
	}
	atexitMap.Lock()
	for _, v := range atexitMap.M {
		v()
	}
	atexitMap.Unlock()

	//关闭mysql数据库
	for _, v := range mysqlManagers {
		v.close()
	}
	waitAllForMysql.Wait()
	//关闭redis数据库
	for _, v := range redisManagers {
		v.close()
	}
	waitAllForRedis.Wait()
	//关闭log
	if !atomic.CompareAndSwapInt32(&stopForLog, 0, 1) {
		return
	}
	close(stopChanForLog)
	waitAllForLog.Wait()
}

//linux后台运行 skip 跳过启动参数
func Daemon(skip ...string) {
	if os.Getppid() == 1 {
		return
	}
	filePath, _ := filepath.Abs(os.Args[0])
	newCmd := []string{os.Args[0]}
	add := 0
	for _, v := range os.Args[1:] {
		if add == 1 {
			add = 0
			continue
		} else {
			add = 0
		}
		for _, s := range skip {
			if strings.Contains(v, s) {
				if strings.Contains(v, "--") {
					add = 2
				} else {
					add = 1
				}
				break
			}
		}
		if add == 0 {
			newCmd = append(newCmd, v)
		}
	}
	LogInfo("go deam args:%v", newCmd)
	cmd := exec.Command(filePath)
	cmd.Args = newCmd
	cmd.Start()
}
