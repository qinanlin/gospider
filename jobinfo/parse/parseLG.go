// parseLG.go
package parse

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	//"io"
	//"log"
	"net/http"
	"regexp"
	"strconv"
	//"strings"
	//"os"
	"sync"
)

var strMutex sync.Mutex
var Urls []string

func urlAdd(url string) {
	strMutex.Lock()

	for _, v := range Urls { //防止重复
		if v == url {
			strMutex.Unlock()
			return
		}
	}
	Urls = append(Urls, url)

	strMutex.Unlock()
}

func lagouGoSearch(ch chan bool, posId string) {
	url := "http://www.lagou.com/jobs/" + posId + ".html"
	doc, err := goquery.NewDocument(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(PosId)

	doc.Find(".job_bt").Each(func(i int, Sel *goquery.Selection) {

		p := Sel.Find("p").Text()
		//fmt.Println("des:", p)

		re, _ := regexp.Compile("[gG][oO]") //go/Go/GO/gO

		if re.Match([]byte(p)) {
			//fmt.Println("test")
			urlAdd(url)
		}
	})

	ch <- true
}

func parseOtherPage(url string, pageId string, ch chan bool) {
	resp, err := http.Get(url + "&pn=" + pageId) //pn=xxx表示第几页
	if err != nil {
		fmt.Println("Get failed...")
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var jVal interface{}
	if err1 := dec.Decode(&jVal); err1 != nil {
		fmt.Println(err)
		return
	}

	var pageSize int //每页列出的职位数
	var chs []chan bool
	ich := 0 //goroutine计数

	//fmt.Println("in page =", pageId)
	//未知结构的json，必须以此方式一层一层解析,层数太多，代码可读性低
	j, ok := jVal.(map[string]interface{}) //j is map[string]interface{}
	if ok {
		for i, v := range j {
			if i == "content" {
				j1, ok1 := v.(map[string]interface{}) //j1 is map[string]interface{}
				if ok1 {
					for i1, v1 := range j1 {
						if i1 == "pageSize" {
							pageSize = int(v1.(float64))
							chs = make([]chan bool, pageSize)
						}
						if i1 == "positionResult" {
							j2, ok2 := v1.(map[string]interface{})
							if ok2 {
								for i2, v2 := range j2 {
									if i2 == "result" {
										j3, ok3 := v2.([]interface{})
										if ok3 {
											for _, v3 := range j3 {
												j4, ok4 :=
													v3.(map[string]interface{})
												if ok4 {
													for i4, v4 := range j4 {
														if i4 == "positionId" {
															//fmt.Println(int(v4.(float64)))
															if ich < pageSize {
																chs[ich] = make(chan bool)
																go lagouGoSearch(chs[ich],
																	strconv.Itoa(int(v4.(float64))))
																ich++
															}
															break
														}
													}
												}
											}
										}
										break
									}
								}
							}
						}
					}
				}
				break
			}
		}
	}

	//等所有goroutine返回才退出
	for i, cha := range chs {
		if i < ich && ich <= len(chs) { //有效goroutine数不一定等于len(chs)
			<-cha
		}
	}
	ch <- true
}

func ParseLagouGo(url string) {
	resp, err := http.Get(url) //pn=xxx表示第几页
	if err != nil {
		fmt.Println("Get failed...")
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	var jVal interface{}
	if err1 := dec.Decode(&jVal); err1 != nil {
		fmt.Println(err)
		return
	}

	var totalCount, pageSize int // 分别为职位总数和每页列出的职位数
	var chsJob, chsPage []chan bool
	ich := 0 //goroutine计数

	/*{
	  "msg":null,
	  "content":{
	  	"pageNo":1,
	  	"pageSize":15,
	  	"positionResult":{
	  		"totalCount":10,
	  		"pageSize":10,
	  		"result":[{
	  			"positionId":1376562,
	  			"companyLogo":"i/image/M00/11/AA/Cgp3O1bicWSAdZyLAAB0E23XERw959.png",
	  			"positionFirstType":"技术",
	  			}, ...]
		}
		......
	  }*/
	//未知结构的json(见上)，必须以此方式一层一层解析,层数太多，代码可读性低
	j, ok := jVal.(map[string]interface{}) //j is map[string]interface{}
	if ok {
		for i, v := range j {
			if i == "content" {
				j1, ok1 := v.(map[string]interface{}) //j1 is map[string]interface{}
				if ok1 {
					for i1, v1 := range j1 {
						if i1 == "pageSize" {
							pageSize = int(v1.(float64))
							//fmt.Println(pageSize)
							chsJob = make([]chan bool, pageSize)
						}
						if i1 == "positionResult" {
							j2, ok2 := v1.(map[string]interface{})
							if ok2 {
								for i2, v2 := range j2 {
									if i2 == "totalCount" {
										totalCount = int(v2.(float64))
									}
									if i2 == "result" {
										j3, ok3 := v2.([]interface{})
										if ok3 {
											for _, v3 := range j3 {
												j4, ok4 :=
													v3.(map[string]interface{})
												if ok4 {
													for i4, v4 := range j4 {
														if i4 == "positionId" {
															//fmt.Println(int(v4.(float64)))
															if ich < pageSize {
																chsJob[ich] = make(chan bool)
																go lagouGoSearch(chsJob[ich],
																	strconv.Itoa(int(v4.(float64))))
																ich++
															}
															break
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
				break
			}
		}
	}

	fmt.Println("pagesize :", pageSize)
	fmt.Println("total :", totalCount)

	page := totalCount / pageSize
	//fmt.Println(page)

	mod := func() int {
		if totalCount%pageSize == 0 {
			return 0
		}
		return 1
	}() //必须加括后，否则mod为函数

	page = func() int {
		if page > 2 { //只检索前三页
			return 2
		}
		return page
	}()

	chsPage = make([]chan bool, page+mod-1) //首页已处理所以必须减一

	for i := 0; i < page; i++ {
		chsPage[i] = make(chan bool)
		go parseOtherPage(url, strconv.Itoa(i+2), chsPage[i]) //从第二页开始
	}

	//等所有goroutine返回才退出
	for i, ch := range chsJob {
		if i < ich && ich <= len(chsJob) { //有效goroutine数不一定等于len(chs)
			<-ch
		}
	}

	for _, ch := range chsPage {
		<-ch
	}
}
