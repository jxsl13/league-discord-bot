package timerutils

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestErrorLoop(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var (
		interval           = time.Second
		numBackoffs        = 3
		cnt                = 0
		allowedMaxDuration = (time.Duration(numBackoffs) * interval)
	)
	f := func() (time.Duration, error) {
		cnt++

		if cnt == numBackoffs {
			cancel()
		}
		return 0, errors.New("test")
	}

	start := time.Now()
	Loop(ctx, interval, interval, f)
	duration := time.Since(start)

	if duration > allowedMaxDuration {
		t.Errorf("Loop took too long: %v, expected less than %v", duration, allowedMaxDuration)
	}
}

func TestSuccessLoop(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var (
		interval           = time.Second
		numLoops           = 3
		cnt                = 0
		allowedMaxDuration = (time.Duration(numLoops) * interval)
	)
	f := func() (time.Duration, error) {
		cnt++

		if cnt == numLoops {
			cancel()
		}
		return 0, nil
	}

	start := time.Now()
	Loop(ctx, interval, interval, f)
	duration := time.Since(start)

	if duration > allowedMaxDuration {
		t.Errorf("Loop took too long: %v, expected less than %v", duration, allowedMaxDuration)
	}
}
