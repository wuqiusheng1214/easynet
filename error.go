/*
@Time       : 2022/1/2
@Author     : wuqiusheng
@File       : error.go
@Description: 错误码
*/
package easynet

import (
	"sync"
)

var idErrMap = sync.Map{}
var errIdMap = sync.Map{}

type Error struct {
	Id  int32
	Str string
}

func (r *Error) Error() string {
	return r.Str
}

func GetError(id int32) *Error {
	if e, ok := idErrMap.Load(id); ok {
		return e.(*Error)
	}
	return ErrErrIdNotFound
}

func GetErrId(err error) int32 {
	if id, ok := errIdMap.Load(err); ok {
		return id.(int32)
	}
	id, _ := errIdMap.Load(ErrErrIdNotFound)
	return id.(int32)
}

func NewError(str string, id int32) *Error {
	err := &Error{id, str}
	idErrMap.Store(id, err)
	errIdMap.Store(err, id)
	return err
}

var (
	ErrOk            = NewError("正确", 0)
	ErrMsgPackPack   = NewError("msgpack打包错误", 1)
	ErrMsgPackUnPack = NewError("msgpack解析错误", 2)
	ErrPBPack        = NewError("pb打包错误", 3)
	ErrPBUnPack      = NewError("pb解析错误", 4)
	ErrJsonPack      = NewError("json打包错误", 5)
	ErrJsonUnPack    = NewError("json解析错误", 6)
	ErrGobPack       = NewError("gob打包错误", 7)
	ErrGobUnPack     = NewError("gob解析错误", 8)
	ErrHttpRequest   = NewError("http请求错误", 9)

	ErrProtoPack      = NewError("协议解析错误", 10)
	ErrProtoUnPack    = NewError("协议打包错误", 11)
	ErrMsgLenTooLong  = NewError("数据过长", 12)
	ErrMsgLenTooShort = NewError("数据过短", 13)

	ErrDBMysqlErr        = NewError("mysql数据库错误", 14)
	ErrDBMysqlDataType   = NewError("mysql数据类型错误", 15)
	ErrDBRedisErr        = NewError("redis数据库错误", 16)
	ErrDBRedisDataType   = NewError("redis数据类型错误", 17)
	ErrDBMongodbErr      = NewError("mongodb数据库错误", 18)
	ErrDBMongodbDataType = NewError("mongodb数据类型错误", 19)

	ErrServePanic          = NewError("服务器内部错误", 20)
	ErrNeedIntraNet        = NewError("需要内网环境", 21)
	ErrConfigPath          = NewError("配置路径错误", 22)
	ErrFileRead            = NewError("文件读取错误", 23)
	ErrNetTimeout          = NewError("网络超时", 24)
	ErrNetUnreachable      = NewError("网络不可达", 25)
	ErrMsgNoHandle         = NewError("消息未注册", 26)
	ErrMicroServerNotFound = NewError("找不到微服务", 27)

	ErrErrIdNotFound = NewError("错误没有对应的错误码", 50)
)
