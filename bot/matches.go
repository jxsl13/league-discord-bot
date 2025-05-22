package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/format"
	"github.com/jxs13/league-discord-bot/internal/options"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

const (
	ReactionEmoji = "🎮"
	// ReactionEmoji = "📆"

	MaxConcurrentMatches = 50 // Category limitation which only allows for up to 50 channels
)

func (b *Bot) commandScheduleMatch(ctx context.Context, data cmdroute.CommandData) (resp *api.InteractionResponseData) {

	var (
		guildID    = data.Event.GuildID
		guildIDStr = guildID.String()
		now        = time.Now()
		nowUnix    = now.Unix()
		userID     = data.Event.SenderID()
		userIDStr  = userID.String()
	)

	// validation still happends synchronously
	err := b.TxQueries(ctx, func(ctx context.Context, q *sqlc.Queries) error {
		err := b.checkAccess(ctx, q, data.Event, WRITE)
		if err != nil {
			return err
		}

		scheduledAt, err := options.FutureTimeInLocation(
			"scheduled_at",
			"location",
			time.Minute,
			data.Options,
		)
		if err != nil {
			return err
		}

		participantsPerTeam, err := options.MinInteger("participants_per_team", data.Options, 0)
		if err != nil {
			return err
		}

		team1, err := options.RoleID("team_1_role", data.Options)
		if err != nil {
			return err
		}

		team2, err := options.RoleID("team_2_role", data.Options)
		if err != nil {
			return err
		}

		moderatorID, err := options.UserID("moderator", data.Options)
		if err != nil {
			return err
		}

		if team1 == team2 {
			err = fmt.Errorf("invalid parameter 'team_1_role' and 'team_2_role': must be different")
			return err
		}

		streamUrl, _, err := options.OptionalUrl("stream_url", data.Options)
		if err != nil {
			return err
		}

		streamerID, okStreamer, err := options.OptionalUserID("streamer", data.Options)
		if err != nil {
			return err
		}

		err = b.checkRoleIDs(guildID, team1, team2)
		if err != nil {
			return err
		}

		err = b.checkUserIDs(guildID, moderatorID)
		if err != nil {
			return err
		}

		if okStreamer {
			err = b.checkUserIDs(guildID, streamerID)
			if err != nil {
				return err
			}
		}

		n, err := q.CountMatches(ctx, guildIDStr)
		if err != nil {
			return fmt.Errorf("error counting matches: %w", err)
		}

		if n >= MaxConcurrentMatches {
			return fmt.Errorf("error: maximum number of concurrent matches reached: %d", MaxConcurrentMatches)
		}

		// validation is finished at this point and the actual creation of the channel begins

		cfg, err := q.GetGuildConfig(ctx, guildIDStr)
		if err != nil {
			return fmt.Errorf("error getting guild config: %w", err)
		}

		intervals, err := parse.ReminderIntervals(cfg.NotificationOffsets)
		if err != nil {
			return err
		}

		categoryID, err := parse.ChannelID(cfg.CategoryID)
		if err != nil {
			return err
		}

		cnt, err := q.NextMatchCounter(ctx, guildID.String())
		if err != nil {
			return fmt.Errorf("error getting next match counter: %w", err)
		}

		everyone, err := b.everyone(guildID)
		if err != nil {
			return err
		}

		createData := api.CreateChannelData{
			Name:       fmt.Sprintf("match-%d", cnt),
			Type:       discord.GuildText,
			CategoryID: categoryID,
			Overwrites: []discord.Overwrite{
				{
					ID:   discord.Snowflake(everyone.ID), // everyone can't access channel
					Type: discord.OverwriteRole,
					Deny: discord.PermissionAllText,
				},
				{
					ID:    discord.Snowflake(b.userID), // bot can access channel
					Type:  discord.OverwriteMember,
					Allow: discord.PermissionAllText,
				},
			},
		}

		c, err := b.state.CreateChannel(guildID, createData)
		if err != nil {
			// category was deleted while hte bot was turned off
			if discordutils.IsStatus(err, http.StatusBadRequest) {
				channels, err := b.state.Channels(guildID)
				if err != nil {
					return fmt.Errorf("failed to list channels: %w", err)
				}
				category, err := b.createMatchCategory(
					guildID,
					discordutils.LastChannelPosition(channels),
				)
				if err != nil {
					return fmt.Errorf("error creating match category: %w", err)
				}
				categoryID = category.ID

				err = q.UpdateCategoryId(ctx, sqlc.UpdateCategoryIdParams{
					CategoryID: categoryID.String(),
					GuildID:    guildIDStr,
				})
				if err != nil {
					return fmt.Errorf("error updating category id: %w", err)
				}

				createData.CategoryID = categoryID
				// category is recreated, now try to create the channel again
				c, err = b.state.CreateChannel(guildID, createData)
				if err != nil {
					return fmt.Errorf("error creating channel: %w", err)
				}
			} else {
				return fmt.Errorf("error creating channel: %w", err)
			}
		}
		defer func() {
			if err != nil {
				// delete the channel if there was an error
				if err := b.state.DeleteChannel(c.ID, api.AuditLogReason(err.Error())); err != nil {
					log.Printf("error deleting channel %s: %v", c.ID, err)
				}
			}
		}()

		var (
			vs                  = ""
			confirmation        = ""
			channelAccessibleAt = scheduledAt.Add(-1 * time.Second * time.Duration(cfg.ChannelAccessOffset))
			channelDeleteAt     = scheduledAt.Add(time.Second * time.Duration(cfg.ChannelDeleteOffset))
		)
		if channelAccessibleAt.Before(now) {
			// if the channel accessible time is in the past, set it to now
			channelAccessibleAt = now
		}

		if participantsPerTeam > 0 {
			vs = fmt.Sprintf("(%don%d)", participantsPerTeam, participantsPerTeam)
			confirmation = fmt.Sprintf("\n\nPlease react with %s to confirm your participation.", ReactionEmoji)
		}

		msg, err := b.state.SendMessage(
			c.ID,
			fmt.Sprintf(
				"Match between %s and %s %s scheduled at %s\n\nThis channel is accessible from %s until %s%s",
				team1.Mention(),
				team2.Mention(),
				vs,
				format.DiscordLongDateTime(scheduledAt),
				format.DiscordLongDateTime(channelAccessibleAt),
				format.DiscordLongDateTime(channelDeleteAt),
				confirmation,
			),
		)
		if err != nil {
			return fmt.Errorf("error sending message: %w", err)
		}
		defer func() {
			if err != nil {
				// delete the message if there was an error
				if err := b.state.DeleteMessage(c.ID, msg.ID, api.AuditLogReason(err.Error())); err != nil {
					log.Printf("error deleting message %s: %v", msg.ID, err)
				}
			}
		}()

		if participantsPerTeam > 0 {
			// only react when there are required participants for the teams
			err = b.state.React(c.ID, msg.ID, ReactionEmoji)
			if err != nil {
				return fmt.Errorf("error reacting to message: %w", err)
			}
		}

		var (
			channelID    = c.ID
			channelIDStr = channelID.String()
			// epoch seconds
			channelAccessibleAtUnix = channelAccessibleAt.Unix()
			channelDeleteAtUnix     = channelDeleteAt.Unix()
			participatonReqDeadline = scheduledAt.Add(-1 * time.Second * time.Duration(cfg.RequirementsOffset)).Unix()
		)

		err = q.AddMatch(ctx, sqlc.AddMatchParams{
			GuildID:             guildID.String(),
			ChannelID:           channelIDStr,
			ChannelAccessibleAt: channelAccessibleAtUnix,
			ChannelDeleteAt:     max(nowUnix, channelDeleteAtUnix),
			MessageID:           msg.ID.String(),
			ScheduledAt:         scheduledAt.Unix(),
			CreatedAt:           nowUnix,
			CreatedBy:           userIDStr,
			UpdatedAt:           nowUnix,
			UpdatedBy:           userIDStr,
		})
		if err != nil {
			return fmt.Errorf("error adding match: %w", err)
		}

		if participantsPerTeam > 0 {
			err = q.AddParticipationRequirements(ctx, sqlc.AddParticipationRequirementsParams{
				ChannelID:           channelIDStr,
				ParticipantsPerTeam: participantsPerTeam,
				DeadlineAt:          max(nowUnix, participatonReqDeadline),
				EntryClosed:         0,
			})
			if err != nil {
				return fmt.Errorf("error adding participation requirements: %w", err)
			}
		}

		// team1
		err = q.AddMatchTeam(ctx, sqlc.AddMatchTeamParams{
			ChannelID: channelIDStr,
			RoleID:    team1.String(),
		})
		if err != nil {
			return fmt.Errorf("error adding match team 1: %w", err)
		}
		// team 2
		err = q.AddMatchTeam(ctx, sqlc.AddMatchTeamParams{
			ChannelID: channelIDStr,
			RoleID:    team2.String(),
		})
		if err != nil {
			return fmt.Errorf("error adding match team 2: %w", err)
		}

		err = q.AddMatchModerator(ctx, sqlc.AddMatchModeratorParams{
			ChannelID: channelIDStr,
			UserID:    moderatorID.String(),
		})
		if err != nil {
			return fmt.Errorf("error adding match moderator: %w", err)
		}

		if okStreamer {
			err = q.AddMatchStreamer(ctx, sqlc.AddMatchStreamerParams{
				ChannelID: channelIDStr,
				UserID:    streamerID.String(),
				Url:       streamUrl,
			})
			if err != nil {
				return fmt.Errorf("error adding match streamer: %w", err)
			}
		}

		// create notifications, can be disabled, in case there are not intervals defined in the guild config
		for _, d := range intervals {
			notifyAt := scheduledAt.Add(-1 * d)
			if now.Sub(notifyAt) >= 0 {
				// if the notification time is in the past, skip it
				continue
			}

			err = q.AddNotification(ctx, sqlc.AddNotificationParams{
				ChannelID:  channelIDStr,
				NotifyAt:   notifyAt.Unix(),
				CustomText: "", // will be automatically generate in case that it is not provided, which is not the case for default notifications
				CreatedBy:  userIDStr,
				CreatedAt:  nowUnix,
				UpdatedBy:  userIDStr,
				UpdatedAt:  nowUnix,
			})
			if err != nil {
				return fmt.Errorf("error adding notification: %w", err)
			}
		}

		err = b.refreshJobSchedules(ctx, q)
		if err != nil {
			return err
		}

		resp = &api.InteractionResponseData{
			Content: option.NewNullableString(
				fmt.Sprintf(
					"Created a new match channel: %s",
					c.ID.Mention(),
				),
			),
			Flags: discord.EphemeralMessage,
		}

		return nil
	})
	if err != nil {
		return errorResponse(err)
	}

	// do not overwrite this response
	// because it is set in the transaction
	return resp

}
