package bilibili

import (
	"../bdb"
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
	"time"
)

func OpenDB() (success bool, db *sql.DB) {
	return bdb.GlobalIsOK(), bdb.GlobalDB()

}

func insertUpToDB(db *sql.DB, mid string, face string, name string) (sql.Result, error) {
	tx, _ := db.Begin()

	stmt, err := tx.Prepare("insert into bbd_up(mid, face, name) values (?,?,?)")

	res, err := stmt.Exec(mid, face, name)
	if err != nil {
		tx.Rollback()

	} else {
		e := tx.Commit()
		if e != nil {
			tx.Rollback()
		}
	}
	return res, err
}
func convert(v int64) string {

	return strconv.FormatInt(v, 10)
}
func insertTopic(tv TopicVideo, db *sql.DB) (sql.Result, error) {
	tx, e := db.Begin()
	stmt, err := tx.Prepare("insert into bbd_topic(mid,aid,title,pic,description) value (?,?,?,?,?)")

	res, err := stmt.Exec(tv.Mid, tv.Aid, tv.Title, tv.Pic, tv.Description)
	if err != nil {
		tx.Rollback()
	} else {
		e = tx.Commit()
		if e != nil {
			tx.Rollback()
		}
	}
	return res, err
}
func parseXiciProxy(c *colly.Collector) (colly.ProxyFunc, error) {

	var pool = dyproxy.AllProxy()
	_, db := OpenDB()

	var wg sync.WaitGroup
	a := []string{}

	for _, v := range pool {
		v := v
		fmt.Println("可用IP", v)
		a = append(a, v.FullIp())

		tx, _ := db.Begin()
		pre, _ := tx.Prepare("insert into bbd_ip(ip,port) value(?,?)")

		_, err := pre.Exec(v.Ip, v.Port)
		if err != nil {
			tx.Rollback()
			fmt.Println("ip错误", err.Error())
		} else {
			e := tx.Commit()
			if e != nil {
				tx.Rollback()
			}
		}
		pre.Close()

	}

	wg.Wait()
	rp, err := proxy.RoundRobinProxySwitcher(a...)
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
	defer wg.Done() //外层的🔐

	tmpVideo := video
	c.OnResponse(func(response *colly.Response) {
		js := string(response.Body)
		fmt.Println(video.Mid, ":up主的视频专辑:", js, tmpVideo)
		var topic = Topic{}
		json.Unmarshal(response.Body, &topic)

		//视频类表下的专辑解析
		var albumWg sync.WaitGroup
		albumWg.Add(len(topic.TopicData.TopicVideo))
		for _, tv := range topic.TopicData.TopicVideo {
			//albumWg.Add(1)
			tv := tv
			res, e := insertTopic(tv, db)
			if e != nil {
				fmt.Println("插入主题", e.Error())
			} else {
				r, _ := res.LastInsertId()

				fmt.Println("插入主题", r)
			}
			go OpenAlbum(tv.Aid, c.Clone(), func(album_owner AlbumOwner) {
				fmt.Println("专辑完成…")
				albumWg.Done()

			}, func(url *url.URL, err error) {
				fmt.Println("专辑失败", err)
				//把失败的专辑入库，恢复的时候优先爬取
				tx, _ := db.Begin()
				stmt, _ := tx.Prepare("insert into bbd.bbd_album_failed(album_url) value(?)")
				if stmt != nil {
					defer stmt.Close()
					res, err := stmt.Exec(url.Scheme + "://" + url.Host + url.Path)
					if err != nil {
						if e != nil {
							tx.Rollback()
						}
						fmt.Println("失败保存失败", err.Error())
					} else {
						r, _ := res.LastInsertId()
						e := tx.Commit()
						if e != nil {
							tx.Rollback()
						}
						fmt.Println("失败保存成功", r)
					}
				}
				albumWg.Done()
			})
		}

		albumWg.Wait() //专辑完成
		//wg.Done() //外层的🔐

	})
	c.OnError(func(response *colly.Response, e error) {
		fmt.Println("❌", e.Error(), string(response.Body))
		//wg.Done()
		// wg.Done() //外层的🔐

	})
	//根据id进行随机的时间休息
	time.Sleep(time.Second * time.Duration(video.Id%10))
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
func OpenAlbum(aid int64, c *colly.Collector, success func(album_owner AlbumOwner), onError func(url *url.URL, err error)) {
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
		}
		result = regexp.MustCompile("image: '(.*?)'").FindAll([]byte(element.Text), -1)
		for _, value := range result {
			fmt.Println("图像封面：", string(value))
		}
		result = regexp.MustCompile("window.__INITIAL_STATE__={(.*?)};").FindAll([]byte(element.Text), -1)
		if len(result) == 0 {
			onError(element.Request.URL, nil)
		} else {
			for _, value := range result {

				dbResult, db := OpenDB()

				info := string(value)

				info = strings.ReplaceAll(info, "window.__INITIAL_STATE__=", "")
				info = strings.ReplaceAll(info, ";", "")
				fmt.Println("专辑详情：", info)

				var album = Album{}
				json.Unmarshal([]byte(info), &album)

				if dbResult {
					tx, _ := db.Begin()
					stmt, e := tx.Prepare("insert into bbd_album(aid,videos,title,state,originTitle,origin) value(?,?,?,?,?,?)")
					if e != nil {
						tx.Rollback()
					} else {
						origin := info
						info := album.AlbumContext.AlbumInfo

						res, e := stmt.Exec(info.Aid, info.Videos, info.Title, info.State, info.OriginTitle, origin)
						defer stmt.Close()
						if e != nil {
							tx.Rollback()
							fmt.Println(e.Error())
						} else {
							r, _ := res.LastInsertId()
							e := tx.Commit()
							if e != nil {
								tx.Rollback()
							}

							fmt.Println("专辑插入成功", r)
						}

					}
				}
				success(album.AlbumContext.AlbumInfo.Owner)

			}
		}
	})
	c.OnError(func(response *colly.Response, e error) {
		//onError(response.Request.URL, e)
	})
	time.Sleep(time.Second * time.Duration(aid%10))
	c.Visit("https://m.bilibili.com/video/av" + convert(aid) + ".html")
}

//打开某一视频 并解析出详情所在专辑中的详细视频列表

func open(video *Video, c *colly.Collector, wg *sync.WaitGroup) {
	tmpVide := video
	OpenAlbum(video.Aid, c.Clone(), func(album_owner AlbumOwner) {
		dbResult, db := OpenDB()
		if dbResult {
			res, err := insertUpToDB(db, convert(album_owner.Mid), album_owner.Face, album_owner.Name)
			if err != nil {
				fmt.Println("插入数据失败", err.Error())
				wg.Done()
			} else {
				id, _ := res.LastInsertId()
				fmt.Println("插入数据成功：", id)
				openUpSubmitVideosFrom(tmpVide, c.Clone(), wg, db)
			}

		} else {
			wg.Done()
		}
	}, func(url *url.URL, err error) {
		fmt.Println(url, err.Error())
		wg.Done()
	})

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
	agents := []string{
		"Mozilla/5.0 (iPhone; U; CPU iPhone OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1",
		"Mozilla/5.0 (iPod; U; CPU iPhone OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko) Version/5.0.2 Mobile/8J2 Safari/6533.18.5",
		"Mozilla/5.0 (Linux; U; Android 2.3.7; en-us; Nexus One Build/FRF91) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
	}
	index := rand.Intn(len(agents) - 1)
	return agents[index]
}

func start(page int, keyword string, mark *chan bool) {
	c := colly.NewCollector(func(collector *colly.Collector) {
		collector.IgnoreRobotsTxt = true
		collector.Async = true
		collector.UserAgent = RandomString()
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

	go engineerSearch(url, &search, cc.Clone(), func(p int, result *SearchResult) {
		//关闭关键词搜索🔍结果log展示
		//fmt.Println(result.Page, "/t", result)
		var wg sync.WaitGroup
		wg.Add(len(result.Result.Video))

		for _, video := range result.Result.Video {
			v := video
			go open(&v, c.Clone(), &wg)
		}
		wg.Wait()
		time.Sleep(time.Second * time.Duration(page))

		go start(int(result.Page)+1, keyword, mark)
	}, func() {
		//close(*mark)
		fmt.Println("获得的🉐", keyword)

		*mark <- true

	})

	cc.Wait()
}

func recover_album(page int, limit int) []string {
	albums := []string{}

	sql := "select album_url from bbd.bbd_album_failed limit ? offset ?"
	stmt, err := bdb.GlobalDB().Prepare(sql)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		rows, err := stmt.Query(limit, limit*(page-1))
		if err != nil {
			fmt.Println(err.Error())
		} else {

			for rows.Next() {
				var url string
				rows.Scan(&url)
				albums = append(albums, url)
			}
			stmt.Close()

		}
	}
	return albums

}
func find(page int, limit int, back func(page int, a []string)) {
	result := recover_album(page, limit)
	back(page, result)
	if len(result) > 0 {
		find(page+1, limit, back)
	}
}
func Bilibili(page int, keyword string, v chan bool) {
	//v := make(chan bool)

	go start(page, keyword, &v)

	//<-v
	//close(v)
}
