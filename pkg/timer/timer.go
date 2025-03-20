package timer

import (
	"runtime"
	"time"
	"mango/pkg/conf"
	"mango/pkg/log"
)

const (
	Invalid     = -1
	LoopForever = 0
)

// one dispatcher per goroutine (goroutine not safe)
type Dispatcher struct {
	ChanTimer chan *Timer
}

func NewDispatcher(l int) *Dispatcher {
	disp := new(Dispatcher)
	disp.ChanTimer = make(chan *Timer, l)
	return disp
}

// Timer
type Timer struct {
	t         *time.Timer
	loopCount int
	cb        func()
}

func (t *Timer) Stop() {
	t.t.Stop()
	t.cb = nil
}

func (t *Timer) Cb() {
	defer func() {
		if t.loopCount == Invalid {
			t.cb = nil
		}
		if r := recover(); r != nil {
			if conf.LenStackBuf > 0 {
				buf := make([]byte, conf.LenStackBuf)
				l := runtime.Stack(buf, false)
				log.Error("Timer", "%v: %s", r, buf[:l])
			} else {
				log.Error("Timer", "%v", r)
			}
		}
	}()

	if t.cb != nil {
		t.cb()
	}
}

func (disp *Dispatcher) AfterFunc(d time.Duration, cb func()) *Timer {
	t := new(Timer)
	t.cb = cb
	t.loopCount = Invalid
	t.t = time.AfterFunc(d, func() {
		disp.ChanTimer <- t
	})
	return t
}

// Cron
type Cron struct {
	t *Timer
}

func (c *Cron) Stop() {
	if c.t != nil {
		c.t.Stop()
	}
}

func (disp *Dispatcher) CronFunc(cronExpr *CronExpr, _cb func()) *Cron {
	c := new(Cron)

	now := time.Now()
	nextTime := cronExpr.Next(now)
	if nextTime.IsZero() {
		return c
	}

	// callback
	var cb func()
	cb = func() {
		defer _cb()

		now := time.Now()
		nextTime := cronExpr.Next(now)
		if nextTime.IsZero() {
			return
		}
		c.t = disp.AfterFunc(nextTime.Sub(now), cb)
	}

	c.t = disp.AfterFunc(nextTime.Sub(now), cb)
	return c
}

func (disp *Dispatcher) LoopFunc(d time.Duration, cb func(), loopCount int) *Timer {
	if loopCount < LoopForever {
		return nil
	}

	t := new(Timer)
	t.loopCount = loopCount
	t.cb = cb
	t.t = time.NewTimer(d)
	go func() {
		for {
			<-t.t.C
			disp.ChanTimer <- t
			if t.loopCount != LoopForever {
				t.loopCount--
				if t.loopCount == 0 {
					t.loopCount = Invalid
					break
				}
			}
			t.t.Reset(d)
		}
	}()
	return t
}
