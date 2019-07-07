package bilibili

import (
	"../dyproxy"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
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
		"//156.235.194.213:8080", "47.52.27.97:31280",
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

		//视频类表下的专辑解析
		var albumWg sync.WaitGroup

		for _, tv := range topic.TopicData.TopicVideo {
			albumWg.Add(1)
			tv := tv
			res, e := insertTopic(tv, db)
			if e != nil {
				fmt.Println("插入主题", e.Error())
			} else {
				r, _ := res.LastInsertId()

				fmt.Println("插入主题", r)
			}
			go openAlbum(tv.Aid, c.Clone(), func() {
				fmt.Println("专辑完成…")
				albumWg.Done()
			}, func(err error) {
				fmt.Println("专辑失败", err.Error())
				albumWg.Done()
			})
		}

		db.Close()
		albumWg.Wait() //专辑完成

		wg.Done() //外层的🔐
	})
	c.OnError(func(response *colly.Response, e error) {
		fmt.Println("❌", e.Error(), string(response.Body))
		wg.Done()
	})
	c.Visit(video.UpSubmitVideosAPI())
}
func downloadFile(url string, name string, fb func(length, downLen int64)) error {
	var (
		fsize   int64
		buf     = make([]byte, 32*1024)
		written int64
	)
	//创建一个http client
	client := new(http.Client)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Refer", "https://static.hdslb.com/play.swf")

	//get方法获取资源
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	//读取服务器返回的文件大小
	fsize, err = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 32)
	if err != nil {
		fmt.Println(err)
	}
	//创建文件
	file, err := os.Create("./video/" + name + ".mp4")
	if err != nil {
		return err
	}
	defer file.Close()
	if resp.Body == nil {
		return errors.New("body is null")
	}
	defer resp.Body.Close()
	//下面是 io.copyBuffer() 的简化版本
	for {
		//读取bytes
		nr, er := resp.Body.Read(buf)
		if nr > 0 {
			//写入bytes
			nw, ew := file.Write(buf[0:nr])
			//数据长度大于0
			if nw > 0 {
				written += int64(nw)
			}
			//写入出错
			if ew != nil {
				err = ew
				break
			}
			//读取是数据长度不等于写入的数据长度
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
		//没有错误了快使用 callback

		fb(fsize, written)
	}
	return err
}
func openAlbum(aid int64, c *colly.Collector, success func(), onError func(err error)) {
	c.OnHTML("html", func(element *colly.HTMLElement) {
		result := regexp.MustCompile("video_url: '(.*?)'").FindAll([]byte(element.Text), -1)
		for _, value := range result {
			videourl := strings.ReplaceAll(string(value), "video_url: '//", "")
			videourl = strings.ReplaceAll(videourl, "'", "")
			yu := strings.Split(videourl, "/")

			ns, err := net.LookupHost(yu[0])
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("ip:", ns)
			}
			var resource = ""
			if len(ns) > 1 {
				for index, ip := range ns {
					ip := ip
					resource = "http://" + ip + "/" + videourl
					fmt.Println("视频地址：", resource)
					e := downloadFile(resource, convert(aid)+"-"+convert(int64(index)), func(length, downLen int64) {
						fmt.Println("视频下载信息：", length, downLen, float32(downLen)/float32(length))
					})
					if e != nil {
						fmt.Println(e.Error())
					}
				}
			} else {
				resource = "http://" + videourl

				fmt.Println("视频地址：", yu, resource)
				go downloadFile(resource, convert(aid), func(length, downLen int64) {
					fmt.Println("视频下载信息：", length, downLen, float32(downLen)/float32(length))
				})

			}

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
		success()
	})
	c.OnError(func(response *colly.Response, e error) {
		onError(e)
	})
	c.Visit("https://m.bilibili.com/video/av" + convert(aid) + ".html")
}

//打开某一视频 并解析出详情所在专辑中的详细视频列表

func open(video *Video, c *colly.Collector, wg *sync.WaitGroup) {
	tmpVide := video
	openAlbum(video.Aid, c.Clone(), func() {
		dbResult, db := OpenDB()
		if dbResult {
			res, err := insertUpToDB(db, video.mIdString())
			if err != nil {
				fmt.Println("插入数据失败", err.Error())
				db.Close()
				wg.Done()
			} else {
				id, _ := res.LastInsertId()

				fmt.Println("插入数据成功：", id)
				openUpSubmitVideosFrom(tmpVide, c.Clone(), wg, db)
			}

		} else {
			wg.Done()
		}
	}, func(err error) {
		fmt.Println(err.Error())
		wg.Done()
	})
	/*
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

	*/
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
		//collector.UserAgent = RandomString()
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
