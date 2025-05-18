package options

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
)

func Duration(name string, min, max time.Duration, options discord.CommandInteractionOptions) (time.Duration, error) {
	s := options.Find(name).String()
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid %q: %w, expected format: 168h, 24h20m10s, 30m, 20m5s, 60s", name, err)
	}

	if d < min {
		return 0, fmt.Errorf("%q must be at least %s: %s", name, min, d)
	} else if d > max {
		return 0, fmt.Errorf("%q must be at most %s: %s", name, max, d)
	}

	return d, nil
}

func DurationOption(name string, min, max time.Duration, options discord.CommandInteractionOptions) (d time.Duration, ok bool, err error) {
	s := options.Find(name).String()
	if s == "" {
		return 0, false, nil
	}

	d, err = time.ParseDuration(s)
	if err != nil {
		return 0, false, fmt.Errorf("invalid %q: %w, expected format: 168h, 24h20m10s, 30m, 20m5s, 60s", name, err)
	}

	if d < min {
		return 0, false, fmt.Errorf("%q must be at least %s: %s", name, min, d)
	} else if d > max {
		return 0, false, fmt.Errorf("%q must be at most %s: %s", name, max, d)
	}

	return d, true, nil
}
