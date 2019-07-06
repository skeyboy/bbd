package main

import (
	"./bilibili"
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
	keywords := []string{"豫剧", "京剧"}

	v := make(chan bool, len(keywords))
	defer close(v)
	for _, value := range keywords {
		keyword := value
		go bilibili.Bilibili(1, keyword, v)

	}
	for i := 0; i < cap(v); i++ {
		<-v
	}
}
