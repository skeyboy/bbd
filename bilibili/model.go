package bilibili

import "strconv"

/*
{"keyword":"yuju"
,"page":2,
"pagesize":20,
"platform":"h5",
"search_type":"all",
"main_ver":"v3",
"order":"totalrank",
"bangumi_num":3,
"movie_num":3}
*/
type Search struct {
	Keyword     string `json:"keyword"`
	Search_type string `json:"search_type"`
	Page        int    `json:"page"`
	Platform    string `json:"platform"`
	Main_ver    string `json:"main_ver"`
	Order       string `json:"order"`
	Bangumi_num int    `json:"bangumi_num"`
	Movie_num   int    `json:"movie_num"`
}
type SearchResult struct {
	Result Result  `json:"result"`
	Seid   string  `json:"seid"`
	Msg    string  `json:"msg"`
	Page   float64 `json:"page"`
}

func (s *SearchResult) IsSuccess() bool {
	return s.Msg == "success"
}

type Result struct {
	Video []Video `json:"video"`
}
type Video struct {
	Tag         string `json:"tag"`
	Id          int64  `json:"id"`
	Title       string `json:"title"`
	Mid         int64  `json:"mid"` //up主id
	Pic         string `json:"pic"`
	Description string `json:"description"`
	Arcurl      string `json:"arcurl"`
	Aid         int64  `json:"aid"`
}
type Topic struct {
	TopicData TopicData `json:"data"`
}
type TopicData struct {
	TopicVideo []TopicVideo `json:"vlist"`
	Pages      int          `json:"pages"`
	Count      int          `json:"count"`
}
type TopicVideo struct {
	Title       string `json:"title"`
	Mid         int64  `json:"mid"` //up主id
	Pic         string `json:"pic"`
	Description string `json:"description"`
	Aid         int64  `json:"aid"`
}
type Album struct {
	AlbumContext AlbumContext `json:"reduxAsyncConnect"`
}
type AlbumContext struct {
	AlbumInfo AlbumInfo `json:"videoInfo"`
}
type AlbumInfo struct {
	Aid         int64       `json:"aid"`
	Videos      int         `json:"videos"`
	Title       string      `json:"title"`
	Desc        string      `json:"desc"`
	State       int         `json:"state"`
	OriginTitle string      `json:"originTitle"`
	Pages       []AlbumPage `json:"pages"`
}
type AlbumPage struct {
}

func (v *Video) aIdString() string {
	return strconv.FormatInt(v.Aid, 10)
}
func (v *Video) mIdString() string {
	//https://space.bilibili.com/95147200
	return strconv.FormatInt(v.Mid, 10)
}

/**
对应up主的主页
*/
func (v *Video) UpHome() string {
	return "https://space.bilibili.com/" + v.mIdString()
}

/*
对应Up主提交的视频列表API默认请求地址
*/
func (v *Video) UpSubmitVideosAPI() string {
	/*
	   https://space.bilibili.com/ajax/member/getSubmitVideos?mid=331734497&pagesize=100&tid=0&page=1&keyword=&order=pubdate
	*/
	return "https://space.bilibili.com/ajax/member/getSubmitVideos?mid=" + v.mIdString() + "&pagesize=100&tid=0&page=1&keyword=&order=pubdate"
}

//视频详情
func (v *Video) VideoHome() string {
	return "https://m.bilibili.com/video/av" + v.aIdString() + ".html"
}
func (v Video) String() string {
	return "\nup主页面：" + v.UpHome() + "\n" + "up主视频列表：" + v.UpSubmitVideosAPI() + "\nup某一视频：" + v.VideoHome() + "\n"
}
