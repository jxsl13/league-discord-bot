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
	"github.com/jxs13/league-discord-bot/sqlc"
)

const (
	MaxConcurrentNotifications = 50
)

func (b *Bot) commandNotificationsList(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {
	var sb strings.Builder
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, READ)
		if err != nil {
			return err
		}

		channelID, err := options.ChannelID("match_channel", data.Options)
		if err != nil {
			return err
		}
		channelIDStr := channelID.String()

		err = b.checkIsGuildChannel(data.Event, channelID)
		if err != nil {
			return err
		}

		_, err = q.GetMatch(ctx, channelIDStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no corresponding match found for %s", channelID.Mention())
			}
			return fmt.Errorf("failed to get match for %s: %w", channelID.Mention(), err)
		}

		notifications, err := q.ListNotifications(ctx, channelIDStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				sb.WriteString(fmt.Sprintf("No notifications found for %s.", channelID.Mention()))
				return nil
			}
			return err
		}
		if len(notifications) == 0 {
			sb.WriteString(fmt.Sprintf("No notifications found for %s.", channelID.Mention()))
			return nil
		}
		sb.Grow((1 + len(notifications)) * 64)
		sb.WriteString(fmt.Sprintf("Notifications for %s:\n", channelID.Mention()))

		// max allowed are 50
		for i, n := range notifications {
			sb.WriteString(fmt.Sprintf("%2d at %s", i+1, format.DiscordLongDateTime(time.Unix(n.NotifyAt, 0))))
			if n.CustomText != "" {
				sb.WriteString("with custom text: ")
				sb.WriteString(format.MarkdownInlineCodeBlock(n.CustomText))
			}
			sb.WriteString("\n")
		}

		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	result := sb.String()
	if len(result) > 2000 {
		result = result[:2000-3] + "..."
	}
	return &api.InteractionResponseData{
		Content: option.NewNullableString(result),
		Flags:   discord.EphemeralMessage,
	}
}

func (b *Bot) commandNotificationsDelete(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {
	result := ""
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, WRITE)
		if err != nil {
			return err
		}

		channelID, err := options.ChannelID("match_channel", data.Options)
		if err != nil {
			return err
		}
		channelIDStr := channelID.String()

		err = b.checkIsGuildChannel(data.Event, channelID)
		if err != nil {
			return err
		}

		n, err := options.MinMaxInteger("list_number", data.Options, 1, 50)
		if err != nil {
			return err
		}

		maxNumber, err := q.CountNotifications(ctx, channelIDStr)
		if err != nil {
			return fmt.Errorf("error counting notifications for %s: %w", channelID.Mention(), err)
		}

		if n > maxNumber {
			return fmt.Errorf("number %d is out of range (1-%d)", n, maxNumber)
		}

		notification, err := q.GetNotificationByOffset(ctx, sqlc.GetNotificationByOffsetParams{
			ChannelID: channelID.String(),
			Offset:    n - 1,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				result = fmt.Sprintf("Notification not found for %s at position %d", channelID.Mention(), n)
				return nil
			}
			return err
		}

		err = q.DeleteNotification(ctx, sqlc.DeleteNotificationParams{
			ChannelID: channelIDStr,
			NotifyAt:  notification.NotifyAt,
		})
		if err != nil {
			return fmt.Errorf("error deleting notification for %s at position %d: %w", channelID.Mention(), n, err)
		}

		err = b.refreshJobSchedules(ctx, q)
		if err != nil {
			return err
		}

		result = fmt.Sprintf(
			"Notification %d (%s) deleted for %s",
			n,
			format.DiscordLongDateTime(time.Unix(notification.NotifyAt, 0)),
			channelID.Mention(),
		)
		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Content: option.NewNullableString(result),
		Flags:   discord.EphemeralMessage,
	}
}

func (b *Bot) commandNotificationsAdd(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {
	result := ""
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, WRITE)
		if err != nil {
			return err
		}

		channelID, err := options.ChannelID("match_channel", data.Options)
		if err != nil {
			return err
		}
		channelIDStr := channelID.String()

		err = b.checkIsGuildChannel(data.Event, channelID)
		if err != nil {
			return err
		}

		customText := data.Options.Find("custom_text").String()

		match, err := q.GetMatch(ctx, channelIDStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("no corresponding match found for %s", channelID.Mention())
			}
			return fmt.Errorf("error getting match for %s: %w", channelID.Mention(), err)
		}
		now := time.Now()

		notifyAt, err := options.TimeBetweenInLocation(
			"notify_at",
			"location",
			now.Add(time.Minute),
			time.Unix(match.ScheduledAt, 0),
			data.Options,
		)
		if err != nil {
			return err
		}

		n, err := q.CountNotifications(ctx, channelIDStr)
		if err != nil {
			return fmt.Errorf("error counting notifications for %s: %w", channelID.Mention(), err)
		}

		if n >= MaxConcurrentNotifications {
			return fmt.Errorf("maximum number of concurrent notifications (%d) reached for %s", MaxConcurrentNotifications, channelID.Mention())
		}

		err = q.AddNotification(ctx, sqlc.AddNotificationParams{
			ChannelID:  channelID.String(),
			NotifyAt:   notifyAt.Unix(),
			CustomText: customText,
		})
		if err != nil {
			return fmt.Errorf("error adding notification for %s: %w", channelID.Mention(), err)
		}

		return b.refreshJobSchedules(ctx, q)
	})
	if err != nil {
		return errorResponse(err)
	}

	return &api.InteractionResponseData{
		Content: option.NewNullableString(result),
		Flags:   discord.EphemeralMessage,
	}
}
