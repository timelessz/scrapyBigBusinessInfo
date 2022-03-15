package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"log"
	"sync"
)

/*
* MysqlCrmConnectiPool
* 数据库连接操作库
* 基于gorm封装开发
 */

type MysqlCrmConnectiPool struct{}

var crmInstance *MysqlCrmConnectiPool
var crmOnce sync.Once

var crmDb *gorm.DB
var crmErrDb error

func GetCrmInstance() *MysqlCrmConnectiPool {
	crmOnce.Do(func() {
		crmInstance = &MysqlCrmConnectiPool{}
	})
	return crmInstance
}

/*
* @fuc 初始化数据库连接(可在mail()适当位置调用)
 */
func (m *MysqlCrmConnectiPool) InitCrmDataPool() (issucc bool) {
	var MysqlHost string = ""
	var MysqlDbname string = ""
	var MysqLUser string = ""
	var MysqlPasswd string = ""
	linkStr := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8&parseTime=True&loc=Local", MysqLUser, MysqlPasswd, MysqlHost, MysqlDbname)
	crmDb, crmErrDb = gorm.Open("mysql", linkStr)
	crmDb.SingularTable(true)
	crmDb.LogMode(true)
	fmt.Println(crmErrDb)
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
func (m *MysqlCrmConnectiPool) GetCrmMysqlDB() (db_con *gorm.DB) {
	return crmDb
}
