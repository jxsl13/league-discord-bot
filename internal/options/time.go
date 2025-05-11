package options

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
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
