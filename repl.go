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
	lazyOutw *acme.Win
	busych   chan bool
	sid      string
}

func newRepl(wi acme.WinInfo) (*repl, error) {
	r := &repl{wi: wi, busych: make(chan bool, 1)}
	b, err := exec.Command("gonrepl", "--clone").Output()
	if err != nil {
		return nil, err
	}
	r.sid = strings.TrimSpace(string(b))
	debugLog("allocated sid %q", r.sid)

	return r, nil
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

	c := exec.Command("gonrepl", "-s", r.sid)
	c.Stdin = strings.NewReader(expr)
	b, err := c.CombinedOutput()
	return string(b), err
}

func (r *repl) report(format string, args ...interface{}) error {
	win, err := r.outWin()
	if err != nil {
		return err
	}
	win.PrintTabbed(fmt.Sprintf(format, args...))
	win.Ctl("clean")
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
		go r.eventLoop(w)
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
	go r.busyController()
}

func (r *repl) Busy() func() {
	r.busych <- true
	return func() {
		r.busych <- false
	}
}
func (r *repl) busyController() {
	var win *acme.Win
	busy := 0

	for b := range r.busych {
		if win == nil {
			w, err := r.outWin()
			if err != nil {
				glog.Errorf("%v", err)
				continue
			}
			win = w
		}
		if b {
			busy++
		} else {
			busy--
		}
		debugLog("busyness is %d, setting flag accordingly", busy)
		if busy > 0 {
			win.Ctl("dirty")
		} else {
			win.Ctl("clean")
		}
	}
}

func (r *repl) eventLoop(win *acme.Win) {
	for e := range win.EventChan() {
		win.WriteEvent(e)
	}

	debugLog("repl closed")
	r.Lock()
	defer r.Unlock()
	r.lazyOutw = nil
}

func (r *repl) close() {
	r.Lock()
	defer r.Unlock()
	if r.lazyOutw != nil {
		win := r.lazyOutw
		win.Del(true)
	}

	_, err := exec.Command("gonrepl", "--close", "-s", r.sid).Output()
	if err != nil {
		glog.Errorf("%v", err)
	}
}
