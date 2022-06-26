/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : db_model.go
@Description: db数据 格式化，model组合DBModel
*/
package easynet

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
)

type DBModel struct{}

func (r *DBModel) DBData(v proto.Message) []byte {
	return DBData(v)
}

func (r *DBModel) DBStr(v proto.Message) string {
	return DBStr(v)
}

func (r *DBModel) PbData(v proto.Message) []byte {
	return PbData(v)
}

func (r *DBModel) PbStr(v proto.Message) string {
	return PbStr(v)
}

func (r *DBModel) ParseDBData(data []byte, v proto.Message) bool {
	return ParseDBData(data, v)
}

func (r *DBModel) ParseDBStr(str string, v proto.Message) bool {
	return ParseDBStr(str, v)
}

func (r *DBModel) ParsePbData(data []byte, v proto.Message) bool {
	return ParsePbData(data, v)
}

func (r *DBModel) ParsePbStr(str string, v proto.Message) bool {
	return ParsePbStr(str, v)
}

func DBData(v proto.Message) []byte {
	data, _ := json.Marshal(v)
	return data
}

func DBStr(v proto.Message) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func PbData(v proto.Message) []byte {
	data, _ := proto.Marshal(v)
	return data
}

func PbStr(v proto.Message) string {
	data, _ := proto.Marshal(v)
	return string(data)
}

func ParseDBData(data []byte, v proto.Message) bool {
	err := json.Unmarshal(data, v)
	return err == nil
}

func ParseDBStr(str string, v proto.Message) bool {
	err := json.Unmarshal([]byte(str), v)
	return err == nil
}

func ParsePbData(data []byte, v proto.Message) bool {
	err := proto.Unmarshal(data, v)
	return err == nil
}

func ParsePbStr(str string, v proto.Message) bool {
	err := proto.Unmarshal([]byte(str), v)
	return err == nil
}
