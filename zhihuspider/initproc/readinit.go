package initproc

import (
	"encoding/json"
	f "fmt"
	"io/ioutil"
)

type initSpider struct { //json struct
	Url     []string
	Num     int
	Account string
	Passwd  string
	DB      int64
	CacheDB []int64
	Threads int64
}

type initRank struct {
	DB      int64
	RankNum int64
}

type InitFile struct {
	Spider initSpider
	Rank   initRank
}

var IniJsonFile InitFile

func LoadJsonFile(filename string, v interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		f.Println("ReadFile error...")
		return err
	}

	if nil != json.Unmarshal(data, v) {
		f.Println("unmarshal failed...")
		return err
	}

	return nil
}
