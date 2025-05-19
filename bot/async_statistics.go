package bot

import (
	"context"
	"log"

	"github.com/jxs13/league-discord-bot/sqlc"
)

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

		numConfiguredAnnouncements, err := q.CountAnnouncements(ctx)
		if err != nil {
			return err
		}

		numEnabledEventCreation, err := q.CountEnabledEventCreation(ctx)
		if err != nil {
			return err
		}

		log.Println("daily statistics:")
		log.Printf("  %d enabled guilds", enabledGuilds)
		log.Printf("  %d disabled guilds", disabledGuilds)
		log.Printf("  %d total guilds", enabledGuilds+disabledGuilds)
		log.Printf("  %d enabled event creation", numEnabledEventCreation)
		log.Printf("  %d total matches", numMatches)
		log.Printf("  %d total notifications", numNotifications)
		log.Printf("  %d total configured announcements", numConfiguredAnnouncements)

		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
