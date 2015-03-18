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
	url := fmt.Sprintf("http://mmb.cn/wap/touch/html/product/id_%s.htm", ti.item.params["id"])

	//get content
	ti.item.loader = NewLoader(url, "Get")
	content, err := ti.item.loader.Send(nil)

	if err != nil {
		ti.item.err = err
		SpiderServer.qerror <- ti.item
		return
	}

	ti.item.htmlParse.LoadData(content).Replace()
	ti.content = ti.item.htmlParse.content
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
	return
}

func (ti *MMB) GetItemTitle() *MMB {
	ti.item.htmlParse.LoadData(ti.content)
	title := ti.item.htmlParse.FindByTagName("title")

	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}
	ti.item.data["title"] = fmt.Sprintf("%s", title[0][2])
	return ti
}

func (ti *MMB) GetItemPrice() *MMB {
	ti.item.htmlParse.LoadData(ti.content)
	// fmt.Println(fmt.Sprintf("%s", hp.content))
	price := ti.item.htmlParse.Partten(`(?U)￥.*(\d+\.\d+)`).FindStringSubmatch()

	if price == nil {
		price = ti.item.htmlParse.Partten(`(?U)￥<em>(.*)</em>`).FindStringSubmatch()
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
	ti.item.htmlParse.LoadData(ti.content)
	img := ti.item.htmlParse.Partten(`(?U)data-original="(.*)"`).FindStringSubmatch()
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
