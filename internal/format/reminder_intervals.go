package format

import (
	"strings"
	"time"
)

// No intervals is also valid, that simply disabled the default notifications for a guild
func ReminderIntervals(intervals []time.Duration) string {
	if len(intervals) == 0 {
		return ""
	}
	// convert time.Duration to string
	ss := make([]string, len(intervals))
	for i, d := range intervals {
		ss[i] = d.String()
	}

	return strings.Join(ss, ",")
}
