package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/internal/sliceutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

func (b *Bot) asyncCheckParticipationDeadline(ctx context.Context) (d time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in check participation requirements routine: %v", err)
		}
	}()
	err = b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		requirements, err := q.ListNowDueParticipationRequirements(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no matches scheduled, nothing to send
				return nil
			}
			return fmt.Errorf("error getting due participation requirements: %w", err)
		}

		orphanedMatches := make([]string, 0)
		for _, req := range requirements {
			err = q.CloseParticipationEntry(ctx, req.ChannelID)
			if err != nil {
				return fmt.Errorf("error closing participation entry: %w", err)
			}

			match, err := q.GetMatch(ctx, req.ChannelID)
			if err != nil {
				return fmt.Errorf("error getting match: %w", err)
			}

			guildID, err := parse.GuildID(match.GuildID)
			if err != nil {
				return err
			}

			channelID, err := parse.ChannelID(match.ChannelID)
			if err != nil {
				return err
			}

			msgID, err := parse.MessageID(match.MessageID)
			if err != nil {
				return err
			}

			teamRoleIDs, err := b.listMatchTeamRoleIDs(ctx, q, channelID)
			if err != nil {
				return err
			}

			modUserIds, err := b.listMatchModeratorUserIDs(ctx, q, channelID)
			if err != nil {
				return err
			}

			streamers, err := b.listMatchStreamerUserIDs(ctx, q, channelID)
			if err != nil {
				return err
			}

			participants, full, err := b.getConfirmedParticipants(
				guildID,
				channelID,
				msgID,
				req.ParticipantsPerTeam,
				teamRoleIDs...,
			)
			if err != nil {
				if discordutils.IsStatus4XX(err) {
					// channel not found -> delete match manually
					log.Printf("channel %s or message %s not found, adding to orphaned list for deletion", channelID, msgID)
					orphanedMatches = append(orphanedMatches, match.ChannelID)
					continue
				}
				return fmt.Errorf("error getting confirmed participants: %w", err)
			}
			if !full {
				// delete future all match notifiactions, because the requirements were not met
				// so the match will not take place
				err = q.DeleteMatchNotifications(ctx, match.ChannelID)
				if err != nil {
					return fmt.Errorf("error deleting match notifications: %w", err)
				}

				msg := FormatNotification(
					fmt.Sprintf("Not enough participants for match %s, closing participation entry", channelID.Mention()),
					"",
					teamRoleIDs,
					modUserIds,
					streamers,
					nil,
				)

				if match.EventID != "" {
					// delete scheduled event
					eventID, err := parse.EventID(match.EventID)
					if err != nil {
						return fmt.Errorf("failed to parse event id for channel %s: %w", channelID, err)
					}

					const reason = "requirements for match not met"
					event, err := b.state.EditScheduledEvent(guildID, eventID, reason, api.EditScheduledEventData{
						Status: discord.CancelledEvent,
					})
					if err != nil && !discordutils.IsStatus4XX(err) {
						return fmt.Errorf("error deleting scheduled event %s in guild %s: %w", eventID, guildID, err)
					}
					log.Printf("cancelled scheduled event %s in guild %s, reason: %s", event.ID, guildID, reason)
				}

				_, err := b.state.SendMessageComplex(channelID, msg)
				if err != nil {
					if discordutils.IsStatus4XX(err) {
						// channel not found -> delete match manually
						log.Printf("channel %s not found, adding to orphaned list for deletion", channelID)
						orphanedMatches = append(orphanedMatches, match.ChannelID)
						continue
					}
					return fmt.Errorf("error sending message: %w", err)
				}
				return nil
			}

			msg := FormatNotification(
				"Closing participation entry, we have reached enough players play the match!",
				"",
				teamRoleIDs,
				modUserIds,
				streamers,
				participants,
			)

			_, err = b.state.SendMessageComplex(channelID, msg)
			if err != nil {
				if discordutils.IsStatus4XX(err) {
					// channel not found -> delete match manually
					log.Printf("channel %s not found, adding to orphaned list for deletion", channelID)
					orphanedMatches = append(orphanedMatches, match.ChannelID)
					continue
				}
				return fmt.Errorf("error sending message: %w", err)
			}

			log.Printf("closed participation entry for match %s, deadline at: %s", channelID, time.Unix(req.DeadlineAt, 0))
		}

		if len(orphanedMatches) > 0 {
			err = b.deleteOphanedMatches(ctx, q, orphanedMatches...)
			if err != nil {
				return err
			}
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

	// initialize buckets for each team role
	buckets := make(map[discord.RoleID][]discord.UserID, len(teamRoles))
	for _, role := range teamRoles {
		buckets[role] = make([]discord.UserID, 0, expectedParticipants)
	}

	if expectedParticipants == 0 {
		// no participants required, return empty buckets
		return buckets, true, nil
	}

	users, err := b.state.Reactions(channelID, messageID, ReactionEmoji, 2*expectedParticipants+1) // +1 for the bot
	if err != nil {
		return nil, false, fmt.Errorf("error getting reactions: %w", err)
	}

	for _, u := range users {
		if b.isMe(u.ID) {
			continue
		}

		member, err := b.state.Member(guildID, u.ID)
		if err != nil {
			if discordutils.IsStatus(err, http.StatusNotFound) {
				continue
			}
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
