package main

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/jonsen/gotld"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

// 截取 mx 后缀的主域名
func getUrlTldDomain(urls string) string {
	u, err := url.Parse(urls)
	if err != nil {
		//panic(err)
	}
	_, domain, err := gotld.GetTld("http://" + u.Host)
	if nil != err {
		fmt.Println(err)
		return ""
	}
	return domain
}

// 截取url 对应的 域名
func getDomainTldDomain(urls string) string {
	_, domain, err := gotld.GetTld(urls)
	if nil != err {
		fmt.Println(err)
		return ""
	}
	return domain
}

func execDigCommand(domain string) string {
	cmd := exec.Command("dig", "-t", "mx", "+short", domain)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
	}
	return string(out)
}

func analyseMxRecord(mxrecord string) string {
	arr := strings.Split(mxrecord, "\r\n")
	var mx string
	prority := 0
	for _, v := range arr {
		if v == "" {
			continue
		}
		arr := strings.Split(v, " ")
		if len(arr) == 2 {
			cprority, _ := strconv.Atoi(arr[0])
			if cprority < prority || prority == 0 {
				mx = arr[1]
			}
		}
	}
	if mx != "" {
		return mx[0 : len(mx)-1]
	}
	return mx
}

func getMxRecordSuffix(suffixMap map[string]MxSuffix, suffix string) (MxSuffix, error) {
	if _, ok := suffixMap[suffix]; ok {
		mss := suffixMap[suffix]
		return mss, nil
	}
	return MxSuffix{}, errors.New("未匹配到MX数据")
}

func saveCustomerMxInfo(db *gorm.DB, mss MxSuffix, domain string, v customer, mxrecord string, i int) {
	fmt.Println(strconv.Itoa(i) + "号消费者：" + v.Name.String + " 域名：" + domain + " 获取mx:" + mxrecord)
	if mss != (MxSuffix{}) {
		// 判断非空 struct 表示匹配到mx 情况
		BId, _ := strconv.ParseInt(mss.BId, 10, 64)
		if v.MxBrandID.Int64 != BId {
			//更新数据
			v.MxBrandID.Int64 = BId
			v.MxBrandName.String = mss.Name
			v.Mxrecord.String = mxrecord
			fmt.Println(strconv.Itoa(i) + "消费者：" + v.Domain.String + "保存mx信息，匹配到邮箱品牌，品牌：" + mss.Name)
			db.Save(v)
		}
	} else {
		// 判断 struct 为空 未匹配到品牌
		if v.Mxrecord.String != mxrecord {
			v.Mxrecord.String = mxrecord
			fmt.Println(strconv.Itoa(i) + "消费者：" + v.Domain.String + "保存mx信息，未匹配到邮箱品牌。")
			db.Save(v)
		}
	}
}

func saveCustomerInfo(db *gorm.DB, v customer, mailTitle string, selfBuildBrandId int, selfbuildBrandName string, domainTitle string, contactBrandId int, contactBrandName string) {
	changeSatus := false
	if v.MailTitle.String != mailTitle {
		v.MailTitle.String = mailTitle
		changeSatus = true
	}
	if v.Title.String != domainTitle {
		v.Title.String = domainTitle
		changeSatus = true
	}
	if v.SelfbuildBrandID.Int64 != int64(selfBuildBrandId) {
		v.SelfbuildBrandID.Int64 = int64(selfBuildBrandId)
		v.SelfbuildBrandName.String = selfbuildBrandName
		changeSatus = true
	}
	if v.ContacttoolBrandID.Int64 != int64(contactBrandId) {
		v.ContacttoolBrandID.Int64 = int64(contactBrandId)
		v.ContacttoolBrandName.String = contactBrandName
		changeSatus = true
	}
	if changeSatus {
		db.Save(v)
	}
}

// 保存客户域名
func saveCustomerDomain(db *gorm.DB, domain string, v customer, i int) {
	fmt.Println(strconv.Itoa(i) + "号消费者：" + v.Name.String + "更新域名：" + domain)
	db.Save(v)
}
