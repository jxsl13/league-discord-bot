package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncDeleteExpiredChannels() (err error) {
	defer func() {
		if err != nil {
			log.Printf("failed to delete expired match channels: %v", err)
		}
	}()

	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) (err error) {
		deletes, err := q.ListNowDeletableChannels(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no channels to delete
				return sql.ErrNoRows
			}
			return fmt.Errorf("error getting next match channel to delete: %w", err)
		}
		if len(deletes) == 0 {
			return nil
		}

		orphanedMatches := make([]string, 0)
		for _, del := range deletes {
			var (
				deleteAt    = time.Unix(del.ChannelDeleteAt, 0).Truncate(time.Second)
				scheduledAt = time.Unix(del.ScheduledAt, 0).Truncate(time.Second)
			)

			if del.EventID != "" {
				// try deleting the scheduled event
				guildID, err := parse.GuildID(del.GuildID)
				if err != nil {
					return fmt.Errorf("failed to parse guild id for channel deletion: %w", err)
				}

				eventID, err := parse.EventID(del.EventID)
				if err != nil {
					return fmt.Errorf("failed to parse event id for channel deletion: %w", err)
				}

				err = b.state.DeleteScheduledEvent(guildID, eventID)
				if err != nil && !discordutils.IsStatus4XX(err) {
					return fmt.Errorf("error deleting scheduled event %s in guild %s: %w", eventID, del.GuildID, err)
				}
			}

			cid, err := parse.ChannelID(del.ChannelID)
			if err != nil {
				return err
			}

			reason := fmt.Sprintf(
				"Match channel is being deleted due to it's lifetime being reached at %s. The corresponding match was at %s.",
				deleteAt,
				scheduledAt,
			)
			err = b.state.DeleteChannel(cid, api.AuditLogReason(reason))
			if err != nil {
				if discordutils.IsStatus4XX(err) {
					// not found -> delete match manually
					log.Printf("channel %s not found (%v), adding to orphaned list for deletion", cid, err)
					orphanedMatches = append(orphanedMatches, del.ChannelID)
					continue
				}
				return err
			}

			log.Printf("deleted expired channel %s, match was scheduled at: %s, channel expired at: %s",
				cid,
				scheduledAt,
				time.Unix(del.ChannelDeleteAt, 0),
			)
		}

		if len(orphanedMatches) > 0 {
			err = b.deleteOphanedMatches(ctx, q, orphanedMatches...)
			if err != nil {
				return fmt.Errorf("error deleting orphaned matches: %w", err)
			}
		} else {
			err = b.refreshJobSchedules(ctx, q)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return nil
}
