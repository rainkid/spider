package spider

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Jd struct {
	item    *Item
	content []byte
}

func (ti *Jd) Item() {
	url := fmt.Sprintf("http://m.jd.com/product/%s.html", ti.item.params["id"])

	//get content
	loader := NewLoader(url, "Get")
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	ti.content = bytes.Replace(content, []byte(`\"`), []byte(`"`), -1)

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
	return
}

func (ti *Jd) GetItemTitle() *Jd {
	hp := NewHtmlParse().LoadData(ti.content).CleanScript().Replace()

	title := hp.Partten(`(?Usm)<title>(.*)-\s`).FindStringSubmatch()

	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}
	ti.item.data["title"] = strings.TrimSpace(string(title[1]))
	return ti
}

func (ti *Jd) GetItemPrice() *Jd {
	hp := NewHtmlParse().LoadData(ti.content)
	price := hp.Partten(`(?U)&yen;(\d+\.\d+)`).FindStringSubmatch()
	if price == nil {
		ti.item.err = errors.New(`get price error`)
		return ti
	}
	iprice, _ := strconv.ParseFloat(fmt.Sprintf("%s", strings.TrimSpace(string(price[1]))), 64)
	ti.item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *Jd) GetItemImg() *Jd {
	hp := NewHtmlParse().LoadData(ti.content)

	img := hp.Partten(`(?U)src="(http://img10.360buyimg.com/.*)"`).FindStringSubmatch()

	if img == nil {
		ti.item.err = errors.New(`get img error`)
		return ti
	}

	ti.item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *Jd) Shop() {

	url := fmt.Sprintf("http://ok.jd.com/m/index-%s.htm", ti.item.params["id"])
	loader := NewLoader(url, "Get")
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	hp := NewHtmlParse()
	hp = hp.LoadData(content).Replace().CleanScript()
	ti.content = hp.content

	if ti.GetShopTitle().CheckError() {
		return
	}

	if ti.GetShopImgs().CheckError() {
		return
	}
	SpiderServer.qfinish <- ti.item
	return
}

func (ti *Jd) GetShopTitle() *Jd {
	hp := NewHtmlParse()
	hp = hp.LoadData(ti.content).Replace()
	title := hp.Partten(`(?U)<div class="name">(.*)</div>`).FindStringSubmatch()
	if title == nil {
		ti.item.err = errors.New(`get jd title error.`)
		return ti
	}
	ti.item.data["title"] = fmt.Sprintf("%s", title[1])
	logo := hp.Partten(`(?U)class="store-logo">.*<img\ssrc="(.*)"`).FindStringSubmatch()
	if logo == nil {
		ti.item.err = errors.New(`get jd shop logo error.`)
		return ti
	}
	ti.item.data["img"] = fmt.Sprintf("%s", logo[1])
	return ti
}

func (ti *Jd) GetShopImgs() *Jd {

	url := fmt.Sprintf("http://ok.jd.com/m/list-%s-0-1-1-10-1.htm", ti.item.params["id"])

	loader := NewLoader(url, "Get")
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		ti.item.err = errors.New(`shop not found.`)
		return ti
	}

	hp := NewHtmlParse().LoadData(content).Replace().CleanScript()
	ti.content = hp.content

	ret := hp.Partten(`(?U)class="p-img">\s<img\ssrc="(.*)"`).FindAllSubmatch()

	if ret == nil {
		ti.item.err = errors.New(`get jd shop images error.`)
		return ti
	}

	l := len(ret)
	if l == 0 {
		ti.item.err = errors.New(`get jd shop images error.`)
		return ti
	}
	var imglist []string
	if l > 3 {
		l = 3
	}
	for i := 1; i < l; i++ {
		imglist = append(imglist, fmt.Sprintf("%s", ret[i][1]))
	}
	ti.item.data["imgs"] = strings.Join(imglist, ",")
	return ti
}

func (ti *Jd) CheckError() bool {
	if ti.item.err != nil {
		SpiderServer.qerror <- ti.item
		return true
	}
	return false
}
