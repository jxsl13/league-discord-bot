package bot

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func (b *Bot) everyone(guildID discord.GuildID) (discord.Role, error) {
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
