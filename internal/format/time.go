package format

import (
	"fmt"
	"time"
)

var (
	durationScalars = []durationUnit{
		{time.Hour, "hour", "hours"},
		{time.Minute, "minute", "minutes"},
		{time.Second, "second", "seconds"},
		{time.Millisecond, "millisecond", "milliseconds"},
		{time.Microsecond, "microsecond", "microseconds"},
		{time.Nanosecond, "nanosecond", "nanoseconds"},
	}
)

type durationUnit struct {
	numeric  time.Duration
	singular string
	plural   string
}

func Duration(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	var scalar durationUnit
	for _, s := range durationScalars {
		truncated := d.Truncate(s.numeric)
		if truncated > 0 {
			scalar = s
			break
		}
	}

	factor := d / scalar.numeric

	plural := factor != 1
	unit := scalar.plural
	if !plural {
		unit = scalar.singular
	}
	return fmt.Sprintf("%d %s", factor, unit)
}
