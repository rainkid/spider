package spider

import (
	"bytes"
	"errors"
	"fmt"
)

type Other struct {
	item    *Item
	content []byte
}

func (ti *Other) Get() {
	//get content

	var content []byte
	var err error

	ti.item.loader = NewLoader(ti.item.params["link"], "Get")
	content, err = ti.item.loader.Send(nil)

	if err != nil && ti.item.tryTimes < TryTime {
		ti.item.err = err
		SpiderServer.qstart <- ti.item
		return
	}

	ti.item.htmlParse.LoadData(content).CleanScript().Replace()
	ct := []byte(ti.item.loader.rheader.Get("Content-Type"))
	ct = bytes.ToLower(ct)

	var needconv bool = true
	if bytes.Index(ct, []byte("utf-8")) > 0 {
		needconv = false
	}

	if needconv && bytes.Index(ct, []byte("gbk")) > 0 {
		ti.item.htmlParse.Convert()
		needconv = false
	}

	if needconv && bytes.Index(ct, []byte("gb2312")) > 0 {
		ti.item.htmlParse.Convert()
		needconv = false
	}

	if needconv && ti.item.htmlParse.IsGbk() {
		ti.item.htmlParse.Convert()
	}

	ti.content = ti.item.htmlParse.content

	//get title and check
	if ti.GetOtherTitle().CheckError() {
		return
	}
	// fmt.Println(ti.item.data)
	SpiderServer.qfinish <- ti.item
}

func (ti *Other) GetOtherTitle() *Other {
	hp := ti.item.htmlParse.LoadData(ti.content)
	title := hp.FindByTagName("title")
	if title == nil {
		ti.item.err = errors.New(`get title error`)
		return ti
	}

	ti.item.data["title"] = fmt.Sprintf("%s", title[0][2])
	return ti
}

func (ti *Other) CheckError() bool {
	if ti.item.err != nil {
		SpiderServer.qerror <- ti.item
		return true
	}
	return false
}
