package main

import (
	"fmt"
	"os/exec"
	"strings"

	"9fans.net/go/acme"
	"github.com/golang/glog"
)

type repl struct {
	wi       acme.WinInfo
	inw      *acme.Win
	lazyOutw *acme.Win
}

func newRepl(wi acme.WinInfo, win *acme.Win) *repl {
	return &repl{wi: wi, inw: win}
}

func (r *repl) enter(expr string) {
	res, err := r.eval(expr)
	r.report("%s", res)
	if err != nil {
		r.report("%v", err)
	}
}

func (r *repl) eval(expr string) (string, error) {
	r.debugLog("evaluating: %s", expr)

	outw, err := r.outWin()
	if err != nil {
		return "", err
	}
	outw.Ctl("dirty")
	defer func() { outw.Ctl("clean") }()

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
}

func (r *repl) eventLoop() {
	for e := range r.inw.EventChan() {
		switch e.C2 {
		case 'X': // execute in body
			if e.Flag&1 == 0 {
				r.debugLog("Got execute event %c %c %q %v q0 f:%d q1 %d (orig q0 %d q1 %d)",
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
					r.debugLog("not executing %q", d)
				} else {
					r.enter(d)
				}
				continue
			}
			fallthrough
		default:
			r.inw.WriteEvent(e)
		}
	}
}

func (r *repl) debugLog(format string, args ...interface{}) {
	glog.Infof(format, args...)
	glog.Flush()
}
