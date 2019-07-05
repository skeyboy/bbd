/*
@author 如梦一般
@date 2019-07-05 14:55
*/
package up

import "sync"

type Topic struct {
	Tilte       string
	Description string
	Pic         string
	Aid         int //视频id
	Mid         int //up主id
	Created     int
	/*
		up上传的一个视频记录合辑，里面可能会有多个视频
			https://m.bilibili.com/video/av54886554.html
	*/
	AvURL  string
	lock   sync.RWMutex
	Videos []Video
}

func (topic *Topic) Add(video Video) {
	topic.lock.Lock()
	defer topic.lock.Unlock()
	topic.Videos = append(topic.Videos, video)
}

func (topic *Topic) FullURL() string {
	return "https://m.bilibili.com/video/av" + string(topic.Aid) + ".html"
}
func (topic *Topic) String() string {
	return "title:" + topic.Tilte + "\t" + topic.FullURL()
}

type Video struct {
}

type TopicPool struct {
	lock   sync.RWMutex
	topics map[string][]Topic // up主id =》 topic
}

var topicPool = TopicPool{topics: make(map[string][]Topic)}

func NewTopicPool() *TopicPool {
	return &topicPool
}

//添加不成功则自动
func (topicPool *TopicPool) Add(upid UpId, topic Topic) bool {
	topicPool.lock.Lock()
	has := false
	for up, t := range topicPool.topics {
		if string(up) == string(upid) {
			has = true
		}
		tt := t
		if len(tt) > 0 {
			has = true
		} else {
			has = false
		}
	}
	if has == false {
		topics := append(topicPool.topics[string(upid)], topic)
		topicPool.topics[string(upid)] = topics
	}
	topicPool.lock.Unlock()
	return !has
}
func (topicPool *TopicPool) FindTopic(upid string, aid int) (Topic, bool) {
	topicPool.lock.RLock()
	var topic Topic
	var find = false
	var topics = topicPool.topics[upid]
	if len(topics) == 0 {
		find = false
	} else {
		for _, v := range topics {
			v := v
			if v.Aid == aid {
				topic = v
			}
		}
	}

	topicPool.lock.RUnlock()
	return topic, find
}
