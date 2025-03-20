package timehelper

import (
	"time"
)

func AfterTimeFunc(d time.Time, fn func()) *time.Timer {
	return time.AfterFunc(time.Duration(d.Sub(time.Now()).Nanoseconds())*time.Nanosecond, func() {
		if fn != nil {
			fn()
		}
	})
}
