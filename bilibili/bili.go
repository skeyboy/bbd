package bilibili

import (
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ProxyIp struct {
	Ip                      string
	Port                    int
	IsHttps                 bool
	UpdateTime              int
	SourceUrl               string
	TimeTolive              int
	AnonymousInfo           string
	Area                    string
	InternetServiceProvider string
	Life                    string
}

var ProxyIpPool []ProxyIp

func parseXiciProxy(c *colly.Collector) (colly.ProxyFunc, error) {
	p := &ProxyIpPool
	SourceUrl := "http://www.xicidaili.com/nt/"
	// Instantiate default collector
	//c := colly.NewCollector(
	//	// MaxDepth is 2, so only the links on the scraped page
	//	// and links on those pages are visited
	//	colly.MaxDepth(1),
	//	colly.Async(true),
	//)

	// Limit the maximum parallelism to 1
	// This is necessary if the goroutines are dynamically
	// created to control the limit of simultaneous requests.
	//
	// Parallelism can be controlled also by spawning fixed
	// number of go routines.
	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 12})

	// On every a element which has href attribute call callback
	c.OnHTML("tr", func(e *colly.HTMLElement) {
		var item ProxyIp
		e.ForEach("td", func(i int, element *colly.HTMLElement) {
			t := element.Text
			switch i {
			case 1:
				item.Ip = t
				break
			case 2:
				p, n := strconv.Atoi(t)
				if n == nil {
					item.Port = p
				}
				break
			case 3:
				item.Area = t
				break
			case 4:
				//是否匿名

			case 5:
				item.IsHttps = strings.Contains(strings.ToLower(t), "https")
				break

			case 6:
				break
			case 8:
				//存活时间 分钟/小时/天
				item.Life = t
				break
			case 9:
				//验证时间

				break

			default:
				break
			}

		})
		item.SourceUrl = SourceUrl
		if len(item.Ip) > 10 && (strings.Contains(item.Life, "天") || strings.Contains(item.Life, "小时")) {
			*p = append(*p, item)
		}
	})

	// Start scraping on https://en.wikipedia.org
	c.Visit(SourceUrl)
	// Wait until threads are finished
	c.Wait()

	fmt.Println(*p)
	fmt.Println("fmt.Println(*p)----------------------------------->")

	var a []string
	for _, v := range *p {
		http := "http"
		if v.IsHttps {
			http = "https"
		}
		if v.Ip != "" && v.Port != 0 && v.IsHttps {
			s := http + "://" + v.Ip + ":" + strconv.Itoa(v.Port)
			fmt.Println(s, "\t", v)
			if len(v.Ip) > 10 {

				a = append(a, s)
			}
		}
	}

	fmt.Println("fmt.Println(*p)<-----------------------------------")

	/*
		c = colly.NewCollector(
			colly.AllowedDomains("cn.sonhoo.com"),
		)*/

	var wg sync.WaitGroup

	for _, v := range ProxyIpPool {
		wg.Add(1)

		http := "http"
		if v.IsHttps {
			http = "https"
		}
		if v.Ip != "" && v.Port != 0 && v.IsHttps {
			s := http + "://" + v.Ip + ":" + strconv.Itoa(v.Port)
			fmt.Println(s, "\t", v)
			if len(v.Ip) > 10 {

				ip, code := ProxyThorn(s, &wg)
				fmt.Println("可用IP", ip, "\t", code)
			}
		}

	}
	wg.Wait()
	rp, err := proxy.RoundRobinProxySwitcher(a...)
	/*
		if err != nil {
			log.Fatal(err)
		}
		c.SetProxyFunc(rp)
	*/

	c.SetProxyFunc(rp)
	return rp, err
}

func ProxyThorn(proxy_addr string, wg *sync.WaitGroup) (ip string, status int) {
	//访问查看ip的一个网址
	httpUrl := "http://icanhazip.com"
	proxy, err := url.Parse(proxy_addr)

	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   10,
		ResponseHeaderTimeout: time.Second * time.Duration(5),
	}
	httpClient := &http.Client{
		Timeout:   time.Second * 10,
		Transport: netTransport,
	}
	res, err := httpClient.Get(httpUrl)
	if err != nil {
		fmt.Println("错误信息：", err)
		return
	}
	defer res.Body.Close()
	defer wg.Done()
	if res.StatusCode != http.StatusOK {
		log.Println(err)
		return
	}
	c, _ := ioutil.ReadAll(res.Body)
	return string(c), res.StatusCode
}

/*
---------------------
作者：Liu-YanLin
来源：CSDN
原文：https://blog.csdn.net/qq_32502511/article/details/90044202
版权声明：本文为博主原创文章，转载请附上博文链接！
*/

/**
up主提交的所有视频
*/
func openUpSubmitVideosFrom(video *Video, c *colly.Collector, wg *sync.WaitGroup) {
	defer wg.Done()
	tmpVideo := video
	c.OnResponse(func(response *colly.Response) {
		js := string(response.Body)
		fmt.Println(video.Mid, ":up主的视频专辑:", js, tmpVideo)
	})
	c.OnError(func(response *colly.Response, e error) {
		fmt.Println("❌", e.Error())
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
			fmt.Println("专辑详情：", string(value))
		}

		//wg.Done()
		openUpSubmitVideosFrom(tmpVide, c.Clone(), wg)
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
			fmt.Println("⚠️", e.Error(), response.Request)
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
		}
		wg.Wait()
		go start(int(result.Page)+1, keyword, mark)
		//<-v
		//close(v)
	}, func() {
		//<-v
		//close(v)
		//os.Exit(0)
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
