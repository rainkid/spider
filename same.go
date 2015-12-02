package spider

import (
	// "bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

type Same struct {
	content []byte
	json    []byte
	items   []Info
}

type Info struct {
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

//	京东，淘宝，1号店，苏宁，国美，亚马逊
//	https://detail.m.tmall.com/item.htm?id=523130215596
//	http://item.m.jd.com/ware/view.action?wareId=1722509764
//	http://item.gome.com.cn/A0005322918-pop8006172148.html
//	http://item.yhd.com/item/34188166
//	http://m.suning.com/product/120956951.html
//	http://www.amazon.cn/gp/aw/d/b00yocbi6k
//	http://app.huihui.cn/price_info.json?product_url=http%3A%2F%2Fitem.jd.com%2F1510479.html

func (s *Same) Item(item *Item) {

	self := Info{
		Channel:item.params["channel"],
		ItemId:item.params["id"],
	}
	self.getItemUrl()

	if self.ItemUrl == "" {
		item.err = errors.New("get item url error")
		SpiderServer.qerror <- item
		return
	}

	//get content
	loader := NewLoader()
	hui_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(self.ItemUrl)
	content, err := loader.Send(hui_url, "Get", nil)
	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}

	//	解析json
	var data_json map[string]interface{}
	if err := json.Unmarshal(content, &data_json); err != nil {
		item.err = errors.New(fmt.Sprintf("parse json error [%s]", hui_url))
		SpiderServer.qerror <- item
		return
	}
	//	判断状态
	succ := data_json["status"].(string)
	if succ != "succ" {
		item.err = errors.New("request error")
		SpiderServer.qerror <- item
		return
	}
	//  获取标题
	data_json = data_json["data"].(map[string]interface{})
	var title string
	if data_json["title"] != nil {
		title = data_json["title"].(string)
	}

	self.parseData()
	if len(self.History)>1 {
		self.Price=self.History[len(self.History)-1].Price
	}

	if self.Price!=""{
		s.items = append(s.items, self)
	}

	other_quotes := data_json["other_quotes"].([]interface{})
	if len(other_quotes) == 0 {
		item.data["data"] = s.items
		SpiderServer.qfinish <- item
		return
	}

	//解析商家信息，获取商家的请求地址
	for _, value := range other_quotes {
		merchant := value.(map[string]interface{})

		info := Info{}
		//获取商品平台
		info.getChannel(merchant["merchant_name"].(string))
		//未获取以及当前平台则跳过
		if info.Channel==""||info.Channel==self.Channel{
			continue
		}
		info.Price = merchant["price"].(string)
		u, err := url.Parse(merchant["purchase_url"].(string))
		if err != nil {
			continue
		}

		info.Title = title
		m, _ := url.ParseQuery(u.RawQuery)
		item_url := merchant["purchase_url"].(string)
		purl, ok := m["purl"]
		if ok {
			item_url = purl[0]
		}
		// 苏宁搜索
		if info.Channel=="suning"{
			resp, err := http.Head(item_url)
			if err != nil {
				continue
			}
			defer resp.Body.Close()
			item_url = fmt.Sprintf("%s", resp.Request.URL)
		}

		//
		if info.Channel=="amazon"{
			purl, ok := m["location"]
			if ok {
				item_url = purl[0]
			}
		}
		info.getItemId(item_url)
		// 爱淘宝搜素
		if info.Channel=="taobao" && info.ItemId==""{
			ts := &Search{}
			ts.url = item_url
			ts.price = self.Price
			ts.Taobao()
			if ts.item_id==""{
				continue
			}
			info.ItemId = ts.item_id
		}

		info.getItemUrl()
		err = info.parseData()
		if err != nil {
			errors.New("parse data channel error")
			continue
		}
		item_price,_ := strconv.ParseFloat(info.Price,64)
		if len(info.History)>1 && item_price==0{
			info.Price=info.History[len(info.History)-1].Price
		}

		if info.Price==""{
			continue
		}

		s.items = append(s.items, info)
	}
	item.data["data"] = s.items

	SpiderServer.qfinish <- item
	return
}

func (i *Info)getItemUrl() {

	item_urls := map[string]string{
		"jd":     "http://m.jd.com/product/%s.html",
		"yhd":    "http://www.yihaodian.com/item/%s",
		"gome":   "http://item.gome.com.cn/%s.html",
		"tmall":  "http://a.m.tmall.com/i%s.htm",
		"taobao": "https://item.taobao.com/item.htm?id=%s",
		"suning": "http://product.suning.com/%s.html",
		"amazon": "http://www.amazon.cn/gp/aw/d/%s",
	}

	item_url, ok := item_urls[i.Channel]
	if !ok {
		return
	}

	item_url = fmt.Sprintf(item_url, i.ItemId)
	i.ItemUrl = item_url
	return
}

func (i *Info)getChannel(channel_name string) {

	 channels := map[string]string{
		"京东商城":"jd",
		"淘宝网":"taobao",
		"国美在线":"gome",
		"苏宁易购":"suning",
		"1号店":"yhd",
		"天猫":"tmall",
		"亚马逊":"amazon",
	}

	channel, ok := channels[channel_name]
	if !ok {
		return
	}
	i.Channel=channel
	return
}
func (i *Info)getItemId(detail_url string) {
	patterns:= map[string]string{
		"jd":`(\d+).html`,
		"gome":`([\w]+)-.*.html`,
		"suning":`(\d+).html`,
		"yhd":`item/(\d+)`,
		"tmall":`i(\d+).htm`,
		"taobao":`id=(\d+)`,
		"amazon":`\/d\/(\w+)`,
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
	i.ItemId=id[1]
	return
}



func (i *Info) execute(mUrl string, channel_name string) {
	// 易迅商城 国美在线 1号店 苏宁易购 天猫 淘宝网
	//	京东，淘宝，1号店，苏宁，国美，亚马逊


}

//根据平台的URL获取相应的历史价格
func (i *Info) parseData() error {

	loader := NewLoader()
	full_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(i.ItemUrl)
	content, err := loader.Send(full_url, "Get", nil)
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

func (ti *Same) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
