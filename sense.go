package spider

import (
	// "bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
)

type Sense struct {
	Channel string
	ItemId  string
	Title   string
	Price   string
	ItemUrl string
	History []History
}

type History struct {
	Price string
	Time  string
}

//func (s *Same)Item(item *Item)  {
//	self := Info{
//		ItemId:item.params["id"],
//		Title:item.params["title"],
//		Channel:item.params["channel"],
//	}
//
//	item_url := self.getItemUrl()
//	fmt.Println(item_url)
//
//}
//	京东，淘宝，1号店，苏宁，国美，亚马逊
//	https://detail.m.tmall.com/item.htm?id=523130215596
//	http://item.m.jd.com/ware/view.action?wareId=1722509764
//	http://item.gome.com.cn/A0005322918-pop8006172148.html
//	http://item.yhd.com/item/34188166
//	http://m.suning.com/product/120956951.html
//	http://www.amazon.cn/gp/aw/d/b00yocbi6k
//	http://app.huihui.cn/price_info.json?product_url=http%3A%2F%2Fitem.jd.com%2F1510479.html

func (i *Sense) getItemUrl() {

	item_urls := map[string]string{
		"jd":     "http://item.jd.com/%s.html",
		"yhd":    "http://www.yihaodian.com/item/%s",
		"gome":   "http://item.gome.com.cn/%s.html",
		"tmall":  "https://detail.tmall.com/item.htm?id=%s",
		"taobao": "https://item.taobao.com/item.htm?id=%s",
		"suning": "http://product.suning.com/%s.html",
		"amazon": "http://www.amazon.cn/mn/detailApp?asin=%s",
	}

	item_url, ok := item_urls[i.Channel]
	if !ok {
		return
	}

	item_url = fmt.Sprintf(item_url, i.ItemId)
	i.ItemUrl = item_url
	return
}

func (i *Sense) GetChannelByName(channel_name string) {

	channels := map[string]string{
		"京东商城": "jd",
		"淘宝网":  "taobao",
		"国美在线": "gome",
		"苏宁易购": "suning",
		"1号店":  "yhd",
		"天猫":   "tmall",
		"亚马逊":  "amazon",
	}

	channel, ok := channels[channel_name]
	if !ok {
		return
	}
	i.Channel = channel
	return
}
func (i *Sense) GetChannelBySite(channel_name string) {

	channels := map[string]string{
		"360buy.com":  "jd",
		"taobao.com":  "taobao",
		"gome.com.cn": "gome",
		"suning.com":  "suning",
		"yhd.com":     "yhd",
		"tmall.com":   "tmall",
		"amazon.cn":   "amazon",
	}

	channel, ok := channels[channel_name]
	if !ok {
		return
	}
	i.Channel = channel
	return
}
func (i *Sense) GetItemID(detail_url string) {
	patterns := map[string]string{
		"jd":     `(\d+).html`,
		"gome":   `([\w]+)-.*.html`,
		"suning": `(\d+).html`,
		"yhd":    `item\/(\d+)`,
		"tmall":  `id\=(\d+)`,
		"taobao": `id\=(\d+)`,
		"amazon": `detailApp\?asin\=(\w+)`,
	}

	pattern, ok := patterns[i.Channel]
	if !ok {
		return
	}
	regex, _ := regexp.Compile(pattern)
	id := regex.FindStringSubmatch(detail_url)
	if id == nil {
		return
	}
	i.ItemId = id[1]
	return
}

//根据平台的URL获取相应的历史价格
func (i *Sense) GetHistoryPrice() error {
	full_url := fmt.Sprintf("http://zhushou.huihui.cn/productSense?phu=%s&type=canvas&t=1448957873849", url.QueryEscape(i.ItemUrl))
	_, content, err := NewLoader().WithProxy().Get(full_url)
	if err != nil {
		return errors.New("request error")
	}

	var json_data map[string]interface{}
	if err := json.Unmarshal(content, &json_data); err != nil {
		return errors.New("parse same json error")
	}
	if json_data["priceHistoryData"] == nil {
		return errors.New("no price History Data with this item")
	}
	//	为了使用解码 map 中的值，我们需要将他们进行适当的类型转换。例如这里我们将 num 的值转换成 float64类型。
	list := json_data["priceHistoryData"].(map[string]interface{})["list"].([]interface{})
	for _, val := range list {
		h := History{}
		row := val.(map[string]interface{})
		h.Price = fmt.Sprintf("%.2f", row["price"].(float64))
		h.Time = row["time"].(string)
		i.History = append(i.History, h)
	}
	return nil
}

//根据平台的URL获取相应的历史价格  removed
func (i *Sense) parseData() error {
	full_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(i.ItemUrl)
	_, content, err := NewLoader().WithProxy().Get(full_url)
	if err != nil {
		return errors.New("request error")
	}

	var dat map[string]interface{}
	if err := json.Unmarshal(content, &dat); err != nil {
		return errors.New("parse same json error")
	}
	//	为了使用解码 map 中的值，我们需要将他们进行适当的类型转换。例如这里我们将 num 的值转换成 float64类型。
	status := dat["status"].(string)
	//判断状态值 succ的时候为成功，其他情况下视为错误
	if status != "succ" {
		return errors.New("get fail")
	}
	data := dat["data"].(map[string]interface{})
	if data["title"] != nil {
		i.Title = data["title"].(string)
	}
	price_history := data["price_history"].([]interface{})
	for _, val := range price_history {
		h := History{}
		row := val.(map[string]interface{})
		h.Price = row["price"].(string)
		h.Time = row["time"].(string)
		i.History = append(i.History, h)
	}
	return nil
}
