package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"9fans.net/go/acme"
	"github.com/golang/glog"
)

type repl struct {
	sync.Mutex
	wi       acme.WinInfo
	inw      *acme.Win
	lazyOutw *acme.Win
	busych   chan bool
}

func newRepl(wi acme.WinInfo, win *acme.Win) *repl {
	return &repl{wi: wi, inw: win, busych: make(chan bool, 1)}
}

func (r *repl) enter(expr string) {
	res, err := r.eval(expr)
	r.report("%s", res)
	if err != nil {
		r.report("%v", err)
	}
}

func (r *repl) eval(expr string) (string, error) {
	debugLog("evaluating: %s", expr)

	defer r.Busy()()

	c := exec.Command("gonrepl")
	c.Stdin = strings.NewReader(expr)
	b, err := c.CombinedOutput()
	return string(b), err
}

func (r *repl) report(format string, args ...interface{}) error {
	outw, err := r.outWin()
	if err != nil {
		return err
	}
	outw.PrintTabbed(fmt.Sprintf(format, args...))
	outw.Ctl("clean")
	return nil
}

func (r *repl) outWin() (*acme.Win, error) {
	r.Lock()
	defer r.Unlock()

	if r.lazyOutw == nil {
		w, err := r.createOutputWindow()
		if err != nil {
			return nil, err
		}
		r.lazyOutw = w
	}
	return r.lazyOutw, nil
}

func (r *repl) createOutputWindow() (*acme.Win, error) {
	w, err := acme.New()
	if err != nil {
		return nil, err
	}
	w.Name("%s+REPL", r.wi.Name)
	w.Ctl("clean")
	w.Ctl("nomark")
	w.Ctl("nomenu")
	return w, nil
}

func (r *repl) start() {
	go r.eventLoop()
	go r.busyController()
}

func (r *repl) Busy() func() {
	r.busych <- true
	return func() {
		r.busych <- false
	}
}
func (r *repl) busyController() {
	var outw *acme.Win
	busy := 0

	for b := range r.busych {
		if outw == nil {
			w, err := r.outWin()
			if err != nil {
				glog.Errorf("%v", err)
				continue
			}
			outw = w
		}
		if b {
			busy++
		} else {
			busy--
		}
		debugLog("busyness is %d, setting flag accordingly", busy)
		if busy > 0 {
			outw.Ctl("dirty")
		} else {
			outw.Ctl("clean")
		}
	}
}

func (r *repl) eventLoop() {
	for e := range r.inw.EventChan() {
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

				d, err := f(r.inw, e.Q0, e.Q1)
				if err != nil {
					r.inw.Errf("%v", err)
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
			r.inw.WriteEvent(e)
		}
	}
}
