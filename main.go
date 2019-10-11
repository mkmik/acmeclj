package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"9fans.net/go/acme"
	"github.com/golang/glog"
)

func handleNew(ev acme.LogEvent) error {
	wi, err := findWindow(ev.Name)
	if err != nil {
		return err
	}
	handleWindow(wi)

	return nil
}

func handleWindow(wi acme.WinInfo) error {
	win, err := acme.Open(wi.ID, nil)
	if err != nil {
		return err
	}
	win.SetErrorPrefix(wi.Name)

	repl := newRepl(wi, win)
	repl.start()

	return nil
}

func around(win *acme.Win, b, e int) (string, error) {
	if b > 0 {
		b--
	}
	e++

	return readRange(win, b, e)
}

func readRange(win *acme.Win, b, e int) (string, error) {
	if err := win.Addr("#%d,#%d", b, e); err != nil {
		return "", err
	}
	d, err := win.ReadAll("xdata")
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func findWindow(name string) (acme.WinInfo, error) {
	winInfos, err := acme.Windows()
	if err != nil {
		return acme.WinInfo{}, err
	}
	for _, winInfo := range winInfos {
		if winInfo.Name == name {
			return winInfo, nil
		}
	}
	return acme.WinInfo{}, fmt.Errorf("cannot find window %q", name)
}

func handleDel(ev acme.LogEvent) error {
	log.Printf("new clojure file closed")

	return nil
}

func isClojure(name string) bool {
	return strings.HasSuffix(name, "clj")
}

func run() error {
	winInfos, err := acme.Windows()
	if err != nil {
		return err
	}
	for _, winInfo := range winInfos {
		if isClojure(winInfo.Name) {
			handleWindow(winInfo)
		}
	}

	l, err := acme.Log()
	if err != nil {
		log.Fatal(err)
	}

	for {
		event, err := l.Read()
		if err != nil {
			return err
		}
		if isClojure(event.Name) {
			var err error
			switch event.Op {
			case "new":
				err = handleNew(event)
			case "del":
				err = handleDel(event)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}
