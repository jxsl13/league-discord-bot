package options

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/parse"
)

func TimeInLocation(datetimeName, locationName string, options discord.CommandInteractionOptions) (time.Time, error) {
	loc, err := parse.Location(options.Find(locationName).String())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid location parameter %q: %w", locationName, err)
	}

	t, err := parse.TimeInLocation(options.Find(datetimeName).String(), loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid datetime parameter %q: %w", datetimeName, err)
	}
	return t, nil
}

// The time must be in the future of at least abs(offset)
func FutureTimeInLocation(datetimeName, locationName string, offset time.Duration, options discord.CommandInteractionOptions) (time.Time, error) {
	t, err := TimeInLocation(datetimeName, locationName, options)
	if err != nil {
		return time.Time{}, err

	}

	if time.Until(t) < offset.Abs() {
		return time.Time{}, fmt.Errorf("invalid parameter %q: must be at least %s in the future", datetimeName, offset)
	}
	return t, nil
}

func TimeBetweenInLocation(datetimeName, locationName string, min, max time.Time, options discord.CommandInteractionOptions) (time.Time, error) {
	t, err := TimeInLocation(datetimeName, locationName, options)
	if err != nil {
		return time.Time{}, err
	}

	if t.Before(min) || t.After(max) {
		return time.Time{}, fmt.Errorf("invalid parameter %q: must be between %s and %s, is %s",
			datetimeName,
			format.DiscordLongDateTime(min),
			format.DiscordLongDateTime(max),
			format.DiscordLongDateTime(t),
		)
	}
	return t, nil
}
