package reminder

import (
	"cmp"
	"errors"
	"slices"
	"time"
)

type Reminder struct {
	intervals        []time.Duration
	maxReminderIndex int64
}

func (r *Reminder) MaxIndex() int64 {
	return r.maxReminderIndex
}

func New(intervals ...time.Duration) (*Reminder, error) {
	if len(intervals) == 0 {
		return nil, errors.New("reminder intervals cannot be empty")
	}

	// from biggest to smallest
	slices.Sort(intervals)
	slices.Reverse(intervals)

	return &Reminder{
		intervals:        intervals,
		maxReminderIndex: int64(len(intervals) - 1),
	}, nil
}

func until(now time.Time, scheduledAt time.Time) time.Duration {
	until := scheduledAt.Sub(now)
	if until < 0 {
		until = -1 * until
	}
	return until
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -1 * d
	}
	return d
}

func (r *Reminder) Next(reminderCnt int64, scheduledAt time.Time) (int64, time.Duration, bool) {
	if reminderCnt > r.maxReminderIndex {
		return reminderCnt, 0, false
	}

	var (
		now = time.Now()
	)

	untilNextReminder := make([]time.Duration, len(r.intervals))
	remindersAt := make([]time.Time, len(r.intervals))
	for i, offset := range r.intervals {
		remindAt := scheduledAt.Add(-abs(offset))

		remindersAt[i] = remindAt
		untilNextReminder[i] = until(now, remindAt)
	}

	sortedIndexList := make([]int, len(untilNextReminder))
	for i := range len(untilNextReminder) {
		sortedIndexList[i] = i
	}
	slices.SortFunc(sortedIndexList, func(a, b int) int {
		ad := int64(untilNextReminder[a])
		bd := int64(untilNextReminder[b])
		return cmp.Compare(ad, bd)
	})

	// first element in that list is the closest reminder
	// let's see if it is in the past or in the future
	for _, i := range sortedIndexList {
		i := int64(i)
		if reminderCnt > i {
			// this reminder is already sent
			continue
		}

		offset := abs(r.intervals[i])
		remindAt := scheduledAt.Add(-offset)
		if now.After(remindAt) {
			// this reminder is in the past, we can skip it
			continue
		}

		nextReminderIn := untilNextReminder[i]
		nextIntervalIn := r.intervals[i]
		triggerReminder := nextReminderIn < nextIntervalIn

		// this reminder is in the future, we can use it
		// but only if we are inside the reminder period
		return i, nextReminderIn, triggerReminder
	}

	return reminderCnt, 0, false
}
