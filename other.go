package spider

import (
	"bytes"
	"errors"
	"fmt"
)

type Other struct {
	content []byte
}

func (ti *Other) Get(item *Item) {
	//get content
	resp, content, err := NewLoader().Get(item.params["link"])

	if err != nil {
		item.err = err
		SpiderServer.qerror <- item
		return
	}
	ti.content = make([]byte, len(content))
	copy(ti.content, content)

	htmlParser := NewHtmlParser()

	htmlParser.LoadData(ti.content).CleanScript().Replace()
	ct := []byte(resp.Header.Get("Content-Type"))
	ct = bytes.ToLower(ct)

	var needconv bool = true
	if bytes.Index(ct, []byte("utf-8")) > 0 {
		needconv = false
	}

	if needconv && bytes.Index(ct, []byte("gbk")) > 0 {
		htmlParser.Convert()
		needconv = false
	}

	if needconv && bytes.Index(ct, []byte("gb2312")) > 0 {
		htmlParser.Convert()
		needconv = false
	}

	if needconv && htmlParser.IsGbk() {
		htmlParser.Convert()
	}

	//get title and check
	if ti.GetOtherTitle(item).CheckError(item) {
		return
	}
	// fmt.Println(item.data)
	SpiderServer.qfinish <- item
}

func (ti *Other) GetOtherTitle(item *Item) *Other {
	htmlParser := NewHtmlParser()

	hp := htmlParser.LoadData(ti.content)
	title := hp.FindByTagName("title")
	if title == nil {
		item.err = errors.New(`get title error`)
		return ti
	}

	item.data["title"] = fmt.Sprintf("%s", title[0][2])
	return ti
}

func (ti *Other) CheckError(item *Item) bool {
	if item.err != nil {
		SpiderServer.qerror <- item
		return true
	}
	return false
}
