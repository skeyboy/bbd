package bilibili

import (
	"../dyproxy"
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	//user:password@/dbname
	DB_Driver = "root:@/bbd"
)

func OpenDB() (success bool, db *sql.DB) {
	var isOpen bool
	db, err := sql.Open("mysql", "root:@/bbd")
	if err != nil {
		isOpen = false
	} else {
		isOpen = true
	}
	return isOpen, db
}
func insertUpToDB(db *sql.DB, mid string) (sql.Result, error) {

	stmt, err := db.Prepare("insert into bbd_up(mid) values (?)")

	res, err := stmt.Exec(mid)

	return res, err
}
func convert(v int64) string {

	return strconv.FormatInt(v, 10)
}
func insertTopic(tv TopicVideo, db *sql.DB) (sql.Result, error) {
	stmt, err := db.Prepare("insert into bbd_topic(mid,aid,title,pic,description) value (?,?,?,?,?)")

	res, err := stmt.Exec(tv.Mid, tv.Aid, tv.Title, tv.Pic, tv.Description)

	return res, err
}
func parseXiciProxy(c *colly.Collector) (colly.ProxyFunc, error) {

	var pool = dyproxy.AllProxy()

	var wg sync.WaitGroup
	a := []string{}
	for _, v := range pool {
		v := v
		fmt.Println("可用IP", v)
		a = append(a, "//"+v.FullIp())

	}

	wg.Wait()
	rp, err := proxy.RoundRobinProxySwitcher(
		"//156.235.194.213:8080", "//207.154.200.199:3128", "//138.201.223.250:31288",
		"//201.217.247.101:80",
	)
	/*
		if err != nil {
			log.Fatal(err)
		}
		c.SetProxyFunc(rp)
	*/
	if err != nil {
		fmt.Println("❎", err.Error())
	}

	c.SetProxyFunc(rp)
	return rp, err
}

/**
up主提交的所有视频
*/
func openUpSubmitVideosFrom(video *Video, c *colly.Collector, wg *sync.WaitGroup, db *sql.DB) {
	tmpVideo := video
	c.OnResponse(func(response *colly.Response) {
		js := string(response.Body)
		fmt.Println(video.Mid, ":up主的视频专辑:", js, tmpVideo)
		var topic = Topic{}
		json.Unmarshal(response.Body, &topic)

		for _, tv := range topic.TopicData.TopicVideo {
			tv := tv
			res, e := insertTopic(tv, db)
			if e != nil {
				fmt.Println("插入主题", e.Error())
			} else {
				r, _ := res.LastInsertId()

				fmt.Println("插入主题", r)
			}
		}
		db.Close()

		wg.Done()
	})
	c.OnError(func(response *colly.Response, e error) {
		fmt.Println("❌", e.Error(), string(response.Body))
		wg.Done()
	})
	c.Visit(video.UpSubmitVideosAPI())
}

//打开某一视频 并解析出详情所在专辑中的详细视频列表

func open(video *Video, c *colly.Collector, wg *sync.WaitGroup) {
	tmpVide := video
	c.OnHTML("html", func(element *colly.HTMLElement) {
		fmt.Println(tmpVide)
		result := regexp.MustCompile("video_url: '(.*?)'").FindAll([]byte(element.Text), -1)
		for _, value := range result {
			fmt.Println("视频地址：", string(value))
		}
		result = regexp.MustCompile("image: '(.*?)'").FindAll([]byte(element.Text), -1)
		for _, value := range result {
			fmt.Println("图像封面：", string(value))
		}
		result = regexp.MustCompile("window.__INITIAL_STATE__={(.*?)};").FindAll([]byte(element.Text), -1)

		for _, value := range result {

			dbResult, db := OpenDB()

			info := string(value)

			info = strings.ReplaceAll(info, "window.__INITIAL_STATE__=", "")
			info = strings.ReplaceAll(info, ";", "")
			fmt.Println("专辑详情：", info)

			var album = Album{}
			json.Unmarshal([]byte(info), &album)

			if dbResult {
				stmt, e := db.Prepare("insert into bbd_album(aid,videos,title,state,originTitle,origin) value(?,?,?,?,?,?)")
				if e != nil {
					db.Close()
				} else {
					origin := info
					info := album.AlbumContext.AlbumInfo

					res, e := stmt.Exec(info.Aid, info.Videos, info.Title, info.State, info.OriginTitle, origin)
					if e != nil {
						fmt.Println(e.Error())
					} else {
						r, _ := res.LastInsertId()
						fmt.Println("专辑插入成功", r)
					}
					db.Close()
				}
			}

		}
		dbResult, db := OpenDB()
		if dbResult {
			res, err := insertUpToDB(db, video.mIdString())
			if err != nil {
				fmt.Println("插入数据失败", err.Error())
				db.Close()
			} else {
				id, _ := res.LastInsertId()

				fmt.Println("插入数据成功：", id)
				openUpSubmitVideosFrom(tmpVide, c.Clone(), wg, db)
			}

		}
		//wg.Done()
	})
	c.OnError(func(response *colly.Response, e error) {
		fmt.Println(e.Error())
		wg.Done()
	})
	c.Visit(video.VideoHome())
}

func engineerSearch(url string, search *Search, c *colly.Collector, callback func(page int, result *SearchResult), finished func()) {
	c.OnRequest(func(request *colly.Request) {
		request.Headers.Set("Content-Type", "application/json")
		request.Headers.Set("Accept", "application/json")
		request.Method = "POST"

	})
	c.OnError(func(response *colly.Response, e error) {
		if e != nil {
			fmt.Println("⚠️", e.Error(), string(response.Body))
			finished()
		}
	})
	c.OnResponse(func(response *colly.Response) {
		//fmt.Println(string(response.Body))
		var searchResult = SearchResult{}
		e := json.Unmarshal(response.Body, &searchResult)
		if e != nil {
			fmt.Println("❌")
			finished()
		}
		if searchResult.IsSuccess() {
			callback(search.Page+1, &searchResult)
		} else {
			fmt.Println("结束🔚", searchResult.Msg, searchResult.Page)
			finished()
		}
	})
	c.OnHTML("html", func(element *colly.HTMLElement) {

	})
	js, e := json.Marshal(search)
	if e != nil {

	}
	c.PostRaw(url, js)
	//c.Wait()
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandomString() string {
	b := make([]byte, rand.Intn(10)+10)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func start(page int, keyword string, mark *chan bool) {
	c := colly.NewCollector(func(collector *colly.Collector) {
		collector.IgnoreRobotsTxt = true
		collector.Async = true
		collector.UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"
	})
	parseXiciProxy(c)
	cc := c.Clone()

	cc.OnResponse(func(response *colly.Response) {
		//关闭其实页面结果
		fmt.Print(response)
	})
	search := Search{Keyword: url.QueryEscape(keyword), Order: "totalrank", Main_ver: "v3", Page: page, Bangumi_num: 3, Movie_num: 3}
	url := "https://m.bilibili.com/search/searchengine"
	//v := make(chan bool)

	go engineerSearch(url, &search, cc.Clone(), func(p int, result *SearchResult) {
		//关闭关键词搜索🔍结果log展示
		//fmt.Println(result.Page, "/t", result)
		var wg sync.WaitGroup
		wg.Add(len(result.Result.Video))
		for _, video := range result.Result.Video {
			v := video
			go open(&v, c.Clone(), &wg)
			//if HasUp(v) == false {
			//	Add(v)
			//}
			//wg.Done()
		}
		wg.Wait()

		go start(int(result.Page)+1, keyword, mark)
		//<-v
		//close(v)
	}, func() {
		//<-v
		//close(v)
		//os.Exit(0)
		fmt.Println("获得的🉐", videos)

		*mark <- true

	})
	//<-v

	//cc.Visit("https://github.com/golang/text")
	cc.Wait()
}
func Bilibili(page int, keyword string, v chan bool) {
	//v := make(chan bool)
	go start(page, keyword, &v)
	//<-v
	//close(v)
}

var videos = []Video{}
var lock sync.RWMutex

func Add(v Video) {
	videos = append(videos, v)
}

func HasUp(v Video) bool {
	lock.Lock()

	result := false
	for _, value := range videos {
		video := value
		if v.Mid == video.Mid {
			result = true
			break
		}
	}

	lock.Unlock()
	return result

}
