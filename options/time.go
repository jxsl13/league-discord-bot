package options

import (
	"fmt"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/parse"
)

func Time(name string, options discord.CommandInteractionOptions) (time.Time, error) {
	t, err := parse.Time(options.Find(name).String())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid datetime parameter %q: %w", name, err)
	}
	return t, nil
}

func MinTime(name string, options discord.CommandInteractionOptions, min time.Time) (time.Time, error) {
	t, err := Time(name, options)
	if err != nil {
		return time.Time{}, err
	}

	if t.Before(min) {
		return time.Time{}, fmt.Errorf("invalid parameter %q: time must be after %s", name, discordutils.ToDiscordTimestamp(min))
	}
	return t, nil
}
