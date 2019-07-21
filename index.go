package main

import (
	"./bdb"
	"./bilibili"
	"encoding/json"
	"log"
	"math"
	"strings"
	"sync"
)

func main() {
	kinds := `越剧
评剧
河南曲剧
皮影戏
龙江剧
莆仙戏
越调
黔剧
昆曲
滇剧
闽剧
蒲剧
赣剧
桂剧
潮剧
上党梆子
淮剧
湖南花鼓戏
秦腔
吉剧
彩调
绍剧
滑稽戏
高甲戏
藏剧
高腔
傣剧
婺剧
梨园戏
河北梆子
吕剧
梆子腔
采茶戏
沪剧
山东梆子
京剧
黄梅戏
湘剧
汉剧
凤阳花鼓戏
雁剧
豫剧
二人台
粤剧
徽剧
川剧
壮剧
祁剧
晋剧`
	kids_id := `1
2
3
4
5
6
7
8
9
10
11
12
13
14
15
16
17
18
19
20
21
22
23
24
25
26
27
28
29
30
31
32
33
34
35
36
37
38
39
40
41
42
43
44
45
46
47
48
49`
	keywords := strings.Split(kinds, "\n")
	keywords_id := strings.Split(kids_id, "\n")

	//数据清洗合并
	merge_db(keywords, keywords_id)
	return
	v := make(chan bool, len(keywords))

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
					log.Println("准备清空")
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
	log.Println("爬取：", keywords, "完成")
	merge_db(keywords, keywords_id)
	bdb.GlobalDB().Close()
}

func clean_before_merge() {
	clearTopic := `DELETE a.* from bbd_topic a WHERE a.mid NOT IN (SELECT c.mid FROM (SELECT b.mid FROM bbd_topic b WHERE b.title REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+"  OR b.description REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+")  c)`
	clearAlbum := `DELETE a.* FROM bbd_album a WHERE a.aid NOT IN (SELECT c.aid FROM (SELECT b.aid FROM bbd_album b WHERE b.title REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+" OR b.origin REGEXP "(豫剧)+|(京剧)+|(秦腔)+|(曲剧)+|(晋剧)+|(评剧)+|(越剧)+|(黄梅戏)+"  ) c)`

	tResult, te := bdb.GlobalDB().Exec(clearTopic)
	if te != nil {
		log.Println(te.Error())
	} else {
		r, e := tResult.RowsAffected()
		if e != nil {
			log.Println(e.Error())
		} else {
			log.Println("清洗数据条目:", r)
		}
	}

	aResult, ae := bdb.GlobalDB().Exec(clearAlbum)
	if ae != nil {
		log.Println(ae.Error())
	} else {
		r, e := aResult.RowsAffected()
		if e != nil {
			log.Println(e.Error())
		} else {
			log.Println("清洗数据条目:", r)
		}
	}
}
func merge_db(keywords []string, keywords_id []string) {
	//清洗数据，然后进行合并
	clean_before_merge()

	var finished = 0
	merges := make(chan string, len(keywords))

	for index, keyword := range keywords {
		keyword := keyword
		id := keywords_id[index]
		go create_keyword_if_not_exists(keyword, merges, id)
	}

merge:
	for {
		select {
		case keyword := <-merges:
			finished += 1
			log.Println("完成一项:", keyword)
			if finished == len(keywords) {
				break merge
			}
		}
	}
	close(merges)
	log.Println("合并完成")
}

type Topic struct {
	Aid   int64
	UpId  int64
	Title string
	Brief string
	Img   string
}
type Page struct {
	TopicId int
	AId     int64
	PageNum int
	Part    string
}

func recover_topic(keyword string) []Topic {
	albums := []Topic{}

	sql := "SELECT t.aid, t.mid,t.title , t.description, t.pic FROM bbd_topic t WHERE t.title REGEXP \"(" + keyword + ")+\""

	stmt, err := bdb.GlobalDB().Prepare(sql)
	if err != nil {
		log.Println(err.Error())
	} else {
		rows, err := stmt.Query()
		if err != nil {
			log.Println(err.Error())
		} else {

			for rows.Next() {
				var t = Topic{}
				rows.Scan(&t.Aid, &t.UpId, &t.Title, &t.Brief, &t.Img)
				albums = append(albums, t)
			}
			stmt.Close()

		}
	}
	return albums

}
func find_topic(keyword string, back func(a []Topic)) {
	result := recover_topic(keyword)
	back(result)
	log.Println(len(result))
}
func merge_up(owner bilibili.AlbumOwner) {
	sql := `insert into ups(mid,face,name) value(?,?,?)`
	stmt, e := bdb.GlobalDB().Prepare(sql)
	if e != nil {
		log.Println(e.Error())
	} else {
		res, e := stmt.Exec(owner.Mid, owner.Face, owner.Name)
		if e != nil {
			log.Println("up主插入失败:", e.Error())
		} else {
			rid, e := res.LastInsertId()
			if e != nil {
				log.Println("up主更新失败:", e.Error())
			} else {
				log.Println("插入up主id为：", rid)
			}
		}
	}
}
func merge_topic_page(aid int64, topicId int64, categoryId int) {
	sql := `SELECT ba.origin FROM bbd_album ba WHERE ba.aid = ?`
	stmt, e := bdb.GlobalDB().Prepare(sql)
	if e != nil {
		log.Println(e.Error())
	} else {
		row := stmt.QueryRow(aid)
		var origin = ""
		row.Scan(&origin)
		var album = bilibili.Album{}
		json.Unmarshal([]byte(origin), &album)
		pages := album.AlbumContext.AlbumInfo.Pages
		stmt.Close()

		//插入up主
		merge_up(album.AlbumContext.AlbumInfo.Owner)

		//插入每一页数据
		for _, page := range pages {
			sql = "insert into topic_videos(topic_id,av,p,title,duration,category_id,cid) values(?,?,?,?,?,?,?)"
			stmt, e := bdb.GlobalDB().Prepare(sql)
			if e != nil {
				log.Println(e.Error())
			} else {
				defer stmt.Close()
				res, e := stmt.Exec(topicId, aid, page.PageNum, page.Part, page.VideoTineLength, categoryId, page.Cid)
				if e != nil {
					log.Println("主题每条列表：", e.Error())
				} else {
					r, e := res.RowsAffected()
					if e != nil {
						log.Println("主题每条列表：", e.Error())
					} else {
						log.Println("插入主题一条：", r)
					}
				}
			}

		}
	}
}
func merge_topic(keyword string, keyid int) {
	find_topic(keyword, func(a []Topic) {

		stmt, e := bdb.GlobalDB().Prepare("insert  into topics(av,title,up_id,img,description,category_id) values (?,?,?,?,?,?)")
		if e == nil {
			for _, v := range a {
				r, e := stmt.Exec(v.Aid, v.Title, v.UpId, v.Img, v.Brief, keyid)
				if e != nil {

					log.Println(e.Error())
				} else {
					lastId, e := r.LastInsertId()
					if e != nil {
						log.Println(e.Error())
					} else {
						//查找子项进行处理
						merge_topic_page(v.Aid, lastId, keyid)
						log.Println(r.RowsAffected())
					}

				}
			}
			stmt.Close()

		} else {
			log.Println(e.Error())
		}

	})

}
func create_keyword_if_not_exists(keyword string, channel chan string, catg_id string) {
	stmt, _ := bdb.GlobalDB().Prepare("insert into  categories(category_name,id) value (?,?)")
	stmt.Exec(keyword, catg_id)
	r := bdb.GlobalDB().QueryRow("select c.id from categories c where c.category_name=?", keyword)
	var keywordId = 0
	r.Scan(&keywordId)

	merge_topic(keyword, keywordId)
	channel <- keyword
}
func stepper(v chan bool, keyword string) {
	go bilibili.Bilibili(1, keyword, v)
}
