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
}

func NewMyLoger() *MyLoger {
	return &MyLoger{
		Handler: log.New(os.Stdout, "", log.Ldate|log.Ltime),
	}
}

func (l *MyLoger) D(infos ...interface{}) {
	l.infoType = "DEBUG"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) I(infos ...interface{}) {
	l.infoType = "INFO"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) E(infos ...interface{}) {
	l.infoType = "ERROR"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) W(infos ...interface{}) {
	l.infoType = "WARN"
	l.Infos = infos
	l.P()
}

func (l *MyLoger) P() {
	var s string
	for _, v := range l.Infos {
		s += fmt.Sprintf("%v ", v)
	}
	// l.Handler.SetPrefix(fmt.Sprintf("%s - "))
	l.Handler.Println(fmt.Sprintf(`SPIDER - %s - "%s"`, l.infoType, s))
}
