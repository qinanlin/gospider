package main

import (
	"fmt"
	"runtime"
	"spider/parse"

	/*"io/ioutil"
	"log"
	"net/http"*/)

type parseFunc func(string)

type spider struct {
	url   string
	pfunc parseFunc
}

var spiders = []spider{
	{"http://www.lagou.com/jobs/positionAjax.json?px=new&city=%E6%B7%B1%E5%9C%B3&kd=c", parse.ParseLagouGo},
	{"http://www.lagou.com/jobs/positionAjax.json?px=new&city=%E6%B7%B1%E5%9C%B3&kd=c++", parse.ParseLagouGo},
}

func spiderFunc(te spider, ch chan bool) {
	te.pfunc(te.url)
	ch <- true
}

func printUrls() {
	if 0 == len(parse.Urls) {
		fmt.Println("No result")
		return
	}

	for _, v := range parse.Urls {
		fmt.Println(v)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	chNum := len(spiders)

	chs := make([]chan bool, chNum)

	for i, page := range spiders {
		chs[i] = make(chan bool)
		go spiderFunc(page, chs[i])
	}

	//等所有goroutine返回才退出
	for _, ch := range chs {
		<-ch
	}
	printUrls() //

	fmt.Println("end------")
}
