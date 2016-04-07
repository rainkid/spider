package spider

import (
	"compress/gzip"
	"crypto/tls"
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
	skey = "OxTNiiS9PjlWIDD1KEgU71ZjZQHNxh"
	ApiProxyURL = "http://proxy.gouwudating.cn/api/fetch/list?key=OxTNiiS9PjlWIDD1KEgU71ZjZQHNxh&total=1000&port=&check_country_group%5B0%5D=0&check_http_type%5B0%5D=0&check_anonymous%5B0%5D=0&check_elapsed=0&check_upcount=0&result_sort_field=1&result_format=json"
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

type Loader struct {
	Runing    int
	proxy     string
	proxyInfo ProxyInfo
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
	num := utils.RandInt(0, len(mobileUserAgentS) - 1)
	loader.myHeader["User-Agent"] = mobileUserAgentS[num]
	return loader
}

func (loader *Loader) WithProxy() *Loader {

	proxyInfo := SpiderProxy.GetProxyServer()
	if proxyInfo != nil {
		loader.proxy = proxyInfo.host
		loader.useProxy = true
	}
	return loader
}

func (loader *Loader) WithPcAgent() *Loader {
	num := utils.RandInt(0, len(pcUserAgentS) - 1)
	loader.myHeader["User-Agent"] = pcUserAgentS[num]
	return loader
}

func (loader *Loader) SetProxyServer(host string) *Loader {
	loader.useProxy = true
	loader.proxy = host
	return loader
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
			return net.DialTimeout(network, addr, time.Second * 15)
		},
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: true,
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
	if (loader.proxy!="") {
		proxyUrl, _ := url.Parse(fmt.Sprintf("http://%s", loader.proxy))
		loader.transport.Proxy = http.ProxyURL(proxyUrl)
	}

	client := &http.Client{
		Transport: loader.transport,
	}

	request := loader.getRequest(target, method, data)
	resp, err := client.Do(request)
	if err != nil {
		loader.Runing--
		return nil, nil, err
	}
	defer resp.Body.Close()

	if loader.useProxy {
		SpiderLoger.D("[Loader->", resp.StatusCode, "][proxy: " + loader.proxy + "] Load [", target, "]")
	} else {
		SpiderLoger.D("[Loader->", resp.StatusCode, "] Load [", target, "]")
	}
	proxy := NewProxy();
	if resp.StatusCode != 200 {
		loader.Runing--
		if (loader.useProxy)&&resp.StatusCode != 404 {
			proxy.DelRow(loader.proxyInfo.index)
		}
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
	//add hit 4 proxy
	if (loader.useProxy) {
		loader.proxyInfo.AddHit()
	}
	loader.Runing--
	return resp, body, nil
}