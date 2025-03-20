package timehelper

import (
	"mango/pkg/util/errorhelper"
	"time"
)

func NewTickerSecond(second time.Duration, fn func()) {
	newTicker(second, time.Second, fn)
}

func NewTicker(millisecond time.Duration, fn func()) {
	newTicker(millisecond, time.Millisecond, fn)
}

func newTicker(t time.Duration, unit time.Duration, fn func()) {
	defer errorhelper.Recover()

	timer := time.NewTicker(t * unit)
	for {
		select {
		case <-timer.C:
			fn()
		}
	}
}
