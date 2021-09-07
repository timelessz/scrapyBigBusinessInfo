package main

import (
	"fmt"
	"github.com/tebeka/selenium"
	"github.com/tebeka/selenium/chrome"
)

const (
	chromeDriver = "C:\\Program Files\\chromedriver_win32\\chromedriver.exe"
	driverport   = 8080
)

func getWebDriver() (*selenium.Service, selenium.WebDriver) {
	ops := []selenium.ServiceOption{
		//selenium.Output(os.Stderr), // Output debug information to STDERR.
	}
	selenium.SetDebug(false)
	service, err := selenium.NewChromeDriverService(chromeDriver, driverport, ops...)
	if err != nil {
		panic(err) // panic is used only as an example and is not otherwise recommended.
	}
	caps := selenium.Capabilities{"browserName": "chrome"}
	//禁止图片加载，加快渲染速度
	imagCaps := map[string]interface{}{
		"profile.managed_default_content_settings.images": 2,
	}
	chromeCaps := chrome.Capabilities{
		Prefs: imagCaps,
		Path:  "",
		Args: []string{
			//"--headless", // 设置Chrome无头模式，在linux下运行，需要设置这个参数，否则会报错
			//"--no-sandbox",
			"--user-agent=Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/69.0.3497.100 Safari/537.36", // 模拟user-agent，防反爬
		},
	}
	//以上是设置浏览器参数
	caps.AddChrome(chromeCaps)
	wd, err := selenium.NewRemote(caps, fmt.Sprintf("http://localhost:%d/wd/hub", driverport))
	if err != nil {
		panic(err)
	}
	return service, wd
}

// 爬取数据 获取mx 信息
func scrapy(wd selenium.WebDriver, url string) (string, string) {
	if err := wd.Get(url); err != nil {
		println(err.Error())
		return "", ""
	}
	title, _ := wd.Title()
	source, _ := wd.PageSource()
	return title, source
}
