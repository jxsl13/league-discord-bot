package model

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/sqlc"
)

type Streamer struct {
	UserID discord.UserID
	Info   sqlc.Streamer
}

func (s Streamer) String() string {
	if s.Info.Url != "" {
		return fmt.Sprintf("%s at %s", s.UserID.Mention(), s.Info.Url)
	} else {
		return s.UserID.Mention()
	}
}

func (s Streamer) Mention() string {
	return s.String()
}
