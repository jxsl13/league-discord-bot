package timerutils

import (
	"time"
)

func NewTimer() (t *time.Timer, drained bool) {
	return time.NewTimer(0), false
}

// resetTimer sets drained to false after resetting the timer.
func ResetTimer(timer *time.Timer, duration time.Duration, drained *bool) {
	if drained == nil {
		panic("drained bool pointer is nil")
	}
	if !timer.Stop() {
		if !*drained {
			<-timer.C
		}
	}
	timer.Reset(duration)
	*drained = false
}

// closeTimer should be used as a deferred function
// in order to cleanly shut down a timer
func CloseTimer(timer *time.Timer, drained *bool) {
	if drained == nil {
		panic("drained bool pointer is nil")
	}
	if !timer.Stop() {
		if *drained {
			return
		}
		<-timer.C
		*drained = true
	}
}
