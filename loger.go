package spider

import (
	"fmt"
	"log"
	"os"
)

type MyLoger struct {
	infoType string
	Handler  *log.Logger
	Infos    []interface{}
	Colors   map[string]string
	ModeColor string
}

type Color map[string]string
type MyColor string

func (c MyColor)String() string {
	colors := map[string]string{}
	colors["none"] = "\033[0m"
	colors["black"] = "\033[0;30m"
	colors["dark_gray"] = "\033[1;30m"
	colors["blue"] = "\033[0;34m"
	colors["light_blue"] = "\033[1;34m"
	colors["green"] = "\033[0;32m"
	colors["light_green"] = "\033[1;32m"
	colors["cyan"] = "\033[0;36m"
	colors["light_cyan"] = "\033[1;36m"
	colors["red"] = "\033[0;31m"
	colors["light_red"] = "\033[1;31m"
	colors["purple"] = "\033[0;35m"
	colors["light_purple"] = "\033[1;35m"
	colors["brown"] = "\033[0;33m"
	colors["yellow"] = "\033[1;33m"
	colors["light_gray"] = "\033[0;37m"
	colors["white"] = "\033[1;37m"
	return colors[string(c)]
}

func NewMyLoger() *MyLoger {
	colors := map[string]string{}
	colors["none"] = "\033[0m"
	colors["black"] = "\033[0;30m"
	colors["dark_gray"] = "\033[1;30m"
	colors["blue"] = "\033[0;34m"
	colors["light_blue"] = "\033[1;34m"
	colors["green"] = "\033[0;32m"
	colors["light_green"] = "\033[1;32m"
	colors["cyan"] = "\033[0;36m"
	colors["light_cyan"] = "\033[1;36m"
	colors["red"] = "\033[0;31m"
	colors["light_red"] = "\033[1;31m"
	colors["purple"] = "\033[0;35m"
	colors["light_purple"] = "\033[1;35m"
	colors["brown"] = "\033[0;33m"
	colors["yellow"] = "\033[1;33m"
	colors["light_gray"] = "\033[0;37m"
	colors["white"] = "\033[1;37m"

	return &MyLoger{
		Handler: log.New(os.Stdout, "", log.Ldate|log.Ltime),
		Colors:colors,
	}
}

func (l *MyLoger) D(infos ...interface{}) {
	l.infoType = "DEBUG"
	l.Infos = infos
	l.ModeColor = l.Colors["light_cyan"]
	l.P()
}

func (l *MyLoger) I(infos ...interface{}) {
	l.infoType = "INFO"
	l.Infos = infos
	l.ModeColor = l.Colors["light_purple"]
	l.P()
}

func (l *MyLoger) E(infos ...interface{}) {
	l.infoType = "ERROR"
	l.Infos = infos
	l.ModeColor = l.Colors["red"]
	l.P()
}

func (l *MyLoger) W(infos ...interface{}) {
	l.infoType = "WARN"
	l.Infos = infos
	l.ModeColor = l.Colors["yellow"]
	l.P()
}

func (l *MyLoger) P() {
	var s string
	for _, v := range l.Infos {
		s += fmt.Sprintf("%v ", v)
	}
	// l.Handler.SetPrefix(fmt.Sprintf("%s - "))
	l.Handler.Println(fmt.Sprintf(`%sSpider->%s%s %s`, l.ModeColor, l.infoType, l.Colors["none"], s))
}
