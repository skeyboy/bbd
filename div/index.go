/*
@author 如梦一般
@date 2019-07-15 10:44
*/
package main

import (
	"fmt"
	"os"
	"time"
)

func test(v chan int, index int, value string) {
	time.Sleep(time.Second * time.Duration(index))
	v <- index
}
func main() {
	keys := []string{"1", "2", "3"}
	var sub = keys[1:3]
	fmt.Println(sub)
	var cuple = 2
	var index = 0
	v := make(chan int, cuple)

	id, value := index, keys[index]
	go test(v, id, value)
	for {
		select {
		case r := <-v:
			index = index + 1
			fmt.Println(r)
			if index < len(keys) {
				go test(v, index, keys[index])
			} else {
				os.Exit(0)
			}

		}
	}

}
