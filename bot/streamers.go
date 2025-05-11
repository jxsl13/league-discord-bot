package bot

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/discordutils"
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

func (b *Bot) listMatchStreamerUserIDs(ctx context.Context, q *sqlc.Queries, channelID discord.ChannelID) (_ []Streamer, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error listing match streamers: %w", err)
		}
	}()

	streamers, err := q.ListMatchStreamers(ctx, channelID.String())
	if err != nil {
		return nil, fmt.Errorf("error getting match streamers: %w", err)
	}

	result := make([]Streamer, 0, len(streamers))
	for _, streamer := range streamers {
		uid, err := discordutils.ParseUserID(streamer.UserID)
		if err != nil {
			return nil, fmt.Errorf("error parsing streamer user ID: %w", err)
		}
		result = append(result, Streamer{
			UserID: uid,
			Info:   streamer,
		})
	}
	return result, nil
}
