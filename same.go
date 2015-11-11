package spider

import (
	// "bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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
	Url     string
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

	detail_url := getUrlString(item.params["channel"], item.params["id"])
	item_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(detail_url)

	if item_url == "" {
		item.err = errors.New("get item url error")
		SpiderServer.qerror <- item
		return
	}

	//get content
	loader := NewLoader()

	content, err := loader.Send(item_url, "Get", nil)
	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}

	//	errors.New(string(content))
	var dat map[string]interface{}
	if err := json.Unmarshal(content, &dat); err != nil {
		item.err = errors.New(fmt.Sprintf("parse json  error [%s]", item_url))
		SpiderServer.qerror <- item
		return
	}
	//	为了使用解码 map 中的值，我们需要将他们进行适当的类型转换。例如这里我们将 num 的值转换成 float64类型。
	succ := dat["status"].(string)
	if succ != "succ" {
		item.err = errors.New("request error")
		SpiderServer.qerror <- item
		return
	}
	//	访问嵌套的值需要一系列的转化。
	//http://www.huihui.cn/proxy?direct=&sid=237&&purl=http%3A%2F%2Fitem.gome.com.cn%2FA0005322918-pop8006172148.html
	data := dat["data"].(map[string]interface{})
	title := data["title"].(string)

	self := Info{}
	self.Channel = item.params["channel"]
	self.ItemId = item.params["id"]
	self.Url = item_url
	self.parseData()
	s.items = append(s.items, self)

	other_quotes := data["other_quotes"].([]interface{})
	if len(other_quotes) == 0 {
		item.data["data"] = s.items
		SpiderServer.qfinish <- item
		return
	}

	fmt.Println(len(other_quotes))

	//解析商家信息，获取商家的请求地址
	for _, value := range other_quotes {
		merchant := value.(map[string]interface{})
		u, err := url.Parse(merchant["purchase_url"].(string))
		if err != nil {
			continue
		}

		m, _ := url.ParseQuery(u.RawQuery)

		purl, ok := m["purl"]
		item_url = merchant["purchase_url"].(string)
		if ok {
			item_url = purl[0]
		}
		info := Info{}

		info.Title = title
		//获取商品id
		err = info.getItemId(item_url, merchant["merchant_name"].(string))
		if err != nil {
			errors.New("get channel error")
			continue
		}
		if info.Channel == item.params["channel"] {
			continue
		}
		err = info.parseData()
		if err != nil {
			errors.New("parse data channel error")
			continue
		}

		info.Price = merchant["price"].(string)
		s.items = append(s.items, info)
	}
	item.data["data"] = s.items

	SpiderServer.qfinish <- item
	return
}

func getUrlString(channel_name string, item_id string) string {

	detail_urls := map[string]string{
		"jd":     "http://m.jd.com/product/%s.html",
		"yhd":    "http://www.yihaodian.com/item/%s",
		"gome":   "http://item.gome.com.cn/%s.html",
		"tmall":  "http://a.m.tmall.com/i%s.htm",
		"taobao": "https://item.taobao.com/item.htm?id=%s",
		"suning": "http://product.suning.com/%s.html",
		"amazon": "http://www.amazon.cn/gp/aw/d/%s",
	}

	detail_url, ok := detail_urls[channel_name]
	if !ok {
		return ""
	}

	detail_url = fmt.Sprintf(detail_url, item_id)
	return detail_url
}

func (i *Info) getItemId(mUrl string, channel_name string) error {
	// 易迅商城 国美在线 1号店 苏宁易购 天猫 淘宝网
	//	京东，淘宝，1号店，苏宁，国美，亚马逊
	var getGoodsId = func(pattern string) string {
		regex, _ := regexp.Compile(pattern)
		id := regex.FindStringSubmatch(mUrl)
		if id == nil {
			return ""
		}
		return id[0]
	}

	switch channel_name {
	case "京东商城":
		i.Channel = "jd"
		i.ItemId = getGoodsId(`(\d+).html`)
		break
	case "国美在线":
		return errors.New("gome not support")
		i.Channel = "gome"
		i.ItemId = getGoodsId(`([\w-]+).html`)
		break
	case "苏宁易购":
		i.Channel = "suning"
		resp, err := http.Head(mUrl)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		mUrl = fmt.Sprintf("%s", resp.Request.URL)
		fmt.Println(mUrl)
		i.ItemId = getGoodsId(`(\d+).html`)
		break
	case "1号店":
		i.Channel = "yhd"
		i.ItemId = getGoodsId(`(\d+)$`)
		break
	case "天猫":
		i.Channel = "tmall"
		i.ItemId = getGoodsId(`i(\d+).htm`)
		break
	case "亚马逊":
		return errors.New("amazon not support")
		i.Channel = "amazon"
		i.ItemId = getGoodsId(`\/d\/(\w+)`)
		break
	case "淘宝网":
		s := &Search{}
		s.keyword = i.Title
		s.Taobao()
		i.Channel = "taobao"
		mUrl = getUrlString("taobao", s.item_id)
		break
	default:
		return errors.New("not support")
	}
	if i.ItemId == "" {
		return errors.New("get item error")
	}
	full_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(mUrl)
	i.Url = full_url
	return nil
}

//根据平台的URL获取相应的历史价格
func (i *Info) parseData() error {

	loader := NewLoader()

	content, err := loader.Send(i.Url, "Get", nil)
	if err != nil {
		return errors.New("request error")
	}

	var dat map[string]interface{}
	if err := json.Unmarshal(content, &dat); err != nil {
		return errors.New("parse json error")
	}
	//	为了使用解码 map 中的值，我们需要将他们进行适当的类型转换。例如这里我们将 num 的值转换成 float64类型。
	status := dat["status"].(string)
	//判断状态值 succ的时候为成功，其他情况下视为错误
	if status != "succ" {
		return errors.New("get fail")
	}
	data := dat["data"].(map[string]interface{})
	title, ok := data["title"]
	if ok {
		i.Title = title.(string)
	}
	i.Title = data["title"].(string)
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
