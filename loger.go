package spider

import (
	"fmt"
	"log"
	"os"
)

type MyLoger struct {
	InfoType string
	Handler  *log.Logger
	Infos    []interface{}
}

func NewMyLoger() *MyLoger {
	return &MyLoger{
		Handler: log.New(os.Stdout, "[SPIDER] ", log.Ldate|log.Ltime),
	}
}

func (l *MyLoger) D(infos ...interface{}) {
	l.InfoType = "DEBUG"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) I(infos ...interface{}) {
	l.InfoType = "INFOS"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) E(infos ...interface{}) {
	l.InfoType = "ERROR"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) P() {
	var s = fmt.Sprintf("[%s] ", l.InfoType)
	for _, v := range l.Infos {
		s += fmt.Sprintf("%v ", v)
	}
	l.Handler.Println(s)
}
