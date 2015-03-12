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
			SpiderLoger.I("Proxy start new runtime")
			proxyNum = 0
			for i := 1; i < 5; i++ {
				go sp.Load(fmt.Sprintf("http://proxy.com.ru/niming/list_%d.html", i))
			}
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

	loader := NewLoader(proxyUrl, "GET").WithPcAgent().WithProxy(false)
	content, err := loader.Send(nil)
	if err != nil {
		SpiderLoger.E("Load proxy error with", proxyUrl)
		return
	}
	hp := NewHtmlParse().LoadData(content).Replace().CleanScript()
	trs := hp.Partten(`(?U)<td>(\d+\.\d+\.\d+\.\d+)</td><td>(\d+)</td>`).FindAllSubmatch()
	l := len(trs)
	if l == 0 {
		SendMail("Load proxy data error.", "load proxy data from "+proxyUrl+" error. ")
		return
	}
	if proxyNum == 0 {
		sp.Servers = make(map[int]*ProxyServerInfo)
	}
	for i := 0; i < l; i++ {
		ip, port := string(trs[i][1]), string(trs[i][2])
		pr := &PingResult{}
		err = Ping(pr, ip, port)
		if err != nil {
			// SpiderLoger.W("Ping error", err.Error())
			continue
		}
		if pr.LostRate == 0 && pr.Average < 500 {
			sp.Servers[proxyNum] = &ProxyServerInfo{proxyNum, ip, port}
			proxyNum++
		}
	}
	if proxyNum <= 5 {
		SendMail("Proxy server less then 5", fmt.Sprintf("spider have %d proxy servers only", proxyNum))
	}
	SpiderLoger.I("The proxy server count", proxyNum)
	return
}
