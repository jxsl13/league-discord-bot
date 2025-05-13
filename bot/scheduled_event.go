package bot

import (
	"context"
	"log"
	"slices"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleScheduledEventDelete(e *gateway.GuildScheduledEventDeleteEvent) {
	eventID := e.ID
	guildID := e.GuildID
	err := b.Queries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		return q.ResetEventID(b.ctx, sqlc.ResetEventIDParams{
			EventID: eventID.String(),
			GuildID: guildID.String(),
		})
	})
	if err != nil {
		log.Printf("failed to reset event ID for scheduled event %s in guild %s: %v", eventID, guildID, err)
	} else {
		log.Printf("reset event ID for scheduled event %s (deleted) in guild %s", eventID, guildID)
	}
}

func (b *Bot) handleScheduledEventUpdate(e *gateway.GuildScheduledEventUpdateEvent) {

	if !slices.Contains([]discord.EventStatus{discord.CompletedEvent, discord.CancelledEvent}, e.Status) {
		return
	}
	eventID := e.ID
	guildID := e.GuildID

	err := b.Queries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		return q.ResetEventID(b.ctx, sqlc.ResetEventIDParams{
			EventID: eventID.String(),
			GuildID: guildID.String(),
		})
	})
	if err != nil {
		log.Printf("failed to reset event ID for scheduled event %s in guild %s: %v", eventID, guildID, err)
	} else {
		log.Printf("reset event ID for scheduled event %s (%d) in guild %s", eventID, e.Status, guildID)
	}
}
