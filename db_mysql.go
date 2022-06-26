/*
@Time       : 2022/1/3
@Author     : wuqiusheng
@File       : db_mysql.go
@Description: mysql数据库
*/
package easynet

import (
	"database/sql"
	"database/sql/driver"
	"easyutil"
	"github.com/go-sql-driver/mysql" // 导入 mysql 驱动包
	"sync"
	"time"
)

const (
	ReadTimeOut          = "30s"           //ReadTimeOut  数据库读取超时时间
	WriteTimeOut         = "2m30s"         //WriteTimeOut 数据库写入超时时间
	MaxAllowedPacket     = 4 * 1024 * 1024 // 最大允许包体大小 16 MiB
	MaxLifeTime      int = 2
	MaxTryCount          = 5
)

// Mysqlfunc 数据库函数
type Mysqlfunc struct {
	Func func(db *sql.DB, param map[string]interface{}, table string) (interface{}, error) // 操作数据库的执行的函数
	Parm map[string]interface{}                                                            // 函数执行的参数
	Tb   string                                                                            // 操作数据库的表名
}

//mysql配置
type MysqlConfig struct {
	AddrList []string //mysql分片地址 dev:dev123@tcp(10.24.14.14:3306)/test_db0?charset=utf8mb4&multiStatements=true&interpolateParams=true
	MaxOpen  int      //最大连接数
	MaxIdle  int      //最大空闲连接
	Cap      int      //mysql分片大小
}

//mysql分片
type MysqlCell struct {
	addr    string //连接地址
	cellId  int    //cellid
	startId int    //分片起始id
	endId   int    //分片结束id

	readDB  *sql.DB //读db
	writeDB *sql.DB //写db
}

//获取分片表名
func (r *MysqlCell) getCellTableById(table string) string {
	if r.cellId != 0 {
		table = easyutil.Sprintf("%v%v", table, r.cellId)
	}
	return table
}

//异步执行SQL读语句
func (r *MysqlCell) ExecSQLReadASync(f *Mysqlfunc) {
	Go(func() {
		table := r.getCellTableById(f.Tb)
		for i := 0; i < MaxTryCount; {
			_, err := f.Func(r.readDB, f.Parm, table)
			if err == mysql.ErrInvalidConn || err == driver.ErrBadConn {
				i++
				LogWarn("[mysql] exec try again, tryCount:%v,err:%v", i, err)
				continue
			}
			if err != nil {
				LogError("[mysql]exec sql failed,table:%v,err:%v", table, err)
			}
			return
		}
	})
}

//同步执行SQL读语句
func (r *MysqlCell) ExecSQLReadSync(f *Mysqlfunc) (interface{}, error) {
	table := r.getCellTableById(f.Tb)
	for i := 0; i <= MaxTryCount; {
		v, err := f.Func(r.readDB, f.Parm, table)
		if err == mysql.ErrInvalidConn || err == driver.ErrBadConn {
			i++
			LogWarn("[mysql] exec try again,tryCount:%v,err:%v", i, err)
			continue
		}
		if err != nil {
			LogError("[mysql]exec sql failed,table:%v,err:%v", table, err)
			return nil, ErrDBMysqlErr
		}
		return v, nil
	}
	return nil, nil
}

//异步执行SQL写语句
func (r *MysqlCell) ExecSQLWriteASync(f *Mysqlfunc) {
	Go(func() {
		table := r.getCellTableById(f.Tb)
		for i := 0; i < MaxTryCount; {
			_, err := f.Func(r.writeDB, f.Parm, table)
			if err == mysql.ErrInvalidConn || err == driver.ErrBadConn {
				i++
				LogWarn("[mysql] exec try again, tryCount:%v,err:%v", i, err)
				continue
			}
			if err != nil {
				LogError("[mysql]exec sql failed,table:%v,err:%v", table, err)
			}
			return
		}
	})
}

//同步执行SQL写语句
func (r *MysqlCell) ExecSQLWriteSync(f *Mysqlfunc) (interface{}, error) {
	table := r.getCellTableById(f.Tb)
	for i := 0; i <= MaxTryCount; {
		v, err := f.Func(r.writeDB, f.Parm, table)
		if err == mysql.ErrInvalidConn || err == driver.ErrBadConn {
			i++
			LogWarn("[mysql] exec try again,tryCount:%v,err:%v", i, err)
			continue
		}
		if err != nil {
			LogError("[mysql]exec sql failed,table:%v,err:%v", table, err)
			return nil, ErrDBMysqlErr
		}
		return v, nil
	}
	return nil, nil
}

//mysql
type Mysql struct {
	cellCount int
	cellMap   map[int]*MysqlCell
	name      string //名字
	conf      *MysqlConfig
	manager   *MysqlManager
	sync.RWMutex
}

func (r *Mysql) GetCellCount() int {
	return r.cellCount
}

func (r *Mysql) GetCellIdByHashId(hashId int) int {
	cellid := hashId % r.cellCount
	return cellid
}

//获取mysqlcell通过固定hashid
func (r *Mysql) GetCellByCellId(cellid int) *MysqlCell {
	if cellid < 0 || cellid >= r.cellCount {
		return nil
	}
	r.RLock()
	cell := r.cellMap[cellid]
	r.RUnlock()
	if cell == nil {
		LogError("[mysql]no mysql cell needCellId:%v,mysql:%v,cellCount:%v,config:%#v", cellid, r.name, len(r.cellMap), r.conf)
	}
	return cell
}

//获取mysqlcell
func (r *Mysql) GetCellById(id int) *MysqlCell {
	cellid := (id - 1) / r.conf.Cap
	r.RLock()
	cell := r.cellMap[cellid]
	r.RUnlock()
	if cell == nil {
		LogError("[mysql]no mysql cell for id:%v,needCellId:%v,mysql:%v,cellCount:%v,config:%#v", id, cellid, r.name, len(r.cellMap), r.conf)
	}
	return cell
}

//获取默认mysqlcell
func (r *Mysql) GetDefaultCell() *MysqlCell {
	r.RLock()
	cell := r.cellMap[0]
	r.RUnlock()
	if cell == nil {
		LogError("[mysql]no default mysql cell,needCellId:0,mysql:%v,cellCount:%v,config:%#v", r.name, len(r.cellMap), r.conf)
	}
	return cell
}

//实时扩容
func (r *Mysql) Add(id int, addr string) (*MysqlCell, error) {
	r.RLock()
	if _, found := r.cellMap[id]; found {
		r.RUnlock()
		LogWarn("[mysql]id:%v is exist", id)
		return nil, nil
	}
	r.RUnlock()

	readDB, err := sql.Open("mysql", easyutil.Sprintf("%s&readTimeout=%s&maxAllowedPacket=%d&interpolateParams=true", addr, ReadTimeOut, MaxAllowedPacket))
	if err != nil {
		LogError("[mysql]init mysql err:%v", err)
		return nil, err
	}
	readDB.SetMaxOpenConns(r.conf.MaxOpen)
	readDB.SetMaxIdleConns(r.conf.MaxIdle)
	// 推荐小于5 分钟
	readDB.SetConnMaxLifetime(time.Duration(MaxLifeTime) * time.Minute)
	if err := readDB.Ping(); err != nil {
		LogError("[mysql]ping readDB addr:%v,error:%v", addr, err)
		readDB.Close()
		return nil, err
	}

	writeDB, err := sql.Open("mysql", easyutil.Sprintf("%s&writeTimeout=%s&maxAllowedPacket=%d&interpolateParams=true", addr, WriteTimeOut, MaxAllowedPacket))
	if err != nil {
		LogError("[mysql]init mysql err:%v", err)
		readDB.Close()
		return nil, err
	}
	writeDB.SetMaxOpenConns(r.conf.MaxOpen)
	writeDB.SetMaxIdleConns(r.conf.MaxIdle)
	// 推荐小于5 分钟
	writeDB.SetConnMaxLifetime(time.Duration(MaxLifeTime) * time.Minute)

	if err := writeDB.Ping(); err != nil {
		LogError("[mysql]ping writeDB addr:%v,error:%v", addr, err)
		readDB.Close()
		writeDB.Close()
		return nil, err
	}
	cell := &MysqlCell{
		cellId:  id,
		addr:    addr,
		startId: r.conf.Cap*id + 1,
		endId:   r.conf.Cap*id + r.conf.Cap,
		readDB:  readDB,
		writeDB: writeDB,
	}

	r.Lock()
	r.cellMap[id] = cell
	r.cellCount = len(r.cellMap)
	r.Unlock()
	return cell, nil
}

func (r *Mysql) close() {
	for _, cell := range r.cellMap {
		cell.readDB.Close()
		cell.writeDB.Close()
	}
}

var mysqlManagers []*MysqlManager

//mysql数据库管理
type MysqlManager struct {
	dbs map[string]*Mysql //map[name]*Mysql
}

func (r *MysqlManager) close() {
	for _, db := range r.dbs {
		db.close()
	}
}

func (r *MysqlManager) GetMysql(name string) *Mysql {
	db := r.dbs[name]
	return db
}

func (r *MysqlManager) AddMysql(name string, config *MysqlConfig) (*Mysql, error) {
	db := &Mysql{
		name:    name,
		conf:    config,
		cellMap: make(map[int]*MysqlCell),
		manager: r,
	}
	for index, addr := range config.AddrList {
		_, err := db.Add(index, addr)
		if err != nil {
			db.close()
			return nil, err
		}
	}
	r.dbs[name] = db
	LogInfo("[mysql]add mysql name:%v,conf:%#v,cell count:%v", db.name, db.conf, len(db.cellMap))
	return db, nil
}

func NewMysqlManager(name string, config *MysqlConfig) (*MysqlManager, error) {
	manager := &MysqlManager{
		dbs: map[string]*Mysql{},
	}
	mysqlManagers = append(mysqlManagers, manager)
	_, err := manager.AddMysql(name, config)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

//分片id连续
func CheckMysqlCell(mysqlManager *MysqlManager) bool {
	for name, db := range mysqlManager.dbs {
		for i := 0; i < db.cellCount; i++ {
			if _, found := db.cellMap[i]; !found {
				LogError("[mysql]cell not found id:%v,name:%v", i, name)
				return false
			}
		}
	}
	return true
}
