package ranking

import (
	"container/heap"
	"strconv"
	//"container/heap"
	f "fmt"
	redis "gopkg.in/redis.v3"
	//"sort"
	"sync"
	"time"
)

var once sync.Once

type ranking struct {
	rankNum  int64
	reClient *redis.Client
	likeRank *RankData
	thsRank  *RankData
}

type com struct {
	url     string
	data    []string
	control int8
}

func NewRanking(DBNum int64, RankNum int64, DBAddr string, DBPasswd string) *ranking {
	ra := ranking{rankNum: RankNum}
	ra.reClient = redis.NewClient(&redis.Options{
		Addr:     DBAddr,
		Password: DBPasswd,
		DB:       DBNum, // use  DB
	})

	pong, err := ra.reClient.Ping().Result()
	if nil != err {
		f.Println(pong, err)
	}

	ra.thsRank = new(RankData)
	ra.likeRank = new(RankData)

	return &ra
}

func (this *ranking) heapAdjust(ra *RankData, unit rankUnit) {
	l := len(*ra)
	if l == 0 {
		*ra = RankData{unit}
		heap.Init(ra)
		return
	}

	if int64(l) >= this.rankNum {
		v := heap.Pop(ra)
		if v.(rankUnit).val >= unit.val {
			heap.Push(ra, v.(rankUnit))
			return
		}
	}
	heap.Push(ra, unit)
}

func (this *ranking) sortData(comChan chan com) {
	var comStr com
	var unit rankUnit
	var like, thks int
	for {
		comStr = <-comChan
		if comStr.control == 0 {
			break
		}
		//HGetAll返回的string数组需判断具体字段才能确定数据
		for i, v := range comStr.data {
			switch v {
			case "name":
				unit.name = comStr.data[i+1]
			case "like":
				like, _ = strconv.Atoi(comStr.data[i+1])
			case "thanks":
				thks, _ = strconv.Atoi(comStr.data[i+1])
			}
		}
		unit.url = comStr.url
		unit.val = like
		this.heapAdjust(this.likeRank, unit)
		unit.val = thks
		this.heapAdjust(this.thsRank, unit)
	}
}

func (this *ranking) processData() {
	var cursor int64
	var val []string
	flag := true
	var comStr com

	comChan := make(chan com)
	go this.sortData(comChan)

	for flag {
		res := this.reClient.Scan(cursor, "", 0)
		if nil != res.Err() {
			f.Println("scan error")
			return
		}

		cursor, val = res.Val()
		if cursor == 0 {
			flag = false
		}

		for _, v := range val {
			hVal := this.reClient.HGetAll(v)
			if nil != hVal.Err() {
				f.Println("HGetAll failed :", v)
				continue
			}
			comStr.url = v
			comStr.data = hVal.Val()
			comStr.control = 1
			comChan <- comStr
		}
		//flag = false //test
	}
	comStr.control = 0
	comChan <- comStr
	close(comChan)
}

func (this *ranking) afterRank() {
	for i := 0; this.likeRank.Len() > 0; i++ {
		v := heap.Pop(this.likeRank)
		f.Printf("No.%d:\t like:%d\t Name:%s\t url:%s\n", this.rankNum-int64(i),
			v.(rankUnit).val, v.(rankUnit).name, v.(rankUnit).url)
	}

	f.Println("**********************************")
	for i := 0; this.thsRank.Len() > 0; i++ {
		v := heap.Pop(this.thsRank)
		f.Printf("No.%d:\t thanks:%d\t Name:%s\t url:%s\n", this.rankNum-int64(i),
			v.(rankUnit).val, v.(rankUnit).name, v.(rankUnit).url)
	}
}

func (this *ranking) Run() {
	start := time.Now()
	this.processData()
	end := time.Now()
	f.Println("Ranking consumes :", float32(end.Sub(start).Nanoseconds())/(1000*1000*1000), "sec")

	this.afterRank()

	this.reClient.Close()
}
