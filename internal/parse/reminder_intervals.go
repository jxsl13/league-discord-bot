package parse

import (
	"fmt"
	"slices"
	"time"
)

const (
	MaxReminerIntervals = 50
)

func ReminderIntervals(input string) ([]time.Duration, error) {
	list, err := DurationList(input)
	if err != nil {
		return nil, fmt.Errorf("invalid reminder intervals: %w", err)
	}

	if len(list) > MaxReminerIntervals {
		return nil, fmt.Errorf("reminder intervals list cannot contain more than %d values", MaxReminerIntervals)
	}

	slices.Sort(list)
	slices.Reverse(list)
	list = slices.Compact(list)

	for _, d := range list {
		if d < time.Second {
			return nil, fmt.Errorf("reminder intervals must be at least 1 second: %s", d)
		}
	}

	return list, nil
}
