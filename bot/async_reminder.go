package bot

import (
	"cmp"
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

var (
	ReminderIntervals = []time.Duration{
		24 * time.Hour,   // one day before
		1 * time.Hour,    // one hour before
		15 * time.Minute, // 15 minutes before
		5 * time.Minute,  // 5 minutes before
		30 * time.Second, // now
	}
	MaxReminderIndex = int64(len(ReminderIntervals) - 1)
)

func until(now time.Time, scheduledAt time.Time) time.Duration {
	until := scheduledAt.Sub(now)
	if until < 0 {
		until = -1 * until
	}
	return until
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -1 * d
	}
	return d
}

func nextReminder(reminderCnt int64, scheduledAt time.Time) (int64, time.Duration, bool) {
	if reminderCnt > MaxReminderIndex {
		return reminderCnt, 0, false
	}

	var (
		now = time.Now()
	)

	untilNextReminder := make([]time.Duration, len(ReminderIntervals))
	remindersAt := make([]time.Time, len(ReminderIntervals))
	for i, offset := range ReminderIntervals {
		remindAt := scheduledAt.Add(-abs(offset))

		remindersAt[i] = remindAt
		untilNextReminder[i] = until(now, remindAt)
	}

	sortedIndexList := make([]int, len(untilNextReminder))
	for i := range len(untilNextReminder) {
		sortedIndexList[i] = i
	}
	slices.SortFunc(sortedIndexList, func(a, b int) int {
		ad := int64(untilNextReminder[a])
		bd := int64(untilNextReminder[b])
		return cmp.Compare(ad, bd)
	})

	// first element in that list is the closest reminder
	// let's see if it is in the past or in the future
	for _, i := range sortedIndexList {
		i := int64(i)
		if reminderCnt > i {
			// this reminder is already sent
			continue
		}

		offset := abs(ReminderIntervals[i])
		remindAt := scheduledAt.Add(-offset)
		if now.After(remindAt) {
			// this reminder is in the past, we can skip it
			continue
		}

		nextReminderIn := untilNextReminder[i]
		nextIntervalIn := ReminderIntervals[i]
		triggerReminder := nextReminderIn < nextIntervalIn

		// this reminder is in the future, we can use it
		// but only if we are inside the reminder period
		return i, nextReminderIn, triggerReminder
	}

	return reminderCnt, 0, false
}

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

	r, err := q.NextMatchReminder(b.ctx, MaxReminderIndex)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no matches scheduled, nothing to send
			return 0, nil
		}
		return 0, fmt.Errorf("error getting next match reminder: %w", err)
	}

	scheduledAt := time.Unix(r.ScheduledAt, 0)
	ridx, untilNextReminder, ok := nextReminder(r.ReminderCount, scheduledAt)
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
