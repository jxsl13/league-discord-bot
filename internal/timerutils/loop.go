package timerutils

import (
	"context"
	"time"
)

func Loop(
	ctx context.Context,
	minBackoff time.Duration,
	interval time.Duration,
	f func(ctx context.Context) (time.Duration, error),
	deferredFunc ...func(),
) {
	defer func() {
		for _, df := range deferredFunc {
			df()
		}
	}()
	minBackoff = max(minBackoff, time.Second)
	interval = max(interval, time.Second)

	var (
		timer, drained = NewTimer()
		backoff        = NewBackoffPolicy(minBackoff, interval)
		retries        = 0
	)
	defer CloseTimer(timer, &drained)

	for {
		select {
		case <-ctx.Done():
			return
		case ct := <-timer.C:
			drained = true
			func() {
				var (
					reset time.Duration
					err   error
				)
				defer func() {
					if err != nil {
						retries++
						b := backoff(retries)
						u := max(0, time.Until(ct.Add(interval)))

						if u < b {
							// when backoff exceeds our next expected interval (point in time)
							// we just abort the backoff and retry regularily
							// tho the retry counter is not reset until the next successful
							// execution, which is why the next backoff will be longer
							// in case that the next interval is also a failure
							// this is to prevent the backoff from being too aggressive
							b = u
						}

						if !(0 < reset && reset <= b) {
							reset = b
						}
					} else {
						retries = 0
						u := max(0, time.Until(ct.Add(interval))) // fire instantly if negative
						if !(0 < reset && reset <= u) {
							reset = u
						}
					}
					ResetTimer(timer, reset, &drained)
				}()

				reset, err = f(ctx)
				if err != nil {
					return
				}
			}()

		}
	}
}
