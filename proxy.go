package spider

import (
	"fmt"
	utils "libs/utils"
	"time"
)

type ProxyServerInfo struct {
	host string
	port string
}

type Proxy struct {
	Servers []*ProxyServerInfo
	Count   int
}

var (
	proxyUrl_0  string = "http://proxy.com.ru/niming/"
	proxyUrl_1  string = "http://proxy.com.ru/niming/list_2.html"
	SpiderProxy *Proxy
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
			go sp.Load(proxyUrl_0)
			go sp.Load(proxyUrl_1)
			time.Sleep(time.Second * 10 * 60)
		}
	}()
}

func (sp *Proxy) GetProxyServer() *ProxyServerInfo {

	if len(sp.Servers) == 0 {
		return nil
	}
	num := utils.RandInt(0, len(sp.Servers))
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
	sp.Servers = nil
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
			SpiderLoger.E("Ping error, ", err.Error())
			continue
		}
		if pr.LostRate == 0 && pr.Average < 300 {
			sp.Servers = append(sp.Servers, &ProxyServerInfo{ip, port})
		}
	}
	j := len(sp.Servers)
	if j <= 5 {
		SendMail("proxy server less then 5", fmt.Sprintf("spider have %d proxy servers only", j))
	}
	SpiderLoger.I("proxy server total", len(sp.Servers))
	return
}
