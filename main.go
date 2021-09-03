package main

// 带缓冲区的channel

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"regexp"
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
func getLimitCustomer(limit int, offset int, db *gorm.DB) []customer {
	var customers []customer
	if err := db.Where("URL != ? AND mxrecord = ?", "", "").Order("id desc").Offset(offset).Limit(limit).Find(&customers).Error; err != nil {
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

func produce(ch chan<- customer, wg *sync.WaitGroup) {
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
			fmt.Println("生产者：" + string(customer.ID) + customer.Name.String + customer.Domain.String)
			ch <- customer
		}
		// 设置已经爬取到的数据
		setScrapyFlag(offset+limit, limit)
	}
	wg.Done()
}

func consumer(ch <-chan customer, wg *sync.WaitGroup, suffixMap map[string]MxSuffix, i int) {
	db := getDB()
	for true {
		v := <-ch
		// 截取小域名 公司域名未截取；
		domain := subDomain(v)
		if v.Domain.String == "" {
			// website 为空
			continue
		}
		if v.Domain.String != domain {
			//保存下  重新获取到的域名
			v.Domain.String = domain
			saveCustomerDomain(db, domain, v, i)
		}
		mxrecord := execDigCommand(domain)
		if mxrecord == "" {
			// website 获取数据为空
			fmt.Println(v.Name.String + v.Domain.String + "获取MXRECORD为空")
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

// 截取出域名
func subDomain(v customer) string {
	if v.URL.String == "" {
		return ""
	}
	domains := execSubDmmain(v.URL.String)
	domain := ""
	if len(domains) >= 1 {
		domain = domains[0]
	}
	return domain
}

// 截取域名
func execSubDmmain(url string) []string {
	if url == "" {
		return nil
	}
	pattern := "[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+\\.?"
	r, _ := regexp.Compile(pattern)
	urls := r.FindAllString(url, -1)
	domains := make([]string, 0)
	tempMap := map[string]byte{} // 存放不重复主键
	for _, value := range urls {
		domain := getDomainTldDomain(value)
		if domain == "" {
			continue
		}
		l := len(tempMap)
		tempMap[domain] = 0    //当e存在于tempMap中时，再次添加是添加不进去的，，因为key不允许重复
		if len(tempMap) != l { // 加入map后，map长度变化，则元素不重复
			domains = append(domains, domain)
		}
	}
	return domains
}

func main() {
	crmdb := getCrmDb()
	MxSuffix := GetCrmSuffixData(crmdb)
	// init database pool
	var wg sync.WaitGroup
	consumerCount := 50
	wg.Add(consumerCount)
	var ch = make(chan customer, consumerCount)
	go produce(ch, &wg)
	for i := 0; i < consumerCount; i++ {
		go consumer(ch, &wg, MxSuffix, i+1)
	}
	wg.Wait()
}
