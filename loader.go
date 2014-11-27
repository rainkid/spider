package spider

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	utils "libs/utils"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Loader struct {
	client    *http.Client
	req       *http.Request
	resp      *http.Response
	data      url.Values
	redirects int64
	rheader   http.Header
	url       string
	method    string
	useProxy  bool
	mheader   map[string]string
}

func NewLoader(url, method string) *Loader {
	l := &Loader{
		redirects: 0,
		url:       url,
		useProxy:  true,
		method:    strings.ToUpper(method),
		mheader: map[string]string{
			"Content-Type": "application/x-www-form-urlencoded",
		},
	}
	l.MobildAgent()
	return l
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
	if len(via) >= 20 {
		return errors.New("stopped after 20 redirects")
	}
	l.redirects++
	return nil
}

func (l *Loader) Sample() ([]byte, error) {
	resp, err := http.Get(l.url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	l.rheader = resp.Header
	return body, nil
}

func (l *Loader) GetResp() (*http.Response, error) {
	if l.method == "POST" {
		l.req, _ = http.NewRequest(l.method, l.url, strings.NewReader(l.data.Encode()))
	} else {
		l.req, _ = http.NewRequest(l.method, l.url, nil)
	}
	l.req.Close = true

	//set headers
	l.header()
	return l.client.Do(l.req)
}

func (l *Loader) Send(v url.Values) ([]byte, error) {
	l.data = v

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{MaxVersion: tls.VersionTLS10, InsecureSkipVerify: true},
	}

	if l.useProxy {
		host, port := SpiderProxy.GetProxyServer()
		if host != nil && port != nil {
			proxyUrl, _ := url.Parse(fmt.Sprintf("http://%s:%s", host, port))
			transport.Proxy = http.ProxyURL(proxyUrl)
			SpiderLoger.D("load with proxy", proxyUrl.String())
		}
	}
	SpiderLoger.D(l.url)
	l.client = &http.Client{
		CheckRedirect: l.CheckRedirect,
		Transport:     transport,
	}

	resp, err := l.GetResp()
	if err != nil {
		return nil, err
	}
	l.resp = resp

	defer l.resp.Body.Close()
	body, err := ioutil.ReadAll(l.resp.Body)
	if err != nil {
		return nil, err
	}
	l.rheader = l.resp.Header
	return body, nil
}

func (l *Loader) GetHeader() http.Header {
	return l.rheader
}

func (l *Loader) SetHeader(key, value string) {
	l.mheader[key] = value
}

func (l *Loader) header() {
	l.req.Close = true
	if l.method == "POST" {
		l.req.Header.Add("Content-Length", strconv.Itoa(len(l.data.Encode())))
	}
	for h, v := range l.mheader {
		l.req.Header.Set(h, v)
	}
}
