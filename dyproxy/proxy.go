package dyproxy

import (
	"fmt"
	"github.com/gocolly/colly"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var proxyPool = []DyIp{}

type DyIp struct {
	Ip             string
	Scheme         string
	Port           string
	Location       string
	Owner          string
	UpdateDateTime string
}

func (d DyIp) String() string {
	return strings.ToLower(d.Scheme) + "://" + d.Ip + ":" + d.Port
}

func (dyIp *DyIp) FullIp() string {
	return dyIp.Ip + ":" + dyIp.Port
}

func ProxyThorn(proxy_addr DyIp, wg *sync.WaitGroup, result func(d DyIp, code int)) (ip string, dyIp DyIp, status int) {
	time.Sleep(time.Second * 1)

	//访问查看ip的一个网址
	httpUrl := "http://icanhazip.com"
	httpUrl = "https://www.baidu.com"
	proxy, err := url.Parse(proxy_addr.FullIp())

	netTransport := &http.Transport{
		Proxy:                 http.ProxyURL(proxy),
		MaxIdleConnsPerHost:   5,
		ResponseHeaderTimeout: time.Second * time.Duration(50),
	}
	httpClient := &http.Client{
		Timeout:   time.Second * 30,
		Transport: netTransport,
	}
	res, err := httpClient.Get(httpUrl)
	if err != nil {
		fmt.Println("错误信息：", err)
		return
	}
	defer res.Body.Close()
	defer wg.Done()
	result(proxy_addr, res.StatusCode)
	if res.StatusCode != http.StatusOK {
		log.Println(err)
		return
	} else {
	}
	c, _ := ioutil.ReadAll(res.Body)
	msg := string(c)
	return msg, proxy_addr, res.StatusCode
}
func xic(dyIp *DyIp, i int, conent string) {
	switch i {
	case 1:
		dyIp.Ip = conent
	case 2:
		dyIp.Port = conent
	case 3:
		dyIp.Location = conent
	case 5:
		dyIp.Scheme = conent

	case 8:
		dyIp.UpdateDateTime = conent
	}
}

var ipPool = []DyIp{}

func AllProxy() []DyIp {
	if len(ipPool) != 0 {
		return ipPool
	}
	c := colly.NewCollector(func(collector *colly.Collector) {
		collector.IgnoreRobotsTxt = true
		collector.Async = true
		collector.UserAgent = "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1"
	})

	cc := c.Clone()
	var tmpPool = []DyIp{}

	cc.OnHTML("tbody", func(element *colly.HTMLElement) {
		element.ForEach("tr", func(i int, element *colly.HTMLElement) {
			dyIp := DyIp{}

			element.ForEach("td", func(i int, e *colly.HTMLElement) {
				content := strings.ReplaceAll(strings.ReplaceAll(e.Text, "\t", ""), "\n", "")
				if e.Request.URL.Host == "www.xicidaili.com" {

					xic(&dyIp, i, content)
				} else {
					switch i {
					case 0:
						dyIp.Ip = content
						break
					case 1:
						dyIp.Port = content
						break
					case 3:
						dyIp.Scheme = content
						break

					case 4:
						dyIp.Location = content
						break
					case 2:
						dyIp.Owner = content
						break
					case 6:
						dyIp.UpdateDateTime = content

					}
				}
			})

			if element.Request.URL.Host == "www.xicidaili.com" {
				//西祠
				if strings.Contains(dyIp.UpdateDateTime, "分钟") {
					tmpPool = append(tmpPool, dyIp)
				}

			} else {
				tmpPool = append(tmpPool, dyIp)
			}

		})
		fmt.Println(tmpPool)

	})
	cc.OnError(func(response *colly.Response, e error) {
		fmt.Println(e.Error())
	})

	//cc.Visit("http://www.89ip.cn")
	cc.Visit("http://www.qydaili.com/free/")
	//cc.Visit("https://www.xicidaili.com/wt/")
	cc.Wait()

	var wg sync.WaitGroup
	for _, v := range tmpPool[0 : len(tmpPool)/2+len(tmpPool)/5] {
		wg.Add(1)
		v := v
		ProxyThorn(v, &wg, func(d DyIp, code int) {
			if code == 200 {
				proxyPool = append(proxyPool, d)
			} else {
				fmt.Println("⚠️", d, code)
			}
		})

	}
	wg.Wait()
	ipPool = tmpPool

	return ipPool
}
