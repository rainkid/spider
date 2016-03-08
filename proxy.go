package spider

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"time"
	"encoding/json"
)

type ProxyServerInfo struct {
	host       string
	rate       float64 //network speed
	area       string  //region
	style      int     //1 http 2 https 3 socket
	status     bool    //region
	anonymous  bool    //0 transparent 1 low 2 high
	last_check int64   //timestamp last check time
}

type Proxy struct {
	Rows  map[uint32]*ProxyServerInfo
	Tbao  []*ProxyServerInfo
	New   []*ProxyServerInfo
	Hhui  []*ProxyServerInfo
	Count int
}

var (
	SpiderProxy *Proxy
	proxyUrl = "http://proxy.gouwudating.cn/api/fetch/list?key=OxTNiiS9PjlWIDD1KEgU71ZjZQHNxh&num=5000&port=80%2C8080%2C8088%2C8888%2C8899&check_country_group%5B0%5D=1&check_http_type%5B0%5D=1&check_http_type%5B1%5D=2&check_anonymous%5B0%5D=3&check_elapsed=0&check_upcount=0&result_sort_field=2&check_result_fields%5B0%5D=2&check_result_fields%5B1%5D=3&check_result_fields%5B2%5D=4&check_result_fields%5B3%5D=5&check_result_fields%5B4%5D=6&check_result_fields%5B5%5D=7&result_format=json"
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

	tick_get := time.NewTicker(60 * 60 * time.Second)
	tick_check := time.NewTicker(120 * 60 * time.Second)

	go func() {
		//		sp.getProxyServer()
		sp.getProxyList(proxyUrl, true)
		for {
			select {
			case <-tick_get.C:
			//				sp.getProxyServer()
				sp.getProxyList(proxyUrl, false)
			case <-tick_check.C:
				sp.Check()
			}
		}
	}()
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
	info := &ProxyServerInfo{}
	for _, item := range sp.Tbao {
		info = item
		break;
	}
	return info
}
func timer() {
	t := time.Tick(time.Second * 30)
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
	for i := 1; i < 10; i++ {
		sp.kuai(fmt.Sprintf("http://www.kuaidaili.com/free/inha/%d/", i))
	}
}

func (sp *Proxy) Check() {
	return
	count := len(sp.Rows)
	SpiderLoger.I("Start checking proxys")
	if count < 500 {
		return
	}

	jobs := make(chan *ProxyServerInfo, 10)
	ch := make(chan bool, 100)
	done := make(chan bool)

	go func() {
		for {
			j, more := <-jobs
			if more {
				ch <- true
				go j.CheckTaobao(ch)
				//				fmt.Println("received jobs",j.host,j.port)
			} else {
				time.Sleep(time.Second * 5)
				SpiderLoger.I("End checking proxys count[", len(sp.Tbao), "]")
				sp.Tbao = []*ProxyServerInfo{}
				for k, i := range sp.Rows {
					if i.status {
						sp.Tbao = append(sp.Tbao, i)
					} else {
						sp.DelProxyServer(k)
					}
				}
				SpiderLoger.I("End checking proxys count[", len(sp.Tbao), "]")
				//				fmt.Println("received all jobs")
				done <- true
				return
			}
		}
	}()

	for _, i := range sp.Rows {
		jobs <- i
		//		fmt.Println("sent job", i)
	}
	close(jobs)
	//	fmt.Println("sent all jobs")
	//We await the worker using the synchronization approach we saw earlier.
	<-done

}




func ChkByTbao(host string) bool {

	var timeout = time.Duration(30 * time.Second)
	proxyUrl, _ := url.Parse(fmt.Sprintf("http://%s", host))
	//	url_proxy := &url.URL{Host: host}
	client := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
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
func ChkByHhui(ip string, port string) bool {

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

func (i *ProxyServerInfo) CheckTaobao(ch chan bool) bool {

	if (time.Now().Unix() - i.last_check) < 30 * 60 {
		if i.status {
			<-ch
			return true
		}
		<-ch
		return false
	}

	i.last_check = time.Now().Unix()

	if ChkByTbao(i.host) {
		<-ch
		return true
	} else {
		<-ch
		return false
	}
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (sp *Proxy) getProxyList(proxyUrl string, isFirst bool) {

	_, body, err := NewLoader().WithPcAgent().Get(proxyUrl)
	if err != nil {
		SpiderLoger.E("Proxy.GetApiProxyList", proxyUrl)
		return
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		SpiderLoger.E("[Proxy.GetApiProxyList]", err.Error())
		return
	}

	success, ok := result["success"].(bool);
	if !ok {
		SpiderLoger.E("[Proxy.GetApiProxyList] json parse error")
		return
	}
	if success == true {
		iplist, _ := result["list"].([]interface{})
		ch := make(chan bool, 10)
		if isFirst {
			for _, val := range iplist {
				tmp := val.(map[string]interface{})
				host := tmp["ip:port"].(string)

				ch<-true
				go func() {
					defer func() { <-ch }()
					if ChkByTbao(host) {
						row := &ProxyServerInfo{host:host, status: true}
						sp.Tbao = append(sp.Tbao, row)
					}
				}()
			}
		} else {
			for _, val := range iplist {
				ch<-true
				tmp := val.(map[string]interface{})
				host := tmp["ip:port"].(string)
				go func() {
					defer func() { <-ch }()
					if ChkByTbao(host) {
						row := &ProxyServerInfo{host:host, status: true}
						sp.New = append(sp.New, row)
					}
					if (len(sp.New) >= 1000) {
						SpiderLoger.I("The proxy server count ", len(sp.New))
						sp.Tbao = sp.New
						sp.New = sp.New[:0]
					}
				}()
			}
		}

	}
	//	SpiderLoger.I("[Proxy.GetApiProxyList] load with", sp.Tbao, "proxy")

}
func (sp *Proxy) kuai(proxyUrl string) {
	_, content, err := NewLoader().WithPcAgent().Get(proxyUrl)
	if err != nil {
		SpiderLoger.E("Load proxy error with", proxyUrl)
		return
	}

	m := make([]byte, len(content))
	copy(m, content)

	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(m)
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
		host := fmt.Sprintf("%s:%s", ip, port)
		h := hash(host)
		_, ok := sp.Rows[h]
		if ok {
			continue
		}
		go func() {
			if ChkByTbao(host) {
				fmt.Println("%s:%s", ip, port)
				sp.Rows[h] = &ProxyServerInfo{host: host, status: true}
				sp.Tbao = append(sp.Tbao, sp.Rows[h])
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

func (sp *Proxy) Load(proxyUrl string) {

	_, content, err := NewLoader().WithPcAgent().Send(proxyUrl, "GET", nil)
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
