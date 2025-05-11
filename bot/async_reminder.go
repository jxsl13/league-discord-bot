package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/format"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncReminder(ctx context.Context) (d time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in reminder routine: %v", err)
		}
	}()
	err = b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		r, err := q.NextMatchReminder(ctx, b.reminder.MaxIndex())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no matches scheduled, nothing to send
				return nil
			}
			return fmt.Errorf("error getting next match reminder: %w", err)
		}

		scheduledAt := time.Unix(r.ScheduledAt, 0)
		ridx, untilNextReminder, ok := b.reminder.Next(r.ReminderCount, scheduledAt)
		if !ok {
			return nil
		}

		channelID, err := discordutils.ParseChannelID(r.ChannelID)
		if err != nil {
			return err
		}

		// we need to remind the teams, moderators and streamers
		teamRoleIDs, err := b.listMatchTeamRoleIDs(ctx, q, channelID)
		if err != nil {
			return err
		}

		modUserIDs, err := b.listMatchModeratorUserIDs(ctx, q, channelID)
		if err != nil {
			return err
		}

		streamers, err := b.listMatchStreamerUserIDs(ctx, q, channelID)
		if err != nil {
			return err
		}

		text := ""
		untilMatch := time.Until(scheduledAt)
		if untilMatch >= time.Minute {
			text = fmt.Sprintf("The match is starting in about %s. ", format.Duration(untilMatch))
		} else {
			text = "The match is starting now!"
		}

		content, allowedMentions := FormatNotification(
			text,
			"",
			teamRoleIDs,
			modUserIDs,
			streamers,
			nil,
		)

		msg := api.SendMessageData{
			Content:         content,
			AllowedMentions: allowedMentions,
		}

		_, err = b.state.SendMessageComplex(channelID, msg)
		if err != nil {
			return fmt.Errorf("error sending reminder message: %w", err)
		}

		// update reminder count
		newReminderCount := max(r.ReminderCount+1, ridx+1)
		err = q.UpdateMatchReminderCount(ctx, sqlc.UpdateMatchReminderCountParams{
			ChannelID:     r.ChannelID,
			ReminderCount: newReminderCount,
		})
		if err != nil {
			return fmt.Errorf("error updating reminder count: %w", err)
		}

		d = untilNextReminder
		return nil
	})
	if err != nil {
		return 0, err
	}

	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return d, nil
}
