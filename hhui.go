package spider
import (
	"fmt"
	"strings"
	"net/url"
	"strconv"
	"time"
	"encoding/json"
	"errors"
)


type Hhui struct {

}

//	京东，淘宝，1号店，苏宁，国美，亚马逊
//	https://detail.m.tmall.com/item.htm?id=523130215596
//	http://item.m.jd.com/ware/view.action?wareId=1722509764
//	http://item.gome.com.cn/A0005322918-pop8006172148.html
//	http://item.yhd.com/item/34188166
//	http://m.suning.com/product/120956951.html
//	http://www.amazon.cn/gp/aw/d/b00yocbi6k
//	http://app.huihui.cn/price_info.json?product_url=http%3A%2F%2Fitem.jd.com%2F1510479.html

func (h *Hhui) Item(item *Item) {

	self := Sense{
		Channel:item.params["channel"],
		Title:item.params["title"],
		ItemId:item.params["id"],
	}
	self.getItemUrl()

	if self.ItemUrl == "" {
		item.err = errors.New("get item url error")
		SpiderServer.qerror <- item
		return
	}

	//get content
	//	item_url := "http://item.jd.com/1510479.html"
	//	title := "创维(Skyworth) 42E5ERS 42英寸 高清LED窄边平板液晶电视(银色)"
	SenseUrl := GetSenseUrl(self.ItemUrl, self.Title)

	loader := NewLoader()
	content, err := loader.Send(SenseUrl, "Get", nil)
	if err != nil {
		return
	}


	//	解析json
	var data_json map[string]interface{}
	if err := json.Unmarshal(content, &data_json); err != nil {
		return
	}
	//	判断状态
	if data_json["thisItem"]==nil{
		thisItem := data_json["thisItem"].(map[string]interface{})
		if thisItem["price"] !=nil{
			self.Price = fmt.Sprintf("%.2f", thisItem["price"].(float64))

		}
	}


	self.GetHistoryPrice()

	result := []Sense{}
	result = append(result, self)

	list := data_json["urlPriceList"].([]interface{})
	for _, val := range list {
		row := val.(map[string]interface{})
		item := row["items"].([]interface{})[0].(map[string]interface{})
		s := Sense{Title:item["name"].(string), Price:item["price"].(string), ItemUrl:item["url"].(string)}
		s.GetChannelBySite(row["site"].(string))
		if s.Channel==""{
			continue
		}
		s.GetItemID(item["url"].(string))
		s.GetHistoryPrice()

		result = append(result, s)

	}
	item.data["data"] = result

	SpiderServer.qfinish <- item
	return
}


func Itemx() {
	item_url := "http://item.jd.com/1510479.html"
	title := "创维(Skyworth) 42E5ERS 42英寸 高清LED窄边平板液晶电视(银色)"
	url_str := GetSenseUrl(item_url, title)

	loader := NewLoader()
	content, err := loader.Send(url_str, "Get", nil)
	if err != nil {
		return
	}

	//	解析json
	var data_json map[string]interface{}
	if err := json.Unmarshal(content, &data_json); err != nil {
		return
	}
	//	判断状态
	thisItem := data_json["thisItem"].(map[string]interface{})
	list := data_json["urlPriceList"].([]interface{})

	for _, val := range list {
		row := val.(map[string]interface{})
		item := row["items"].([]interface{})[0].(map[string]interface{})
		inf := Sense{Title:item["name"].(string), Price:item["price"].(string), ItemUrl:item["url"].(string)}
		inf.GetChannelBySite(row["site"].(string))
		inf.GetItemID(item["url"].(string))
		inf.GetHistoryPrice()
		fmt.Println(inf)
	}

	fmt.Println(thisItem["price"].(float64))
}


func GetSenseUrl(item_url string, title string) string {
	url_query := url.QueryEscape(item_url)
	m := Encrypt(url_query, 2, true)
	title_map := []string{"t=" + title, "k=lxsx", "d=ls"}
	title_param_str := strings.Join(title_map, "^&")
	k := Encrypt(title_param_str, 4, false)

	parameters := url.Values{}
	parameters.Add("m", m)
	parameters.Add("k", k)
	parameters.Add("av", "3.0", )
	parameters.Add("vendor", "chrome", )
	parameters.Add("browser", "chrome")
	parameters.Add("version", "3.7.5.2", )
	parameters.Add("extensionid", "e8170cff-a3e7-039c-3865-44b1c126227e", )
	parameters.Add("t", fmt.Sprintf("%d", time.Now().UnixNano()))

	var Url *url.URL
	Url, _ = url.Parse("http://zhushou.huihui.cn/productSense")
	Url.RawQuery = parameters.Encode()
	return Url.String()
}



func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes) - 1; i < j; i, j = i + 1, j - 1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
func Encrypt(msg string, with int, reverse bool) string {
	var s_arr []string
	for _, s_chr := range msg {
		s_int := int(s_chr)
		s_arr = append(s_arr, ToHex(s_int, with))
	}
	ret := strings.Join(s_arr, "")
	if reverse {
		return Reverse(ret)
	}
	return ret
}


func ToHex(c int, width int) string {
	return fmt.Sprintf("%0" + strconv.Itoa(width) + "x", c + 88)
}
