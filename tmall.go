package spider

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Tmall struct {
	content []byte
}

func (ti *Tmall) Item(item *Item) {
	url := fmt.Sprintf("http://detail.m.tmall.com/item.htm?id=%s", item.params["id"])

	//get content
	loader := NewLoader()

	content, err := loader.Send(url, "Get", nil)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).Convert().Replace()

	if ti.GetItemTitle(item).CheckError(item) {
		return
	}
	//check price
	if ti.GetItemPrice(item).CheckError(item) {
		return
	}
	if ti.GetItemImg(item).CheckError(item) {
		return
	}
	// fmt.Println(item.data)
	SpiderServer.qfinish <- item
	return
}

func (ti *Tmall) GetItemTitle(item *Item) *Tmall {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	title := htmlParser.Partten(`(?U)"title":"(.*)"`).FindStringSubmatch()

	if title == nil {
		item.err = errors.New(`get title error`)
		return ti
	}
	item.data["title"] = fmt.Sprintf("%s", title[1])
	return ti
}

func (ti *Tmall) GetItemPrice(item *Item) *Tmall {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)

	defaultPriceArr := htmlParser.FindByAttr("b", "class", "ui-yen")
	defaultPriceStr := bytes.Replace(defaultPriceArr[0][2], []byte("&yen;"), []byte(""), -1)

	var price float64
	if bytes.Contains(defaultPriceStr, []byte("-")) {
		defaultPrices := bytes.Split(defaultPriceStr, []byte(" - "))
		price, _ = strconv.ParseFloat(fmt.Sprintf("%s", defaultPrices[0]), 64)
	} else {
		price, _ = strconv.ParseFloat(fmt.Sprintf("%s", defaultPriceStr), 64)
	}

	jsonData := htmlParser.Partten(`"defaultPriceInfoDO"(.*)"detailPageTipsDO"`).FindStringSubmatch()

	if jsonData != nil {
		htmlParser.LoadData(jsonData[0])
		prices := htmlParser.FindJsonStr("price")

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
	item.data["price"] = fmt.Sprintf("%.2f", price)
	return ti
}

func (ti *Tmall) GetItemImg(item *Item) *Tmall {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	data := htmlParser.FindByAttr("section", "id", "s-showcase")
	if data == nil {
		item.err = errors.New(`get imgs error`)
		return ti
	}
	pdata := htmlParser.LoadData(data[0][2]).Partten(`(?U)src="(.*)"`).FindStringSubmatch()
	if pdata == nil {
		item.err = errors.New(`get imgs error`)
		return ti
	}
	item.data["img"] = fmt.Sprintf("%s", pdata[1])
	return ti
}

func (ti *Tmall) Shop(item *Item) {
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

func (ti *Tmall) GetShopTitle(item *Item) *Tmall {
	url := fmt.Sprintf("http://shop.m.tmall.com/?shop_id=%s", item.params["id"])
	//get content
	loader := NewLoader()

	content, err :=loader.Send(url, "Get", nil)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return ti
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)
	

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	shopname := htmlParser.FindByTagName("title")
	if shopname == nil {
		item.err = errors.New("get shop title error")
		SpiderServer.qerror <- item
		return ti

	}
	title := bytes.Replace(shopname[0][2], []byte("-"), []byte(""), -1)
	title = bytes.Replace(title, []byte("天猫触屏版"), []byte(""), -1)
	title = bytes.Trim(title, " ")
	item.data["title"] = fmt.Sprintf("%s", title)
	return ti
}

func (ti *Tmall) GetShopImgs(item *Item) *Tmall {
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
			htmlParser = htmlParser.LoadData(val)
			imgs = htmlParser.Partten(`(?U)src="(.*)"`).FindAllSubmatch()
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

func (ti *Tmall) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
