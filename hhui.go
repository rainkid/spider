package spider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
	"math/rand"
)

type Hhui struct {
}

//	京东，淘宝，1号店，苏宁，国美，亚马逊

func (h *Hhui) Item(item *Item) {

	self := Sense{
		Channel: item.params["channel"],
		Title:   item.params["title"],
		ItemId:  item.params["id"],
	}
	self.getItemUrl()

	if self.ItemUrl == "" {
		item.err = errors.New("get sense url error")
		SpiderServer.qerror <- item
		return
	}

	//get content
	//	item_url := "http://item.jd.com/1510479.html"
	//	title := "创维(Skyworth) 42E5ERS 42英寸 高清LED窄边平板液晶电视(银色)"
	SenseUrl := GetSenseUrl(self.ItemUrl, self.Title)
	_, content, err := NewLoader().WithProxy().Get(SenseUrl)
	if err != nil {
		item.err = errors.New("get sense content error")
		SpiderServer.qerror <- item
		return
	}

	//	解析json
	var data_json map[string]interface{}
	if err := json.Unmarshal(content, &data_json); err != nil {
		item.err = errors.New("parse sense content error")
		SpiderServer.qerror <- item
		return
	}
	//	判断状态
	if data_json["thisItem"] != nil {
		thisItem := data_json["thisItem"].(map[string]interface{})
		if thisItem["price"] != nil {
			self.Price = fmt.Sprintf("%.2f", thisItem["price"].(float64))
		}
	}
	//	判断状态
	if data_json["priceHistoryData"] != nil {
		priceHistory := data_json["priceHistoryData"].(map[string]interface{})["list"].([]interface{})
		self.GetHistoryPrice(priceHistory)
	}


	result := []Sense{}
	result = append(result, self)

	list := data_json["urlPriceList"].([]interface{})
	for _, val := range list {
		row := val.(map[string]interface{})
		item := row["items"].([]interface{})[0].(map[string]interface{})
		s := Sense{Title: item["name"].(string), Price: item["price"].(string), ItemUrl: item["url"].(string)}
		s.GetChannelBySite(row["site"].(string))
		if s.Channel == "" {
			continue
		}
		s.GetItemID(item["url"].(string))
		result = append(result, s)

	}
	item.data["data"] = result

	SpiderServer.qfinish <- item
	return
}

func GetSenseUrl(item_url string, title string) string {
	url_query := url.QueryEscape(item_url)
	m := Encrypt(url_query, 2, true)
	title_map := []string{"t=" + title, "k=lxsx", "d=ls"}
	title_param_str := strings.Join(title_map, "^&")
	k := Encrypt(title_param_str, 4, false)

	ExtensionId :=GetExtensionId()
	parameters := url.Values{}
	parameters.Add("av", "3.0")
	parameters.Add("vendor", "chromenew")
	parameters.Add("browser", "chrome")
	parameters.Add("version", "4.2.9.1")
	parameters.Add("extensionid", ExtensionId)
	parameters.Add("m", m)
	parameters.Add("k", k)
	parameters.Add("t", fmt.Sprintf("%d", time.Now().UnixNano()))

	var Url *url.URL
	Url, _ = url.Parse("http://zhushou.huihui.cn/productSense")
	Url.RawQuery = parameters.Encode()
	return Url.String()
}


func GetExtensionId() string {
	s4 := func() string {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		str := ToHex(r.Intn(rand.Int()), 2)
		return str[1:5]

	}
	s := s4() + s4() + "-" + s4() + "-" + s4() + "-" + s4() + "-" + s4() + s4() + s4()
	return s;
}

//字符串反转
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
//遍历msg的字符,转换成相应with位16进制,然后连接在一起
func Encrypt(msg string, with int, reverse bool) string {
	var ch_arr []string
	for _, ch := range msg {
		ch_int := int(ch)
		ch_arr = append(ch_arr, ToHex(ch_int+88, with))
	}
	ret := strings.Join(ch_arr, "")
	if reverse {
		return Reverse(ret)
	}
	return ret
}
//变成对应位数的16进制
func ToHex(ch int, width int) string {
	return fmt.Sprintf("%0"+strconv.Itoa(width)+"x", ch)
}
