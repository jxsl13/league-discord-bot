package bot

import (
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
)

func (b *Bot) checkUserIDs(guildID discord.GuildID, userIDs ...discord.UserID) (err error) {
	if len(userIDs) == 0 {
		return nil
	} else if len(userIDs) == 1 {
		member, err := b.state.Member(guildID, userIDs[0])
		if err != nil {
			return fmt.Errorf("failed to check user id %s: %w", userIDs[0], err)
		}
		if member.User.ID != userIDs[0] {
			return fmt.Errorf("user %s not found", userIDs[0])
		}
		return nil
	}

	// > 1

	members, err := b.state.Members(guildID)
	if err != nil {
		return fmt.Errorf("failed to check user ids: %w", err)
	}

	available := make(map[discord.UserID]bool, len(members))
	for _, member := range members {
		available[member.User.ID] = true
	}

	for _, id := range userIDs {
		if _, ok := available[id]; !ok {
			return fmt.Errorf("user %s not found", id)
		}
	}

	return nil
}
