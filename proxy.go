package spider

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
	"encoding/json"
	"math/rand"
	"container/list"
)

type ProxyInfo struct {
	hit        int    //network speed
	area       string //region
	host       string
	style      int    //1 http 2 https 3 socket
	index      int    //1 http 2 https 3 socket
	status     bool   //region
	anonymous  bool   //0 transparent 1 low 2 high
	last_check int64  //timestamp last check time
}

type Proxy struct {
	Rows  map[uint32]*ProxyInfo
	Tbao  []*ProxyInfo
	New   []*ProxyInfo
	Hhui  []*ProxyInfo
	Count int
}

var (
	SpiderProxy *Proxy
	proxyUrl = "http://proxy.tebiere.com/api/fetch/list?key=OxTNiiS9PjlWIDD1KEgU71ZjZQHNxh&num=5000&port=&check_country_group%5B0%5D=1&check_http_type%5B0%5D=1&check_http_type%5B1%5D=2&check_http_type%5B2%5D=4&check_anonymous%5B0%5D=3&check_elapsed=1&check_upcount=500&result_sort_field=3&result_format=json"
	count = 0
	has_proxy = false
	proxyList     *list.List = list.New()
	proxyListLock bool = false
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
		sp.getProxyList(proxyUrl, true)
		for {
			select {
			case <-tick_get.C:
			//sp.getProxyServer()
				sp.getProxyList(proxyUrl, false)
			case <-tick_check.C:
				sp.Check()
			}
		}
	}()
}
func (sp *Proxy)  DelProxyServer(index uint32) {
	SpiderLoger.D("delete proxyserver", index)
	delete(sp.Rows, index)
}

func (sp *Proxy) DelRow(index int) {
	SpiderLoger.D("delete proxyserver", index)
	count := len(sp.Tbao)

	if (count <= 1) {
		sp.Tbao = sp.Tbao[:0]
	}
	if (index >= count - 1 ) {
		sp.Tbao = sp.Tbao[:(count - 2)]
		return
	}
	if index == 0 {
		sp.Tbao = sp.Tbao[1:]
		return
	}

	sp.Tbao = append(sp.Tbao[:(index - 1)], sp.Tbao[(index + 1):]...)
	return
}

func getRandIndex(len int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(len)
}

func (pi *ProxyInfo)AddHit() {
	pi.hit = pi.hit + 1
}

func (sp *Proxy) GetProxyServer() *ProxyInfo {
	count := len(sp.Tbao)
	if count == 0 {
		return nil
	}
	info := &ProxyInfo{}
	proxy_index := getRandIndex(count)
	proxy_index = 0
	info = sp.Tbao[proxy_index]
	info.index = proxy_index
	return info
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

	jobs := make(chan *ProxyInfo, 10)
	ch := make(chan bool, 100)
	done := make(chan bool)

	go func() {
		for {
			j, more := <-jobs
			if more {
				ch <- true
				go j.CheckAli(ch)
				//fmt.Println("received jobs",j.host,j.port)
			} else {
				time.Sleep(time.Second * 5)
				SpiderLoger.I("End checking proxys count[", len(sp.Tbao), "]")
				sp.Tbao = []*ProxyInfo{}
				for k, i := range sp.Rows {
					if i.status {
						sp.Tbao = append(sp.Tbao, i)
					} else {
						sp.DelProxyServer(k)
					}
				}
				SpiderLoger.I("End checking proxys count[", len(sp.Tbao), "]")
				//fmt.Println("received all jobs")
				done <- true
				return
			}
		}
	}()

	for _, i := range sp.Rows {
		jobs <- i
		//fmt.Println("sent job", i)
	}
	close(jobs)
	//	fmt.Println("sent all jobs")
	//We await the worker using the synchronization approach we saw earlier.
	<-done

}

func CheckByAli(host string) bool {

	testUrl := "https://err.taobao.com/error1.html"
	_, body, err := NewLoader().WithPcAgent().SetProxyServer(host).Get(testUrl)
	if err != nil {
		//SpiderLoger.I("[Proxy.GetApiProxyList] check ", host, MyColor("red"), "failed", MyColor("none"))
		return false
	}
	if !strings.Contains(string(body), "alibaba.com") {
		//SpiderLoger.I("[Proxy.GetApiProxyList] check ", host, MyColor("red"), "failed", MyColor("none"))
		return false
	}
	SpiderLoger.I("[Proxy.GetApiProxyList] check ", host, MyColor("green"), "OK", MyColor("none"))

	return true
}

func CheckByAliX(host string) bool {

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
	SpiderLoger.I("[Proxy.GetApiProxyList] check ", host, MyColor("green"), "OK")

	return true
}
func CheckByHuiHui(ip string, port string) bool {

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

func (i *ProxyInfo) CheckAli(ch chan bool) bool {

	if (time.Now().Unix() - i.last_check) < 30 * 60 {
		if i.status {
			<-ch
			return true
		}
		<-ch
		return false
	}

	i.last_check = time.Now().Unix()

	if CheckByAli(i.host) {
		<-ch
		return true
	} else {
		<-ch
		return false
	}
}

func (sp *Proxy) getProxyList(proxyUrl string, isFirst bool) {

	_, body, err := NewLoader().WithPcAgent().Get(proxyUrl)
	if err != nil {
		SpiderLoger.E("Proxy.GetApiProxyList", proxyUrl)
		return
	}
	byteContent := make([]byte, len(body))
	copy(byteContent, body)
	result := make(map[string]interface{})
	err = json.Unmarshal(byteContent, &result)
	if err != nil {
		SpiderLoger.E("[Proxy.GetApiProxyList]", err.Error())
		return
	}

	success, ok := result["success"].(bool);
	if !ok || success == false {
		SpiderLoger.E("[Proxy.GetApiProxyList] json parse error")
		return
	}

	proxyIpList, _ := result["list"].([]interface{})

	ch := make(chan bool, 100)
	SpiderLoger.I("[Proxy.GetApiProxyList] get proxy list")
	for _, val := range proxyIpList {
		ch <- true
		tmp := val.(map[string]interface{})
		host := tmp["ip:port"].(string)
		go func() {
			defer func() {
				<-ch
			}()
			if CheckByAli(host) {
				row := &ProxyInfo{host:host, status: true}
				if (isFirst) {
					sp.Tbao = append(sp.Tbao, row)

				}else {
					sp.New = append(sp.New, row)
				}
				if (has_proxy == false) {
					has_proxy = true
					SpiderLoger.I("[Proxy.GetApiProxyList] add first ", row.host)
				}
			}

			if (len(sp.New) >= 10) {
				SpiderLoger.I("The proxy server count ", len(sp.New))
				sp.Tbao = append(sp.Tbao, sp.New...)
				sp.New = sp.New[:0]
			}
		}()
	}

}

func (loader *Loader) getApiProxyList() {
	if proxyListLock == true {
		return
	}
	if proxyList.Len() > 20 {
		return
	}

	proxyListLock = true
	time.AfterFunc(time.Duration(30) * time.Second, func() {
		proxyListLock = false
	})

	_, body, err := loader.Get(ApiProxyURL)
	if err != nil {
		SpiderLoger.E("[Loader.GetApiProxyList] ", err.Error())
		return
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		SpiderLoger.E("[Loader.GetApiProxyList]", err.Error())
		return
	}

	if success, ok := result["success"].(bool); ok {
		if success == true {
			iplist, _ := result["data"].([]interface{})
			for _, val := range iplist {
				ipport, _ := val.(map[string]interface{})
				proxyList.PushBack(ipport["ip:port"].(string))
			}
		}
		SpiderLoger.I("[Loader.GetApiProxyList] load with", proxyList.Len(), "proxy")
	} else {
		SpiderLoger.E("[Loader.GetApiProxyList] json parse error")
		return
	}
}
func (loader *Loader) getProxyServer() *list.Element {
	if el := proxyList.Front(); el != nil {
		proxyList.MoveToBack(el)
	}
	return proxyList.Front()
}
func (sp *Proxy) kuai(proxyUrl string) {
	_, content, err := NewLoader().WithPcAgent().Get(proxyUrl)
	fmt.Println(string(content))
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
		sp.Rows = make(map[uint32]*ProxyInfo)
	}

	for i := 0; i < l; i++ {
		if string(trs[i][3]) != "高匿名" {
			continue
		}
		ip, port := string(trs[i][1]), string(trs[i][2])
		host := fmt.Sprintf("%s:%s", ip, port)
		go func() {
			if CheckByAli(host) {
				fmt.Println("%s:%s", ip, port)
				row := &ProxyInfo{host: host, status: true}
				sp.Tbao = append(sp.Tbao, row)
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
