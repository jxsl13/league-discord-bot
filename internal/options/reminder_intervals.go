package options

import (
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/parse"
)

func ReminderIntervals(name string, options discord.CommandInteractionOptions) ([]time.Duration, error) {
	s := options.Find(name).String()
	return parse.ReminderIntervals(s)
}

func ReminderIntervalsOption(name string, options discord.CommandInteractionOptions) ([]time.Duration, bool, error) {
	s := options.Find(name).String()
	if s == "" {
		return nil, false, nil
	}

	durations, err := parse.ReminderIntervals(s)
	if err != nil {
		return nil, false, err
	}

	return durations, true, nil
}
