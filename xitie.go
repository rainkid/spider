package spider

import (
	// "bytes"
	"errors"
	"fmt"
	"github.com/qiniu/iconv"
	"strconv"
	"strings"
)

type Xitie struct {
	content []byte
	json    []byte
}

func (ti *Xitie) Item(item *Item) {
	url := "http://www.xitie.com/jd.php?no=592892"
	fmt.Println(item.params["channel"])
	switch item.params["channel"] {
	case "jd":
		url = fmt.Sprintf("http://www.xitie.com/jd.php?no=%s", item.params["id"])
		break
	case "tmall":
		url = fmt.Sprintf("http://www.xitie.com/tmall.php?no=%s", item.params["id"])
		break
	case "taobao":
		url = fmt.Sprintf("http://www.xitie.com/taobao.php?no=%s", item.params["id"])
		break
	}
	fmt.Println(url)
	//get content
	_, content, err := NewLoader().Get(url)
	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	//	fmt.Println(content)
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	// ti.content = bytes.Replace(ti.content, []byte(`\"`), []byte(`"`), -1)

	if ti.getItemCleanContent(item).CheckError(item) {
		return
	}
	if ti.getItemX(item).CheckError(item) {
		return
	}
	if ti.getItemY(item).CheckError(item) {
		return
	}
	fmt.Println(item.data["json"])
	// fmt.Println(item.data)
	SpiderServer.qfinish <- item
	return
}

func (ti *Xitie) getItemCleanContent(item *Item) *Xitie {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).CleanScript().Replace()
	con := htmlParser.Partten(`(?U)highcharts\((.*)\)\;\s\}\)\;`).FindStringSubmatch()
	if con == nil {
		item.err = errors.New(`get json error`)
		return ti
	}

	cd, err := iconv.Open("utf-8", "gbk") // convert utf-8 to gbk
	if err != nil {
		fmt.Println("iconv.Open failed!")
		return ti
	}
	defer cd.Close()

	conv := cd.ConvString(strings.TrimSpace(string(con[1])))
	ti.json = make([]byte, len(conv))
	copy(ti.json, conv)
	return ti
}

func (ti *Xitie) getItemX(item *Item) *Xitie {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.json).CleanScript().Replace()
	x := htmlParser.Partten(`(?U)categories:(.*)]`).FindStringSubmatch()
	if x == nil {
		item.err = errors.New(`get X error`)
		return ti
	}

	space := strings.Trim(strings.TrimSpace(string(x[1])), "['")
	space = strings.Replace(space, ".", "-", len(space))
	tm := strings.Split(space, "','")
	item.data["x"] = tm
	return ti
}

func (ti *Xitie) getItemY(item *Item) *Xitie {
	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.json).CleanScript().Replace()
	y := htmlParser.Partten(`(?U)data:(.*)]`).FindStringSubmatch()
	if y == nil {
		item.err = errors.New(`get Y error`)
		return ti
	}
	fmt.Println(y)
	space := strings.Trim(strings.TrimSpace(string(y[1])), "['")
	tm := strings.Split(space, ",")
	item.data["y"] = tm
	fmt.Println(item.data["y"])
	return ti
}

func (ti *Xitie) GetItemPrice(item *Item) *Xitie {
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

func (ti *Xitie) GetItemImg(item *Item) *Xitie {
	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(ti.content)

	img := hp.Partten(`(?U)<img class="unit-pic J_ping".*src="(.*)">`).FindStringSubmatch()

	if img == nil {
		item.err = errors.New(`get img error`)
		return ti
	}

	item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *Xitie) Shop(item *Item) {

	url := fmt.Sprintf("http://ok.jd.com/m/index-%s.htm", item.params["id"])

	_, content, err := NewLoader().Get(url)

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

func (ti *Xitie) GetShopTitle(item *Item) *Xitie {
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

func (ti *Xitie) GetShopImgs(item *Item) *Xitie {

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

func (ti *Xitie) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
