package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"sync"
)

/*
* MysqlConnectiPool
* 数据库连接操作库
* 基于gorm封装开发
 */
type MysqlConnectiPool struct{}

var instance *MysqlConnectiPool
var once sync.Once

var db *gorm.DB
var err_db error

func GetInstance() *MysqlConnectiPool {
	once.Do(func() {
		instance = &MysqlConnectiPool{}
	})
	return instance
}

/*
* @fuc 初始化数据库连接(可在mail()适当位置调用)
 */
func (m *MysqlConnectiPool) InitDataPool() (issucc bool) {
	var MysqlHost string = "rdsfjnifbfjnifbo.mysql.rds.aliyuncs.com"
	var MysqlDbname string = "madeinchina"
	var MysqLUser string = "madeinchina"
	var MysqlPasswd string = "LiuRui123$%^"
	linkStr := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8&parseTime=True&loc=Local", MysqLUser, MysqlPasswd, MysqlHost, MysqlDbname)
	db, err_db = gorm.Open("mysql", linkStr)
	db.SingularTable(true)
	fmt.Println(err_db)
	if err_db != nil {
		log.Fatal(err_db)
		return false
	}
	//关闭数据库，db会被多个goroutine共享，可以不调用
	//defer db.Close()
	return true
}


/*
* @fuc  对外获取数据库连接对象db
 */
func (m *MysqlConnectiPool) GetMysqlDB() (db_con *gorm.DB) {
	return db
}
