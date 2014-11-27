package spider

import (
	"encoding/json"
	"fmt"
	utils "libs/utils"
	"net/url"
)

var (
	SpiderServer *Spider
	SpiderProxy  *Proxy
	spiderErrors *SpiderErrors = &SpiderErrors{}
	SpiderLoger  *MyLoger      = NewMyLoger()
	TryTime                    = 5
)

type SpiderErrors struct {
	errorStr   string
	errorTotal int
}

type Spider struct {
	qstart  chan *Item
	qfinish chan *Item
	qerror  chan *Item
}

type Item struct {
	id       string
	callback string
	data     map[string]string
	tag      string
	tryTimes int
	err      error
}

func NewSpider() *Spider {
	SpiderServer = &Spider{
		qstart:  make(chan *Item),
		qfinish: make(chan *Item),
		qerror:  make(chan *Item),
	}
	return SpiderServer
}

func Start() *Spider {
	if SpiderServer == nil {
		SpiderServer = NewSpider()
		SpiderServer.Daemon()
	}
	if SpiderProxy == nil {
		SpiderProxy = NewProxy()
		SpiderProxy.Daemon()
	}
	return SpiderServer
}

func SendMail(title, content string) error {
	return utils.SendMail("rainkid@163.com", "Rainkid,.0.", "smtp.163.com:25", "liaohu@gionee.com", title, content, "html")
}

func (spider *Spider) Do(item *Item) {
	item.tryTimes++
	SpiderLoger.I(fmt.Sprintf("item.id:%s,item.tag:%s try with %d times.", item.id, item.tag, item.tryTimes))
	switch item.tag {
	case "TmallItem":
		ti := &Tmall{item: item}
		go ti.Item()
		break
	case "TaobaoItem":
		ti := &Taobao{item: item}
		go ti.Item()
		break
	case "MmbItem":
		ti := &MMB{item: item}
		go ti.Item()
		break
	case "TmallShop":
		ti := &Tmall{item: item}
		go ti.Shop()
		break
	case "TaobaoShop":
		ti := &Taobao{item: item}
		go ti.Shop()
		break
	case "Other":
		ti := &Other{item: item}
		go ti.Get()
		break
	}
	return
}

func (spider *Spider) Error(item *Item) {
	if item.err != nil {
		sbody := fmt.Sprintf("id:%s tag:%s %s", item.id, item.tag, item.err.Error())
		if spiderErrors.errorTotal == 10 {
			err := SendMail("spider load data error.", sbody)
			if err != nil {
				SpiderLoger.E("send mail fail.")
			}
		}
		spiderErrors.errorStr += sbody + "\n"
		spiderErrors.errorTotal++
		SpiderLoger.E(sbody)
		item.err = nil
	}
	return
}

func (spider *Spider) Finish(item *Item) {
	output, err := json.Marshal(item.data)
	if err != nil {
		SpiderLoger.E("error with json output")
		return
	}
	v := url.Values{}
	v.Add("id", item.id)
	v.Add("data", fmt.Sprintf("%s", output))

	url, _ := url.QueryUnescape(item.callback)
	loader := NewLoader(url, "Post").WithProxy(false)
	_, err = loader.Send(v)
	if err != nil {
		SpiderLoger.E("callback with error", err.Error())
		return
	}
	SpiderLoger.I("success with", fmt.Sprintf("tag:%s,id:%s,url:%s", item.tag, item.id, url))
	return
}

func (spider *Spider) Add(tag, id, callback string) {
	item := &Item{
		tag:      tag,
		id:       id,
		tryTimes: 0,
		callback: callback,
		data:     make(map[string]string),
		err:      nil,
	}
	spider.qstart <- item
}

func (spider *Spider) Daemon() {
	go func() {
		for {
			select {
			case item := <-spider.qstart:
				go spider.Do(item)
				break
			case item := <-spider.qfinish:
				go spider.Finish(item)
				break
			case item := <-spider.qerror:
				go spider.Error(item)
				break
			}
		}
	}()
}
