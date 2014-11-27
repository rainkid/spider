package spider

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Tmall struct {
	item    *Item
	content []byte
}

func (ti *Tmall) Item() {
	url := fmt.Sprintf("http://detail.m.tmall.com/item.htm?id=%s", ti.item.id)

	//get content
	loader := NewLoader(url, "Get")
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	hp := NewHtmlParse()
	hp = hp.LoadData(content).Convert().Replace()
	ti.content = hp.content

	if ti.GetItemTitle().CheckError() {
		return
	}
	//check price
	if ti.GetItemPrice().CheckError() {
		return
	}
	if ti.GetItemImg().CheckError() {
		return
	}
	// fmt.Println(ti.item.data)
	SpiderServer.qfinish <- ti.item
}

func (ti *Tmall) GetItemTitle() *Tmall {
	hp := NewHtmlParse().LoadData(ti.content)
	title := hp.FindJsonStr("title")

	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}
	ti.item.data["title"] = fmt.Sprintf("%s", title[0][1])
	return ti
}

func (ti *Tmall) GetItemPrice() *Tmall {
	hp := NewHtmlParse().LoadData(ti.content)

	defaultPriceArr := hp.FindByAttr("b", "class", "ui-yen")
	defaultPriceStr := bytes.Replace(defaultPriceArr[0][2], []byte("&yen;"), []byte(""), -1)

	var price float64
	if bytes.Contains(defaultPriceStr, []byte("-")) {
		defaultPrices := bytes.Split(defaultPriceStr, []byte(" - "))
		price, _ = strconv.ParseFloat(fmt.Sprintf("%s", defaultPrices[0]), 64)
	} else {
		price, _ = strconv.ParseFloat(fmt.Sprintf("%s", defaultPriceStr), 64)
	}

	jsonData := hp.Partten(`{"isSuccess":true.*"serviceDO"`).FindStringSubmatch()

	if jsonData != nil {
		hp.LoadData(jsonData[0])
		prices := hp.FindJsonStr("price")

		lp := len(prices)
		if prices != nil {
			for i := 0; i < lp; i++ {
				p, _ := strconv.ParseFloat(fmt.Sprintf("%s", prices[i][1]), 64)
				if p > 0 {
					if p < price {
						price = p
					}
				}
			}
		}
	}
	ti.item.data["price"] = fmt.Sprintf("%.2f", price)
	return ti
}

func (ti *Tmall) GetItemImg() *Tmall {
	hp := NewHtmlParse().LoadData(ti.content)
	data := hp.FindByAttr("section", "id", "s-showcase")
	if data == nil {
		ti.item.err = errors.New(`get imgs error`)
		return ti
	}
	pdata := hp.LoadData(data[0][2]).Partten(`(?U)src="(.*)"`).FindStringSubmatch()
	if pdata == nil {
		ti.item.err = errors.New(`get imgs error`)
		return ti
	}
	ti.item.data["img"] = fmt.Sprintf("%s", pdata[1])
	return ti
}

func (ti *Tmall) Shop() {
	if ti.GetShopTitle().CheckError() {
		return
	}
	url := fmt.Sprintf("http://s.taobao.com/search?q=%s&app=shopsearch", ti.item.data["title"])
	//get content
	loader := NewLoader(url, "Get").WithPcAgent()
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	hp := NewHtmlParse()
	hp = hp.LoadData(content).CleanScript().Replace().Convert()
	ti.content = hp.content

	if ti.GetShopImgs().CheckError() {
		return
	}
	// fmt.Println(ti.item.data)
	SpiderServer.qfinish <- ti.item
}

func (ti *Tmall) GetShopTitle() *Tmall {
	url := fmt.Sprintf("http://shop.m.tmall.com/?shop_id=%s", ti.item.id)
	//get content
	loader := NewLoader(url, "Get")
	shop, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return ti
	}

	hp := NewHtmlParse()
	hp = hp.LoadData(shop)
	shopname := hp.FindByTagName("title")
	if shopname == nil {
		ti.item.err = errors.New("get shop title error")
		SpiderServer.qerror <- ti.item
		return ti

	}
	uid := hp.Partten(`G_msp_userId = "(.*)"`).FindStringSubmatch()
	if uid == nil {
		ti.item.err = errors.New("get shop uid error")
		SpiderServer.qerror <- ti.item
		return ti
	}
	ti.item.data["uid"] = fmt.Sprintf("%s", uid[1])
	title := bytes.Replace(shopname[0][2], []byte("-"), []byte(""), -1)
	title = bytes.Replace(title, []byte("天猫触屏版"), []byte(""), -1)
	title = bytes.Trim(title, " ")
	ti.item.data["title"] = fmt.Sprintf("%s", title)
	return ti
}

func (ti *Tmall) GetShopImgs() *Tmall {
	hp := NewHtmlParse().LoadData(ti.content)
	ret := hp.Partten(`(?U)<li class="list-item">(.*)</p> </li>`).FindAllSubmatch()
	l := len(ret)

	if l == 0 {
		ti.item.err = errors.New(`shop not found.`)
		return ti
	}
	var imgs [][][]byte
	for i := 0; i < l; i++ {
		val := ret[i][1]
		sep := []byte(fmt.Sprintf(`data-item="%s"`, ti.item.id))
		if bytes.Index(val, sep) > 0 {
			hp1 := NewHtmlParse().LoadData(val)
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

func (ti *Tmall) CheckError() bool {
	if ti.item.err != nil {
		SpiderServer.qerror <- ti.item
		return true
	}
	return false
}
