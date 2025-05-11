package bot

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) listMatchModeratorUserIDs(ctx context.Context, q *sqlc.Queries, channelID discord.ChannelID) (_ []discord.UserID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error listing match moderators: %w", err)
		}
	}()

	// we also need to get access to moderators
	mods, err := q.ListMatchModerators(ctx, channelID.String())
	if err != nil {
		return nil, fmt.Errorf("error getting match moderators: %w", err)
	}

	modUserIDs := make([]discord.UserID, 0, len(mods))
	for _, mod := range mods {
		uid, err := parse.UserID(mod.UserID)
		if err != nil {
			return nil, fmt.Errorf("error parsing moderator role ID: %w", err)
		}
		modUserIDs = append(modUserIDs, uid)
	}

	return modUserIDs, nil
}
