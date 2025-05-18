package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/options"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) commandAnnouncementConfiguration(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {
	guildID := data.Event.GuildID.String()

	var content string
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, READ)
		if err != nil {
			return err
		}

		a, err := q.GetAnnouncement(ctx, guildID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				content = "No announcements configured for this server."
				return nil
			}
			return err
		}

		targetChannelID, err := parse.ChannelID(a.ChannelID)
		if err != nil {
			return err
		}

		interval := time.Duration(a.Interval) * time.Second

		startsAt := time.Unix(a.StartsAt, 0)
		endAt := time.Unix(a.EndsAt, 0)

		var sb strings.Builder
		sb.WriteString("Announcements are configured for this server:\n")
		sb.WriteString("announcement_channel: ")
		sb.WriteString(targetChannelID.Mention())
		sb.WriteString("\n")
		sb.WriteString("starts_at: ")
		sb.WriteString(format.DiscordLongDateTime(startsAt))
		sb.WriteString("\n")
		sb.WriteString("ends_at: ")
		sb.WriteString(format.DiscordLongDateTime(endAt))
		sb.WriteString("\n")
		sb.WriteString("interval: ")
		sb.WriteString(format.MarkdownInlineCodeBlock(interval.String()))
		sb.WriteString("\n")

		if a.CustomTextBefore != "" {
			sb.WriteString("custom_text_before: ")
			sb.WriteString(format.MarkdownInlineCodeBlock(a.CustomTextBefore))
			sb.WriteString("\n")
		}
		if a.CustomTextAfter != "" {
			sb.WriteString("custom_text_after: ")
			sb.WriteString(format.MarkdownInlineCodeBlock(a.CustomTextAfter))
			sb.WriteString("\n")
		}

		content = sb.String()
		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Flags:   discord.EphemeralMessage,
		Content: option.NewNullableString(content),
	}
}

func (b *Bot) commandAnnouncementsDisable(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {
	guildID := data.Event.GuildID.String()

	var content string
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, WRITE)
		if err != nil {
			return err
		}

		_, err = q.GetAnnouncement(ctx, guildID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				content = "No announcements configured for this server."
				return nil
			}
			return err
		}

		err = q.DeleteAnnouncement(ctx, guildID)
		if err != nil {
			return err
		}
		content = "Announcements disabled for this server."
		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Flags:   discord.EphemeralMessage,
		Content: option.NewNullableString(content),
	}
}

func (b *Bot) commandAnnouncementsEnable(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {

	var content string
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, WRITE)
		if err != nil {
			return err
		}

		targetChannelID, err := options.ChannelID("announcement_channel", data.Options)
		if err != nil {
			return err
		}

		err = b.checkIsGuildChannel(data.Event, targetChannelID)
		if err != nil {
			return err
		}

		interval, err := options.Duration("interval", time.Hour, 8760*time.Hour, data.Options)
		if err != nil {
			return err
		}

		startsAt, err := options.FutureTimeInLocation(
			"starts_at",
			"location",
			max(time.Minute, b.loopInterval*2),
			data.Options,
		)
		if err != nil {
			return err
		}

		endsAt, err := options.FutureTimeInLocation(
			"ends_at",
			"location",
			max(time.Minute, b.loopInterval*2),
			data.Options,
		)
		if err != nil {
			return err
		}

		if startsAt.After(endsAt) {
			return errors.New("starts_at must be before ends_at")
		}

		customTextBefore := data.Options.Find("custom_text_before").String()
		customTextAfter := data.Options.Find("custom_text_after").String()

		err = q.AddAnnouncement(ctx, sqlc.AddAnnouncementParams{
			GuildID:          data.Event.GuildID.String(),
			ChannelID:        targetChannelID.String(),
			Interval:         int64(interval / time.Second),
			StartsAt:         startsAt.Unix(),
			EndsAt:           endsAt.Unix(),
			CustomTextBefore: customTextBefore,
			CustomTextAfter:  customTextAfter,
			LastAnnouncedAt:  startsAt.Unix() - int64(interval/time.Second), // needs to be correcty initialized
		})
		if err != nil {
			return err
		}

		content = fmt.Sprintf(
			"Announcements enabled for this server. First announcement will be at %s in channel %s",
			format.DiscordLongDateTime(startsAt),
			targetChannelID.Mention(),
		)
		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Flags:   discord.EphemeralMessage,
		Content: option.NewNullableString(content),
	}
}
