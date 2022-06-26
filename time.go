/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : time.go
@Description: 时间系统
*/
package easynet

import (
	"time"
)

var (
	StartMs   int64 //启动时间 ms
	NowMs     int64 //当前时间 ms
	Timestamp int64 //当前时间 s
)

func timerTick() {
	StartMs = time.Now().UnixNano() / 1000000
	NowMs = StartMs
	Timestamp = NowMs / 1000
	var ticker = time.NewTicker(time.Millisecond)
	Go(func() {
		for IsRuning() {
			select {
			case <-ticker.C:
				NowMs = time.Now().UnixNano() / 1000000
				Timestamp = NowMs / 1000
			}
		}
		ticker.Stop()
	})
}

//定时执行函数fn
func SetTimeTick(inteval int, fn func(...interface{}), args ...interface{}) {
	if inteval <= 0 {
		LogError("new TimeTick error inteval:%v ms", inteval)
		return
	}
	LogInfo("new TimeTick inteval:%v ms", inteval)
	Go2(func(cstop chan struct{}) {
		tick := time.NewTicker(time.Millisecond * time.Duration(inteval))
		for IsRuning() {
			select {
			case <-tick.C:
				fn(args...)
			}
		}
		tick.Stop()
	})
}

//超时执行函数fn，可不定间隔执行
func SetTimeout(inteval int, fn func(...interface{}) int, args ...interface{}) {
	if inteval < 0 {
		LogError("new timerout inteval:%v ms", inteval)
		return
	}
	LogInfo("new timerout inteval:%v ms", inteval)

	Go2(func(cstop chan struct{}) {
		var tick *time.Timer
		for inteval > 0 {
			tick = time.NewTimer(time.Millisecond * time.Duration(inteval))
			select {
			case <-cstop:
				tick.Stop()
				inteval = 0
			case <-tick.C:
				tick.Stop()
				inteval = fn(args...)
			}
		}
	})
}

func ParseTimeRFC3339Sec(str string) int64 {
	t, e := time.Parse(time.RFC3339, str)
	if e != nil {
		return 0
	}
	return t.Unix()
}

func ParseTimeSec(str string) int64 {
	t, e := time.Parse("2006-01-02 15:04:05", str)
	if e != nil {
		return 0
	}
	return t.Unix()
}

func ParseTimeRFC3339(str string) (time.Time, error) {
	return time.Parse(time.RFC3339, str)
}

func ParseTime(str string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", str)
}

func Date() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func UnixTime(sec, nsec int64) time.Time {
	return time.Unix(sec, nsec)
}

func UnixMs() int64 {
	return time.Now().UnixNano() / 1000000
}

func UnixNano() int64 {
	return time.Now().UnixNano()
}

func Now() time.Time {
	return time.Now()
}

func NewTimer(ms int) *time.Timer {
	return time.NewTimer(time.Millisecond * time.Duration(ms))
}

func NewTicker(ms int) *time.Ticker {
	return time.NewTicker(time.Millisecond * time.Duration(ms))
}

func After(ms int) <-chan time.Time {
	return time.After(time.Millisecond * time.Duration(ms))
}

func Tick(ms int) <-chan time.Time {
	return time.Tick(time.Millisecond * time.Duration(ms))
}

func Sleep(ms int) {
	time.Sleep(time.Millisecond * time.Duration(ms))
}

/**
* @brief 获得timestamp距离下个小时的时间，单位s
*
* @return uint32_t 距离下个小时的时间，单位s
 */
func GetNextHourIntervalS() int {
	return int(3600 - (Timestamp % 3600))
}

/**
 * @brief 获得timestamp距离下个小时的时间，单位ms
 *
 * @return uint32_t 距离下个小时的时间，单位ms
 */
func GetNextHourIntervalMS() int {
	return GetNextHourIntervalS() * 1000
}

/**
* @brief 时间戳转换为小时，24小时制，0点用24表示
*
* @param timestamp 时间戳
* @param timezone  时区
* @return uint32_t 小时 范围 1-24
 */
func GetHour24(timestamp int64, timezone int) int {
	hour := (int((timestamp%86400)/3600) + timezone)
	if hour > 24 {
		return hour - 24
	}
	return hour
}

/**
 * @brief 时间戳转换为小时，24小时制，0点用0表示
 *
 * @param timestamp 时间戳
 * @param timezone  时区
 * @return uint32_t 小时 范围 0-23
 */
func GetHour23(timestamp int64, timezone int) int {
	hour := GetHour24(timestamp, timezone)
	if hour == 24 {
		return 0 //24点就是0点
	}
	return hour
}

func GetHour(timestamp int64, timezone int) int {
	return GetHour23(timestamp, timezone)
}

/**
* @brief 判断两个时间戳是否是同一天
*
* @param now 需要比较的时间戳
* @param old 需要比较的时间戳
* @param timezone 时区
* @return uint32_t 返回不同的天数
 */
func IsDiffDay(now, old int64, timezone int) int {
	now += int64(timezone * 3600)
	old += int64(timezone * 3600)
	return int((now / 86400) - (old / 86400))
}

/**
* @brief 判断时间戳是否处于两个小时之内
*
* @param now 需要比较的时间戳
* @param hour_start 起始的小时 0 -23
* @param hour_end 结束的小时 0 -23
* @param timezone 时区
* @return bool true表示处于两个小时之间
 */
func IsBetweenHour(now int64, hour_start, hour_end int, timezone int) bool {
	hour := GetHour23(now, timezone)
	return (hour >= hour_start) && (hour < hour_end)
}

/**
* @brief 判断时间戳是否处于一个小时的两边，即一个时间错大于当前的hour，一个小于
*
* @param now 需要比较的时间戳
* @param old 需要比较的时间戳
* @param hour 小时，0-23
* @param timezone 时区
* @return bool true表示时间戳是否处于一个小时的两边
 */
func IsDiffHour(now, old int64, hour, timezone int) bool {
	diff := IsDiffDay(now, old, timezone)
	if diff == 1 {
		if GetHour23(old, timezone) > hour {
			return GetHour23(now, timezone) >= hour
		} else {
			return true
		}
	} else if diff >= 2 {
		return true
	}

	return (GetHour23(now, timezone) >= hour) && (GetHour23(old, timezone) < hour)
}

/**
* @brief 判断时间戳是否处于跨周, 在周一跨天节点的两边
*
* @param now 需要比较的时间戳
* @param old 需要比较的时间戳
* @param hour 小时，0-23
* @param timezone 时区
* @return bool true表示时间戳是否处于跨周, 在周一跨天节点的两边
 */
func IsDiffWeek(now, old int64, hour, timezone int) bool {
	diffHour := IsDiffHour(now, old, hour, timezone)
	now += int64(timezone * 3600)
	old += int64(timezone * 3600)
	// 使用UTC才能在本地时间采用周一作为一周的开始
	_, nw := time.Unix(now, 0).UTC().ISOWeek()
	_, ow := time.Unix(old, 0).UTC().ISOWeek()
	return nw != ow && diffHour
}

/* 今天零点
 * timezone 时区
 * return 零点时间
 */
func ZeroTime(timezone int) int64 {
	timeStr := time.Now().Format("2006-01-02")
	t, _ := time.Parse("2006-01-02", timeStr)
	return t.Unix() - int64(timezone*3600)
}

/* 年月日
 * timestamp 时间
 * timezone 时区
 * return year,month,day
 */
func YearMonthDay(timestamp int64, timezone int) (int32, int32, int32) {
	timestamp += int64(timezone * 3600)
	year, month, day := time.Unix(timestamp, 0).UTC().Date()
	return int32(year), int32(month), int32(day)
}
