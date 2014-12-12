package spider

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Taobao struct {
	item    *Item
	content []byte
}

func (ti *Taobao) Item() {
	url := fmt.Sprintf("http://hws.m.taobao.com/cache/wdetail/5.0/?id=%s", ti.item.params["id"])

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
}

func (ti *Taobao) GetItemTitle() *Taobao {
	hp := NewHtmlParse().LoadData(ti.content)
	title := hp.Partten(`(?U)"itemId":"\d+","title":"(.*)"`).FindStringSubmatch()

	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}
	ti.item.data["title"] = fmt.Sprintf("%s", title[1])

	favcount := hp.Partten(`(?U)"favcount":"(\d+)"`).FindStringSubmatch()
	if favcount == nil {
		ti.item.err = errors.New(`get favcount error`)
		return ti
	}
	ti.item.data["favcount"] = fmt.Sprintf("%s", favcount[1])

	totalSoldQuantity := hp.Partten(`(?U)"totalSoldQuantity":"(\d+)"`).FindStringSubmatch()
	if totalSoldQuantity == nil {
		ti.item.err = errors.New(`get totalSoldQuantity error`)
		return ti
	}
	ti.item.data["totalSoldQuantity"] = fmt.Sprintf("%s", totalSoldQuantity[1])

	goodRatePercentage := hp.Partten(`(?U)"goodRatePercentage":"(.*)"`).FindStringSubmatch()
	if goodRatePercentage == nil {
		ti.item.err = errors.New(`get goodRatePercentage error`)
		return ti
	}
	ti.item.data["goodRatePercentage"] = fmt.Sprintf("%s", goodRatePercentage[1])

	return ti
}

func (ti *Taobao) GetItemPrice() *Taobao {
	hp := NewHtmlParse().LoadData(ti.content)
	price := hp.Partten(`(?U)"rangePrice":".*","price":"(.*)"`).FindStringSubmatch()

	if price == nil {
		price = hp.Partten(`(?U)"price":"(.*)"`).FindStringSubmatch()
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

func (ti *Taobao) GetItemImg() *Taobao {
	hp := NewHtmlParse().LoadData(ti.content)
	img := hp.Partten(`(?U)"picsPath":\["(.*)"`).FindStringSubmatch()

	if img == nil {
		ti.item.err = errors.New(`get img error`)
		return ti
	}
	ti.item.data["img"] = fmt.Sprintf("%s", img[1])
	return ti
}

func (ti *Taobao) Shop() {
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

func (ti *Taobao) GetShopTitle() *Taobao {
	url := fmt.Sprintf("http://shop%s.m.taobao.com/", ti.item.params["id"])
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

func (ti *Taobao) GetShopImgs() *Taobao {
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
		sep := []byte(fmt.Sprintf(`data-item="%s"`, ti.item.params["id"]))
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

func (ti *Taobao) SameStyle() {
	var result []map[string]string
	url := fmt.Sprintf("http://s.taobao.com/list?tab=all&sort=sale-desc&type=samestyle&uniqpid=-%s&app=i2i&nid=%s", ti.item.params["pid"], ti.item.params["id"])
	loader := NewLoader(url, "Get").WithPcAgent().WithProxy(false)
	content, err := loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	hp := NewHtmlParse().LoadData(content).Replace().Convert()
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

	for i := 1; i < l; i++ {
		var sortScore = 10 - i + 1
		data := map[string]string{"istmall": "0", "comment_num": "0", "pay_num": "0", "sortScore": "0"}
		val := ret[i][1]
		hp1 := NewHtmlParse().LoadData(val)

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
		data["img"] = fmt.Sprintf("%s", imgs[1])

		title := hp1.Partten(`(?U)title="(.*)"`).FindStringSubmatch()
		data["title"] = fmt.Sprintf("%s", title[1])

		address := hp1.Partten(`(?U)<div class="seller-loc">(.*)</div>`).FindStringSubmatch()
		data["address"] = fmt.Sprintf("%s", address[1])

		istmall := bytes.Index(val, []byte(`icon-service-tianmao-large`))
		if istmall > 0 {
			sortScore += 1
		}

		comment_num := hp1.Partten(`(?U)(\d+) 条评论`).FindStringSubmatch()
		if comment_num != nil {
			data["comment_num"] = fmt.Sprintf("%s", comment_num[1])
		}

		data["sortScore"] = fmt.Sprintf("%d", sortScore)

		if i == 10 {
			break
		}
		result = append(result, data)
	}
	if len(result) == 0 {
		ti.item.err = errors.New("has none samestyle")
		SpiderServer.qerror <- ti.item
		return
	}
	ti.item.data["unipid"] = ti.item.params["pid"]
	ti.item.data["nid"] = ti.item.params["id"]
	ti.item.data["list"] = result
	// fmt.Println(ti.item.data)
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
