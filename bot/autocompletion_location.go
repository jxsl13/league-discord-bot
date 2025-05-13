package bot

import (
	"log"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/internal/tz"
	"github.com/lithammer/fuzzysearch/fuzzy"
)

func (b *Bot) handleAutocompletionLocationInteraction(e *gateway.InteractionCreateEvent) {
	d, ok := e.Data.(*discord.AutocompleteInteraction)
	if !ok {
		return
	}
	focused := d.Options.Focused()
	if focused.Name != "location" {
		return
	}

	searchTerm := focused.String()

	ranks := fuzzy.RankFindFold(searchTerm, tz.TimeZones)
	if len(ranks) > 25 {
		ranks = ranks[:25]
	}

	choices := make(api.AutocompleteStringChoices, 0, len(ranks))
	for _, r := range ranks {
		choices = append(choices, discord.StringChoice{
			Name:  r.Target,
			Value: r.Target,
		})
	}
	resp := api.InteractionResponse{
		Type: api.AutocompleteResult,
		Data: &api.InteractionResponseData{
			Choices: &choices,
		},
	}

	if err := b.state.RespondInteraction(e.ID, e.Token, resp); err != nil {
		log.Println("failed to send interaction callback:", err)
	}
}
