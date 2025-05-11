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
