package bot

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func (b *Bot) checkRoleIDs(guildID discord.GuildID, roleIDs ...discord.RoleID) (err error) {
	roles, err := b.state.Roles(guildID)
	if err != nil {
		return fmt.Errorf("failed to check role ids: %w", err)
	}

outer:
	for _, id := range roleIDs {
	inner:
		for _, role := range roles {
			if role.Name == "@everyone" {
				if id == role.ID {
					return fmt.Errorf("invalid role %s", role.Name)
				}

				continue inner
			}

			if id == role.ID {
				continue outer
			}
		}
		return fmt.Errorf("role %s not found", discord.Snowflake(id))
	}

	return nil
}

func (b *Bot) resolveRoles(guildID discord.GuildID, roleMentions []string) (result []discord.Role, err error) {

	roles, err := b.state.Roles(guildID)
	if err != nil {
		return nil, err
	}

	result = make([]discord.Role, 0, len(roleMentions))
outer:
	for _, m := range roleMentions {
		for _, role := range roles {
			if m == role.Mention() {
				result = append(result, role)
				continue outer
			} else if m == role.ID.String() {
				result = append(result, role)
				continue outer
			} else if m == role.Name {
				result = append(result, role)
				continue outer
			}
		}
		return nil, fmt.Errorf("role %s not found", m)
	}

	return result, nil
}

func (b *Bot) resolveRoleIDs(guildID discord.GuildID, roleIDs []discord.RoleID) (result map[discord.RoleID]discord.Role, err error) {

	roles, err := b.state.Roles(guildID)
	if err != nil {
		return nil, err
	}

	result = make(map[discord.RoleID]discord.Role, len(roleIDs))
outer:
	for _, id := range roleIDs {
		for _, role := range roles {
			if id == role.ID {
				result[id] = role
				continue outer
			}
		}
		return nil, fmt.Errorf("role %s not found", id)
	}

	return result, nil
}
