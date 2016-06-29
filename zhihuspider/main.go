package main

import (
	f "fmt"
	//"os"
	"runtime"
	//"runtime/pprof" // 引用pprof package
	"zhihuspider/initproc"
	//"zhihuspider/ranking"
	"zhihuspider/spider"
)

func main() {
	f.Println("CPUNUM :", runtime.GOMAXPROCS(runtime.NumCPU()))
	/*file, _ := os.Create("profile_file")
	pprof.StartCPUProfile(file)  // 开始cpu profile，结果写到文件f中
	defer pprof.StopCPUProfile() // 结束profile*/

	if nil != initproc.LoadJsonFile("init.json", &initproc.IniJsonFile) {
		return
	}

	jsonSpider := initproc.IniJsonFile.Spider
	jsonRank := initproc.IniJsonFile.Rank
	f.Println(jsonSpider)
	f.Println(jsonRank)
	f.Println("Threads: ", jsonSpider.Threads)

	if jsonSpider.Threads <= 0 {
		f.Println("Threads <= 0")
		return
	}

	if int(jsonSpider.Threads) != len(jsonSpider.CacheDB) {
		f.Println("Threads != CacheDB num")
		return
	}

	var sp []*spider.Spider
	chs := make([]chan bool, int(jsonSpider.Threads))

	for i := 0; i < int(jsonSpider.Threads); i++ {
		sp = append(sp, spider.NewSpider(jsonSpider.Url[i], jsonSpider.Num, jsonSpider.DB,
			jsonSpider.CacheDB[i]))
	}

	sp[0].SimulateLogin(jsonSpider.Account, jsonSpider.Passwd)
	for i := 1; i < int(jsonSpider.Threads); i++ { //copy cookies
		sp[i].CopyLoginInfo(sp[0])
	}

	for i := 0; i < int(jsonSpider.Threads); i++ {
		chs[i] = make(chan bool)
		go sp[i].Run(i, chs[i])
	}

	for _, ch := range chs {
		<-ch
	}

	//ranking.NewRanking(jsonRank.DB, jsonRank.RankNum).Run()
}
