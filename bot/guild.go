package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/jxs13/league-discord-bot/config"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/options"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) commandGuildConfiguration(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {

	text := ""
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, ADMIN)
		if err != nil {
			return err
		}

		cfg, err := q.GetGuildConfig(ctx, data.Event.GuildID.String())
		if err != nil {
			return err
		}

		var sb strings.Builder
		sb.Grow(512)

		var (
			accessOffset        = time.Duration(cfg.ChannelAccessOffset) * time.Second
			notificationOffsets = cfg.NotificationOffsets
			requirementsOffset  = time.Duration(cfg.RequirementsOffset) * time.Second
			deleteOffset        = time.Duration(cfg.ChannelDeleteOffset) * time.Second
		)

		sb.WriteString("Guild configuration:\n")
		sb.WriteString("enabled: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(strconv.FormatBool(int64ToBool(cfg.Enabled))))
		sb.WriteString(" whether the bot is enabled in this server or not\n\n")
		sb.WriteString("channel_access_offset: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(accessOffset.String()))
		sb.WriteString(" point in time before the match at which participants gain access to the channel\n\n")
		sb.WriteString("event_creation_enabled: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(strconv.FormatBool(int64ToBool(cfg.EventCreationEnabled))))
		sb.WriteString(" whether to create events in case there is a streamer with a stream_url available\n\n")
		sb.WriteString("notification_offsets: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(notificationOffsets))
		sb.WriteString(" list of points in time before the match, at which automatic notifications are created for the participants\n\n")
		sb.WriteString("requirements_offset: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(requirementsOffset.String()))
		sb.WriteString(" point in time before the match at which the participation requirements need to be met.\n\n")
		sb.WriteString("channel_delete_offset: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(deleteOffset.String()))
		sb.WriteString(" point in time after the match, at which the match channel is deleted and the Discord event ends.\n\n")

		text = sb.String()
		if len(text) > 2000 {
			text = text[:2000]
		}
		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Content: option.NewNullableString(text),
		Flags:   discord.EphemeralMessage,
	}
}

func (b *Bot) commandGuildConfigure(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {

	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		// disable guild enabled check for this command
		err := b.checkAccess(ctx, q, data.Event, ADMIN, true)
		if err != nil {
			return err
		}

		cfg, err := q.GetGuildConfig(ctx, data.Event.GuildID.String())
		if err != nil {
			return err
		}

		atLeastOneOption := false

		enabled, enabledOk, err := options.BoolInt64Option("enabled", data.Options)
		if err != nil {
			return err
		}
		atLeastOneOption = enabledOk || atLeastOneOption

		if enabledOk {
			cfg.Enabled = enabled
		}

		accessOffset, accessOffsetOk, err := options.DurationOption("channel_access_offset", 0, 720*time.Hour, data.Options)
		if err != nil {
			return err
		}
		atLeastOneOption = accessOffsetOk || atLeastOneOption

		if accessOffsetOk {
			cfg.ChannelAccessOffset = int64(accessOffset / time.Second)
		}

		deleteOffset, deleteOffsetOk, err := options.DurationOption("channel_delete_offset", 0, 8760*time.Hour, data.Options)
		if err != nil {
			return err
		}
		atLeastOneOption = deleteOffsetOk || atLeastOneOption

		if deleteOffsetOk {
			cfg.ChannelDeleteOffset = int64(deleteOffset / time.Second)
		}

		requirementsOffset, requirementsOffsetOk, err := options.DurationOption("requirements_offset", 0, 720*time.Hour, data.Options)
		if err != nil {
			return err
		}
		atLeastOneOption = requirementsOffsetOk || atLeastOneOption

		if requirementsOffsetOk {
			cfg.RequirementsOffset = int64(requirementsOffset / time.Second)
		}

		intervals, intervalsOk, err := options.ReminderIntervalsOption("notification_offsets", data.Options)
		if err != nil {
			return err
		}
		atLeastOneOption = intervalsOk || atLeastOneOption

		if intervalsOk {
			cfg.NotificationOffsets = format.ReminderIntervals(intervals)
		}

		eventCreationEnabled, eventCreationOK, err := options.BoolInt64Option("event_creation_enabled", data.Options)
		if err != nil {
			return err
		}
		atLeastOneOption = eventCreationOK || atLeastOneOption

		if eventCreationOK {
			cfg.EventCreationEnabled = eventCreationEnabled
		}

		if !atLeastOneOption {
			return errors.New("no options were provided, please provide at least one option to update")
		}

		// reuse validation logic from config
		err = config.ValidatableGuildConfig(
			time.Duration(cfg.ChannelAccessOffset)*time.Second,
			time.Duration(cfg.RequirementsOffset)*time.Second,
			time.Duration(cfg.ChannelDeleteOffset)*time.Second,
		)
		if err != nil {
			return err
		}

		err = q.UpdateGuildConfig(ctx, sqlc.UpdateGuildConfigParams{
			GuildID:              data.Event.GuildID.String(),
			Enabled:              cfg.Enabled,
			EventCreationEnabled: cfg.EventCreationEnabled,
			ChannelAccessOffset:  cfg.ChannelAccessOffset,
			NotificationOffsets:  cfg.NotificationOffsets,
			RequirementsOffset:   cfg.RequirementsOffset,
			ChannelDeleteOffset:  cfg.ChannelDeleteOffset,
		})
		if err != nil {
			err = fmt.Errorf("error adding guild config: %w", err)
			log.Println(err)
			return fmt.Errorf("%w, please contact the owner of the bot", err)
		}

		resp = &api.InteractionResponseData{
			Content:         option.NewNullableString("Guild configuration was updated. New match schedules will be created accordingly."),
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
		i, err := q.IsGuildEnabled(ctx, e.Guild.ID.String())
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
		err = q.AddGuildConfig(ctx, sqlc.AddGuildConfigParams{
			GuildID:             e.Guild.ID.String(),
			Enabled:             boolToInt64(created),
			CategoryID:          categoryID.String(),
			ChannelAccessOffset: int64(b.defaultChannelAccessOffset / time.Second),
			ChannelDeleteOffset: int64(b.defaultChannelDeleteOffset / time.Second),
			NotificationOffsets: b.DefaultReminderIntervals(),
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
	err := b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := q.DeleteGuildConfig(ctx, e.ID.String())
		if err != nil {
			return fmt.Errorf("error deleting guild %d: %v", e.ID, err)
		}
		log.Printf("deleted guild %d", e.ID)

		return b.refreshJobSchedules(ctx, q)
	})

	if err != nil {
		fmt.Println(err)
	}

}
