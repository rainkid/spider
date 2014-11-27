package spider

import (
	"bytes"
	"fmt"
	iconv "github.com/qiniu/iconv"
	"regexp"
)

type HtmlParse struct {
	url      string
	content  []byte
	partten  string
	replaces [][]string
}

func NewHtmlParse() *HtmlParse {
	return &HtmlParse{
		replaces: [][]string{
			{`\s+`, " "},           //过滤多余回车
			{`<[ ]+`, "<"},         //过滤<__("<"号后面带空格)
			{`<\!–.*?–>`, ""},      // //注释
			{`<(\!.*?)>`, ""},      //过滤DOCTYPE
			{`<(\/?html.*?)>`, ""}, //过滤html标签
			{`<(\/?br.*?)>`, ""},   //过滤br标签
			{`<(\/?head.*?)>`, ""}, //过滤head标签
			// {`<(\/?meta.*?)>`, ""},                    //过滤meta标签
			{`<(\/?body.*?)>`, ""},                    //过滤body标签
			{`<(\/?link.*?)>`, ""},                    //过滤link标签
			{`<(\/?form.*?)>`, ""},                    //过滤form标签
			{`<(applet.*?)>(.*?)<(\/applet.*?)>`, ""}, //过滤applet标签
			{`<(\/?applet.*?)>`, ""},
			{`<(style.*?)>(.*?)<(\/style.*?)>`, ""}, //过滤style标签
			{`<(\/?style.*?)>`, ""},
			// {`<(title.*?)>(.*?)<(\/title.*?)>`, ""}, //过滤title标签
			// {`<(\/?title.*?)>`, ""},
			{`<(object.*?)>(.*?)<(\/object.*?)>`, ""}, //过滤object标签
			{`<(\/?objec.*?)>`, ""},
			{`<(noframes.*?)>(.*?)<(\/noframes.*?)>`, ""}, //过滤noframes标签
			{`<(\/?noframes.*?)>`, ""},
			{`<(i?frame.*?)>(.*?)<(\/i?frame.*?)>`, ""},   //过滤frame标签
			{`<(noscript.*?)>(.*?)<(\/noscript.*?)>`, ""}, //过滤noframes标签
			// {`on([a-z]+)\s*="(.*?)"`, ""},                 //过滤dom事件
			// {`on([a-z]+)\s*='(.*?)'`, ""},
		},
	}
}

func (hp *HtmlParse) CleanScript() *HtmlParse {
	hp.replaces = append(hp.replaces, []string{`<(script.*?)>(.*?)<(\/script.*?)>`, ""})
	hp.replaces = append(hp.replaces, []string{`<(\/?script.*?)>`, ""})
	return hp
}

func (hp *HtmlParse) IsGbk() bool {
	d := bytes.ToLower(hp.content)
	if bytes.Index(d, []byte(`text/html; charset=gbk`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`text/html; charset="gbk"`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`text/html; charset=gb2312`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`text/html; charset="gb2312"`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`charset=gbk`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`charset="gbk"`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`charset="gb2312"`)) > 0 {
		return true
	}
	if bytes.Index(d, []byte(`charset=gb2312`)) > 0 {
		return true
	}
	return false
}

func (hp *HtmlParse) Convert() *HtmlParse {
	cd, err := iconv.Open("UTF-8//IGNORE", "GB2312")
	if err != nil {
		fmt.Println("iconv.Open failed!")
		return hp
	}
	defer cd.Close()

	hp.content = []byte(cd.ConvString(fmt.Sprintf("%s", hp.content)))
	return hp
}

func (hp *HtmlParse) LoadData(content []byte) *HtmlParse {
	hp.content = content
	return hp
}

func (hp *HtmlParse) Replace() *HtmlParse {
	length := len(hp.replaces)
	for i := 0; i < length; i++ {
		if l := len(hp.replaces[i]); l > 0 {
			p, r := hp.replaces[i][:1], hp.replaces[i][1:2]
			hp.content = regexp.MustCompile(p[0]).ReplaceAll(hp.content, []byte(r[0]))
		}
	}
	return hp
}

func (hp *HtmlParse) Partten(p string) *HtmlParse {
	hp.partten = p
	return hp
}

func (hp *HtmlParse) FindStringSubmatch() [][]byte {
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindSubmatch(hp.content)
}

func (hp *HtmlParse) FindSubmatch(tagName string) [][]byte {
	hp.partten = fmt.Sprintf(`((?U)<%s+.*>(.*)</%s>).*?`, tagName, tagName)
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindSubmatch(hp.content)
}

func (hp *HtmlParse) FindAllSubmatch() [][][]byte {
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindAllSubmatch(hp.content, -1)
}

func (hp *HtmlParse) FindByAttr(tagName, attr, value string) [][][]byte {
	hp.partten = fmt.Sprintf(`((?U)<%s+.*%s=['"]%s['"]+.*>(.*)</%s>).*?`, tagName, attr, value, tagName)
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindAllSubmatch(hp.content, -1)
}

func (hp *HtmlParse) FindByTagName(tagName string) [][][]byte {
	hp.partten = fmt.Sprintf(`((?U)<%s+.*>(.*)</%s>).*?`, tagName, tagName)
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindAllSubmatch(hp.content, -1)
}

func (hp *HtmlParse) FindJsonStr(nodeName string) [][][]byte {
	hp.partten = fmt.Sprintf(`(?U)"%s":\s*?['"](.*)['"]`, nodeName)
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindAllSubmatch(hp.content, -1)
}

func (hp *HtmlParse) FindJsonInt(nodeName string) [][][]byte {
	hp.partten = fmt.Sprintf(`(?U)"%s":(.*),`, nodeName)
	re := regexp.MustCompile(hp.partten)
	// fmt.Println(re.String())
	return re.FindAllSubmatch(hp.content, -1)
}
