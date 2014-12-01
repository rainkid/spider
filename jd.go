package spider

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"io/ioutil"
)

type Jd struct {
	item    *Item
	content []byte
}

func (ti *Jd) Item() {
	url := fmt.Sprintf("http://m.jd.com/product/%s.html", ti.item.id)

	//get content
	loader := NewLoader(url, "Get")
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	ti.content = bytes.Replace(content, []byte(`\"`), []byte(`"`), -1)
	//  fmt.Println(string(ti.content));
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
	fmt.Println(ti.item.data)
	SpiderServer.qfinish <- ti.item
}

func (ti *Jd) GetItemTitle() *Jd {
	hp := NewHtmlParse().LoadData(ti.content)

	ioutil.WriteFile("./xxx.html", ti.content, 0666)  //写入文件(字节数组)

	title := hp.Partten(`(?Usm)<title>(.*)-\s`).FindStringSubmatch()

	fmt.Printf("[%q]", strings.TrimSpace(string(title[1])))


	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}

	return ti
}

func (ti *Jd) GetItemPrice() *Jd {
	hp := NewHtmlParse().LoadData(ti.content)
	price := hp.Partten(`(?U)id="price">&yen;(.*)\s</span>`).FindStringSubmatch()
	fmt.Printf("[%q]", string(price[1]))
	iprice, _ := strconv.ParseFloat(fmt.Sprintf("%s", strings.TrimSpace(string(price[1]))), 64)
	ti.item.data["price"] = fmt.Sprintf("%.2f", iprice)
	return ti
}

func (ti *Jd) GetItemImg() *Jd {
	hp := NewHtmlParse().LoadData(ti.content)
	img := hp.Partten(`(?U)"tbl-cell"><img src="(.*)"`).FindStringSubmatch()
	fmt.Printf("[%q]", string(img[1]))
	if img == nil {
		ti.item.err = errors.New(`get img error`)
		return ti
	}
	ti.item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *Jd) Shop() {
	if ti.GetShopTitle().CheckError() {
		return
	}
	url := fmt.Sprintf("http://s.jd.com/search?q=%s&app=shopsearch", ti.item.data["title"])
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

func (ti *Jd) GetShopTitle() *Jd {
	url := fmt.Sprintf("http://shop%s.m.jd.com/", ti.item.id)
	//get content
	loader := NewLoader(url, "Get")
	shop, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return ti
	}

	hp := NewHtmlParse()
	hp = hp.LoadData(shop).Replace()
	shopname := hp.FindByTagName("title")
	uid := hp.Partten(`G_msp_userId="(\d+)"`).FindStringSubmatch()

	if shopname == nil {
		ti.item.err = errors.New("get shop title error")
		SpiderServer.qerror <- ti.item
		return ti

	}
	if uid == nil {
		ti.item.err = errors.New("get shop uid error")
		SpiderServer.qerror <- ti.item
		return ti
	}
	ti.item.data["uid"] = fmt.Sprintf("%s", uid[1])
	title := bytes.Replace(shopname[0][2], []byte("首页"), []byte(""), -1)
	title = bytes.Replace(title, []byte("淘宝网"), []byte(""), -1)
	title = bytes.Replace(title, []byte("天猫"), []byte(""), -1)
	title = bytes.Replace(title, []byte("Tmall.com"), []byte(""), -1)
	title = bytes.Replace(title, []byte("-"), []byte(" "), -1)
	title = bytes.Trim(title, " ")
	ti.item.data["title"] = fmt.Sprintf("%s", title)
	return ti
}

func (ti *Jd) GetShopImgs() *Jd {
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

func (ti *Jd) CheckError() bool {
	if ti.item.err != nil {
		SpiderServer.qerror <- ti.item
		return true
	}
	return false
}
