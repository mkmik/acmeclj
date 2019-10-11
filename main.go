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

	repl := newRepl(wi)
	repl.start()
	go watchSourceWindow(win, repl)

	return nil
}

func watchSourceWindow(win *acme.Win, r *repl) {
	for e := range win.EventChan() {
		switch e.C2 {
		case 'X': // execute in body
			if e.Flag&1 == 0 {
				debugLog("Got execute event %c %c %q %v q0 f:%d q1 %d (orig q0 %d q1 %d)",
					e.C1, e.C2, e.Text, e.Flag,
					e.Q0, e.Q1, e.OrigQ0, e.OrigQ1)

				var (
					f        func(*acme.Win, int, int) (string, error)
					expanded bool
				)
				if e.Q0 == e.OrigQ0 && e.Q1 == e.OrigQ1 {
					f = readRange
				} else {
					expanded = true
					f = around
				}

				d, err := f(win, e.Q0, e.Q1)
				if err != nil {
					win.Errf("%v", err)
				}

				if expanded && !strings.HasSuffix(d, ")") {
					debugLog("not executing %q", d)
				} else {
					go r.enter(d)
				}
				continue
			}
			fallthrough
		default:
			win.WriteEvent(e)
		}
	}
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

func debugLog(format string, args ...interface{}) {
	glog.Infof(format, args...)
	glog.Flush()
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
