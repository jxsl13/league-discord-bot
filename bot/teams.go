package bot

import (
	"context"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) listMatchTeamRoleIDs(ctx context.Context, q *sqlc.Queries, channelID discord.ChannelID) (_ []discord.RoleID, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error listing match teams: %w", err)
		}
	}()
	// we need to give access to the corresponding teams
	teams, err := q.ListMatchTeams(b.ctx, channelID.String())
	if err != nil {
		return nil, fmt.Errorf("error getting match teams: %w", err)
	}

	teamRoleIDs := make([]discord.RoleID, 0, len(teams))
	for _, team := range teams {
		rid, err := discordutils.ParseRoleID(team.RoleID)
		if err != nil {
			return nil, fmt.Errorf("error parsing team role ID: %w", err)
		}
		teamRoleIDs = append(teamRoleIDs, rid)
	}

	return teamRoleIDs, nil
}
