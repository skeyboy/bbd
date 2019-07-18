package main

import (
	"./bdb"
	"./bilibili"
	"fmt"
	"math"
	"sync"
)

func main() {

	/*
		f, e := os.Create("./bilibili/bi.txt")
		defer f.Close()
		if e != nil {
			os.Exit(0)
		}
		w := csv.NewWriter(f)

		w.WriteAll([][]string{
			{"A"},
			{"B"},
		})
		w.Flush()
	*/
	//keywords := []string{"豫剧", "京剧",  }
	keywords := []string{"豫剧", "京剧", "秦腔",
		"曲剧", "晋剧", "评剧", "越剧", "黄梅戏",
	}

	v := make(chan bool, len(keywords))
	//defer close(v)
	//for _, value := range keywords {
	//	keyword := value
	//	go bilibili.Bilibili(1, keyword, v)
	//
	//}
	//for i := 0; i < cap(v); i++ {
	//	<-v
	//}

	//控制后续启动之后的并发量
	var step = 2
	var index = 0
	var lock sync.RWMutex
	next := int(math.Min(float64(index+step), float64(len(keywords))))
	sub := keywords[index:next]
	for _, value := range sub {
		value := value
		go stepper(v, value)
	}
	index += step
loop:
	for {
		select {
		case <-v:

			//保证总是有N个在执行
			if index < len(keywords) {
				lock.Lock()
				var next = index
				index = index + 1
				lock.Unlock()
				go stepper(v, keywords[next])
			} else {
				if len(v) == 0 {
					fmt.Println("准备清空")
					//os.Exit(0)
					lock.Lock()
					defer lock.Unlock()
					for range v {
						close(v)
					}
					break loop
				}
			}
		}
	}
	fmt.Println("爬取：", keywords, "完成")
	clearTopic := `DELETE a.* from bbd_topic a WHERE a.mid NOT IN (SELECT c.mid FROM (SELECT b.mid FROM bbd_topic b WHERE b.title REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+"  OR b.description REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+")  c)`
	clearAlbum := `DELETE a.* FROM bbd_album a WHERE a.aid NOT IN (SELECT c.aid FROM (SELECT b.aid FROM bbd_album b WHERE b.title REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+" OR b.origin REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+"  ) c)`

	tResult, te := bdb.GlobalDB().Exec(clearTopic)
	if te != nil {
		fmt.Println(te.Error())
	} else {
		r, e := tResult.RowsAffected()
		if e != nil {
			fmt.Println(e.Error())
		} else {
			fmt.Println("清洗数据条目:", r)
		}
	}

	aResult, ae := bdb.GlobalDB().Exec(clearAlbum)
	if ae != nil {
		fmt.Println(ae.Error())
	} else {
		r, e := aResult.RowsAffected()
		if e != nil {
			fmt.Println(e.Error())
		} else {
			fmt.Println("清洗数据条目:", r)
		}
	}
	bdb.GlobalDB().Close()
}
func stepper(v chan bool, keyword string) {
	go bilibili.Bilibili(1, keyword, v)

}
