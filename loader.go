package spider

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	utils "libs/utils"
	"net/http"
	"compress/gzip"
	"net/url"
	"io"
	"time"
	"net"
	"strconv"
	"strings"
)

type Loader struct {
	client    *http.Client
	request   *http.Request
	transport *http.Transport
	data      url.Values
	rheader   http.Header
	url       string
	method    string
	useProxy  bool
	mheader   map[string]string
}



func NewLoader() *Loader {
	transport := NewTransPort(30)
	l := &Loader{
		transport: transport,
		useProxy:  true,
		mheader: map[string]string{
			"Accept-Charset":"utf-8",
			"Accept-Encoding": "gzip, deflate",
			"Content-Type": "application/x-www-form-urlencoded",
			"Connection":"close",
		},
	}

	time.AfterFunc(time.Duration(30)*time.Second, func() {
		l.Close()
		recover()
	})
	l.MobildAgent()
	return l
}

func NewTransPort(timeout int) *http.Transport{
	duration := time.Duration(timeout) * time.Second
	transport :=  &http.Transport{
		TLSClientConfig: &tls.Config{MaxVersion: tls.VersionTLS10, InsecureSkipVerify: true},
		Dial: func(netw, addr string) (net.Conn, error) { 
			deadline := time.Now().Add(duration)
			c, err := net.DialTimeout(netw, addr, duration) 
			if err != nil { 
				SpiderLoger.E("http transport dail timeout", err) 
		 		return nil, err 
			} 
			c.SetDeadline(deadline)
		    return c, nil 
		}, 
		DisableKeepAlives:true,
		// MaxIdleConnsPerHost:10, 
		ResponseHeaderTimeout: duration, 
	}
	return transport
}

func (l *Loader) WithProxy(val bool) *Loader {
	l.useProxy = val
	return l
}

func (l *Loader) MobildAgent() *Loader {
	agents := []string{
		"Mozilla/5.0 (Linux; U; Android 4.0.2; en-us; Galaxy Nexus Build/ICL53F) AppleWebKit/534.30 (KHTML, like Gecko) Version/4.0 Mobile Safari/534.30",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 5_0_1 like Mac OS X) AppleWebKit/534.46 (KHTML, like Gecko)",
		"Mozilla/5.0 (iPad; U; CPU OS 4_3_3 like Mac OS X; en-us) AppleWebKit/533.17.9 (KHTML, like Gecko)",
		"Mozilla/5.0 (Linux; U; Android 2.3.5; zh-cn; MI-ONE Plus Build/GINGERBREAD) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
		"Mozilla/5.0 (Linux; U; Android 2.3.3; zh-cn; HTC_WildfireS_A510e Build/GRI40) AppleWebKit/533.1 (KHTML, like Gecko) Version/4.0 Mobile Safari/533.1",
	}
	num := utils.RandInt(0, len(agents)-1)
	l.SetHeader("User-Agent", agents[num])
	return l
}

func (l *Loader) WithPcAgent() *Loader {
	agents := []string{
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/40.0.2214.115 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/36.0.1941.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/37.0.2062.94 Chrome/37.0.2062.94 Safari/537.36",
		"Mozilla/5.0 (Windows; U; Windows NT 5.2) Gecko/2008070208 Firefox/3.0.1",
		"Mozilla/5.0 (Windows; U; Windows NT 5.2) AppleWebKit/525.13 (KHTML, like Gecko) Version/3.1 Safari/525.13",
		"Mozilla/5.0 (Windows; U; Windows NT 5.2) AppleWebKit/525.13 (KHTML, like Gecko) Chrome/0.2.149.27 Safari/525.13",
	}
	num := utils.RandInt(0, len(agents)-1)
	l.SetHeader("User-Agent", agents[num])
	return l
}

func (l *Loader) CheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	return nil
}

func (l *Loader) GetRequest() {
	if l.method == "POST" {
		l.request, _ = http.NewRequest(l.method, l.url, strings.NewReader(l.data.Encode()))
	} else {
		l.request, _ = http.NewRequest(l.method, l.url, nil)
	}
	l.request.Close = true

	//set headers
	l.header()
	return
}

func (l *Loader) Close() {
	if l.transport != nil {
		SpiderLoger.D("Loader closed request [", l.url, "]")
		l.transport.CloseIdleConnections()
		l.transport.CancelRequest(l.request)
	}
	return
}

func (l *Loader) Send(urlStr, method string, data url.Values) ([]byte, error) {
	l.url = urlStr
	l.method = strings.ToUpper(method)
	l.data = data
	proxy_addr:= "none"
	if l.useProxy {
		proxyServerInfo := SpiderProxy.GetProxyServer()
		fmt.Println(proxyServerInfo)
		if proxyServerInfo != nil {
			proxyUrl, _ := url.Parse(fmt.Sprintf("http://%s:%s", proxyServerInfo.host, proxyServerInfo.port))
			l.transport.Proxy = http.ProxyURL(proxyUrl)
			proxy_addr = fmt.Sprintf("%s:%s", proxyServerInfo.host, proxyServerInfo.port)
		}
	}

	l.client = &http.Client{
		CheckRedirect: l.CheckRedirect,
		Transport:     l.transport,
	}

	l.GetRequest()
	resp, err := l.client.Do(l.request)
	if err != nil{
		return nil, err
	}
	defer resp.Body.Close()

	SpiderLoger.D(fmt.Sprintf("[%d] Loader [%s] with proxy[%s].", resp.StatusCode,l.url, proxy_addr))

	if resp.StatusCode != 200 {
		return nil, err
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
		case "gzip":
			reader, err = gzip.NewReader(resp.Body)
			if err != nil {
				return nil, err
			}
			defer reader.Close()
		default:
			reader = resp.Body
	}

	body, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	l.rheader = resp.Header
	return body, nil
}

func (l *Loader) SetHeader(key, value string) {
	l.mheader[key] = value
}

func (l *Loader) header() {
	l.request.Close = true
	if l.method == "POST" {
		l.request.Header.Add("Content-Length", strconv.Itoa(len(l.data.Encode())))
	}
	for h, v := range l.mheader {
		l.request.Header.Set(h, v)
	}
}
