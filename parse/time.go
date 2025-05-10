package parse

import (
	"fmt"
	"time"
)

const (
	LayoutDateTimeWithZone = "2006-01-02 15:04 -07:00"
)

func Time(in string) (time.Time, error) {
	if in == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	t, err := time.Parse(LayoutDateTimeWithZone, in)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing time: %w: allowed time format with timezone offset ±hh:mm: %s", err, LayoutDateTimeWithZone)
	}

	return t, nil
}
