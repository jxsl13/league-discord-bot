package parse

import (
	"fmt"
	"time"
)

const (
	LayoutDateTimeWithZone = "2006-01-02 15:04"
)

func Time(in string) (time.Time, error) {
	if in == "" {
		return time.Time{}, fmt.Errorf("empty time string")
	}

	t, err := time.Parse(LayoutDateTimeWithZone, in)
	if err != nil {
		return time.Time{},
			fmt.Errorf("error parsing time: %w: allowed time format with timezone offset ±hh:mm: %s (examples: 2006-01-02 15:04 CEST, 2006-01-02 15:04 GMT-0100)", err, LayoutDateTimeWithZone)
	}

	return t, nil
}

func Location(location string) (*time.Location, error) {
	l, err := time.LoadLocation(location)
	if err != nil {
		return nil, fmt.Errorf("invalid location (example: Europe/Germany): %s: %w", location, err)
	}
	return l, nil
}

func TimeInLocation(datetime string, loc *time.Location) (time.Time, error) {
	t, err := time.ParseInLocation(LayoutDateTimeWithZone, datetime, loc)
	if err != nil {
		return time.Time{},
			fmt.Errorf("invalid time: `%s`: expected the following format: `%s`: %w",
				datetime,
				LayoutDateTimeWithZone,
				err,
			)
	}

	return t, nil
}
