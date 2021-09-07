package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ScrapyInfo struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// 读取 文件到数据库中
func readFile(fileName string) ScrapyInfo {
	absPath, _ := filepath.Abs(fileName)
	file, err := os.Open(absPath)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var sinfo ScrapyInfo
	fileContent, err := ioutil.ReadAll(file)
	if err := json.Unmarshal(fileContent, &sinfo); err != nil {
		fmt.Println("Read file error =", err)
	} else {
		fmt.Println("Read file success =", sinfo)
	}
	fmt.Println(string(fileContent))
	return sinfo
}

// 写文件 到数据库中
func writeFile(fileName string, sinfo ScrapyInfo) bool {
	//s, _ := json.Marshal(sinfo)
	//fmt.Println(string(s))
	fileContent, _ := json.Marshal(sinfo)
	absPath, _ := filepath.Abs(fileName)
	err := ioutil.WriteFile(absPath, fileContent, 0644)
	if err != nil {
		panic(err)
	}
	return true
}
