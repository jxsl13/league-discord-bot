package bot

import (
	"context"
	"fmt"
	"log"
	"slices"

	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) deleteOphanedMatches(ctx context.Context, q *sqlc.Queries, channelIDs ...string) (err error) {
	if len(channelIDs) == 0 {
		return nil
	}

	log.Printf("deleting %d orphaned matches", len(channelIDs))
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to delete %d orphaned matches: %w", len(channelIDs), err)
		} else {
			log.Printf("deleted %d orphaned matches", len(channelIDs))
		}
	}()

	if len(channelIDs) == 1 {
		// delete single match
		err = q.DeleteMatch(ctx, channelIDs[0])
		if err != nil {
			return
		}
		return nil
	}

	slices.Sort(channelIDs)
	channelIDs = slices.Compact(channelIDs)

	return q.DeleteMatchList(ctx, channelIDs)
}
