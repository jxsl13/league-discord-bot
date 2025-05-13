package bot

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/model"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) listMatchStreamerUserIDs(ctx context.Context, q *sqlc.Queries, channelID discord.ChannelID) (_ []model.Streamer, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error listing match streamers: %w", err)
		}
	}()

	streamers, err := q.ListMatchStreamers(ctx, channelID.String())
	if err != nil {
		return nil, fmt.Errorf("error getting match streamers: %w", err)
	}

	result := make([]model.Streamer, 0, len(streamers))
	for _, streamer := range streamers {
		uid, err := parse.UserID(streamer.UserID)
		if err != nil {
			return nil, fmt.Errorf("error parsing streamer user ID: %w", err)
		}
		result = append(result, model.Streamer{
			UserID: uid,
			Info:   streamer,
		})
	}
	return result, nil
}
