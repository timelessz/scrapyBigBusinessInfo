package main

// 带缓冲区的channel

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"regexp"
	"strings"
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
func getScrapyFlag(file string) (int, int) {
	sinfo := readFile(file)
	return sinfo.Offset, sinfo.Limit
}

// 设置爬取标记
func setScrapyFlag(file string, offset int, limit int) bool {
	sInfo := ScrapyInfo{
		Offset: offset,
		Limit:  limit,
	}
	return writeFile(file, sInfo)
}

func produce(ch chan<- customer, wg *sync.WaitGroup) {
	db := getDB()
	fileName := "lock.txt"
	for true {
		offset, limit := getScrapyFlag(fileName)
		customers := getLimitCustomer(limit, offset, db)
		if len(customers) == 0 {
			// 表示未获取到数据
			fmt.Println("mx 数据爬取生产者，未获取到数据")
			setScrapyFlag(fileName, 0, 10)
			continue
		}
		for _, customer := range customers {
			fmt.Println("生产者：" + string(customer.ID) + customer.Name.String + customer.Domain.String)
			ch <- customer
		}
		// 设置已经爬取到的数据
		setScrapyFlag(fileName, offset+limit, limit)
	}
	wg.Done()
}

func consumer(ch <-chan customer, wg *sync.WaitGroup, suffixMap map[string]MxSuffix, i int) {
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
		/////////////////////////////////////////
		// 获取mx记录
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
		/////////////////////////////////////////
	}
	wg.Done()
}

// 获取 使用 selenium 的数据
func scrapyProduce(ch chan<- customer, wg *sync.WaitGroup) {
	db := getDB()
	fileName := "scrapylock.txt"
	for true {
		offset, limit := getScrapyFlag(fileName)
		customers := getLimitCustomer(limit, offset, db)
		if len(customers) == 0 {
			// 表示未获取到数据
			fmt.Println("域名邮箱主页爬取生产者，未获取到数据")
			setScrapyFlag(fileName, 0, 10)
			continue
		}
		for _, customer := range customers {
			fmt.Println("生产者：" + string(customer.ID) + customer.Name.String + customer.Domain.String)
			ch <- customer
		}
		// 设置已经爬取到的数据
		setScrapyFlag(fileName, offset+limit, limit)
	}
	wg.Done()
}

func scrapyConsumer(ch <-chan customer, wg *sync.WaitGroup, contactMap map[string]map[int]string, mailSelfBuildMap map[string]map[int]string, i int) {
	//db := getDB()
	service, wd := getWebDriver()
	defer service.Stop()
	defer wd.Quit()
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
		///////////////////////////////////////////////
		// 爬取mail 网站信息  自建邮箱数据获取
		mailTitle, mailSource := scrapy(wd, "http://mail."+domain)
		selfBuildBrandId, selfbuildBrandName := matchSelfBuild(&mailSource, mailSelfBuildMap)
		fmt.Println(mailTitle, selfBuildBrandId, selfbuildBrandName)
		// 爬取www 网站标题  加获取 咨询工具 记录
		domainTitle, domainSource := scrapy(wd, "http://"+domain)
		contactBrandId, contactBrandName := matchSelfBuild(&domainSource, contactMap)
		fmt.Println(domainTitle, contactBrandId, contactBrandName)
		saveCustomerInfo(db, v, mailTitle, selfBuildBrandId, selfbuildBrandName, domainTitle, contactBrandId, contactBrandName)
		///////////////////////////////////////////
	}
	wg.Done()
}

// 匹配邮箱自建客户数据
func matchSelfBuild(pageSource *string, maps map[string]map[int]string) (int, string) {
	for domains, brandInfo := range maps {
		if find := strings.Contains(*pageSource, domains); find {
			for key, value := range brandInfo {
				return key, value
			}
		}
	}
	return 0, ""
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
	// 聊天方式
	conactMap := map[string]map[int]string{
		"qiyukf.com":    {1: "七鱼智能客服"},
		"53kf.com":      {2: "53kf"},
		"udesk.cn":      {3: "U-desk"},
		"easemob.com":   {4: "环信"},
		"meiqia.com":    {5: "美洽"},
		"sobot.com":     {6: "智齿"},
		"xiaoneng.cn":   {7: "小能"},
		"youkesdk.com":  {8: "有客云"},
		"live800.com":   {9: "Live800"},
		"b.qq.com":      {10: "营销QQ"},
		"bizapp.qq.com": {10: "营销QQ2"},
		"workec.com":    {11: "EC企信"},
		"looyu.com":     {12: "乐语"},
		"tq.cn":         {13: "TQ洽谈通"},
		"zoosnet.net":   {14: "网站商务通"},
		"talk99.cn":     {15: "Talk99"},
		"kf5.com":       {16: "逸创云客服"},
	}

	// 域名自建相关数据
	mailSelfBuildMap := map[string]map[int]string{
		"coremail":        {1: "盈世"},
		"fangmail":        {2: "方向标"},
		"winmail":         {3: "winmail"},
		"anymacro":        {4: "安宁"},
		"turbomail":       {5: "TurboMail"},
		"u-mail":          {6: "U-Mail"},
		"exchange":        {7: "Exchange"},
		"microsoftonline": {8: "微软Office365"},
		"NiceWebMail":     {9: "NiceWebMail"},
		"/owa/auth.owa":   {10: "微软outlook"},
	}
	crmdb := getCrmDb()
	MxSuffix := GetCrmSuffixData(crmdb)
	// init database pool
	var wg sync.WaitGroup
	consumerCount := 50
	wg.Add(consumerCount)
	var ch = make(chan customer, consumerCount)
	go produce(ch, &wg)
	for i := 0; i < consumerCount; i++ {
		go consumer(ch, &wg, MxSuffix, i)
	}
	// 网站 www 爬取
	scrapyConsumerCount := 10
	wg.Add(scrapyConsumerCount)
	var scrapych = make(chan customer, scrapyConsumerCount)
	go scrapyProduce(scrapych, &wg)
	for i := 0; i < scrapyConsumerCount; i++ {
		go scrapyConsumer(scrapych, &wg, conactMap, mailSelfBuildMap, i)
	}
	// 等待程序都结束才停止执行
	wg.Wait()
}
