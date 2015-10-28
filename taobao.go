package spider

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"encoding/json"
)

type Taobao struct {
	content []byte
	json map[string]interface{}
}

func (ti *Taobao) Item(item *Item) {
	url := fmt.Sprintf("http://hws.m.taobao.com/cache/wdetail/5.0/?id=%s", item.params["id"])
	//get content
	loader := NewLoader()

	content, err := loader.Send(url, "Get", nil)
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	//json praise
	if  err := json.Unmarshal(ti.content, &ti.json); err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}

	_,err = ti.CheckResponse(item)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}

	if ti.GetBasicInfo(item).CheckError(item) {
		return
	}

	// fmt.Println(item.data)
	SpiderServer.qfinish <- item
	return
}

func (ti *Taobao) CheckResponse(item *Item)(*Taobao, error ){

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


func (ti *Taobao) GetBasicInfo(item *Item) *Taobao {

	data := ti.json["data"].(map[string]interface{})

	itemInfoModel :=data["itemInfoModel"].(map[string]interface{})
	seller :=data["seller"].(map[string]interface{})
	apiStack := data["apiStack"].([]interface {})[0].(map[string]interface {})["value"]

	var api_stack map[string]interface {}
	stack_data:= []byte(apiStack.(string))
	if  err := json.Unmarshal(stack_data, &api_stack); err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return ti
	}

	info := api_stack["data"].(map[string]interface {})["itemInfoModel"].(map[string]interface {})
	priceUnits := info["priceUnits"].([]interface{})[0].(map[string]interface {})
	price_byte :=[]byte(priceUnits["price"].(string))

	var price float64
	if bytes.Index(price_byte, []byte("-")) > 0 {
		price_map := bytes.Split(price_byte, []byte("-"))
		price, _ = strconv.ParseFloat(fmt.Sprintf("%s", price_map[0]), 64)
	} else {
		price, _ = strconv.ParseFloat(fmt.Sprintf("%s", price_byte), 64)
	}

	item.data["price"]              = fmt.Sprintf("%.2f", price)
	item.data["title"]              = fmt.Sprintf("%s", itemInfoModel["title"])
	item.data["favcount"]           = fmt.Sprintf("%s", itemInfoModel["favcount"])
	item.data["img"]                = fmt.Sprintf("%s", itemInfoModel["picsPath"].([]interface{})[0])
	item.data["goodRatePercentage"] = fmt.Sprintf("%s", seller["goodRatePercentage"])
	item.data["totalSoldQuantity"]  = fmt.Sprintf("%s", info["totalSoldQuantity"])

	return ti
}

func (ti *Taobao) GetItemPrice(item *Item) *Taobao {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	price := htmlParser.Partten(`(?U)"rangePrice":".*","price":"(.*)"`).FindStringSubmatch()

	if price == nil {
		price = htmlParser.Partten(`(?U)"price":"(.*)"`).FindStringSubmatch()
	}
	if price == nil {
		item.err = errors.New(`get price error`)
		return ti
	}

	var iprice float64
	if bytes.Index(price[1], []byte("-")) > 0 {
		price = bytes.Split(price[1], []byte("-"))
		iprice, _ = strconv.ParseFloat(fmt.Sprintf("%s", price[0]), 64)
	} else {
		iprice, _ = strconv.ParseFloat(fmt.Sprintf("%s", price[1]), 64)
	}

	item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *Taobao) Shop(item *Item) {
	if ti.GetShopTitle(item).CheckError(item) {
		return
	}
	url := fmt.Sprintf("http://s.taobao.com/search?q=%s&app=shopsearch", item.data["title"])
	//get content
	loader := NewLoader()

	content, err := loader.WithPcAgent().Send(url, "Get", nil)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).CleanScript().Replace().Convert()

	if ti.GetShopImgs(item).CheckError(item) {
		return
	}
	// fmt.Println(item.data)
	SpiderServer.qfinish <- item
	return
}

func (ti *Taobao) GetShopTitle(item *Item) *Taobao {
	url := fmt.Sprintf("http://shop%s.m.taobao.com/", item.params["id"])
	//get content
	loader := NewLoader()

	content, err := loader.Send(url, "Get", nil)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return ti
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()
	
	htmlParser.LoadData(ti.content).Replace()
	shopname := htmlParser.FindByTagName("title")

	if shopname == nil {
		item.err = errors.New("get shop title error")
		SpiderServer.qerror <- item
		return ti

	}
	title := bytes.Replace(shopname[0][2], []byte("首页"), []byte(""), -1)
	title = bytes.Replace(title, []byte("淘宝网"), []byte(""), -1)
	title = bytes.Replace(title, []byte("天猫"), []byte(""), -1)
	title = bytes.Replace(title, []byte("Tmall.com"), []byte(""), -1)
	title = bytes.Replace(title, []byte("-"), []byte(" "), -1)
	title = bytes.Trim(title, " ")
	item.data["title"] = fmt.Sprintf("%s", title)
	return ti
}

func (ti *Taobao) GetShopImgs(item *Item) *Taobao {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	ret := htmlParser.Partten(`(?U)<li class="list-item">(.*)</p> </li>`).FindAllSubmatch()
	l := len(ret)

	if l == 0 {
		item.err = errors.New(`shop not found.`)
		return ti
	}
	var imgs [][][]byte
	for i := 0; i < l; i++ {
		val := ret[i][1]
		sep := []byte(fmt.Sprintf(`data-item="%s"`, item.params["id"]))
		if bytes.Index(val, sep) > 0 {
			hp1 := htmlParser.LoadData(val)
			imgs = hp1.Partten(`(?U)src="(.*)"`).FindAllSubmatch()
		}

	}
	imgl := len(imgs)
	if imgl == 0 {
		item.err = errors.New(`get shop imgs error`)
		return ti
	}

	var imglist []string
	if imgl > 4 {
		imgl = 4
	}
	for i := 1; i < imgl; i++ {
		imglist = append(imglist, fmt.Sprintf("%s", imgs[i][1]))
	}
	item.data["img"] = fmt.Sprintf("%s", imgs[0][1])
	item.data["imgs"] = strings.Join(imglist, ",")
	return ti
}

func (ti *Taobao) SameStyle(item *Item) {
	var result []map[string]string
	url := fmt.Sprintf("http://s.taobao.com/list?tab=all&sort=sale-desc&type=samestyle&uniqpid=-%s&app=i2i&nid=%s", item.params["pid"], item.params["id"])

	loader := NewLoader()
	content, err := loader.WithPcAgent().WithProxy(false).Send(url, "Get", nil)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(ti.content).Replace()
	ret := hp.Partten(`(?U)"nid".*"pid_info"`).FindAllSubmatch()

	l := len(ret) - 1
	if l <= 0 {
		item.err = errors.New(`Can't found samestyle goods`)
		SpiderServer.qerror <- item
		return
	}
	var (
		totalPrice      float64 = 0
		totalCount      float64 = 0
		avgPrice        float64 = 0
		uniquePricesArr []float64
		pricesMap       map[float64]bool = make(map[float64]bool)
	)
	prices := hp.Partten(`(?U)"view_price":"(.*)"`).FindAllSubmatch()

	if len(prices) == 0 {
		return
	}
	for _, v := range prices {
		p, err := strconv.ParseFloat(string(v[1]), 64)
		if err != nil {
			continue
		}
		if pricesMap[p] != true {
			uniquePricesArr = append(uniquePricesArr, p)
			pricesMap[p] = true
		}
		totalPrice += p
		totalCount++
		if totalCount == 10 {
			break
		}
	}
	sort.Float64s(uniquePricesArr)
	//计算平均价格
	avgPrice, _ = strconv.ParseFloat(fmt.Sprintf("%.2f", totalPrice/totalCount), 64)
	lret := len(ret)
	for i := 1; i < l; i++ {
		var sortScore = lret - i
		data := map[string]string{"channel": "taobao", "comment_num": "0", "pay_num": "0", "sortScore": "0", "express": "0.00"}
		val := ret[i][0]

		htmlParser := NewHtmlParser()

		hp1 := htmlParser.LoadData(val)

		id := hp1.Partten(`(?U)"nid":"(\d+)"`).FindStringSubmatch()
		data["id"] = fmt.Sprintf("%s", id[1])

		score := hp1.Partten(`(?U)"dsr_scores":\["(.*)","(.*)","(.*)"\]`).FindStringSubmatch()
		if score != nil {
			data["score"] = fmt.Sprintf("%s", score[1])
		}
		//评分低于4.7分的
		p1, _ := strconv.ParseFloat(data["score"], 64)
		if p1 < 4.7 {
			// SpiderLoger.D(data["id"], "score lesslen 4.7")
			continue
		}

		pay_num := hp1.Partten(`(?U)"view_sales":"(\d+).*"`).FindStringSubmatch()
		if pay_num != nil {
			data["pay_num"] = fmt.Sprintf("%s", pay_num[1])
		}
		//销量低于3件
		p3, _ := strconv.ParseFloat(data["pay_num"], 64)
		if p3 < 5 {
			// SpiderLoger.D(data["id"], "pay_num len 5")
			continue
		}

		price := hp1.Partten(`(?U)"reserve_price":"(.*)"`).FindStringSubmatch()
		data["price"] = fmt.Sprintf("%s", price[1])
		//价格低于平均价格40%
		p2, _ := strconv.ParseFloat(data["price"], 64)
		if p2 < avgPrice*0.4 {
			// SpiderLoger.D(data["id"], "price len aveprice off 40%")
			continue
		}
		//价格按低到高，加分10递减
		pos := sort.SearchFloat64s(uniquePricesArr, p2)
		sortScore += (10 - pos)

		imgs := hp1.Partten(`(?U)"pic_url":"(.*)"`).FindStringSubmatch()
		if imgs != nil {
			data["img"] = fmt.Sprintf("%s", imgs[1])
		}

		title := hp1.Partten(`(?U)"title":"(.*)"`).FindStringSubmatch()
		if title != nil {
			data["title"] = fmt.Sprintf("%s", title[1])
		}

		area := hp1.Partten(`(?U)"item_loc":"(.*)"`).FindStringSubmatch()
		if area != nil {
			data["area"] = fmt.Sprintf("%s", area[1])
		}

		istmall := bytes.Index(val, []byte(`detail.tmall.com`))
		if istmall > 0 {
			data["channel"] = "tmall"
			sortScore += 1
		}

		shop_title := hp1.Partten(`(?U)"nick":"(.*)"`).FindStringSubmatch()
		if shop_title != nil {
			data["shop_title"] = fmt.Sprintf("%s", shop_title[1])
		}

		shop_level := hp1.Partten(`(?U)<span class="icon-supple-level-(.*)"></span>`).FindAllSubmatch()
		if shop_level != nil {
			data["shop_level"] = fmt.Sprintf("%d-%s", len(shop_level), shop_level[0][1])
		}

		express := hp1.Partten(`(?U)"view_fee":"(.*)"`).FindStringSubmatch()
		if express != nil {
			data["express"] = fmt.Sprintf("%s", express[1])
		}

		comment_num := hp1.Partten(`(?U)"comment_count":"(\d+)"`).FindStringSubmatch()
		if comment_num != nil {
			data["comment_num"] = fmt.Sprintf("%s", comment_num[1])
		}

		data["sortScore"] = fmt.Sprintf("%d", sortScore)
		result = append(result, data)
		if len(result) == 5 {
			break
		}
	}
	if len(result) == 0 {
		item.err = errors.New(fmt.Sprintf("%d result load and %d result matched", l, len(result)))
		SpiderServer.qerror <- item
		return
	}
	item.data["unipid"] = item.params["pid"]
	item.data["nid"]    = item.params["id"]
	item.data["list"]   = result
	SpiderServer.qfinish <- item
	return
}

func (ti *Taobao) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
