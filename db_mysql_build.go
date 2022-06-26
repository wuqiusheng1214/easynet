/*
@Time       : 2022/1/14
@Author     : wuqiusheng
@File       : db_mysql_build.go
@Description: sql语句构造
*/
package easynet

import "github.com/smartwalle/dbs"

type MysqlBuilder struct {
	*dbs.RawBuilder
}

func NewMysqlBuilder(sql string, args ...interface{}) *MysqlBuilder {
	return &MysqlBuilder{RawBuilder: dbs.NewBuilder(sql, args...)}
}

type MysqlUpdateBuilder struct {
	*dbs.UpdateBuilder
}

func NewMysqlUpdateBuilder() *MysqlUpdateBuilder {
	return &MysqlUpdateBuilder{UpdateBuilder: dbs.NewUpdateBuilder()}
}

type MysqlInsertBuilder struct {
	*dbs.InsertBuilder
}

func NewMysqlInsertBuilder() *MysqlInsertBuilder {
	return &MysqlInsertBuilder{InsertBuilder: dbs.NewInsertBuilder()}
}

type MysqlDeleteBuilder struct {
	*dbs.DeleteBuilder
}

func NewMysqlDeleteBuilder() *MysqlDeleteBuilder {
	return &MysqlDeleteBuilder{DeleteBuilder: dbs.NewDeleteBuilder()}
}

type MysqlSelectBuilder struct {
	*dbs.SelectBuilder
}

func NewMysqlSelectBuilder() *MysqlSelectBuilder {
	return &MysqlSelectBuilder{SelectBuilder: dbs.NewSelectBuilder()}
}
