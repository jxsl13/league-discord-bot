package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/internal/sliceutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) getConfirmedParticipants(
	guildID discord.GuildID,
	channelID discord.ChannelID,
	messageID discord.MessageID,
	participantsPerTeam int64,
	teamRoles ...discord.RoleID,
) (teamParticipants map[discord.RoleID][]discord.UserID, full bool, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error getting confirmed participants: %w", err)
		}
	}()

	if len(teamRoles) == 0 {
		return nil, false, errors.New("no team roles provided")
	}

	expectedParticipants := uint(participantsPerTeam) * uint(len(teamRoles))
	users, err := b.state.Reactions(channelID, messageID, ReactionEmoji, 2*expectedParticipants+1) // +1 for the bot
	if err != nil {
		return nil, false, fmt.Errorf("error getting reactions: %w", err)
	}

	// initialize buckets for each team role
	buckets := make(map[discord.RoleID][]discord.UserID, len(teamRoles))
	for _, role := range teamRoles {
		buckets[role] = make([]discord.UserID, 0, expectedParticipants)
	}

	for _, u := range users {
		if b.isMe(u.ID) {
			continue
		}
		member, err := b.state.Member(guildID, u.ID)
		if err != nil {
			return nil, false, fmt.Errorf("error getting member: %w", err)
		}

		r, ok := sliceutils.ContainsOne(member.RoleIDs, teamRoles...)
		if !ok {
			// not in any of the team roles
			continue
		}

		if len(buckets[r]) == int(participantsPerTeam) {
			// bucket for that team is full, skip this user
			continue
		}
		buckets[r] = append(buckets[r], member.User.ID)
	}

	full = true
	for _, b := range buckets {
		if len(b) < int(participantsPerTeam) {
			// not enough participants in this team
			full = false
			break
		}
	}

	return buckets, full, nil
}

func (b *Bot) asyncCheckParticipationDeadline() (d time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in check participation deadline routine: %v", err)
		}
	}()
	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		deadline, err := q.NextParticipationConfirmationDeadline(b.ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no matches scheduled, nothing to send
				return nil
			}
			return fmt.Errorf("error getting next participation deadline: %w", err)
		}

		err = q.CloseParticipationEntry(b.ctx, deadline.ChannelID)
		if err != nil {
			return fmt.Errorf("error closing participation entry: %w", err)
		}

		guildID, err := discordutils.ParseGuildID(deadline.GuildID)
		if err != nil {
			return fmt.Errorf("error parsing guild ID: %w", err)
		}

		channelID, err := discordutils.ParseChannelID(deadline.ChannelID)
		if err != nil {
			return fmt.Errorf("error parsing channel ID: %w", err)
		}

		msgID, err := discordutils.ParseMessageID(deadline.MessageID)
		if err != nil {
			return fmt.Errorf("error parsing message ID: %w", err)
		}

		teamRoleIDs, err := b.listMatchTeamRoleIDs(b.ctx, q, channelID)
		if err != nil {
			return err
		}

		modUserIds, err := b.listMatchModeratorUserIDs(b.ctx, q, channelID)
		if err != nil {
			return err
		}

		streamers, err := b.listMatchStreamerUserIDs(b.ctx, q, channelID)
		if err != nil {
			return err
		}

		participants, full, err := b.getConfirmedParticipants(
			guildID,
			channelID,
			msgID,
			deadline.RequiredParticipantsPerTeam,
			teamRoleIDs...,
		)
		if err != nil {
			return fmt.Errorf("error getting confirmed participants: %w", err)
		}
		if !full {
			// disable match reminders, because we do not have enough participants
			err = q.UpdateMatchReminderCount(b.ctx, sqlc.UpdateMatchReminderCountParams{
				ChannelID:     deadline.ChannelID,
				ReminderCount: b.reminder.DisabledIndex(),
			})
			if err != nil {
				return fmt.Errorf("error disabling match reminder: %w", err)
			}

			content, am := FormatNotification(
				fmt.Sprintf("Not enough participants for match %s, closing participation entry", channelID.Mention()),
				"",
				teamRoleIDs,
				modUserIds,
				streamers,
				nil,
			)

			_, err := b.state.SendMessageComplex(
				channelID,
				api.SendMessageData{
					Content:         content,
					AllowedMentions: am,
				},
			)
			if err != nil {
				return fmt.Errorf("error sending message: %w", err)
			}
			return nil
		}

		content, am := FormatNotification(
			"Closing participation entry, we have reached enough players play the match!",
			"",
			teamRoleIDs,
			modUserIds,
			streamers,
			participants,
		)

		_, err = b.state.SendMessageComplex(
			channelID,
			api.SendMessageData{
				Content:         content,
				AllowedMentions: am,
			},
		)
		if err != nil {
			return fmt.Errorf("error sending message: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return d, nil
}
