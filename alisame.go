package spider

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"net/http"
	"io/ioutil"
)

type AliSame struct {
	content []byte
	json    map[string]interface{}
}

type AliSameItem struct {
	Id           string  `json:"id"`
	Area         string  `json:"area"`
	Title        string  `json:"title"`
	Price        float64 `json:"price"`
	Score        float64 `json:"score"`
	PayNum       uint64  `json:"pay_num"`
	Img          string  `json:"img"`
	Channel      string  `json:"channel"`
	Express      float64 `json:"express"`
	SortScore    uint64  `json:"sortScore"`
	ShopTitle    string  `json:"shop_title"`
	CommentNum   uint64  `json:"comment_num"`
	ReservePrice float64 `json:"reserve_price"`
}

func (ti *AliSame) getPid(item *Item) {
	ali_search_url := fmt.Sprintf("http://s.taobao.com/search?q=%s", item.params["title"])
	resp, err := http.Get(ali_search_url)
	if err != nil {
		item.err = errors.New("load same page error")
		SpiderServer.qerror <- item
		return
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		item.err = errors.New("load same page error")
		SpiderServer.qerror <- item
		return
	}
	search_content :=make([]byte,len(body))
	copy(search_content, body)
	shp := NewHtmlParser().LoadData(search_content)

	ret := shp.Partten(`(?U)"nid":"`+item.params["id"]+`","category":"\d+","pid":"-(\d+)"`).FindStringSubmatch()
	if ret != nil && len(ret) > 0 {
		item.params["pid"] = string(ret[1])
	}
}

func (ti *AliSame) Same(item *Item) {

	if item.params["pid"] == "" || item.params["pid"] == "0" {
		ti.getPid(item)
	}
	result := []AliSameItem{}
	url := fmt.Sprintf("http://s.taobao.com/list?tab=all&sort=sale-desc&type=samestyle&uniqpid=-%s&app=i2i&nid=%s", item.params["pid"], item.params["id"])
	url = fmt.Sprintf("https://s.taobao.com/search?type=samestyle&app=i2i&rec_type=1&uniqpid=-%s&nid=%s", item.params["pid"], item.params["id"])
	_, content, err := NewLoader().WithPcAgent().Get(url)

	if err != nil {
		item.err = errors.New("load same page error")
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()
	htmlParser.LoadData(ti.content)
	sub_content := htmlParser.Partten(`(?U)g_page_config\s+=\s+({.*})\;`).FindStringSubmatch()
	if len(sub_content) < 2 {
		item.err = errors.New("get same json content error")
		SpiderServer.qerror <- item
		return
	}
	//json parse
	if err := json.Unmarshal(sub_content[1], &ti.json); err != nil {
		item.err = errors.New("parse same json error")
		SpiderServer.qerror <- item
		return
	}

	mods := ti.json["mods"].(map[string]interface{})
	//当前款
	single, ok := mods["singleauction"].(map[string]interface{})["data"]
	var current_reserve_price, current_view_price float64
	if ok {
		singleauction := single.(map[string]interface{})
		current_reserve_price, _ = strconv.ParseFloat(singleauction["reserve_price"].(string), 64)
		current_view_price, _ = strconv.ParseFloat(singleauction["view_price"].(string), 64)
	}

	data, ok := mods["recitem"].(map[string]interface{})["data"]
	if !ok {
		item.err = errors.New(`Having none same style goods`)
		SpiderServer.qerror <- item
		return
	}
	items, ok := data.(map[string]interface{})["items"]
	if !ok {
		item.err = errors.New(`Having none same style goods`)
		SpiderServer.qerror <- item
		return
	}

	rows := items.([]interface{})
	if len(rows) < 2 {
		item.err = errors.New(`Can't found same style goods`)
		SpiderServer.qerror <- item
		return
	}
	for _, row := range rows {
		v := row.(map[string]interface{})
		s := AliSameItem{}
		s.Id = v["nid"].(string)
		s.Area = v["item_loc"].(string)
		s.Title = v["title"].(string)
		s.Img = v["pic_url"].(string)
		s.Channel = "taobao"
		s.SortScore = 0
		s.ShopTitle = v["nick"].(string)

		//销量小于3的不要
		view_sales := strings.Replace(v["view_sales"].(string), "人付款", "", -1)
		s.PayNum, _ = strconv.ParseUint(view_sales, 0, 64)
		if s.PayNum < 3 {
			continue
		}
		//包邮+1
		s.Express, _ = strconv.ParseFloat(v["view_fee"].(string), 64)
		if s.Express == 0 {
			s.SortScore += 1
		}
		//原价过滤
		s.ReservePrice, _ = strconv.ParseFloat(v["reserve_price"].(string), 64)
		if s.ReservePrice > current_reserve_price * 1.5 || s.ReservePrice < current_reserve_price * 0.5 {
			continue
		}
		//现价过滤
		s.Price, _ = strconv.ParseFloat(v["view_price"].(string), 64)
		if s.Price > current_view_price * 1.5 || s.Price < current_view_price * 0.5 {
			continue
		}
		//低价+2
		if s.Price < current_view_price {
			s.SortScore += 2
		}
		//评论+1
		s.CommentNum, _ = strconv.ParseUint(v["comment_count"].(string), 0, 64)
		if s.CommentNum > 0 {
			s.SortScore += 1
		}

		//分值计算
		var sum_score float64
		dsr_scores := v["dsr_scores"].([]interface{})
		for _, score := range dsr_scores {
			f_score, _ := strconv.ParseFloat(score.(string), 64)
			if f_score < 4.60 {
				continue
			}
			sum_score += f_score
		}
		s.Score = Round(sum_score / 3, 2)
		//如果三项平均分低于4.7,排除
		if s.Score < 4.65 {
			continue
		}

		//价格按低到高，加分10递减
		istmall := strings.Contains(v["detail_url"].(string), "detail.tmall.com")
		if istmall {
			s.Channel = "tmall"
			s.SortScore += 1
		}

		result = append(result, s)
		if len(result) == 5 {
			break
		}
	}
	item.data["list"] = result
	item.data["nid"] = item.params["id"]
	item.data["unipid"] = item.params["pid"]
	SpiderServer.qfinish <- item
	return
}

func (ti *AliSame) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}

func (ti *AliSame) CheckResponse(item *Item) (*AliSame, error) {

	tmp := ti.json["ret"].([]interface{})
	ret := tmp[0].(string)
	if ret == "ERRCODE_QUERY_DETAIL_FAIL::宝贝不存在" {
		item.err = errors.New(`not found`)
		item.method = "delete"
		return ti, errors.New("not found")
	}
	item.method = "post"
	return ti, nil
}

