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
	v := make(chan bool)
	go bilibili.Bilibili(1, "雄兵连")
	go bilibili.Bilibili(1, "斗罗大陆")
	<-v
	close(v)
}
