package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/options"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) commandGuildConfigure(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {

	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, ADMIN)
		if err != nil {
			return err
		}

		accessOffset, err := options.Duration("channel_access_offset", time.Hour, 730*time.Hour, data.Options)
		if err != nil {
			return err
		}

		deleteOffset, err := options.Duration("channel_delete_offset", time.Hour, 8760*time.Hour, data.Options)
		if err != nil {
			return err
		}

		err = q.UpdateGuildConfig(b.ctx, sqlc.UpdateGuildConfigParams{
			GuildID:             data.Event.GuildID.String(),
			ChannelDeleteOffset: int64(deleteOffset / time.Second),
			ChannelAccessOffset: int64(accessOffset / time.Second),
		})
		if err != nil {
			err = fmt.Errorf("error adding guild config: %w", err)
			log.Println(err)
			return fmt.Errorf("%w, please contact the owner of the bot", err)
		}

		resp = &api.InteractionResponseData{
			Content:         option.NewNullableString("Guild configuration was updated. New match schedules will be created accordingly"),
			Flags:           discord.EphemeralMessage,
			AllowedMentions: &api.AllowedMentions{ /* none */ },
		}
		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	return resp

}

func (b *Bot) handleAddGuild(e *gateway.GuildCreateEvent) {
	err := b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		i, err := q.IsGuildEnabled(b.ctx, e.Guild.ID.String())
		if err == nil && i != 0 {
			log.Printf("guild %s is already enabled, skipping", e.Guild.ID.String())
			return nil
		} else if err == nil && i == 0 {
			log.Printf("guild %s is disabled, skipping", e.Guild.ID.String())
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("error checking if guild %d is enabled: %v", e.ID, err)
		}
		// guild is unknown, so we need to add and initialize is
		lastPos := discordutils.LastChannelPosition(e.Channels)

		category, err := b.createMatchCategory(e.Guild.ID, lastPos)
		var (
			created    = err == nil
			categoryID = discord.NullChannelID
		)
		if created {
			categoryID = category.ID
		}

		// add default config
		err = q.AddGuildConfig(b.ctx, sqlc.AddGuildConfigParams{
			GuildID:             e.Guild.ID.String(),
			Enabled:             boolToInt64(created),
			CategoryID:          categoryID.String(),
			ChannelAccessOffset: int64(b.defaultChannelAccessOffset / time.Second),
			ChannelDeleteOffset: int64(b.defaultChannelDeleteOffset / time.Second),
		})
		if err != nil {
			return fmt.Errorf("error adding guild %d (%s): %v", e.ID, e.Name, err)
		}

		if created {
			log.Printf("added enabled guild %d (%s)", e.ID, e.Name)
		} else {
			log.Printf("added disabled guild %d (%s)", e.ID, e.Name)
		}
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) createMatchCategory(guildID discord.GuildID, pos int) (*discord.Channel, error) {
	everyone, err := b.everyone(guildID)
	if err != nil {
		return nil, err
	}
	category, err := b.state.CreateChannel(
		guildID,
		api.CreateChannelData{
			Name:     "matches",
			Type:     discord.GuildCategory,
			Position: option.NewInt(pos),
			Overwrites: []discord.Overwrite{
				{
					ID:   discord.Snowflake(everyone.ID),
					Type: discord.OverwriteRole,
					Deny: discord.PermissionAllText,
				},
				{
					ID:    discord.Snowflake(b.userID), // bot can access category
					Type:  discord.OverwriteMember,
					Allow: discord.PermissionAllText,
				},
			},
		})
	if err != nil {
		return nil, fmt.Errorf("error creating category: %w", err)
	}
	return category, nil
}

func (b *Bot) handleRemoveGuild(e *gateway.GuildDeleteEvent) {
	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Printf("error getting queries: %v", err)
		return
	}
	defer q.Close()

	err = q.DeleteGuildConfig(b.ctx, e.ID.String())
	if err != nil {
		log.Printf("error deleting guild %d: %v", e.ID, err)
		return
	}
	log.Printf("deleted guild %d", e.ID)
}
