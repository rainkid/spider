package spider

import (
	"fmt"
	utils "libs/utils"
	"time"
)

type ProxyServerInfo struct {
	id   int
	host string
	port string
}

type Proxy struct {
	Servers map[int]*ProxyServerInfo
	Count   int
}

var (
	proxyUrl_1  string = "http://proxy.com.ru/niming/list_1.html"
	proxyUrl_2  string = "http://proxy.com.ru/niming/list_2.html"
	SpiderProxy *Proxy
	proxyNum    = 0
)

func NewProxy() *Proxy {
	return &Proxy{}
}

func StartProxy() *Proxy {
	if SpiderProxy == nil {
		SpiderLoger.I("SpiderProxy Daemon.")
		SpiderProxy = NewProxy()
		SpiderProxy.Daemon()
	}
	return SpiderProxy
}

func (sp *Proxy) Daemon() {
	go func() {
		for {
			proxyNum = 0
			sp.Servers = make(map[int]*ProxyServerInfo)
			go sp.Load(proxyUrl_1)
			go sp.Load(proxyUrl_2)
			time.Sleep(time.Second * 10 * 60)
		}
	}()
}

func (sp *Proxy) DelProxyServer(index int) {
	SpiderLoger.D("delete proxyserver", index)
	delete(sp.Servers, index)
}

func (sp *Proxy) GetProxyServer() *ProxyServerInfo {
	if proxyNum == 0 {
		return nil
	}
	num := utils.RandInt(0, proxyNum)
	return sp.Servers[num]
}

func (sp *Proxy) Load(proxyUrl string) {
	SpiderLoger.I("load proxy data from", proxyUrl)

	loader := NewLoader(proxyUrl, "GET").WithProxy(false)
	content, err := loader.Send(nil)
	if err != nil {
		SpiderLoger.E("load proxy error with", proxyUrl)
		SendMail("load proxy data error.", err.Error())
		return
	}
	hp := NewHtmlParse().LoadData(content).Replace().Convert()
	trs := hp.Partten(`(?U)<td>(\d+\.\d+\.\d+\.\d+)</td><td>(\d+)</td>`).FindAllSubmatch()
	l := len(trs)
	if l == 0 {
		SendMail("load proxy data error.", "load proxy data from "+proxyUrl+" error. ")
		return
	}
	for i := 0; i < l; i++ {
		ip, port := string(trs[i][1]), string(trs[i][2])
		pr := &PingResult{}
		err = Ping(pr, ip, port)
		if err != nil {
			SpiderLoger.E("ping error", err.Error())
			continue
		}
		if pr.LostRate == 0 && pr.Average < 500 {
			sp.Servers[proxyNum] = &ProxyServerInfo{proxyNum, ip, port}
			proxyNum++
		}
	}
	if proxyNum <= 5 {
		SendMail("proxy server less then 5", fmt.Sprintf("spider have %d proxy servers only", proxyNum))
	}
	SpiderLoger.I("proxy server total", proxyNum)
	return
}
