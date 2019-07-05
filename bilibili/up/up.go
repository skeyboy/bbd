/*
@author 如梦一般
@date 2019-07-05 14:34
*/
package up

import "sync"

//Up主信息
type Up struct {
	UpId   UpId //mid
	Author string
}

func (up *Up) UpHome() string {
	return string("https://space.bilibili.com/" + up.UpId)
}
func (up *Up) String() string {
	return "Author:\t" + up.Author + "\t" + "Home:" + up.UpHome()
}

type UpId string

type UpPool struct {
	lock  sync.RWMutex
	upMap map[Up]UpId
}

var uppool = UpPool{upMap: make(map[Up]UpId)}

func NewUpPool() *UpPool {
	return &uppool
}
func (pool *UpPool) add(up Up) bool {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	result := false
	for innerUp := range pool.upMap {
		tUp := innerUp
		result = tUp.UpId == up.UpId && tUp.Author == up.Author
	}
	return result
}
func (pool *UpPool) Add(upid UpId, author string, result func(upid UpId, isNew bool)) bool {

	success := pool.add(Up{UpId: upid, Author: author})
	result(upid, success)
	return success
}
