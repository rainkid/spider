package spider

import (
	"encoding/json"
	"fmt"
	"net/url"
)

var (
	SpiderServer *Spider
	SpiderLoger  *MyLoger = NewMyLoger()
)

type Spider struct {
	qstart  chan *Item
	qfinish chan *Item
	qerror  chan *Item
}

type Item struct {
	params   map[string]string
	data     map[string]interface{}
	tag      string
	method   string
	tryTimes int
	err      error
}

func NewSpider() *Spider {
	SpiderServer = &Spider{
		qstart:  make(chan *Item, 1),
		qfinish: make(chan *Item, 1),
		qerror:  make(chan *Item, 1),
	}
	return SpiderServer
}

func Start() *Spider {
	if SpiderServer == nil {
		SpiderLoger.I("SpiderServer Daemon.")
		SpiderServer = NewSpider()
		SpiderServer.Daemon()
	}
	return SpiderServer
}

func (spider *Spider) Do(item *Item) {
	item.tryTimes++
	SpiderLoger.I(fmt.Sprintf("tag: <%s>, params: %v try with (%d) times.", item.tag, item.params, item.tryTimes))
	switch item.tag {
	case "TmallItem":
		tmall := &Tmall{}
		go tmall.Item(item)
		break
	case "TaobaoItem":
		taobao := &Taobao{}
		go taobao.Item(item)
		break
	case "JdItem":
		jd := &Jd{}
		go jd.Item(item)
		break
	case "Same":
		hh := &Hhui{}
		go hh.Item(item)
		break
	case "MmbItem":
		mmb := &MMB{}
		go mmb.Item(item)
		break
	case "TmallShop":
		tmall := &Tmall{}
		go tmall.Shop(item)
		break
	case "TmallSearch":
		tmall := &Tmall{}
		go tmall.Search(item)
		break
	case "JdShop":
		jd := &Jd{}
		go jd.Shop(item)
		break
	case "TaobaoShop":
		taobao := &Taobao{}
		go taobao.Shop(item)
		break
	case "SameStyle":
		taobao := &Taobao{}
		go taobao.SameStyle(item)
	case "Other":
		other := &Other{}
		go other.Get(item)
		break
	}
	return
}

func (spider *Spider) Error(item *Item) {
	if item.err != nil {
		err := fmt.Sprintf("tag:<%s>, params: [%v] error :{%v}", item.tag, item.params["id"], item.err.Error())
		SpiderLoger.E(err)
		// if item.tryTimes < 2 {
		// 	SpiderServer.qstart <- item
		// 	return
		// }
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
	v.Add("id", item.params["id"])
	v.Add("data", fmt.Sprintf("%s", output))
	v.Add("method", fmt.Sprintf("%s", item.method))
	SpiderLoger.D(v)
	url, _ := url.QueryUnescape(item.params["callback"])

	_, _, err = NewLoader().Post(url, v)
	if err != nil {
		SpiderLoger.E("Callback with error", err.Error())
		return
	}
	SpiderLoger.I("-- callback --", fmt.Sprintf("tag:<%s> params:%v", item.tag, item.params))
	return
}

func (spider *Spider) Add(tag string, params map[string]string) {
	item := &Item{
		tag:      tag,
		method:   tag,
		params:   params,
		tryTimes: 0,
		data:     make(map[string]interface{}),
		err:      nil,
	}
	spider.qstart <- item
	return
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
