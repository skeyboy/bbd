package main

import (
	"bbd/bilibili"
	"bbds/db"
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
	keywords := []string{"豫剧", "京剧", "昆曲", "坠子戏", "粤剧", "淮剧", "川剧", "秦腔", "沪剧", "晋剧", "汉剧", "河北梆子",
		"河南越调", "河南坠子", "湘剧", "湖南花鼓戏", "穆桂英", "沙家浜", "苏武牧羊", "小二黑", "梅艳芳"}

	v := make(chan bool, len(keywords))
	defer close(v)
	for _, value := range keywords {
		keyword := value
		go bilibili.Bilibili(1, keyword, v)

	}
	for i := 0; i < cap(v); i++ {
		<-v
	}

	db.Close()
}
