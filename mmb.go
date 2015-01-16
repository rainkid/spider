package spider

import (
	"errors"
	"fmt"
	"strconv"
)

type MMB struct {
	item    *Item
	content []byte
}

func (ti *MMB) Item() {
	url := fmt.Sprintf("http://mmb.cn/wap/touch/html/product/id_%s.htm", ti.item.id)

	//get content
	loader := NewLoader(url, "Get")
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < 3 {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	hp := NewHtmlParse()
	hp = hp.LoadData(content).Replace()
	ti.content = hp.content
	// ti.content = fmt.Sprintf("%s", content)
	//get title and check
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
	SpiderServer.qfinish <- ti.item
}

func (ti *MMB) GetItemTitle() *MMB {
	hp := NewHtmlParse().LoadData(ti.content)
	title := hp.FindByTagName("title")

	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}
	ti.item.data["title"] = fmt.Sprintf("%s", title[0][2])
	return ti
}

func (ti *MMB) GetItemPrice() *MMB {
	hp := NewHtmlParse().LoadData(ti.content)
	price := hp.Partten(`(?U)￥.*(\d{1,10}\.\d{1,2})`).FindStringSubmatch()

	if price == nil {
		price = hp.Partten(`(?U)￥<em>(.*)</em>`).FindStringSubmatch()
	}
	if price == nil {
		ti.item.err = errors.New(`get price error`)
		return ti
	}
	iprice, _ := strconv.ParseFloat(fmt.Sprintf("%s", price[1]), 64)
	ti.item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *MMB) GetItemImg() *MMB {
	hp := NewHtmlParse().LoadData(ti.content)
	img := hp.Partten(`(?U)"(http://rep.mmb.cn/wap/upload/productImage/+.*)"`).FindStringSubmatch()
	if img == nil {
		img = hp.Partten(`(?U)"(.*/wap/upload/productImage/+.*)"`).FindStringSubmatch()
	}
	if img == nil {
		ti.item.err = errors.New(`get img error`)
		return ti
	}
	ti.item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *MMB) CheckError() bool {
	if ti.item.err != nil {
		SpiderServer.qerror <- ti.item
		return true
	}
	return false
}
