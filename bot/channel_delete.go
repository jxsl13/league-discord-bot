package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleChannelDelete(e *gateway.ChannelDeleteEvent) {
	if e.Type != discord.GuildText && e.Type != discord.GuildCategory {
		return
	}

	var (
		guildID      = e.GuildID
		guildIDStr   = guildID.String()
		channelID    = e.Channel.ID
		channelIDStr = channelID.String()
	)

	err := b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		if e.Type == discord.GuildText {
			// just delete match channel if it matches the channel id
			err := q.DeleteMatch(ctx, channelIDStr)
			if err != nil {
				return fmt.Errorf("error deleting match for channel %s: %w", channelID, err)
			}
			return nil
		}

		// category channel -> guild config was modified

		r, err := q.GetGuildConfigByCategory(ctx, channelIDStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no config found, ignore
				return nil
			}
			return fmt.Errorf("error getting guild config for category %s: %w", channelIDStr, err)
		}

		if r.Enabled == 0 {
			return errors.New("guild config is disabled, ignoring")
		}

		// config found
		channels, err := b.state.Channels(guildID)
		if err != nil {
			return fmt.Errorf("error getting guild %s: %w", guildIDStr, err)
		}
		lastPos := discordutils.LastChannelPosition(channels)

		matches, err := q.ListGuildMatches(ctx, guildIDStr)
		if err != nil {
			return fmt.Errorf("error getting matches for guild %s: %w", guildIDStr, err)
		}

		channelIDs := make([]discord.ChannelID, 0, len(matches))
		for _, m := range matches {
			id, err := parse.ChannelID(m.ChannelID)
			if err != nil {
				return fmt.Errorf("error parsing channel ID %s: %w", m.ChannelID, err)
			}
			channelIDs = append(channelIDs, id)
		}

		category, err := b.createMatchCategory(e.GuildID, lastPos)
		if err != nil {
			err = fmt.Errorf("error creating category for guild %s: %v", e.GuildID.String(), err)

			derr := q.DisableGuild(ctx, guildIDStr)
			if derr != nil {
				return fmt.Errorf("error disabling guild %s: %w (%w)", guildIDStr, derr, err)
			}
			return err
		}

		err = q.UpdateCategoryId(ctx, sqlc.UpdateCategoryIdParams{
			GuildID:    guildIDStr,
			CategoryID: category.ID.String(),
		})
		if err != nil {
			return fmt.Errorf("error updating category id for guild %s: %w", guildIDStr, err)
		}

		// recreated category. look for all channels and move them back into the new category
		for _, id := range channelIDs {
			err = b.state.Client.ModifyChannel(id, api.ModifyChannelData{
				CategoryID: category.ID,
			})
			if err != nil {
				return fmt.Errorf("error modifying channel %s: %v", id, err)
			}
		}

		return nil
	})
	if err != nil {
		log.Println(err)
	}
}
