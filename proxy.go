package spider

import (
	"fmt"
	utils "libs/utils"
	"time"
)

type Proxy struct {
	Servers [][][]byte
	Count   int
}

var (
	proxyUrl string = "http://proxy.com.ru/"
)

func NewProxy() *Proxy {
	return &Proxy{}
}

func (sp *Proxy) Daemon() {
	go func() {
		sp.Load()
		for {
			time.Sleep(time.Second * 60 * 10)
		}
	}()
}

func (sp *Proxy) GetProxyServer() (host, port []byte) {

	if len(sp.Servers) == 0 {
		return nil, nil
	}
	num := utils.RandInt(0, len(sp.Servers))
	return []byte(sp.Servers[num][0]), []byte(sp.Servers[num][1])
}

func (sp *Proxy) Load() {
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
		ip, port := trs[i][1], trs[i][2]
		pr := &PingResult{}
		err := Ping(pr, fmt.Sprintf("%s", ip))
		if err != nil {
			SpiderLoger.E("Ping error, ", err.Error())
			continue
		}
		if pr.LostRate == 0 && pr.Average < 200 {
			sp.Servers = append(sp.Servers, [][]byte{ip, port})
		}
	}
	j := len(sp.Servers)
	if j <= 5 {
		SendMail("proxy server less then 5", fmt.Sprintf("spider have %d proxy servers only", j))
	}
	SpiderLoger.I("proxy server total", len(sp.Servers))
	return
}
