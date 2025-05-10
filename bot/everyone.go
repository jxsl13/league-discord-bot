package bot

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func (b *Bot) everyone(guildID discord.GuildID) (discord.Role, error) {
	/*
		guild, err := b.state.Guild(guildID)
		if err != nil {
			return nil, fmt.Errorf("error getting guild %d: %w", guildID, err)
		}

		if len(guild.Roles) == 0 {
			return nil, fmt.Errorf("guild %d has no roles", guildID)
		}

		everyone := guild.Roles[0]
		if everyone.Name != "@everyone" {
			return nil, fmt.Errorf("guild %d has no @everyone role", guildID)
		}
	*/
	roles, err := b.state.Roles(guildID)
	if err != nil {
		return discord.Role{}, err
	}

	for _, role := range roles {
		if role.Name == "@everyone" {
			return role, nil
		}
	}

	return discord.Role{}, fmt.Errorf("failed to get @everyone role: guild %d has no @everyone role", guildID)
}
