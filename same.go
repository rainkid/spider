package spider

import (
// "bytes"
	"errors"
	"fmt"
	"encoding/json"
	"net/url"
	"regexp"
	"net/http"
)

type Same struct {
	content []byte
	json    []byte
	items   []Info
}

type Info struct {
	Channel     string
	ChannelName string
	ItemId      string
	Title       string
	Price       string
	Url         string
	History     []History
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

	item_url := getUrlString(item.params["channel"],item.params["id"])

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
	//	fmt.Println(string(content))

	var dat map[string]interface{}
	if err := json.Unmarshal(content, &dat); err != nil {
		item.err = errors.New("parse json  error")
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
	//		http://www.huihui.cn/proxy?direct=&sid=237&&purl=http%3A%2F%2Fitem.gome.com.cn%2FA0005322918-pop8006172148.html
	data := dat["data"].(map[string]interface{})
	other_quotes := data["other_quotes"].([]interface{})

	var sid string
	for _, value := range other_quotes {
		//解析商家信息，获取商家的请求地址
		merchant := value.(map[string]interface{})
		u, err := url.Parse(merchant["purchase_url"].(string))
		if err != nil {
			continue
		}

		m, _ := url.ParseQuery(u.RawQuery)

		if sid == "" {
			same_id, ok := m["sid"]
			if !ok {
				continue
			}
			sid = same_id[0]
		}

		purl, ok := m["purl"]
		if !ok {
			continue
		}
		info := Info{}
		//获取商品id
		err = info.getItemId(purl[0], merchant["merchant_name"].(string))
		if err != nil {
			fmt.Println("get channel error")
			continue
		}

		err = info.parseData()
		if err != nil {
			fmt.Println("parse data channel error")
			continue
		}

		info.Price = merchant["price"].(string)
		//		fmt.Println(info)
		s.items = append(s.items, info)
	}
	item.data["data"] = s.items
	item.data["sid"] = sid

	SpiderServer.qfinish <- item
	return
}

func getUrlString(channel_name string,item_id string) string {

	 detail_urls :=map[string]string{
		 "jd":"http://m.jd.com/product/%s.html",
		 "gome":"http://item.gome.com.cn/%s.html",
		 "yhd":"http://www.yihaodian.com/item/%s",
		 "taobao":"https://item.taobao.com/item.htm?id=%s",
		 "tmall":"http://a.m.tmall.com/i%s.htm",
		 "suning":"http://product.suning.com/%s.html",
		 "amzon":"http://www.amazon.cn/gp/aw/d/%s",
	 }

	detail_url,ok:=detail_urls[channel_name]
	if !ok{
		return ""
	}

	detail_url = fmt.Sprintf(detail_url,item_id)
	full_detail_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(detail_url)
	return full_detail_url
}

func (i *Info)getItemId(mUrl string, channel_name string) (error) {
	// 易迅商城 国美在线 1号店 苏宁易购 天猫 淘宝网
	//	京东，淘宝，1号店，苏宁，国美，亚马逊
	i.Channel = channel_name
	var getGoodsId = func(pattern string) string {
		regex, _ := regexp.Compile(pattern)
		id := regex.FindStringSubmatch(mUrl)
		return id[1]
	}

	switch channel_name {
	case "京东商城":
		i.ChannelName = "jd"
		i.ItemId = getGoodsId(`(\d+).html`)
		break
	case "国美在线":
		return errors.New("not support")
		i.ChannelName = "gome"
		i.ItemId = getGoodsId(`([\w-]+).html`)
		break
	case "苏宁易购":
		i.ChannelName = "suning"
		resp, err := http.Head(mUrl)
		if err != nil {
			fmt.Println("err:", err)
		}
		defer resp.Body.Close()
		mUrl = fmt.Sprintf("%s", resp.Request.URL)
		i.ItemId = getGoodsId(`(\d+).html`)
		break
	case "1号店":
		i.ChannelName = "yhd"
		i.ItemId = getGoodsId(`(\d+)$`)
		break
	case "天猫":
		i.ChannelName = "tmall"
		i.ItemId = getGoodsId(`i(\d+).htm`)
		break
	case "亚马逊":
		return errors.New("not support")
		i.ChannelName = "amazon"
		i.ItemId = getGoodsId(`\/d\/(\w+)`)
		break
	case "淘宝网":
		i.ChannelName = "taobao"
		break
	default:
		return errors.New("not support")
	}

	full_url := "http://app.huihui.cn/price_info.json?product_url=" + url.QueryEscape(mUrl)
	i.Url = full_url
	return nil
}
//根据平台的URL获取相应的历史价格
func (i *Info)parseData() error {

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
	//	fmt.Println(data["title"].(string))
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
