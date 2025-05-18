package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/maputils"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncNotifications(ctx context.Context) (d time.Duration, err error) {
	started := time.Now()
	defer func() {
		if err != nil {
			log.Printf("error in reminder routine (started at %s): %v", started, err)
		}
	}()
	err = b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		notifications, err := q.ListNowDueNotifications(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no matches scheduled, nothing to send
				return nil
			}
			return fmt.Errorf("error getting next notifications: %w", err)
		}

		if len(notifications) == 0 {
			// no matches scheduled, nothing to send
			return nil
		}

		cnm := make(map[discord.ChannelID][]sqlc.Notification, len(notifications))
		for _, n := range notifications {
			channelID, err := parse.ChannelID(n.ChannelID)
			if err != nil {
				return err
			}
			l, ok := cnm[channelID]
			if !ok {
				l = make([]sqlc.Notification, 0, 2)
			}
			l = append(l, n)
			cnm[channelID] = l
		}

		orphanedMatches := make([]string, 0)
	outer:
		for _, channelID := range maputils.SortedKeys(cnm) {
			notifications := cnm[channelID]
			channelIDStr := channelID.String()

			match, err := q.GetMatch(ctx, channelIDStr)
			if err != nil {
				return fmt.Errorf("error getting match: %w", err)
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
			scheduledAt := time.Unix(match.ScheduledAt, 0)

			for _, n := range notifications {

				var msg api.SendMessageData
				if n.CustomText == "" {
					text := ""
					untilMatch := time.Until(scheduledAt)
					if untilMatch >= time.Minute {
						text = fmt.Sprintf("The match is starting in about %s. ", format.Duration(untilMatch))
					} else {
						text = "The match is starting now!"
					}

					msg = FormatNotification(
						text,
						"",
						teamRoleIDs,
						modUserIDs,
						streamers,
						nil,
					)
				} else {
					am := AllowedMentions(
						teamRoleIDs,
						modUserIDs,
						streamers,
						nil,
					)

					// allow mentioning all participants
					msg = api.SendMessageData{
						Content:         n.CustomText,
						AllowedMentions: am,
					}
				}

				_, err = b.state.SendMessageComplex(channelID, msg)
				if err != nil {
					if discordutils.IsStatus4XX(err) {
						// channel not found -> delete match manually
						log.Printf("channel %s not found, adding to orphaned list for deletion", channelID)
						orphanedMatches = append(orphanedMatches, channelIDStr)

						// we do not need to delete the notifications, because we will delete the match
						// and everything referencing it, especially all the notifications
						continue outer
					}

					return fmt.Errorf("error sending reminder message: %w", err)
				}

				err = q.DeleteNotification(ctx, sqlc.DeleteNotificationParams{
					ChannelID: channelIDStr,
					NotifyAt:  n.NotifyAt,
				})
				if err != nil {
					return fmt.Errorf("error deleting notification: %w", err)
				}

				log.Printf("sent notification (%s) for match %s, scheduled at %s",
					time.Unix(n.NotifyAt, 0),
					channelID,
					scheduledAt,
				)
			}
		}

		if len(orphanedMatches) > 0 {
			err = b.deleteOphanedMatches(ctx, q, orphanedMatches...)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return d, nil
}
