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
	item    *Item
	content []byte
	json map[string]interface{}
}

func (ti *Taobao) Item() {
	url := fmt.Sprintf("http://hws.m.taobao.com/cache/wdetail/5.0/?id=%s", ti.item.params["id"])
	//get content
	ti.item.loader = NewLoader(url, "Get")
	content, err := ti.item.loader.Send(nil)
	ti.content = content

	if err != nil {
		ti.item.err = err
		SpiderServer.qerror <- ti.item
		return
	}
	//json praise
	if  err := json.Unmarshal(content, &ti.json); err != nil {
		panic(err)
	}

	_,err = ti.CheckResponse()

	if err != nil {
		ti.item.err = err
		SpiderServer.qfinish <- ti.item
		return
	}

	if ti.GetBasicInfo().CheckError() {
		return
	}

	// fmt.Println(ti.item.data)
	SpiderServer.qfinish <- ti.item
	return
}

func (ti *Taobao) CheckResponse()(*Taobao, error ){

	tmp := ti.json["ret"].([]interface{})
	ret := tmp[0].(string)
	if ret =="ERRCODE_QUERY_DETAIL_FAIL::宝贝不存在" {
		ti.item.err = errors.New(`not found`)
		ti.item.method="delete"
		return ti,errors.New("not found");
	}
	ti.item.method="post"
	return ti,nil;
}


func (ti *Taobao) GetBasicInfo() *Taobao {

	data := ti.json["data"].(map[string]interface{})

	itemInfoModel :=data["itemInfoModel"].(map[string]interface{})
	seller :=data["seller"].(map[string]interface{})
	apiStack := data["apiStack"].([]interface {})[0].(map[string]interface {})["value"]

	var api_stack map[string]interface {}
	stack_data:= []byte(apiStack.(string))
	if  err := json.Unmarshal(stack_data, &api_stack); err != nil {
		panic(err)
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

	ti.item.data["price"]              = fmt.Sprintf("%.2f", price)
	ti.item.data["title"]              = fmt.Sprintf("%s", itemInfoModel["title"])
	ti.item.data["favcount"]           = fmt.Sprintf("%s", itemInfoModel["favcount"])
	ti.item.data["img"]                = fmt.Sprintf("%s", itemInfoModel["picsPath"].([]interface{})[0])
	ti.item.data["goodRatePercentage"] = fmt.Sprintf("%s", seller["goodRatePercentage"])
	ti.item.data["totalSoldQuantity"]  = fmt.Sprintf("%s", info["totalSoldQuantity"])

	return ti
}

func (ti *Taobao) GetItemPrice() *Taobao {
	ti.item.htmlParse.LoadData(ti.content)
	price := ti.item.htmlParse.Partten(`(?U)"rangePrice":".*","price":"(.*)"`).FindStringSubmatch()

	if price == nil {
		price = ti.item.htmlParse.Partten(`(?U)"price":"(.*)"`).FindStringSubmatch()
	}
	if price == nil {
		ti.item.err = errors.New(`get price error`)
		return ti
	}

	var iprice float64
	if bytes.Index(price[1], []byte("-")) > 0 {
		price = bytes.Split(price[1], []byte("-"))
		iprice, _ = strconv.ParseFloat(fmt.Sprintf("%s", price[0]), 64)
	} else {
		iprice, _ = strconv.ParseFloat(fmt.Sprintf("%s", price[1]), 64)
	}

	ti.item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *Taobao) Shop() {
	if ti.GetShopTitle().CheckError() {
		return
	}
	url := fmt.Sprintf("http://s.taobao.com/search?q=%s&app=shopsearch", ti.item.data["title"])
	//get content
	ti.item.loader = NewLoader(url, "Get").WithPcAgent()
	content, err := ti.item.loader.Send(nil)

	if err != nil {
		ti.item.err = err
		SpiderServer.qerror <- ti.item
		return
	}

	ti.item.htmlParse.LoadData(content).CleanScript().Replace().Convert()
	ti.content = ti.item.htmlParse.content

	if ti.GetShopImgs().CheckError() {
		return
	}
	// fmt.Println(ti.item.data)
	SpiderServer.qfinish <- ti.item
	return
}

func (ti *Taobao) GetShopTitle() *Taobao {
	url := fmt.Sprintf("http://shop%s.m.taobao.com/", ti.item.params["id"])
	//get content
	ti.item.loader = NewLoader(url, "Get")
	shop, err := ti.item.loader.Send(nil)

	if err != nil {
		ti.item.err = err
		SpiderServer.qerror <- ti.item
		return ti
	}

	ti.item.htmlParse.LoadData(shop).Replace()
	shopname := ti.item.htmlParse.FindByTagName("title")

	if shopname == nil {
		ti.item.err = errors.New("get shop title error")
		SpiderServer.qerror <- ti.item
		return ti

	}
//	uid := hp.Partten(`G_msp_userId="(\d+)"`).FindStringSubmatch()
//	if uid == nil {
//		ti.item.err = errors.New("get shop uid error")
//		SpiderServer.qerror <- ti.item
//		return ti
//	}
//	ti.item.data["uid"] = fmt.Sprintf("%s", uid[1])
	title := bytes.Replace(shopname[0][2], []byte("首页"), []byte(""), -1)
	title = bytes.Replace(title, []byte("淘宝网"), []byte(""), -1)
	title = bytes.Replace(title, []byte("天猫"), []byte(""), -1)
	title = bytes.Replace(title, []byte("Tmall.com"), []byte(""), -1)
	title = bytes.Replace(title, []byte("-"), []byte(" "), -1)
	title = bytes.Trim(title, " ")
	ti.item.data["title"] = fmt.Sprintf("%s", title)
	return ti
}

func (ti *Taobao) GetShopImgs() *Taobao {
	ti.item.htmlParse.LoadData(ti.content)
	ret := ti.item.htmlParse.Partten(`(?U)<li class="list-item">(.*)</p> </li>`).FindAllSubmatch()
	l := len(ret)

	if l == 0 {
		ti.item.err = errors.New(`shop not found.`)
		return ti
	}
	var imgs [][][]byte
	for i := 0; i < l; i++ {
		val := ret[i][1]
		sep := []byte(fmt.Sprintf(`data-item="%s"`, ti.item.params["id"]))
		if bytes.Index(val, sep) > 0 {
			hp1 := ti.item.htmlParse.LoadData(val)
			imgs = hp1.Partten(`(?U)src="(.*)"`).FindAllSubmatch()
		}

	}
	imgl := len(imgs)
	if imgl == 0 {
		ti.item.err = errors.New(`get shop imgs error`)
		return ti
	}

	var imglist []string
	if imgl > 4 {
		imgl = 4
	}
	for i := 1; i < imgl; i++ {
		imglist = append(imglist, fmt.Sprintf("%s", imgs[i][1]))
	}
	ti.item.data["img"] = fmt.Sprintf("%s", imgs[0][1])
	ti.item.data["imgs"] = strings.Join(imglist, ",")
	return ti
}

func (ti *Taobao) SameStyle() {
	var result []map[string]string
	url := fmt.Sprintf("http://s.taobao.com/list?tab=all&sort=sale-desc&type=samestyle&uniqpid=-%s&app=i2i&nid=%s", ti.item.params["pid"], ti.item.params["id"])
	ti.item.loader = NewLoader(url, "Get").WithPcAgent().WithProxy(false)
	content, err := ti.item.loader.Send(nil)

	if err != nil {
		ti.item.err = err
		SpiderServer.qerror <- ti.item
		return
	}

	hp := ti.item.htmlParse.LoadData(content).Replace().Convert()
	ret := hp.FindByAttr("div", "class", "row item icon-datalink")

	l := len(ret) - 1
	if l <= 0 {
		ti.item.err = errors.New(`Can't found samestyle goods`)
		SpiderServer.qerror <- ti.item
		return
	}
	var (
		totalPrice      float64 = 0
		totalCount      float64 = 0
		avgPrice        float64 = 0
		uniquePricesArr []float64
		pricesMap       map[float64]bool = make(map[float64]bool)
	)
	prices := hp.Partten(`(?U)<i>￥</i>(.*)</span>`).FindAllSubmatch()
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
		val := ret[i][1]
		hp1 := ti.item.htmlParse.LoadData(val)

		id := hp1.Partten(`(?U)data-item="(\d+)"`).FindStringSubmatch()
		data["id"] = fmt.Sprintf("%s", id[1])

		score := hp1.Partten(`(?U)<span class="feature-dsr-num">(.*)</span>`).FindStringSubmatch()
		if score != nil {
			data["score"] = fmt.Sprintf("%s", score[1])
		}
		//评分低于4.8分的
		p1, _ := strconv.ParseFloat(data["score"], 64)
		if p1 < 4.8 {
			// SpiderLoger.D(data["id"], "score lesslen 4.8")
			continue
		}

		pay_num := hp1.Partten(`(?U)(\d+) 人付款`).FindStringSubmatch()
		if pay_num != nil {
			data["pay_num"] = fmt.Sprintf("%s", pay_num[1])
		}
		//销量低于3件
		p3, _ := strconv.ParseFloat(data["pay_num"], 64)
		if p3 < 5 {
			// SpiderLoger.D(data["id"], "pay_num len 5")
			continue
		}

		price := hp1.Partten(`(?U)<i>￥</i>(.*)</span>`).FindStringSubmatch()
		data["price"] = fmt.Sprintf("%s", price[1])
		//价格低于平均价格30%
		p2, _ := strconv.ParseFloat(data["price"], 64)
		if p2 < avgPrice*0.3 {
			// SpiderLoger.D(data["id"], "price len aveprice off 30%")
			continue
		}
		//价格按低到高，加分10递减
		pos := sort.SearchFloat64s(uniquePricesArr, p2)
		sortScore += (10 - pos)

		imgs := hp1.Partten(`(?U)data-ks-lazyload="(.*)"`).FindStringSubmatch()
		if imgs != nil {
			data["img"] = fmt.Sprintf("%s", imgs[1])
		}

		title := hp1.Partten(`(?U)title="(.*)">`).FindStringSubmatch()
		if title != nil {
			data["title"] = fmt.Sprintf("%s", title[1])
		}

		area := hp1.Partten(`(?U)<div class="seller-loc">(.*)</div>`).FindStringSubmatch()
		if area != nil {
			data["area"] = fmt.Sprintf("%s", area[1])
		}

		istmall := bytes.Index(val, []byte(`icon-service-tianmao-large`))
		if istmall > 0 {
			data["channel"] = "tmall"
			sortScore += 1
		}

		shop_title := hp1.Partten(`(?U)<a class="feature-dsc-tgr popup-tgr" trace="srpwwnick" target="_blank" href=".*"> (.*) </a>`).FindStringSubmatch()
		if shop_title != nil {
			data["shop_title"] = fmt.Sprintf("%s", shop_title[1])
		}

		shop_level := hp1.Partten(`(?U)<span class="icon-supple-level-(.*)"></span>`).FindAllSubmatch()
		if shop_level != nil {
			data["shop_level"] = fmt.Sprintf("%d-%s", len(shop_level), shop_level[0][1])
		}

		express := hp1.Partten(`(?U)<div class="shipping">(.*)</div>`).FindStringSubmatch()
		if express != nil {
			data["express"] = fmt.Sprintf("%s", express[1])
		}

		comment_num := hp1.Partten(`(?U)(\d+) 条评论`).FindStringSubmatch()
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
		ti.item.err = errors.New(fmt.Sprintf("%d result load and %d result matched", l, len(result)))
		SpiderServer.qerror <- ti.item
		return
	}
	ti.item.data["unipid"] = ti.item.params["pid"]
	ti.item.data["nid"] = ti.item.params["id"]
	ti.item.data["list"] = result
	SpiderServer.qfinish <- ti.item
	return
}

func (ti *Taobao) CheckError() bool {
	if ti.item.err != nil {
		SpiderServer.qerror <- ti.item
		return true
	}
	return false
}
