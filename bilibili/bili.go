package bilibili

import (
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"net/url"
	"regexp"
	"sync"
)

func open(video *Video, c *colly.Collector, wg *sync.WaitGroup) {
	c.OnHTML("html", func(element *colly.HTMLElement) {
		result := regexp.MustCompile("video_url: '(.*?)'").FindAll([]byte(element.Text), -1)
		for _, value := range result {
			fmt.Println(string(value))
		}
		wg.Done()
	})
	c.OnError(func(response *colly.Response, e error) {
		fmt.Println(e.Error())
		wg.Done()
	})
	c.Visit(video.Arcurl)
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
	cc := c.Clone()

	cc.OnResponse(func(response *colly.Response) {
		fmt.Print(response)
	})
	search := Search{Keyword: url.QueryEscape(keyword), Order: "totalrank", Main_ver: "v3", Page: page, Bangumi_num: 3, Movie_num: 3}
	url := "https://m.bilibili.com/search/searchengine"
	//v := make(chan bool)

	go engineerSearch(url, &search, cc.Clone(), func(p int, result *SearchResult) {
		fmt.Println(result.Page, "/t", result)
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
func Bilibili(page int, keyword string) {
	v := make(chan bool)
	go start(page, keyword, &v)
	<-v
	close(v)
}
