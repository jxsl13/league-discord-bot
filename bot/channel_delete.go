package bot

import (
	"database/sql"
	"errors"
	"log"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleChannelDelete(e *gateway.ChannelDeleteEvent) {
	if e.Type != discord.GuildText && e.Type != discord.GuildCategory {
		return
	}

	var (
		guildID      = e.GuildID
		guildIDStr   = guildID.String()
		channelID    = e.Channel.ID
		channelIDStr = channelID.String()
	)

	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Println("error getting queries:", err)
		return
	}
	defer q.Close()

	if e.Type == discord.GuildText {
		// just delete match channel if it matches the channel id
		err = q.DeleteMatch(b.ctx, channelIDStr)
		if err != nil {
			log.Printf("error deleting match for channel %s: %v", channelID, err)
			return
		}
		return
	}

	// category channel -> guild config was modified

	r, err := q.GetGuildConfigByCategory(b.ctx, channelIDStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no config found, ignore
			return
		}
		log.Printf("error getting guild config for category %s: %v", channelIDStr, err)
		return
	}

	if r.Enabled == 0 {
		log.Println("guild config is disabled, ignoring")
		return
	}

	// config found
	channels, err := b.state.Channels(guildID)
	if err != nil {
		log.Printf("error getting guild %s: %v", guildIDStr, err)
		return
	}
	lastPos := discordutils.LastChannelPosition(channels)

	matches, err := q.ListGuildMatches(b.ctx, guildIDStr)
	if err != nil {
		log.Printf("error getting matches for guild %s: %v", guildIDStr, err)
		return
	}

	channelIDs := make([]discord.ChannelID, 0, len(matches))
	for _, m := range matches {
		id, err := discordutils.ParseChannelID(m.ChannelID)
		if err != nil {
			log.Printf("error parsing channel id %s: %v", m.ChannelID, err)
			continue
		}
		channelIDs = append(channelIDs, id)
	}

	category, err := b.createMatchCategory(e.GuildID, lastPos)
	if err != nil {
		log.Printf("error creating category for guild %s: %v", e.GuildID.String(), err)

		err = q.DisableGuild(b.ctx, guildIDStr)
		if err != nil {
			log.Printf("error disabling guild %s: %v", guildIDStr, err)
			return
		}
		log.Printf("disabled guild %s", guildIDStr)
		return
	}

	err = q.UpdateCategoryId(b.ctx, sqlc.UpdateCategoryIdParams{
		GuildID:    guildIDStr,
		CategoryID: category.ID.String(),
	})
	if err != nil {
		log.Printf("error updating category id for guild %s: %v", guildIDStr, err)
		return
	}

	// recreated category. look for all channels and move them back into the new category
	for _, id := range channelIDs {
		err = b.state.Client.ModifyChannel(id, api.ModifyChannelData{
			CategoryID: category.ID,
		})
		if err != nil {
			log.Printf("error modifying channel %s: %v", id, err)
			continue
		}
	}
}
