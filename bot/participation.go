package bot

import (
	"database/sql"
	"errors"
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) handleAddParticipationReaction(e *gateway.MessageReactionAddEvent) {
	if b.isMe(e.UserID) || e.Emoji.Name != ReactionEmoji || e.Member == nil {
		return
	}

	var (
		channelID = e.ChannelID
		roleIDs   = e.Member.RoleIDs
	)

	ids := make([]string, 0, len(roleIDs))
	for _, rid := range roleIDs {
		ids = append(ids, rid.String())
	}

	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Println("error getting queries:", err)
		return
	}
	defer q.Close()

	team, err := q.GetMatchTeamByRoles(b.ctx, sqlc.GetMatchTeamByRolesParams{
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

	q, err := b.Queries(b.ctx)
	if err != nil {
		log.Println("error getting queries:", err)
		return
	}
	defer q.Close()

	team, err := q.GetMatchTeamByRoles(b.ctx, sqlc.GetMatchTeamByRolesParams{
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

	// found match, remove user from match
	err = q.DecreaseMatchTeamConfirmedParticipants(
		b.ctx,
		sqlc.DecreaseMatchTeamConfirmedParticipantsParams{
			ChannelID: team.ChannelID,
			RoleID:    team.RoleID,
		})
	if err != nil {
		log.Printf("error decreasing match team confirmed participants for channel %s: %v", channelID, err)
		return
	}
	log.Printf("reoved user %s from match %s", member.User.Username, channelID)
}
