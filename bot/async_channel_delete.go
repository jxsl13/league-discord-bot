package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/jxs13/league-discord-bot/discordutils"
)

func (b *Bot) asyncDeleteExpiredMatchChannel() (d time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in channel delete routine: %v", err)
		}
	}()
	q, err := b.Queries(b.ctx)
	if err != nil {
		return 0, err
	}
	defer q.Close()

	mc, err := q.NextMatchChannelDelete(b.ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no channels to delete
			return 0, nil
		}
		return 0, fmt.Errorf("error getting next match channel to delete: %w", err)
	}

	var (
		deleteAt    = time.Unix(mc.ChannelDeleteAt, 0).Truncate(time.Second)
		scheduledAt = time.Unix(mc.ScheduledAt, 0).Truncate(time.Second)
	)

	cid, err := discordutils.ParseChannelID(mc.ChannelID)
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

	log.Printf("deleted channel %s, match was scheduled at: %s", cid, scheduledAt)

	return 0, nil
}
