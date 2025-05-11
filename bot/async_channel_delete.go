package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncDeleteExpiredChannels(ctx context.Context) (d time.Duration, err error) {
	log.Println("checking for expired match channels")
	defer func() {
		if err != nil {
			log.Printf("failed to delete expired match channels: %v", err)
		}
	}()

	var deletes []sqlc.ListNowDeletableChannelsRow
	err = b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) (err error) {
		deletes, err = q.ListNowDeletableChannels(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no channels to delete
				return sql.ErrNoRows
			}
			return fmt.Errorf("error getting next match channel to delete: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no channels to delete
			return 0, nil
		}
		return 0, err
	}

	if len(deletes) == 0 {
		return 0, nil
	}

	for _, del := range deletes {
		var (
			deleteAt    = time.Unix(del.ChannelDeleteAt, 0).Truncate(time.Second)
			scheduledAt = time.Unix(del.ScheduledAt, 0).Truncate(time.Second)
		)

		cid, err := parse.ChannelID(del.ChannelID)
		if err != nil {
			return 0, fmt.Errorf("error parsing channel ID: %w", err)
		}

		reason := fmt.Sprintf(
			"Match channel is being deleted due to it's lifetime being reached at %s. The corresponding match was at %s.",
			deleteAt,
			scheduledAt,
		)
		err = b.state.DeleteChannel(cid, api.AuditLogReason(reason))
		if err != nil {
			return 0, err
		}

		log.Printf("deleted expired channel %s, match was scheduled at: %s, channel expired at: %s",
			cid,
			scheduledAt,
			time.Unix(del.ChannelDeleteAt, 0),
		)
	}

	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return d, nil
}
