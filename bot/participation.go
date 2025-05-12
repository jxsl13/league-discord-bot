package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/internal/sliceutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleAddParticipationReaction(e *gateway.MessageReactionAddEvent) {
	if b.isMe(e.UserID) || e.Emoji.Name != ReactionEmoji || e.Member == nil {
		return
	}

	var (
		channelID   = e.ChannelID.String()
		userRoleIDs = e.Member.RoleIDs
	)

	err := b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		req, err := q.GetParticipationRequirements(ctx, channelID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no match found, ignore
				return nil
			}
			return fmt.Errorf("error getting match for channel %s: %w", channelID, err)
		}

		if req.EntryClosed == 1 {
			// removing emoji reacion, the participation entry is closed.
			err = b.state.DeleteUserReaction(e.ChannelID, e.MessageID, e.UserID, ReactionEmoji)
			if err != nil {
				return fmt.Errorf("error removing reaction %s from message %s: %w", ReactionEmoji, e.MessageID, err)
			}
			return nil
		}

		rids := make([]string, 0, len(userRoleIDs))
		for _, rid := range userRoleIDs {
			rids = append(rids, rid.String())
		}

		teams, err := q.GetMatchTeamByRoles(ctx, sqlc.GetMatchTeamByRolesParams{
			ChannelID: channelID,
			RoleIds:   rids,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no match found, ignore
				return nil
			}
			return fmt.Errorf("error getting match for channel %s: %w", channelID, err)
		} else if len(teams) == 0 {
			return nil
		} else if len(teams) > 1 {
			// removing emoji reacion, because the user has both teams as roles
			err = b.state.DeleteUserReaction(e.ChannelID, e.MessageID, e.UserID, ReactionEmoji)
			if err != nil {
				return fmt.Errorf("error removing reaction %s from message %s: %w", ReactionEmoji, e.MessageID, err)
			}
			return nil
		}

		// has exactly one team associated with the user in the match
		team := teams[0]

		// found match, add user to match
		err = q.IncreaseMatchTeamConfirmedParticipants(
			ctx,
			sqlc.IncreaseMatchTeamConfirmedParticipantsParams{
				ChannelID: team.ChannelID,
				RoleID:    team.RoleID,
			})
		if err != nil {
			return fmt.Errorf("error increasing match team confirmed participants for channel %s: %w", channelID, err)
		}
		log.Printf("added user %s to match %s", e.Member.User.Username, channelID)
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}

func (b *Bot) handleRemoveParticipationReaction(e *gateway.MessageReactionRemoveEvent) {
	if b.isMe(e.UserID) || e.Emoji.Name != ReactionEmoji {
		return
	}

	var (
		guildID   = e.GuildID
		channelID = e.ChannelID
		userID    = e.UserID
	)

	member, err := b.state.Member(guildID, userID)
	if err != nil {
		log.Println(fmt.Errorf("error getting member %s: %w", userID, err))
		return
	}
	userRoleIDs := member.RoleIDs

	rids := make([]string, 0, len(userRoleIDs))
	for _, rid := range userRoleIDs {
		rids = append(rids, rid.String())
	}

	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {

		teams, err := q.GetMatchTeamByRoles(ctx, sqlc.GetMatchTeamByRolesParams{
			ChannelID: channelID.String(),
			RoleIds:   rids,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no match found, ignore
				return nil
			}
			return fmt.Errorf("error getting match for channel %s: %v", channelID, err)
		} else if len(teams) == 0 {
			// no match team found
			return nil
		}

		var teamRoleID string
		if len(teams) == 1 {
			teamRoleID = teams[0].RoleID
		} else if len(teams) > 1 {

			teamRoleIDs := make([]discord.RoleID, 0, len(teams))
			for _, team := range teams {
				roleID, err := parse.RoleID(team.RoleID)
				if err != nil {
					return fmt.Errorf("error parsing role ID %s: %v", team.RoleID, err)
				}
				teamRoleIDs = append(teamRoleIDs, roleID)
			}
			// we cannot recreate a user's reaction, which is why we need to try to guess as best as we can, where
			// to remove the user from.
			// this case should not happen, because we try to prevent the user from creating reactions when he has both teams assigned.
			roleID, ok := sliceutils.ContainsOne(userRoleIDs, teamRoleIDs...)
			if !ok {
				return fmt.Errorf("invalid state, user does not have role ids, even tho he should have them: expected to have one of %v, but has %v", teamRoleIDs, userRoleIDs)
			}
			teamRoleID = roleID.String()
		}

		// found match, remove user from match
		err = q.DecreaseMatchTeamConfirmedParticipants(
			ctx,
			sqlc.DecreaseMatchTeamConfirmedParticipantsParams{
				ChannelID: channelID.String(),
				RoleID:    teamRoleID,
			})
		if err != nil {
			return fmt.Errorf("error decreasing match team confirmed participants for channel %s: %v", channelID, err)
		}
		log.Printf("removed user %s from match %s", member.User.Username, channelID)
		return nil
	})
	if err != nil {
		log.Println(err)
	}
}
