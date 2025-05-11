package bot

import (
	"database/sql"
	"errors"
	"log"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/internal/sliceutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleAddParticipationReaction(e *gateway.MessageReactionAddEvent) {
	if b.isMe(e.UserID) || e.Emoji.Name != ReactionEmoji || e.Member == nil {
		return
	}

	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Println("error getting queries:", err)
		return
	}
	defer q.Close()

	var (
		channelID = e.ChannelID.String()
		roleIDs   = e.Member.RoleIDs
	)

	m, err := q.GetMatch(b.ctx, channelID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no match found, ignore
			return
		}
		log.Printf("error getting match for channel %s: %v", channelID, err)
		return
	}

	if m.ParticipationEntryClosed == 1 {
		// removing emoji reacion, the participation entry is closed.
		err = b.state.DeleteUserReaction(e.ChannelID, e.MessageID, e.UserID, ReactionEmoji)
		if err != nil {
			log.Printf("error removing reaction %s from message %s: %v", ReactionEmoji, e.MessageID, err)
		}
		return
	}

	ids := make([]string, 0, len(roleIDs))
	for _, rid := range roleIDs {
		ids = append(ids, rid.String())
	}

	teams, err := q.GetMatchTeamByRoles(b.ctx, sqlc.GetMatchTeamByRolesParams{
		ChannelID: channelID,
		RoleIds:   ids,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no match found, ignore
			return
		}
		log.Printf("error getting match for channel %s: %v", channelID, err)
		return
	}

	if len(teams) > 1 {
		// removing emoji reacion, because the user has both teams as roles
		err = b.state.DeleteUserReaction(e.ChannelID, e.MessageID, e.UserID, ReactionEmoji)
		if err != nil {
			log.Printf("error removing reaction %s from message %s: %v", ReactionEmoji, e.MessageID, err)
		}
		return
	}

	team := teams[0]

	// found match, add user to match
	err = q.IncreaseMatchTeamConfirmedParticipants(
		b.ctx,
		sqlc.IncreaseMatchTeamConfirmedParticipantsParams{
			ChannelID: team.ChannelID,
			RoleID:    team.RoleID,
		})
	if err != nil {
		log.Printf("error increasing match team confirmed participants for channel %s: %v", channelID, err)
		return
	}
	log.Printf("added user %s to match %s", e.Member.User.Username, channelID)
}

func (b *Bot) handleRemoveParticipationReaction(e *gateway.MessageReactionRemoveEvent) {
	if b.isMe(e.UserID) || e.Emoji.Name != ReactionEmoji {
		return
	}

	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Println("error getting queries:", err)
		return
	}
	defer q.Close()

	var (
		guildID   = e.GuildID
		channelID = e.ChannelID
		userID    = e.UserID
	)

	member, err := b.state.Member(guildID, userID)
	if err != nil {
		log.Printf("error getting member %s: %v", userID, err)
		return
	}
	roleIDs := member.RoleIDs

	ids := make([]string, 0, len(roleIDs))
	for _, rid := range roleIDs {
		ids = append(ids, rid.String())
	}

	teams, err := q.GetMatchTeamByRoles(b.ctx, sqlc.GetMatchTeamByRolesParams{
		ChannelID: channelID.String(),
		RoleIds:   ids,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no match found, ignore
			return
		}
		log.Printf("error getting match for channel %s: %v", channelID, err)
		return
	}

	var teamRoleID string
	if len(teams) == 1 {
		teamRoleID = teams[0].RoleID
	} else if len(teams) > 1 {

		teamRoleIDs := make([]discord.RoleID, 0, len(teams))
		for _, team := range teams {
			roleID, err := discordutils.ParseRoleID(team.RoleID)
			if err != nil {
				log.Printf("error parsing role ID %s: %v", team.RoleID, err)
				return
			}
			teamRoleIDs = append(teamRoleIDs, roleID)
		}
		// we cannot recreate a user's reaction, which is why we need to try to guess as best as we can, where
		// to remove the user from.
		// this case should not happen, because we try to prevent the user from creating reactions when he has both teams assigned.
		roleID, ok := sliceutils.ContainsOne(roleIDs, teamRoleIDs...)
		if !ok {
			log.Printf("invalid state, user does not have role ids, even tho he should have them: expected to have one of %v, but has %v", teamRoleIDs, roleIDs)
			return
		}
		teamRoleID = roleID.String()
	}

	// found match, remove user from match
	err = q.DecreaseMatchTeamConfirmedParticipants(
		b.ctx,
		sqlc.DecreaseMatchTeamConfirmedParticipantsParams{
			ChannelID: channelID.String(),
			RoleID:    teamRoleID,
		})
	if err != nil {
		log.Printf("error decreasing match team confirmed participants for channel %s: %v", channelID, err)
		return
	}
	log.Printf("removed user %s from match %s", member.User.Username, channelID)
}
