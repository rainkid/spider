package spider

import (
	"errors"
	"fmt"
	"sort"
	"encoding/json"
)

type Search struct {
	content []byte
	json map[string]interface{}
	keyword string
	item_id string
}

type Row struct {
	ItemId    string
	Title     string
	RealPrice float64
	Biz       int
	SaleCount int
	FreeShipping bool
}

type rows  []Row
type RowByBiz  []Row

func (a RowByBiz) Len() int           { return len(a) }
func (a RowByBiz) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a RowByBiz) Less(i, j int) bool { return a[i].Biz > a[j].Biz }

func (ti *Search) Taobao() {
	url  := fmt.Sprintf("http://ai.taobao.com/search/index.htm?source_id=search&key=%s", ti.keyword)
	fmt.Println(url)
	//get content
	loader := NewLoader()
	content, err := loader.WithPcAgent().Send(url, "Get", nil)
	ti.content = make([]byte, len(content))
	copy(ti.content, content)
//	fmt.Println(string(content))
	if err != nil {
		return
	}

	htmlParser := NewHtmlParser()
	htmlParser.LoadData(ti.content).Convert().CleanScript().Replace()
	sub_content := htmlParser.Partten(`(?U)_pageResult\s+=\s+({.*})\;`).FindStringSubmatch()
	fmt.Println(len(sub_content))
	if len(sub_content)<2 {
		return
	}
	//json parse
	if  err := json.Unmarshal(sub_content[1], &ti.json); err != nil {
		return
	}

	data := ti.json["result"].(map[string]interface{})
	auction := data["auction"].([]interface{})

	if len(auction)<1 {
		return
	}

	rows := rows{}
	for _,val :=range auction  {
		row := val.(map[string]interface{})
		if(row["tagType"].(string)=="1"){
			continue
		}
		//排除月销量为0的商家
		if row["biz30Day"].(float64)==0 {
			continue
		}
		r  := Row{}
		r.ItemId    =fmt.Sprintf("%.0f",row["itemId"].(float64))
		r.RealPrice =row["realPrice"].(float64)

		r.SaleCount =int(row["saleCount"].(float64))
		r.Biz       =int(row["biz30Day"].(float64))
		rows = append(rows,r)
	}
	if len(rows)<1 {
		return
	}
	sort.Sort(RowByBiz(rows))
	ti.item_id = rows[0].ItemId
	return
}

func (ti *Search) CheckResponse(item *Item)(*Search, error ){

	tmp := ti.json["ret"].([]interface{})
	ret := tmp[0].(string)
	if ret =="ERRCODE_QUERY_DETAIL_FAIL::宝贝不存在" {
		item.err = errors.New(`not found`)
		item.method="delete"
		return ti,errors.New("not found");
	}
	item.method="post"
	return ti,nil;
}
