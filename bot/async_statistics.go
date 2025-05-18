package bot

import (
	"context"
	"log"
	"time"

	"github.com/jxs13/league-discord-bot/internal/timerutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func untilMidnight(from ...time.Time) time.Duration {
	now := time.Now()
	if len(from) > 0 {
		now = from[0]
	}
	_, offset := now.Zone()
	midnight := now.Add(24 * time.Hour).Truncate(24 * time.Hour)
	d := midnight.Sub(now) - time.Duration(offset)*time.Second
	log.Printf("until next statistics: %s", d)

	return d
}

func (b *Bot) asyncStatistics() {
	timer, drained := timerutils.NewTimer()
	defer timerutils.CloseTimer(timer, &drained)

	for {
		select {
		case <-b.ctx.Done():
			log.Println("stopping daily statistics routine")
			return
		case t := <-timer.C:
			drained = true

			_ = b.printDailyStatistics()
			timerutils.ResetTimer(timer, untilMidnight(t), &drained)
		}
	}
}

func (b *Bot) printDailyStatistics() (err error) {
	defer func() {
		if err != nil {
			log.Printf("error while printing daily statistics: %v", err)
		}
	}()

	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		enabledGuilds, err := q.CountEnabledGuilds(ctx)
		if err != nil {
			return err
		}

		disabledGuilds, err := q.CountDisabledGuilds(ctx)
		if err != nil {
			return err
		}

		numMatches, err := q.CountAllMatches(ctx)
		if err != nil {
			return err
		}

		numNotifications, err := q.CountAllNotifications(ctx)
		if err != nil {
			return err
		}

		log.Println("daily statistics:")
		log.Printf("  %d enabled guilds", enabledGuilds)
		log.Printf("  %d disabled guilds", disabledGuilds)
		log.Printf("  %d total guilds", enabledGuilds+disabledGuilds)
		log.Printf("  %d total matches", numMatches)
		log.Printf("  %d total notifications", numNotifications)

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
