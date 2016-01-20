package spider

import (
	"compress/gzip"
	"container/list"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	utils "libs/utils"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	skey             = "OxTNiiS9PjlWIDD1KEgU71ZjZQHNxh"
	ApiProxyURL      = "http://proxy.gouwudating.cn/api/fetch/list?key=OxTNiiS9PjlWIDD1KEgU71ZjZQHNxh&total=1000&port=&check_country_group%5B0%5D=0&check_http_type%5B0%5D=0&check_anonymous%5B0%5D=0&check_elapsed=0&check_upcount=0&result_sort_field=1&result_format=json"
	mobileUserAgentS = []string{
		"Mozilla/5.0 (Linux; U; Android 4.0.2; en-us; Galaxy Nexus Build/ICL53F) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 5_0_1 like Mac OS X) AppleWebKit/534.46 (KHTML, like Gecko)",
		"Mozilla/5.0 (iPad; U; CPU OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko)",
		"Mozilla/5.0 (Linux; U; Android 2.3.5; zh-cn; MI-ONE Plus Build/GINGERBREAD) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
		"Mozilla/5.0 (Linux; U; Android 2.3.3; zh-cn; HTC_WildfireS_A510e Build/GRI40) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
	}
	pcUserAgentS = []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.8; rv:21.0) Gecko/20100101 Firefox/21.0",
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:21.0) Gecko/20130331 Firefox/21.0",
		"Mozilla/5.0 (Windows NT 6.2; WOW64; rv:21.0) Gecko/20100101 Firefox/21.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/27.0.1453.93 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/535.11 (KHTML, like Gecko) Ubuntu/11.10 Chromium/27.0.1453.93 Chrome/27.0.1453.93 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/27.0.1453.94 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.2; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/27.0.1453.94 Safari/537.36",
	}
)

var (
	proxyList     *list.List = list.New()
	proxyListLock bool       = false
)

type Loader struct {
	Runing    int
	proxyInfo *list.Element
	useProxy  bool
	transport *http.Transport
	myHeader  map[string]string
}

func NewLoader() *Loader {
	loader := &Loader{
		myHeader: map[string]string{
			"Accept-Charset":  "utf-8",
			"Accept-Encoding": "gzip",
			"Content-Type":    "application/x-www-form-urlencoded",
			"Connection":      "close",
		},
		useProxy: false,
	}
	loader.WithMobileAgent()
//	loader.getApiProxyList()
	return loader
}

func (loader *Loader) SetHeader(head, value string) *Loader {
	loader.myHeader[head] = value
	return loader
}

func (loader *Loader) WithMobileAgent() *Loader {
	num := utils.RandInt(0, len(mobileUserAgentS)-1)
	loader.myHeader["User-Agent"] = mobileUserAgentS[num]
	return loader
}

func (loader *Loader) WithProxy() *Loader {
	loader.useProxy = true
	return loader
}

func (loader *Loader) WithPcAgent() *Loader {
	num := utils.RandInt(0, len(pcUserAgentS)-1)
	loader.myHeader["User-Agent"] = pcUserAgentS[num]
	return loader
}



func (loader *Loader) getProxyServer() *list.Element {
	if el := proxyList.Front(); el != nil {
		proxyList.MoveToBack(el)
	}
	return proxyList.Front()
}

func (loader *Loader) getApiProxyList() {
	if proxyListLock == true {
		return
	}
	if proxyList.Len() > 20 {
		return
	}

	proxyListLock = true
	time.AfterFunc(time.Duration(30)*time.Second, func() {
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

func (loader *Loader) getRequest(target, method string, data url.Values) *http.Request {
	var request *http.Request
	if strings.ToUpper(method) == "POST" {
		encodeData := data.Encode()
		request, _ = http.NewRequest(method, target, strings.NewReader(encodeData))
		request.Header.Add("Content-Length", strconv.Itoa(len(encodeData)))
	} else {
		request, _ = http.NewRequest(method, target, nil)
	}
	request.Close = true

	for h, v := range loader.myHeader {
		request.Header.Set(h, v)
	}
	return request
}

func (loader *Loader) getTransport() *http.Transport {
	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, time.Second*15)
		},
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
	}

	if loader.useProxy == true {
//		loader.proxyInfo = loader.getProxyServer()
//		if loader.proxyInfo != nil {
//			proxy := fmt.Sprintf("http://%s", loader.proxyInfo.Value.(string))
//			proxyUrl, _ := url.Parse(proxy)
//			transport.Proxy = http.ProxyURL(proxyUrl)
//		}

		proxyServerInfo := SpiderProxy.GetProxyServer()
		if proxyServerInfo != nil {
			proxyUrl, _ := url.Parse(fmt.Sprintf("http://%s:%s", proxyServerInfo.host, proxyServerInfo.port))
			transport.Proxy = http.ProxyURL(proxyUrl)
		}
	}
	return transport
}

func (loader *Loader) SetTransport(transport *http.Transport) *Loader {
	loader.transport = transport
	return loader
}

func (loader *Loader) Get(target string) (*http.Response, []byte, error) {
	return loader.Send(target, "GET", nil)
}

func (loader *Loader) Post(target string, data url.Values) (*http.Response, []byte, error) {
	return loader.Send(target, "POST", data)
}

func (loader *Loader) Send(target, method string, data url.Values) (*http.Response, []byte, error) {
	loader.Runing++
	loader.transport = loader.getTransport()

	client := &http.Client{
		Transport: loader.transport,
	}

//	utime := int32(time.Now().Unix())
//	if strings.Contains(target, "?") {
//		target += fmt.Sprintf("&t=%d", utime)
//	} else {
//		target += fmt.Sprintf("?t=%d", utime)
//	}
	request := loader.getRequest(target, method, data)
	resp, err := client.Do(request)
	if err != nil {
		loader.Runing--
		return nil, nil, err
	}
	defer resp.Body.Close()

	if loader.useProxy {
		SpiderLoger.D("[Loader.Send][", resp.StatusCode, "] Loader [", target, "] with proxy")
	} else {
		SpiderLoger.D("[Loader.Send][", resp.StatusCode, "] Loader [", target, "]")
	}

	if resp.StatusCode != 200 {
		loader.Runing--
//		if loader.proxyInfo != nil {
//			proxyList.Remove(loader.proxyInfo)
//		}
		return resp, nil, err
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			loader.Runing--
			return nil, nil, err
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	body, err := ioutil.ReadAll(reader)

	if err != nil {
		loader.Runing--
		return nil, nil, err
	}
	loader.Runing--
	return resp, body, nil
}