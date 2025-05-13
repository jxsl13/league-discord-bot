package bot

import (
	"log"
	"slices"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleScheduledEventDelete(e *gateway.GuildScheduledEventDeleteEvent) {
	eventID := e.ID
	guildID := e.GuildID
	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Printf("error getting queries for scheduled event %s in guild %s: %v", eventID, guildID, err)
		return
	}

	err = q.ResetEventID(b.ctx, sqlc.ResetEventIDParams{
		EventID: eventID.String(),
		GuildID: guildID.String(),
	})
	if err != nil {
		log.Printf("error resetting event ID for scheduled event %s in guild %s: %v", eventID, guildID, err)
		return
	}
}

func (b *Bot) handleScheduledEventUpdate(e *gateway.GuildScheduledEventUpdateEvent) {

	if !slices.Contains([]discord.EventStatus{discord.CompletedEvent, discord.CancelledEvent}, e.Status) {
		return
	}
	eventID := e.ID
	guildID := e.GuildID
	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Printf("error getting queries for scheduled event %s in guild %s: %v", eventID, guildID, err)
		return
	}

	err = q.ResetEventID(b.ctx, sqlc.ResetEventIDParams{
		EventID: eventID.String(),
		GuildID: guildID.String(),
	})
	if err != nil {
		log.Printf("error resetting event ID for scheduled event %s in guild %s: %v", eventID, guildID, err)
		return
	}
}
