package main

// 带缓冲区的channel

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"sync"
)

func getDB() *gorm.DB {
	issucc := GetInstance().InitDataPool()
	if !issucc {
		log.Println("init customer database pool failure...")
		os.Exit(1)
	}
	db := GetInstance().GetMysqlDB()
	return db
}

// 获取limit offset 指定数量客户
func getLimitCustomer(limit int, offset int, db *gorm.DB) []Customer {
	var customers []Customer
	if err := db.Where("website != ? AND mxrecord = ?", "", "").Offset(offset).Limit(limit).Find(&customers).Error; err != nil {
		// 数据报错
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 返回空数组
			return customers
		}
		fmt.Println("获取数据异常")
	}
	return customers
}

func getCrmDb() *gorm.DB {
	issucc := GetCrmInstance().InitCrmDataPool()
	if !issucc {
		log.Println("init crm database pool failure...")
		os.Exit(1)
	}
	db := GetCrmInstance().GetCrmMysqlDB()
	return db
}

type MxSuffix struct {
	BId    string `json:"b_id"`
	Suffix string `json:"suffix"`
	Name   string `json:"name"`
}

// map

// 获取limit 客户数据
func GetCrmSuffixData(crmdb *gorm.DB) map[string]MxSuffix {
	suffixMap := make(map[string]MxSuffix)
	rows, err := crmdb.Table("sm_mx_suffix as s").Select("s.mxsuffix as suffix,s.brand_id as b_id, b.name as name").Joins("left join sm_mx_brand as b on b.id=s.brand_id").Rows()
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()
	for rows.Next() {
		var suffix, b_id, name string
		rows.Scan(&suffix, &b_id, &name)
		s := MxSuffix{
			BId:    b_id,
			Suffix: suffix,
			Name:   name,
		}
		suffixMap[suffix] = s
	}
	return suffixMap
}

// 获取爬取标记
func getScrapyFlag() (int, int) {
	sinfo := readFile()
	return sinfo.Offset, sinfo.Limit
}

// 设置爬取标记
func setScrapyFlag(offset int, limit int) bool {
	sInfo := ScrapyInfo{
		Offset: offset,
		Limit:  limit,
	}
	return writeFile(sInfo)
}

func produce(ch chan<- Customer, wg *sync.WaitGroup) {
	db := getDB()
	for true {
		offset, limit := getScrapyFlag()
		customers := getLimitCustomer(limit, offset, db)
		if len(customers) == 0 {
			// 表示未获取到数据
			fmt.Println("生产者，未获取到数据")
			setScrapyFlag(0, 10)
			continue
		}
		for _, customer := range customers {
			fmt.Println("生产者：" + string(customer.ID) + customer.EnName.String + customer.Website.String)
			ch <- customer
		}
		// 设置已经爬取到的数据
		setScrapyFlag(offset+limit, limit)
	}
	wg.Done()
}

func consumer(ch <-chan Customer, wg *sync.WaitGroup, suffixMap map[string]MxSuffix, i int) {
	db := getDB()
	for true {
		v := <-ch
		if v.Website.String == "" {
			// website 为空
			continue
		}
		domain := getUrlTldDomain(v.Website.String)
		mxrecord := execDigCommand(domain)
		if mxrecord == "" {
			// website 获取数据为空
			fmt.Println(v.EnName.String + v.Website.String + "获取MXRECORD为空")
		}
		suffix := analyseMxRecord(mxrecord)
		if suffix == "" {
			continue
		}
		subsuffix := getUrlTldDomain("http://" + suffix)
		// 获取mx后缀 对应的品牌
		mss, err := getMxRecordSuffix(suffixMap, subsuffix)
		if err != nil {
			fmt.Println(err.Error())
		}
		saveCustomerMxInfo(db, mss, domain, v, mxrecord, i)
	}
	wg.Done()
}

func main() {
	crmdb := getCrmDb()
	MxSuffix := GetCrmSuffixData(crmdb)
	// init database pool
	var wg sync.WaitGroup
	consumerCount := 1
	wg.Add(consumerCount)
	var ch = make(chan Customer, consumerCount)
	go produce(ch, &wg)
	for i := 0; i < consumerCount; i++ {
		go consumer(ch, &wg, MxSuffix, i+1)
	}
	wg.Wait()
}
