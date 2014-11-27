package main

import (
	"fmt"
	"net/http"
	spider "spider"
)

var sp *spider.Spider

// var spider = s.NewSpider()
func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Println("response.")
	/*sp.Add("tmall", 21827332489)
	sp.Add("taobao", 38681387627)
	sp.Add("mmb", 191513)*/
}
func main() {
	sp = spider.Start()

	http.HandleFunc("/hello", hello)         //设置访问的路由
	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		fmt.Println("ListenAndServe: ", err)
	}
	// spider.Daemon()
}
