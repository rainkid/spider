package spider

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Search struct {
	content []byte
	url     string
	item_id string
	price   string
}

type Row struct {
	ItemId       string
	Title        string
	view_price   float64
	Biz          int
	SaleCount    int
	FreeShipping bool
}

type rows []Row
type RowByBiz []Row

func (a RowByBiz) Len() int           { return len(a) }
func (a RowByBiz) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RowByBiz) Less(i, j int) bool { return a[i].Biz > a[j].Biz }

func (ti *Search) Taobao() {
	//get content
	_, content, err := NewLoader().WithPcAgent().Get(ti.url)
	ti.content = make([]byte, len(content))
	copy(ti.content, content)
	if err != nil {
		return
	}

	htmlParser := NewHtmlParser()
	htmlParser.LoadData(ti.content)
	sub_content := htmlParser.Partten(`(?U)g_page_config\s+=\s+({.*})\;`).FindStringSubmatch()

	if len(sub_content) < 2 {
		fmt.Println("get taobao search list error")
		return
	}
	var data_json map[string]interface{}
	//json parse
	if err := json.Unmarshal(sub_content[1], &data_json); err != nil {
		fmt.Println("parse taobao search json error")
		return
	}

	auction := data_json["mods"].(map[string]interface{})["itemlist"].(map[string]interface{})["data"].(map[string]interface{})["auctions"].([]interface{})

	if len(auction) < 1 {
		fmt.Println("get taobao search auction error")
		return
	}

	rows := rows{}
	for _, val := range auction {
		row := val.(map[string]interface{})
		if row["shopcard"].(map[string]interface{})["isTmall"] == "true" {
			continue
		}
		//排除月销量为0的商家
		r := Row{}
		r.ItemId = row["nid"].(string)
		view_price := row["view_price"].(string)
		view_price_float64, _ := strconv.ParseFloat(view_price, 64)
		org_price, _ := strconv.ParseFloat(ti.price, 64)
		if org_price*1.5 < view_price_float64 || org_price*0.5 > view_price_float64 {
			continue
		}
		r.view_price = view_price_float64
		biz_str := row["view_sales"].(string)
		biz := strings.Replace(biz_str, "人付款", "", -1)
		biz_num, _ := strconv.ParseFloat(biz, 64)
		r.Biz = int(biz_num)

		rows = append(rows, r)
	}
	if len(rows) < 1 {
		return
	}
	sort.Sort(RowByBiz(rows))
	ti.item_id = rows[0].ItemId
	return
}

func (ti *Search) AiTaobao() {
	//get content
	_, content, err := NewLoader().WithPcAgent().Get(ti.url)
	ti.content = make([]byte, len(content))
	copy(ti.content, content)
	if err != nil {
		return
	}

	htmlParser := NewHtmlParser()
	htmlParser.LoadData(ti.content)
	sub_content := htmlParser.Partten(`(?U)_pageResult\s+=\s+({.*})\;`).FindStringSubmatch()
	if len(sub_content) < 2 {
		return
	}
	var data_json map[string]interface{}
	//json parse
	if err := json.Unmarshal(sub_content[1], &data_json); err != nil {
		return
	}

	auction := data_json["result"].(map[string]interface{})["auction"].([]interface{})

	if len(auction) < 1 {
		return
	}

	rows := rows{}
	for _, val := range auction {
		row := val.(map[string]interface{})
		if row["tagType"].(string) == "1" {
			continue
		}
		//排除月销量为0的商家
		if row["biz30Day"].(float64) == 0 {
			continue
		}
		r := Row{}
		r.ItemId = fmt.Sprintf("%.0f", row["itemId"].(float64))
		r.view_price = row["realPrice"].(float64)
		org_price, _ := strconv.ParseFloat(ti.price, 64)
		if org_price*1.5 < r.view_price || org_price*0.5 > r.view_price {
			continue
		}

		r.SaleCount = int(row["saleCount"].(float64))
		r.Biz = int(row["biz30Day"].(float64))
		rows = append(rows, r)
	}
	if len(rows) < 1 {
		return
	}
	sort.Sort(RowByBiz(rows))
	ti.item_id = rows[0].ItemId
	return
}
