package bilibili

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
	Tag         string  `json:"tag"`
	Id          float64 `json:"id"`
	Title       string  `json:"title"`
	Mid         float64 `json:"mid"`
	Pic         string  `json:"pic"`
	Description string  `json:"description"`
	Arcurl      string  `json:"arcurl"`
}
