package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncAnnouncements() (err error) {
	defer func() {
		if err != nil {
			log.Printf("failed to announce matches: %v", err)
		}
	}()

	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) (err error) {
		announcements, err := q.ListNowDueAnnouncements(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		if len(announcements) == 0 {
			return nil
		}

		// we got announcements to handle

		for _, a := range announcements {
			err = b.sendGuildAnnouncement(ctx, q, a)
			if err != nil {
				return err
			}
		}

		return b.refreshAnnouncementJob(ctx, q)
	})
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) sendGuildAnnouncement(ctx context.Context, q *sqlc.Queries, announcement sqlc.Announcement) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error sending match pre announcement for guild %s and channel %s: %w", announcement.GuildID, announcement.ChannelID, err)
		}
	}()

	targetChannelID, err := parse.ChannelID(announcement.ChannelID)
	if err != nil {
		return fmt.Errorf("error parsing channel ID: %w", err)
	}

	msgs, ok, err := b.generateGuildAnnouncement(ctx, q, announcement)
	if err != nil {
		return err
	}
	if !ok {
		// no matches to announce
		// maybe there are matches next time
		// for now continue to the next announcement interval.
		return q.ContinueAnnouncement(ctx, announcement.GuildID)
	}

	for _, text := range msgs {
		_, err = b.state.SendMessageComplex(targetChannelID, api.SendMessageData{
			Content: text,
			Flags:   discord.SuppressEmbeds,
			AllowedMentions: &api.AllowedMentions{
				Parse: []api.AllowedMentionType{
					api.AllowUserMention,
					api.AllowRoleMention,
					api.AllowEveryoneMention,
				},
			},
		})
		if err != nil {
			if discordutils.IsStatus4XX(err) {
				// channel not found or bot not in channel, disable preannouncements
				log.Printf("sending announcement message failed: channel not found or bot not in channel %s: %v", targetChannelID, err)
				err = q.DeleteAnnouncement(ctx, announcement.GuildID)
				if err != nil {
					return fmt.Errorf("error deleting pre announcement: %w", err)
				}

				// we cannot send -> disable pre announcements
				// do not return an error, because we have more messages to send
				return nil
			}
			return fmt.Errorf("error sending announcement message: %w", err)
		}
	}

	// move last_announcement to the next point in time which is now but more exact w/o time drift(last_annoncement + interval)
	return q.ContinueAnnouncement(ctx, announcement.GuildID)
}

func (b *Bot) generateGuildAnnouncement(ctx context.Context, q *sqlc.Queries, announcement sqlc.Announcement) (_ []string, ok bool, err error) {
	matches, err := q.ListGuildMatchesScheduledBetween(ctx, sqlc.ListGuildMatchesScheduledBetweenParams{
		GuildID: announcement.GuildID,
		MinAt:   announcement.LastAnnouncedAt + announcement.Interval,
		MaxAt:   announcement.LastAnnouncedAt + 2*announcement.Interval, // 1st for current time and 2nd for next time
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if len(matches) == 0 {
		return nil, false, nil
	}

	result := make([]string, 0, 1)
	// we got matches to announce
	var sb strings.Builder
	sb.Grow(min(2000, len(announcement.CustomTextAfter)+len(announcement.CustomTextBefore)+len(matches)*256))
	sb.WriteString(announcement.CustomTextBefore)

	intervalStart := time.Unix(announcement.LastAnnouncedAt+announcement.Interval, 0)
	intervalEnd := time.Unix(announcement.LastAnnouncedAt+2*announcement.Interval, 0)

	startYear, startMonth, startDay := intervalStart.Date()
	endYear, endMonth, endDay := intervalEnd.Date()

	// same day -> use long datetime
	// different day -> use long date
	formatFunc := format.DiscordLongDateTime
	if startYear != endYear || startMonth != endMonth || startDay != endDay {
		formatFunc = format.DiscordLongDate
	}

	sb.WriteString("\n")
	sb.WriteString(
		fmt.Sprintf("**Upcoming matches:**\n\n%s - %s\n",
			formatFunc(intervalStart),
			formatFunc(intervalEnd),
		),
	)

	for _, m := range matches {
		guildID, err := parse.GuildID(m.GuildID)
		if err != nil {
			return nil, false, err
		}

		channelID, err := parse.ChannelID(m.ChannelID)
		if err != nil {
			return nil, false, err
		}

		teams, err := b.listMatchTeamRoleIDs(ctx, q, channelID)
		if err != nil {
			return nil, false, err
		}

		moderators, err := b.listMatchModeratorUserIDs(ctx, q, channelID)
		if err != nil {
			return nil, false, err
		}

		streamers, err := b.listMatchStreamerUserIDs(ctx, q, channelID)
		if err != nil {
			return nil, false, err
		}

		roleMap, err := b.resolveRoleIDs(guildID, teams)
		if err != nil {
			return nil, false, err
		}

		scheduledAt := time.Unix(m.ScheduledAt, 0)
		teamNames := make([]string, 0, len(teams))

		for _, id := range teams {
			team, ok := roleMap[id]
			if !ok {
				return nil, false, fmt.Errorf("team %s not found in role id map", id)
			}
			teamNames = append(teamNames, format.MarkdownFat(team.Name))
		}

		moderatorNames := make([]string, 0, len(moderators))
		for _, id := range moderators {
			moderator, err := b.state.Member(guildID, id)
			if err != nil {
				if discordutils.IsStatus4XX(err) {
					// user not found, ignore
					continue
				}
				return nil, false, fmt.Errorf("error getting moderator %s: %w", id, err)

			}
			moderatorNames = append(moderatorNames, moderator.User.Username)
		}

		streamerLines := make([]string, 0, len(streamers))
		for _, s := range streamers {
			if s.Info.Url == "" {
				continue
			}

			streamer, err := b.state.Member(guildID, s.UserID)
			if err != nil {
				if discordutils.IsStatus4XX(err) {
					// user not found, ignore
					continue
				}
				return nil, false, fmt.Errorf("error getting streamer %s: %w", s.UserID, err)
			}

			streamerLines = append(streamerLines, fmt.Sprintf("%s at %s", streamer.User.DisplayName, s.Info.Url))
		}

		var mb strings.Builder
		mb.Grow(256)

		mb.WriteString("\n")
		mb.WriteString("* ")
		mb.WriteString(format.DiscordLongDateTime(scheduledAt))
		mb.WriteString("\n")
		if len(teams) > 0 {
			if len(teams) == 1 {
				mb.WriteString("Team: ")
			} else {
				mb.WriteString("Teams: ")
			}
			mb.WriteString(strings.Join(teamNames, " vs "))
			mb.WriteString("\n")
		}

		if len(moderatorNames) > 0 {
			if len(moderatorNames) == 1 {
				mb.WriteString("Moderator: ")
			} else {
				mb.WriteString("Moderators: ")
			}
			mb.WriteString(strings.Join(moderatorNames, ", "))
			mb.WriteString("\n")
		}
		if len(streamerLines) > 0 {
			if len(streamerLines) == 1 {
				mb.WriteString("Streamer: ")
			} else {
				mb.WriteString("Streamers:\n")
			}
			mb.WriteString(strings.Join(streamerLines, "\n"))
			mb.WriteString("\n\n")
		}

		if sb.Len()+len(announcement.CustomTextAfter)+mb.Len() > 2000 {
			result = append(result, sb.String())
			sb.Reset()
		}
		sb.WriteString(mb.String())
	}

	if sb.Len()+len(announcement.CustomTextAfter) > 2000 {
		result = append(result, sb.String())
		sb.Reset()
	}
	sb.WriteString(announcement.CustomTextAfter)

	if sb.Len() > 2000 {
		result = append(result, sb.String()[:2000-3]+"...")
	} else {
		result = append(result, sb.String())
	}

	return result, true, nil
}
