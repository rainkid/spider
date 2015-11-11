package spider

import (
	"fmt"
	"io/ioutil"
	utils "libs/utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ProxyServerInfo struct {
	id         int
	host       string
	port       string
	rate       float64 //network speed
	style      int     //1 http 2 https 3 socket
	anonymous  bool    //0 transparent 1 low 2 high
	last_check int64   //timestamp last check time
	area       string  //region
	status     bool    //region
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
		//		for {
		//			SpiderLoger.I("Proxy start new runtime")
		//			proxyNum = 0
		//			for i := 1; i < 5; i++ {
		//				go sp.Load(fmt.Sprintf("http://proxy.com.ru/list_%d.html", i))
		//			}
		//			time.Sleep(time.Second * 10 * 60)
		//		}
		//		for {
		//			SpiderLoger.I("Proxy start new runtime with xicidaili")
		//			proxyNum = 0
		//			for i := 1; i < 3; i++ {
		////				xicidaili
		//				go sp.Xici(fmt.Sprintf("http://www.xicidaili.com/nn/%d", i))
		//			}
		//			time.Sleep(time.Second * 10 * 60)
		//		}
		//		for {
		//			SpiderLoger.I("Proxy start new runtime with haodailiip")
		//			proxyNum = 0
		//			for i := 1; i < 3; i++ {
		////				xicidaili
		//				go sp.Xici(fmt.Sprintf("http://www.xicidaili.com/nn/%d", i))
		//			}
		//			time.Sleep(time.Second * 10 * 60)
		//		}

		for {
			SpiderLoger.I("Proxy start new runtime with kuaidaili")
			for i := 1; i < 5; i++ {
				go sp.kuai(fmt.Sprintf("http://www.kuaidaili.com/proxylist/%d", i))
			}
			time.Sleep(time.Second * 1)
			if sp.Count > 0 {
				go sp.Check()
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
	if !sp.Servers[num].status {
		return sp.GetProxyServer()
	}
	return sp.Servers[num]

}

func (i *ProxyServerInfo) CheckTaobao(c chan bool) {

	start_ts := time.Now()
	if (time.Now().Unix() - i.last_check) < 30*60 {
		c <- false
		return
	}

	if i.status {
		c <- false
		return
	}

	i.last_check = time.Now().Unix()
	var timeout = time.Duration(30 * time.Second)

	host := fmt.Sprintf("%s:%s", i.host, i.port)
	url_proxy := &url.URL{Host: host}

	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(url_proxy)},
		Timeout:   timeout}

	resp, err := client.Get("https://err.taobao.com/error1.html")
	if err != nil {
		c <- false
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c <- false
	}

	body, _ := ioutil.ReadAll(resp.Body)

	time_diff := time.Now().UnixNano() - start_ts.UnixNano()
	if strings.Contains(string(body), "alibaba.com") {
		i.rate = float64(time_diff) / 1e9
		i.status = true

		SpiderLoger.I("Proxy :[", host, "] OK")
		c <- true
	} else {
		i.status = false
		c <- false
	}
}

func (sp *Proxy) Check() {

	SpiderLoger.I("Start checking proxys")
	if sp.Count < 1 {
		return
	}

	resp_chan := make(chan bool, 10)

	for _, i := range sp.Servers {
		go i.CheckTaobao(resp_chan)
	}

	<-resp_chan

}

func (sp *Proxy) kuai(proxyUrl string) {
	loader := NewLoader()

	content, err := loader.WithPcAgent().WithProxy(false).Send(proxyUrl, "GET", nil)
	if err != nil {
		SpiderLoger.E("Load proxy error with", proxyUrl)
		return
	}

	mcontent := make([]byte, len(content))
	copy(mcontent, content)

	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(mcontent).Replace().CleanScript()
	trs := hp.Partten(`(?U)<td>(\d+\.\d+\.\d+\.\d+)</td>\s<td>(\d+)</td>`).FindAllSubmatch()
	l := len(trs)
	if l == 0 {
		SpiderLoger.E("load proxy data from " + proxyUrl + " error. ")
		return
	}
	if proxyNum == 0 {
		sp.Servers = make(map[int]*ProxyServerInfo)
	}
	for i := 0; i < l; i++ {
		ip, port := string(trs[i][1]), string(trs[i][2])
		sp.Servers[proxyNum] = &ProxyServerInfo{id: proxyNum, host: ip, port: port}
		proxyNum++
	}
	sp.Count = proxyNum
	if proxyNum <= 5 {
		SpiderLoger.E("proxy servers only ", proxyNum)
	}
	SpiderLoger.I("The proxy server count", proxyNum)
	return

}

func (sp *Proxy) Xici(proxyUrl string) {
	loader := NewLoader()

	content, err := loader.WithPcAgent().WithProxy(false).Send(proxyUrl, "GET", nil)
	if err != nil {
		SpiderLoger.E("Load proxy error with", proxyUrl)
		return
	}
	mcontent := make([]byte, len(content))
	copy(mcontent, content)

	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(mcontent).Replace().CleanScript()
	pattern := `(?U)<tr class="\w?">.*alt="(\w+)".*<td>(\d+\.\d+\.\d+\.\d+)</td>\s<td>(\d+)</td>.*a>\s</td>\s<td>(.*)</td>\s<td>(.*)</td>.*title="(.*)秒".*.*title=".*秒".*</div>.*<td>(.*)</td>\s</tr>`
	trs := hp.Partten(pattern).FindAllSubmatch()
	l := len(trs)
	if l == 0 {
		SpiderLoger.E("load proxy data from " + proxyUrl + " error. ")
		return
	}

	if proxyNum == 0 {
		sp.Servers = make(map[int]*ProxyServerInfo)
	}
	for i := 0; i < l; i++ {
		area, ip, port, anonymous, style, rate, _ := string(trs[i][1]), string(trs[i][2]), string(trs[i][3]), string(trs[i][4]), string(trs[i][5]), string(trs[i][6]), string(trs[i][7])
		info := ProxyServerInfo{}

		style_map := map[string]int{"http": 1, "https": 2, "socket": 3}

		info.id = proxyNum
		info.host = ip
		info.port = port
		info.rate, _ = strconv.ParseFloat(rate, 64)
		info.anonymous = (anonymous == "高匿")
		info.style = style_map[strings.ToLower(style)]
		info.area = strings.ToLower(area)
		sp.Servers[proxyNum] = &info
		proxyNum++
	}
	if proxyNum <= 5 {
		SpiderLoger.E("proxy servers only ", proxyNum)
	}
	SpiderLoger.I("The proxy server count", proxyNum)
	return
}

func (sp *Proxy) Load(proxyUrl string) {
	loader := NewLoader()

	content, err := loader.WithPcAgent().WithProxy(false).Send(proxyUrl, "GET", nil)
	if err != nil {
		SpiderLoger.E("Load proxy error with", proxyUrl)
		return
	}
	mcontent := make([]byte, len(content))
	copy(mcontent, content)

	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(mcontent).Replace().CleanScript()
	trs := hp.Partten(`(?U)<td>(\d+\.\d+\.\d+\.\d+)</td><td>(\d+)</td>`).FindAllSubmatch()
	l := len(trs)
	if l == 0 {
		SpiderLoger.E("load proxy data from " + proxyUrl + " error. ")
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
			//			sp.Servers[proxyNum] = &ProxyServerInfo{proxyNum, ip, port}
			proxyNum++
		}
	}
	if proxyNum <= 5 {
		SpiderLoger.E("proxy servers only ", proxyNum)
	}
	SpiderLoger.I("The proxy server count", proxyNum)
	return
}
