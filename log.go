/*
@Time       : 2021/12/31 15:05
@Author     : wuqiusheng
@File       : log.go
@Description: 日志系统
			支持文本和json文本和json格式,支持控制台log,文件log,http log
			性能比较 相对github.com/sirusen/logrus
			性能 text:json:logrus.json = 2:3:15
*/

package easynet

import (
	"easyutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

var (
	DefLog     *Log //默认日志
	stopForLog int32
)

type ILogger interface {
	Write(str string)
}

type ConsoleLogger struct {
}

func (r *ConsoleLogger) Write(str string) {
	easyutil.Println(str)
}

type OnFileLogFull func(path string)
type OnFileLogTimeout func(path string) int
type OnFileRename func(dirName, fileName, extName string) string
type FileLogger struct {
	Path         string
	Timeout      int //0表示不设置, 单位s
	MaxSize      int //0表示不限制，最大大小
	OnFull       OnFileLogFull
	OnTimeout    OnFileLogTimeout
	OnRenameFile OnFileRename

	size     int
	file     *os.File
	filename string
	extname  string
	dirname  string
}

func (r *FileLogger) Write(str string) {
	if r.file == nil {
		return
	}

	newsize := r.size
	newsize += len(str) + 1

	if r.MaxSize > 0 && newsize >= r.MaxSize {
		r.file.Close()
		r.file = nil
		newpath := r.dirname + "/" + r.filename + easyutil.Sprintf("_%v", Date()) + r.extname
		if r.OnRenameFile != nil {
			newpath = r.OnRenameFile(r.dirname+"/", r.filename, r.extname)
		}
		os.Rename(r.Path, newpath)
		file, err := os.OpenFile(r.Path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err == nil {
			r.file = file
		}
		r.size = 0
		if r.OnFull != nil {
			r.OnFull(newpath)
		}
	}

	if r.file == nil {
		return
	}

	r.file.WriteString(str + "\n")
	r.size += len(str) + 1
}

type HttpLogger struct {
	Url     string
	Get     bool
	GetKey  string
	Timeout int
}

func (r *HttpLogger) Write(str string) {
	if r.Url == "" {
		return
	}
	Go(func() {
		if r.Timeout == 0 {
			r.Timeout = 5
		}
		c := http.Client{
			Timeout: time.Duration(r.Timeout) * time.Second,
		}
		var resp *http.Response
		if r.Get {
			if r.GetKey == "" {
				r.GetKey = "log"
			}
			resp, _ = c.Get(r.Url + "/?" + r.GetKey + "=" + str)
		} else {
			resp, _ = c.Post(r.Url, "application/x-www-form-urlencoded", strings.NewReader(str))
		}

		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	})
}

type LogLevel int

const (
	LogLevelDebug  LogLevel = iota //调试信息
	LogLevelInfo                   //资讯讯息
	LogLevelWarn                   //警告状况发生
	LogLevelError                  //一般错误，可能导致功能不正常
	LogLevelFatal                  //严重错误，会导致进程退出
	LogLevelAllOff                 //关闭所有日志
)

var LogLevelNameMap = map[string]LogLevel{
	"debug": LogLevelDebug,
	"info":  LogLevelInfo,
	"warn":  LogLevelWarn,
	"error": LogLevelError,
	"fatal": LogLevelFatal,
	"off":   LogLevelAllOff,
}

type LogField struct {
	FieldList []string
	ValueList []interface{}
}

func NewField(field string, v interface{}) *LogField {
	r := &LogField{
		FieldList: []string{field},
		ValueList: []interface{}{v},
	}
	return r
}

func NewFieldMap(fieldMap map[string]interface{}) *LogField {
	r := &LogField{
		FieldList: make([]string, 0, len(fieldMap)),
		ValueList: make([]interface{}, 0, len(fieldMap)),
	}
	for k, v := range fieldMap {
		r.FieldList = append(r.FieldList, k)
		r.ValueList = append(r.ValueList, v)
	}
	return r
}

func (r *LogField) WithField(field string, v interface{}) *LogField {
	r.FieldList = append(r.FieldList, field)
	r.ValueList = append(r.ValueList, v)
	return r
}

type Log struct {
	logger         [8]ILogger
	cwrite         chan string
	ctimeout       chan *FileLogger
	bufsize        int
	stop           int32
	preLoggerCount int32
	loggerCount    int32
	level          LogLevel
	isJson         bool //默认文本格式,是否json格式
}

func (r *Log) initFileLogger(f *FileLogger) *FileLogger {
	if f.file == nil {
		f.Path, _ = filepath.Abs(f.Path)
		f.Path = easyutil.StrReplace(f.Path, "\\", "/")
		f.dirname = path.Dir(f.Path)
		f.extname = path.Ext(f.Path)
		f.filename = filepath.Base(f.Path[0 : len(f.Path)-len(f.extname)])
		os.MkdirAll(f.dirname, 0666)
		file, err := os.OpenFile(f.Path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
		if err == nil {
			f.file = file
			info, err := f.file.Stat()
			if err != nil {
				return nil
			}

			f.size = int(info.Size())
			if f.Timeout > 0 {
				SetTimeout(f.Timeout*1000, func(...interface{}) int {
					defer func() { recover() }()
					r.ctimeout <- f
					return 0
				})
			}

			return f
		}
	}
	return nil
}

func (r *Log) start() {
	goForLog(func(cstop chan struct{}) {
		var i int32
		for !r.IsStop() {
			select {
			case s, ok := <-r.cwrite:
				if ok {
					for i = 0; i < r.loggerCount; i++ {
						r.logger[i].Write(s)
					}
				}
			case c, ok := <-r.ctimeout:
				if ok {
					c.file.Close()
					c.file = nil
					newpath := c.dirname + "/" + c.filename + easyutil.Sprintf("_%v", Date()) + c.extname
					if c.OnRenameFile != nil {
						newpath = c.OnRenameFile(c.dirname+"/", c.filename, c.extname)
					}
					os.Rename(c.Path, newpath)
					file, err := os.OpenFile(c.Path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
					if err == nil {
						c.file = file
					}
					c.size = 0
					if c.OnTimeout != nil {
						nc := c.OnTimeout(newpath)
						if nc > 0 {
							SetTimeout(nc*1000, func(...interface{}) int {
								defer func() { recover() }()
								r.ctimeout <- c
								return 0
							})
						}
					}
				}
			case <-cstop:
			}
		}

		for s := range r.cwrite {
			for i = 0; i < r.loggerCount; i++ {
				r.logger[i].Write(s)
			}
		}

		// 关闭打开的文件fd
		for i = 0; i < r.loggerCount; i++ {
			if f, ok := r.logger[i].(*FileLogger); ok {
				if f.file != nil {
					f.file.Close()
					f.file = nil
				}
			}
		}
	})
}

func (r *Log) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		close(r.cwrite)
		close(r.ctimeout)
	}
}

func (r *Log) SetLogger(logger ILogger) bool {
	if r.preLoggerCount >= 7 {
		return false
	}
	if f, ok := logger.(*FileLogger); ok {
		if r.initFileLogger(f) == nil {
			return false
		}
	}
	r.logger[atomic.AddInt32(&r.preLoggerCount, 1)] = logger
	atomic.AddInt32(&r.loggerCount, 1)
	return true
}

func (r *Log) Level() LogLevel {
	return r.level
}
func (r *Log) SetLevel(level LogLevel) {
	r.level = level
}

func (r *Log) SetLevelByName(name string) bool {
	level, ok := LogLevelNameMap[name]
	if ok {
		r.SetLevel(level)
	}
	return ok
}

func (r *Log) UseJson(useJson bool) {
	r.isJson = useJson
}

func isLogStop() bool {
	return stopForLog == 1
}

func (r *Log) IsStop() bool {
	if r.stop == 0 {
		if isLogStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *Log) write(levstr string, field *LogField, v ...interface{}) {
	defer func() { recover() }()
	if r.IsStop() {
		return
	}

	prefix := levstr
	_, file, line, ok := runtime.Caller(3)
	if ok {
		i := strings.LastIndex(file, "/") + 1
		prefix = easyutil.Sprintf("[%s][%s][%s:%d]:", levstr, Date(), (string)(([]byte(file))[i:]), line)
	}

	if field != nil {
		for i := 0; i < len(field.FieldList); i++ {
			switch reflect.TypeOf(field.ValueList[i]).Kind() {
			case reflect.Interface,
				reflect.Array,
				reflect.Slice,
				reflect.Map,
				reflect.Ptr:
				data, _ := easyutil.JsonPack(field.ValueList[i])
				prefix += easyutil.Sprintf(field.FieldList[i]+":%v,", string(data))
			default:
				prefix += easyutil.Sprintf(field.FieldList[i]+":%v,", field.ValueList[i])
			}
		}
	}

	if len(v) > 1 {
		r.cwrite <- prefix + easyutil.Sprintf(v[0].(string), v[1:]...)
	} else if len(v) == 1 {
		r.cwrite <- prefix + easyutil.Sprint(v[0])
	} else {
		r.cwrite <- prefix
	}
}

func (r *Log) writeJson(levstr string, field *LogField, logList ...interface{}) {
	defer func() { recover() }()
	if r.IsStop() {
		return
	}

	prefix := levstr
	_, file, line, ok := runtime.Caller(3)
	if ok {
		i := strings.LastIndex(file, "/") + 1
		prefix = easyutil.Sprintf("{\"level\":\"%s\",\"date\":\"%s\",\"file\":\"%s:%d\"", levstr, Date(), (string)(([]byte(file))[i:]), line)
	}

	if field != nil {
		for i := 0; i < len(field.FieldList); i++ {
			if field.ValueList[i] == nil {
				prefix += easyutil.Sprintf(",\"%s\":\"%v\"", field.FieldList[i], nil)
				continue
			}
			switch reflect.TypeOf(field.ValueList[i]).Kind() {
			case reflect.Interface,
				reflect.Array,
				reflect.Slice,
				reflect.Map,
				reflect.Ptr,
				reflect.Struct,
				reflect.String:
				data, _ := easyutil.JsonPack(field.ValueList[i])
				prefix += easyutil.Sprintf(",\"%s\":%v", field.FieldList[i], string(data))
			default:
				prefix += easyutil.Sprintf(",\"%s\":%v", field.FieldList[i], field.ValueList[i])
			}
		}
	}
	if len(logList) > 1 {
		data, _ := easyutil.JsonPack(easyutil.Sprintf(logList[0].(string), logList[1:]...))
		r.cwrite <- prefix + easyutil.Sprintf(",\"log\":%v}", string(data))
	} else if len(logList) == 1 {
		r.cwrite <- prefix + easyutil.Sprintf(",\"log\":\"%v\"}", easyutil.Sprint(logList[0]))
	} else {
		r.cwrite <- prefix + "}"
	}
}

func (r *Log) Debug(v ...interface{}) {
	if r.level <= LogLevelDebug {
		if r.isJson {
			r.writeJson("Debug", nil, v...)
		} else {
			r.write("D", nil, v...)
		}
	}
}

func (r *Log) Info(v ...interface{}) {
	if r.level <= LogLevelInfo {
		if r.isJson {
			r.writeJson("Info", nil, v...)
		} else {
			r.write("I", nil, v...)
		}
	}
}

func (r *Log) Warn(v ...interface{}) {
	if r.level <= LogLevelWarn {
		if r.isJson {
			r.writeJson("Warn", nil, v...)
		} else {
			r.write("W", nil, v...)
		}
	}
}

func (r *Log) Error(v ...interface{}) {
	if r.level <= LogLevelError {
		if r.isJson {
			r.writeJson("Error", nil, v...)
		} else {
			r.write("E", nil, v...)
		}
	}
}

func (r *Log) Fatal(v ...interface{}) {
	if r.level <= LogLevelFatal {
		if r.isJson {
			r.writeJson("Fatal", nil, v...)
		} else {
			r.write("Fatal", nil, v...)
		}
	}
}

func (r *Log) DebugWithField(field *LogField, v ...interface{}) {
	if r.level <= LogLevelDebug {
		if r.isJson {
			r.writeJson("Debug", field, v...)
		} else {
			r.write("D", field, v...)
		}
	}
}

func (r *Log) InfoWithField(field *LogField, v ...interface{}) {
	if r.level <= LogLevelInfo {
		if r.isJson {
			r.writeJson("Info", field, v...)
		} else {
			r.write("I", field, v...)
		}
	}
}

func (r *Log) WarnWithField(field *LogField, v ...interface{}) {
	if r.level <= LogLevelWarn {
		if r.isJson {
			r.writeJson("Warn", field, v...)
		} else {
			r.write("W", field, v...)
		}
	}
}

func (r *Log) ErrorWithField(field *LogField, v ...interface{}) {
	if r.level <= LogLevelError {
		if r.isJson {
			r.writeJson("Error", field, v...)
		} else {
			r.write("E", field, v...)
		}
	}
}

func (r *Log) FatalWithField(field *LogField, v ...interface{}) {
	if r.level <= LogLevelFatal {
		if r.isJson {
			r.writeJson("Fatal", field, v...)
		} else {
			r.write("Fatal", field, v...)
		}
	}
}

func NewLog(isJson bool, bufsize int, logger ...ILogger) *Log {
	log := &Log{
		bufsize:        bufsize,
		cwrite:         make(chan string, bufsize),
		ctimeout:       make(chan *FileLogger, 32),
		level:          LogLevelDebug,
		preLoggerCount: -1,
		isJson:         isJson,
	}
	for _, l := range logger {
		log.SetLogger(l)
	}
	log.start()
	return log
}

func LogInfo(v ...interface{}) {
	DefLog.Info(v...)
}

func LogDebug(v ...interface{}) {
	DefLog.Debug(v...)
}

func LogError(v ...interface{}) {
	DefLog.Error(v...)
}

func LogFatal(v ...interface{}) {
	DefLog.Fatal(v...)
}

func LogWarn(v ...interface{}) {
	DefLog.Warn(v...)
}

func LogInfoWithField(field *LogField, v ...interface{}) {
	DefLog.InfoWithField(field, v...)
}

func LogDebugWithField(field *LogField, v ...interface{}) {
	DefLog.DebugWithField(field, v...)
}

func LogErrorWithField(field *LogField, v ...interface{}) {
	DefLog.ErrorWithField(field, v...)
}

func LogFatalWithField(field *LogField, v ...interface{}) {
	DefLog.FatalWithField(field, v...)
}

func LogWarnWithField(field *LogField, v ...interface{}) {
	DefLog.WarnWithField(field, v...)
}

func LogStack() {
	buf := make([]byte, 1<<12)
	LogError(string(buf[:runtime.Stack(buf, false)]))
}

func LogSimpleStack() string {
	_, file, line, _ := runtime.Caller(2)
	i := strings.LastIndex(file, "/") + 1
	i = strings.LastIndex((string)(([]byte(file))[:i-1]), "/") + 1

	return easyutil.Sprintf("%s:%d", (string)(([]byte(file))[i:]), line)
}
