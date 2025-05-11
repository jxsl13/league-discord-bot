package parse

import (
	"fmt"
	"strings"
	"time"
)

// DurationList parses a comma-separated list of durations in the format
// "24h30m3s, 1h, 15m, 10m30s, 20s, 1s" into a slice of time.Duration.
// If th eprovided string is empty, it returns an empty slice.
func DurationList(input string) ([]time.Duration, error) {
	if input == "" {
		return []time.Duration{}, nil
	}

	durations := strings.Split(input, ",")
	var result []time.Duration
	for _, d := range durations {
		duration, err := time.ParseDuration(strings.TrimSpace(d))
		if err != nil {
			return nil, fmt.Errorf("invalid duration value %q: %w (allowed format: 24h30m3s, 1h, 15m, 10m30s, 20s, 1s)", d, err)
		}
		result = append(result, duration)
	}
	return result, nil
}
