package timeutils

import "time"

func Ceil(t time.Time, d time.Duration) time.Time {
	// Round up to the next multiple of d
	return t.Add(d - (t.Sub(t.Truncate(d)) % d))
}
