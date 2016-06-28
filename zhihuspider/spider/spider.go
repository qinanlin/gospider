package spider

import (
	f "fmt"
	"github.com/PuerkitoBio/goquery"
	redis "gopkg.in/redis.v3"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
)

var once sync.Once

type Spider struct {
	baseUrl     string //基准url
	urlNum      int    //url数目
	count       int    //爬数据过程中的用户计数
	client      http.Client
	cookies     []*http.Cookie
	reClient    *redis.Client
	cacheClient *redis.Client
	file        *os.File
	urlMap      map[string]bool
	cacheDB     int64
}

type userInfo struct {
	url  string
	name string //用户名
	like int    //被赞同数
	thks int    //被感谢数
}

type Jar struct {
	cookies []*http.Cookie
}

func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	jar.cookies = cookies
}

func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
	return jar.cookies
}

func (this *Spider) SimulateLogin(account string, passwd string) error { //模拟登录
	v := url.Values{}
	v.Set("password", passwd)
	v.Set("email", account)
	v.Set("remember_me", "true")

	this.client = http.Client{Jar: new(Jar)}

	resp, err := this.client.PostForm("http://www.zhihu.com/login/email", v)
	if err != nil {
		f.Println("post failed")
		return err
	}

	this.cookies = resp.Cookies()

	defer resp.Body.Close()

	return nil
}

func (this *Spider) CopyLoginInfo(sp *Spider) {
	this.client = http.Client{Jar: new(Jar)}
	this.cookies = sp.cookies
}

func (this *Spider) myGoQuery(url string) (*goquery.Document, error) {
	req, _ := http.NewRequest("GET", url, nil)

	//对付反爬虫，伪造User-Agent
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:37.0) Gecko/20100101 Firefox/37.0")
	req.Header.Add("Connection", "keep-alive")
	this.client.Jar.SetCookies(req.URL, this.cookies)
	res, _ := this.client.Do(req)
	return goquery.NewDocumentFromResponse(res)
}

func (this *Spider) addUserInfo(url string, user map[string]string) error {
	//f.Println(user)
	/*this.file.WriteString(url + "\t")
	for _, v := range user {
		this.file.WriteString(v + "\t")
	}
	this.file.WriteString("\n")*/

	ret := this.reClient.HMSetMap(url, user)
	//f.Println(ret.Result())
	if nil != ret.Err() {
		f.Println("HMSetMap failed")
		return ret.Err()
	}
	return nil
}

func (this *Spider) isExist(url string) bool {
	//直接判断是否在Reis中
	ret := this.reClient.Exists(url)

	return ret.Val() //存在则为true
}

func (this *Spider) getPageInfo(doc *goquery.Document, url string) bool {
	userMap := make(map[string]string)

	userMap["name"] = doc.Find(".title-section a").First().Text()
	if len(userMap["name"]) == 0 {
		userMap["name"], _ = doc.Find(".title-section span").Html()
	}

	if len(userMap["name"]) == 0 {
		f.Println("name is empty :", url)
		this.file.WriteString(url + "\n")
		return false
	}

	userMap["like"] = doc.Find(".zm-profile-header-user-agree").Find("strong").Text()

	userMap["thanks"] = doc.Find(".zm-profile-header-user-thanks").Find("strong").Text()

	if nil != this.addUserInfo(url, userMap) {
		return false
	}

	this.count++

	return true
}

func (this *Spider) processor(seq int, url string) {
	if true == this.isExist(url) {
		return
	}

	doc, err := this.myGoQuery(url) //需模拟登录
	//doc, err := goquery.NewDocument(url) //无登录
	if err != nil {
		f.Println(err, url)
		this.file.WriteString(url + "\n")
		return
	}

	once.Do(func() {
		htmlString, _ := doc.Html()
		html := []byte(htmlString)
		ioutil.WriteFile("html/"+strconv.Itoa(this.count+seq)+".html", html, 0666)
	})

	if this.getPageInfo(doc, url) == true {
		doc.Find(".zm-list-content-medium").Each(func(i int, Sel *goquery.Selection) {
			href, _ := Sel.Find("a").Attr("href")
			this.urlMap[href+"/followees"] = true
		})
	}

	delete(this.urlMap, url) //如果键不存在，那么这个调用将什么都不发生

	return
}

func NewSpider(url string, num int, DBNum int64, CacheDB int64) *Spider {
	sp := Spider{
		baseUrl: url,
		urlNum:  num,
		cacheDB: CacheDB,
	}
	sp.reClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",    // no password set
		DB:       DBNum, // use  DB
	})

	pong, err := sp.reClient.Ping().Result()
	if nil != err {
		f.Println(pong, err)
	}

	file, err1 := os.OpenFile("nameempty.txt", os.O_RDWR|os.O_APPEND, 0666)
	if nil != err1 {
		f.Println("open file error..")
	}
	sp.file = file
	return &sp
}

func (this *Spider) Run(seq int, ch chan bool) {
	var lenth int
	this.urlMap = make(map[string]bool)
	memProfile = "memprofile"

	//this.simulateLogin(account, passwd) //

	if false == this.loadUrlMap(seq) {
		f.Println("LoadURlMap failed")
	}

	lenth = len(this.urlMap)
	f.Println("UrlMap lenth : ", lenth)
	if lenth == 0 {
		this.urlMap[this.baseUrl] = true
	}

	start := time.Now()
	for {
		if this.count >= this.urlNum {
			break
		}

		lenth = len(this.urlMap)

		if lenth == 0 {
			break
		}

		for key, _ := range this.urlMap {
			this.processor(seq, key)
			//time.Sleep(100)
			break
		}

	}
	end := time.Now()

	f.Println(seq, " the mapsize : ", len(this.urlMap))
	f.Println(seq, " total num : ", this.count)

	this.saveUrlMap()

	f.Println(seq, " Spider consumes :", float32(end.Sub(start).Nanoseconds())/(1000*1000*1000*60), "min")

	this.reClient.Close()
	this.cacheClient.Close()
	//stopMemProfile()
	this.file.Close()

	ch <- true
}

func (this *Spider) loadUrlMap(seq int) bool {
	this.cacheClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",           // no password set
		DB:       this.cacheDB, // use  DB
	})

	pong, err := this.cacheClient.Ping().Result()
	if nil != err {
		f.Println(pong, err)
		return false
	}

	/*res := this.cacheClient.LLen("urlcache")
	if res.Val() > 0 {
		cacheRes := this.cacheClient.LRange("urlcache", 0, res.Val())
		if nil != cacheRes.Err() {
			f.Println("LRange error")
			return false
		}
		for _, v := range cacheRes.Val() {
			this.urlMap[v] = true
		}
	}*/
	res := this.cacheClient.SMembers("urlcache")
	if nil != res.Err() {
		f.Println("SMembers error")
		return false
	}
	for _, v := range res.Val() {
		this.urlMap[v] = true
	}

	this.cacheClient.Del("urlcache") //读取后全部删除
	f.Println(seq, " loadUrlMap succ")
	return true
}

func (this *Spider) saveUrlMap() {
	if len(this.urlMap) == 0 {
		return
	}

	for key, _ := range this.urlMap {
		this.cacheClient.SAdd("urlcache", key)
		//this.cacheClient.LPush("urlcache", key)
	}
}

/**************************************************************/
//memory profile

var memProfile string

func startMemProfile(memProfile string, memProfileRate int) {
	if memProfile != "" && memProfileRate > 0 {
		runtime.MemProfileRate = memProfileRate
	}
}

func stopMemProfile(memProfile string) {
	if memProfile != "" {
		file, err := os.Create(memProfile)
		if err != nil {
			f.Fprintf(os.Stderr, "Can not create mem profile output file: %s", err)
			return
		}
		if err = pprof.WriteHeapProfile(file); err != nil {
			f.Fprintf(os.Stderr, "Can not write %s: %s", memProfile, err)
		}
		file.Close()
	}
}
