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
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncDeleteExpiredMatchChannel() (d time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in channel delete routine: %v", err)
		}
	}()
	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		mc, err := q.NextMatchChannelDelete(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no channels to delete
				return nil
			}
			return fmt.Errorf("error getting next match channel to delete: %w", err)
		}

		var (
			deleteAt    = time.Unix(mc.ChannelDeleteAt, 0).Truncate(time.Second)
			scheduledAt = time.Unix(mc.ScheduledAt, 0).Truncate(time.Second)
		)

		cid, err := discordutils.ParseChannelID(mc.ChannelID)
		if err != nil {
			return fmt.Errorf("error parsing channel ID: %w", err)
		}

		reason := fmt.Sprintf(
			"Match channel is being deleted due to it's lifetime being reached at %s. The corresponding match was at %s.",
			deleteAt,
			scheduledAt,
		)
		err = b.state.DeleteChannel(cid, api.AuditLogReason(reason))
		if err != nil {
			return err
		}

		log.Printf("deleted channel %s, match was scheduled at: %s", cid, scheduledAt)

		return nil
	})
	if err != nil {
		return 0, err
	}

	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return d, nil
}
