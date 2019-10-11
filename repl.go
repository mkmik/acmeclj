package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"9fans.net/go/acme"
)

type repl struct {
	wi        acme.WinInfo
	inw, outw *acme.Win
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
	c := exec.Command("gonrepl")
	c.Stdin = strings.NewReader(expr)
	b, err := c.CombinedOutput()
	return string(b), err
}

func (r *repl) report(format string, args ...interface{}) error {
	if r.outw == nil {
		w, err := acme.New()
		if err != nil {
			return err
		}
		r.outw = w
		w.Name("%s+REPL", r.wi.Name)
	}
	r.outw.PrintTabbed(fmt.Sprintf(format, args...))
	return nil
}

func (r *repl) start() {
	go r.eventLoop()
}

func (r *repl) eventLoop() {
	for e := range r.inw.EventChan() {
		switch e.C2 {
		case 'X': // execute in body
			if e.Flag&1 == 0 {
				if false {
					log.Printf("Got execute event %c %c %q %v q0 f:%d q1 %d (orig q0 %d q1 %d)",
						e.C1, e.C2, e.Text, e.Flag,
						e.Q0, e.Q1, e.OrigQ0, e.OrigQ1)
				}

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
					//	log.Printf("not executing %q", d)
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
