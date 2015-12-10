package spider

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Jd struct {
	content []byte
}

func (ti *Jd) Item(item *Item) {
	url := fmt.Sprintf("http://item.m.jd.com/ware/view.action?wareId=%s", item.params["id"])

	//get content
	_, content, err := NewLoader().WithProxy().Get(url)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	// ti.content = bytes.Replace(ti.content, []byte(`\"`), []byte(`"`), -1)

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

func (ti *Jd) GetItemTitle(item *Item) *Jd {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content)
	title := htmlParser.Partten(`(?U)class="title-text">(.*)\s+<i`).FindStringSubmatch()

	if title == nil {
		item.err = errors.New(`get title error`)
		return ti
	}
	item.data["title"] = strings.TrimSpace(string(title[1]))
	return ti
}

func (ti *Jd) GetItemPrice(item *Item) *Jd {
	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(ti.content)
	price := hp.Partten(`(?U)&yen;(\d+\.\d+)`).FindStringSubmatch()
	if price == nil {
		item.err = errors.New(`get price error`)
		return ti
	}
	iprice, _ := strconv.ParseFloat(fmt.Sprintf("%s", strings.TrimSpace(string(price[1]))), 64)
	item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *Jd) GetItemImg(item *Item) *Jd {
	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(ti.content)

	img := hp.Partten(`(?U)<img.*src="(.*)" class="J_ping"`).FindStringSubmatch()

	if img == nil {
		item.err = errors.New(`get img error`)
		return ti
	}

	item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *Jd) Shop(item *Item) {

	url := fmt.Sprintf("http://ok.jd.com/m/index-%s.htm", item.params["id"])
	_, content, err := NewLoader().WithProxy().Get(url)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).Replace().CleanScript()

	if ti.GetShopTitle(item).CheckError(item) {
		return
	}

	if ti.GetShopImgs(item).CheckError(item) {
		return
	}
	SpiderServer.qfinish <- item
	return
}

func (ti *Jd) GetShopTitle(item *Item) *Jd {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).Replace()
	title := htmlParser.Partten(`(?U)<div class="name">(.*)</div>`).FindStringSubmatch()
	if title == nil {
		item.err = errors.New(`get jd title error.`)
		return ti
	}
	item.data["title"] = fmt.Sprintf("%s", title[1])
	logo := htmlParser.Partten(`(?U)class="store-logo">.*<img\ssrc="(.*)"`).FindStringSubmatch()
	if logo == nil {
		item.err = errors.New(`get jd shop logo error.`)
		return ti
	}
	item.data["img"] = fmt.Sprintf("%s", logo[1])
	return ti
}

func (ti *Jd) GetShopImgs(item *Item) *Jd {

	url := fmt.Sprintf("http://ok.jd.com/m/list-%s-0-1-1-10-1.htm", item.params["id"])
	_, content, err := NewLoader().WithProxy().Get(url)
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return ti
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).Replace().CleanScript()
	ret := htmlParser.Partten(`(?U)class="p-img">\s<img\ssrc="(.*)"`).FindAllSubmatch()

	if ret == nil {
		item.err = errors.New(`get jd shop images error.`)
		return ti
	}

	l := len(ret)
	if l == 0 {
		item.err = errors.New(`get jd shop images error.`)
		return ti
	}
	var imglist []string
	if l > 3 {
		l = 3
	}
	for i := 1; i < l; i++ {
		imglist = append(imglist, fmt.Sprintf("%s", ret[i][1]))
	}
	item.data["imgs"] = strings.Join(imglist, ",")
	return ti
}

func (ti *Jd) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
