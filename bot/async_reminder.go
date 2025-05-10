package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/format"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncReminder() (_ time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in reminder routine: %v", err)
		}
	}()
	q, err := b.Queries(b.ctx)
	if err != nil {
		return 0, err
	}
	defer q.Close()

	r, err := q.NextMatchReminder(b.ctx, b.reminder.MaxIndex())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no matches scheduled, nothing to send
			return 0, nil
		}
		return 0, fmt.Errorf("error getting next match reminder: %w", err)
	}

	scheduledAt := time.Unix(r.ScheduledAt, 0)
	ridx, untilNextReminder, ok := b.reminder.Next(r.ReminderCount, scheduledAt)
	if !ok {
		return 0, nil
	}

	channelID, err := discordutils.ParseChannelID(r.ChannelID)
	if err != nil {
		return 0, err
	}

	// we need to remind the teams, moderators and streamers
	teamRoleIDs, err := b.listMatchTeamRoleIDs(b.ctx, q, channelID)
	if err != nil {
		return 0, err
	}

	modUserIds, err := b.listMatchModeratorUserIDs(b.ctx, q, channelID)
	if err != nil {
		return 0, err
	}

	streamers, err := b.listMatchStreamerUserIDs(b.ctx, q, channelID)
	if err != nil {
		return 0, err
	}

	text := ""
	untilMatch := time.Until(scheduledAt)
	if untilMatch >= time.Minute {
		text = fmt.Sprintf("The match is starting in about %s. ", format.Duration(untilMatch))
	} else {
		text = "The match is starting now!"
	}

	var sb strings.Builder
	sb.WriteString(text)
	sb.WriteString("\n")

	roleIDs := make([]discord.RoleID, 0, len(teamRoleIDs))
	userIDs := make([]discord.UserID, 0, len(modUserIds)+len(streamers))

	if len(teamRoleIDs) > 0 {
		sb.WriteString("Teams: ")
		for idx, rid := range teamRoleIDs {
			roleIDs = append(roleIDs, rid)

			sb.WriteString(rid.Mention())
			if idx < len(teamRoleIDs)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("\n")
	}

	if len(modUserIds) > 0 {
		sb.WriteString("Moderators: ")
		for idx, uid := range modUserIds {
			userIDs = append(userIDs, uid)

			sb.WriteString(uid.Mention())
			if idx < len(teamRoleIDs)-1 {
				sb.WriteString(", ")
			}
		}
		sb.WriteString("\n")
	}

	if len(streamers) > 0 {
		if len(streamers) > 1 {
			sb.WriteString("Streamers: \n")
		} else {
			sb.WriteString("Streamer: ")
		}
		for idx, s := range streamers {
			userIDs = append(userIDs, s.UserID)

			sb.WriteString("\t")
			sb.WriteString(s.Mention())
			if idx < len(streamers)-1 {
				sb.WriteString("\n")
			}
		}
	}

	slices.Sort(userIDs)
	slices.Sort(roleIDs)

	msg := api.SendMessageData{
		Content: sb.String(),
		AllowedMentions: &api.AllowedMentions{

			/*
				Parse: []api.AllowedMentionType{
					api.AllowRoleMention,
					api.AllowUserMention,
				},
			*/
			Roles: slices.Compact(roleIDs),
			Users: slices.Compact(userIDs),
		},
	}

	_, err = b.state.SendMessageComplex(channelID, msg)
	if err != nil {
		return 0, fmt.Errorf("error sending reminder message: %w", err)
	}

	// update reminder count
	newReminderCount := max(r.ReminderCount+1, ridx+1)
	err = q.UpdateMatchReminderCount(b.ctx, sqlc.UpdateMatchReminderCountParams{
		ChannelID:     r.ChannelID,
		ReminderCount: newReminderCount,
	})
	if err != nil {
		return 0, fmt.Errorf("error updating reminder count: %w", err)
	}

	return untilNextReminder, nil
}
