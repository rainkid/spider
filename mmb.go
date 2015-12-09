package spider

import (
	"errors"
	"fmt"
	"strconv"
)

type MMB struct {
	content []byte
}

func (ti *MMB) Item(item *Item) {
	url := fmt.Sprintf("http://mmb.cn/wap/touch/html/product/id_%s.htm", item.params["id"])
	_, content, err := NewLoader().WithProxy().Get(url)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}

	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).Replace()

	// ti.content = fmt.Sprintf("%s", content)
	//get title and check
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
	SpiderServer.qfinish <- item
	return
}

func (ti *MMB) GetItemTitle(item *Item) *MMB {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	title := htmlParser.FindByTagName("title")

	if title == nil {
		item.err = errors.New(`get title error`)
		return ti
	}
	item.data["title"] = fmt.Sprintf("%s", title[0][2])
	return ti
}

func (ti *MMB) GetItemPrice(item *Item) *MMB {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	// fmt.Println(fmt.Sprintf("%s", hp.content))
	price := htmlParser.Partten(`(?U)￥.*(\d+\.\d+)`).FindStringSubmatch()

	if price == nil {
		price = htmlParser.Partten(`(?U)￥<em>(.*)</em>`).FindStringSubmatch()
	}
	if price == nil {
		item.err = errors.New(`get price error`)
		return ti
	}
	iprice, _ := strconv.ParseFloat(fmt.Sprintf("%s", price[1]), 64)
	item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *MMB) GetItemImg(item *Item) *MMB {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	img := htmlParser.Partten(`(?U)data-original="(.*)"`).FindStringSubmatch()
	if img == nil {
		item.err = errors.New(`get img error`)
		return ti
	}
	item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *MMB) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
