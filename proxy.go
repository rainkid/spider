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
	"hash/fnv"
	"runtime"
)

type ProxyServerInfo struct {
	host       string
	port       string
	rate       float64 //network speed
	style      int     //1 http 2 https 3 socket
	anonymous  bool    //0 transparent 1 low 2 high
	last_check int64   //timestamp last check time
	area       string  //region
	TbStatus   bool    //region
	HhStatus   bool    //region
}

type Proxy struct {
	Rows  map[uint32]*ProxyServerInfo
	Tbao  []*ProxyServerInfo
	Hhui  []*ProxyServerInfo
	Count int
}

var (
	SpiderProxy *Proxy
	count = 0
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

	tick_get := time.NewTicker(20 * 60 * time.Second)
	tick_check := time.NewTicker(120 * 60 * time.Second)

	go func() {
		sp.getProxyServer()
		for {
			select {
			case <-tick_get.C:
				sp.getProxyServer()
			case <-tick_check.C:
				sp.Check()
			}
		}
	}()
}

func timer()  {
	t := time.Tick(time.Second*30)
	go func() {
		for {
			select {
			case <-t:
				SpiderLoger.I(fmt.Sprintf("NumGoroutine: %d", runtime.NumGoroutine()))
			}
		}
	}()
}

func (sp *Proxy) getProxyServer() {
	SpiderLoger.I("Proxy start new runtime with kuaidaili")
	for i := 1; i < 6; i++ {
		sp.kuai(fmt.Sprintf("http://www.kuaidaili.com/proxylist/%d", i))
	}
}

func (sp *Proxy) Check() {

	return
	count := len(sp.Rows)
	SpiderLoger.I("Start checking proxys")
	if count < 10 {
		return
	}

	jobs := make(chan *ProxyServerInfo,10)
	ch := make(chan bool,100)
	done := make(chan bool)

	go func() {
		for {
			j, more := <-jobs
			if more {
				ch<-true
				go j.CheckTaobao(ch)
//				fmt.Println("received jobs",j.host,j.port)
			} else {
				time.Sleep(time.Second*5)
				SpiderLoger.I("End checking proxys count[",len(sp.Tbao),"]")
				sp.Tbao =[]*ProxyServerInfo{}
				for k,i:= range sp.Rows {
					if i.TbStatus {
						sp.Tbao = append(sp.Tbao,i)
					}else{
						sp.DelProxyServer(k)
					}
				}
				SpiderLoger.I("End checking proxys count[",len(sp.Tbao),"]")
				//				fmt.Println("received all jobs")
				done <- true
				return
			}
		}
	}()

	for _,i := range sp.Rows {
		jobs <- i
		//		fmt.Println("sent job", i)
	}
	close(jobs)
	//	fmt.Println("sent all jobs")
	//We await the worker using the synchronization approach we saw earlier.
	<-done

}

func (sp *Proxy) DelProxyServer(index uint32) {
	SpiderLoger.D("delete proxyserver", index)
	delete(sp.Rows, index)
}

func (sp *Proxy) GetProxyServer() *ProxyServerInfo {
	count := len(sp.Tbao)
	if count == 0 {
		return nil
	}
	num := utils.RandInt(0, count-1)
	return sp.Tbao[num]

}

func ChkByTbao(ip string ,port string) bool{

	var timeout = time.Duration(10 * time.Second)
	host := fmt.Sprintf("%s:%s", ip, port)
	url_proxy := &url.URL{Host: host}
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(url_proxy)},
		Timeout:   timeout}

	resp, err := client.Get("https://err.taobao.com/error1.html")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if !strings.Contains(string(body), "alibaba.com") {
		return false
	}
	SpiderLoger.I("Proxy :[", host, "] OK")
	return true
}
func ChkByHhui(ip string ,port string) bool{

	var timeout = time.Duration(10 * time.Second)
	host := fmt.Sprintf("%s:%s", ip, port)
	url_proxy := &url.URL{Host: host}
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(url_proxy)},
		Timeout:   timeout}

	resp, err := client.Get("https://err.taobao.com/error1.html")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false
	}
	body, _ := ioutil.ReadAll(resp.Body)
	if !strings.Contains(string(body), "alibaba.com") {
		return false
	}
	SpiderLoger.I("Proxy :[", host, "] OK")
	return true
}

func (i *ProxyServerInfo) CheckTaobao(ch chan bool)bool {

	if (time.Now().Unix() - i.last_check) < 30 * 60 {
		if i.TbStatus {
			<-ch
			return true
		}
		<-ch
		return false
	}


	i.last_check = time.Now().Unix()

	if(ChkByTbao(i.host,i.port)){
		<-ch
		return true
	}else {
		<-ch
		return false
	}
}


func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
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

	hp := htmlParser.LoadData(mcontent)
	trs := hp.Partten(`(?U)<td>(\d+\.\d+\.\d+\.\d+)</td>\s+<td>(\d+)</td>\s+<td>(.*)</td>`).FindAllSubmatch()
	l := len(trs)
	if l == 0 {
		SpiderLoger.E("load proxy data from " + proxyUrl + " error. ")
		return
	}
	count = len(sp.Rows)
	if count == 0 {
		sp.Rows = make(map[uint32]*ProxyServerInfo)
	}

	for i := 0; i < l; i++ {
		if string(trs[i][3]) != "高匿名" {
			continue
		}
		ip, port := string(trs[i][1]), string(trs[i][2])
		h := hash(fmt.Sprintf("%s:%s", ip, port))
		_,ok := sp.Rows[h]
		if ok {
			continue
		}
		go func() {
			if ChkByTbao(ip,port){
				sp.Rows[h] = &ProxyServerInfo{host: ip, port: port, TbStatus:true}
				sp.Tbao = append(sp.Tbao,sp.Rows[h])
			}
		}()
		count++
	}
	sp.Count = count
	if count <= 5 {
		SpiderLoger.E("The proxy servers only ", count)
	}
	SpiderLoger.I("The proxy server count", count)
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

	if count == 0 {
		sp.Rows = make(map[uint32]*ProxyServerInfo)
	}
	for i := 0; i < l; i++ {
		area, ip, port, anonymous, style, rate, _ := string(trs[i][1]), string(trs[i][2]), string(trs[i][3]), string(trs[i][4]), string(trs[i][5]), string(trs[i][6]), string(trs[i][7])
		info := ProxyServerInfo{}

		style_map := map[string]int{"http": 1, "https": 2, "socket": 3}

		info.host = ip
		info.port = port
		h := hash(fmt.Sprintf("%s:%s",ip,port))
		info.rate, _ = strconv.ParseFloat(rate, 64)
		info.anonymous = (anonymous == "高匿")
		info.style = style_map[strings.ToLower(style)]
		info.area = strings.ToLower(area)
		sp.Rows[h] = &info
		count++
	}
	if count <= 5 {
		SpiderLoger.E("proxy servers only ", count)
	}
	SpiderLoger.I("The proxy server count", count)
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
	if count == 0 {
		sp.Rows = make(map[uint32]*ProxyServerInfo)
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
			count++
		}
	}
	if count <= 5 {
		SpiderLoger.E("proxy servers only ", count)
	}
	SpiderLoger.I("The proxy server count", count)
	return
}
