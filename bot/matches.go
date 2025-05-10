package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/api/cmdroute"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/jxs13/league-discord-bot/discordutils"
	"github.com/jxs13/league-discord-bot/options"
	"github.com/jxs13/league-discord-bot/sqlc"
)

const (
	ReactionEmoji = "🎮"
	// ReactionEmoji = "📆"
)

func (b *Bot) commandScheduleMatch(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {

	var (
		guildID   = data.Event.GuildID
		now       = time.Now()
		nowUnix   = now.Unix()
		userID    = data.Event.SenderID()
		userIDStr = userID.String()
	)

	q, err := b.Queries(b.ctx)
	if err != nil {
		err = fmt.Errorf("error getting queries: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}
	defer q.Close()

	err = b.checkAccess(ctx, q, data.Event, WRITE)
	if err != nil {
		return errorResponse(err)
	}

	scheduledAt, err := options.TimeInLocation("scheduled_at", "location", data.Options) // TODO: change check to options.MinTime
	if err != nil {
		return errorResponse(err)
	}

	participantsPerTeam, err := options.MinInteger("participants_per_team", data.Options, 1)
	if err != nil {
		return errorResponse(err)
	}

	team1, err := options.RoleID("team_1_role", data.Options)
	if err != nil {
		return errorResponse(err)
	}

	team2, err := options.RoleID("team_2_role", data.Options)
	if err != nil {
		return errorResponse(err)
	}

	moderatorID, err := options.UserID("moderator", data.Options)
	if err != nil {
		return errorResponse(err)
	}

	if team1 == team2 {
		err = fmt.Errorf("invalid parameter 'team_1_role' and 'team_2_role': must be different")
		return errorResponse(err)
	}

	err = b.checkRoleIDs(guildID, team1, team2)
	if err != nil {
		return errorResponse(err)
	}

	err = b.checkUserIDs(guildID, moderatorID)
	if err != nil {
		return errorResponse(err)
	}

	streamUrl, _, err := options.OptionalUrl("stream_url", data.Options)
	if err != nil {
		return errorResponse(err)
	}

	streamerID, okStreamer, err := options.OptionalUserID("streamer", data.Options)
	if err != nil {
		return errorResponse(err)
	}

	if okStreamer {
		err = b.checkUserIDs(guildID, streamerID)
		if err != nil {
			return errorResponse(err)
		}
	}

	cfg, err := q.GetGuildConfig(ctx, guildID.String())
	if err != nil {
		err = fmt.Errorf("error getting guild config: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	cid, err := discord.ParseSnowflake(cfg.CategoryID)
	if err != nil {
		err = fmt.Errorf("error parsing category ID: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}
	categoryID := discord.ChannelID(cid)

	cnt, err := q.NextMatchCounter(ctx, guildID.String())
	if err != nil {
		err = fmt.Errorf("error getting next match counter: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	everyone, err := b.everyone(guildID)
	if err != nil {
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	c, err := b.state.CreateChannel(guildID, api.CreateChannelData{
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
	})
	if err != nil {
		err = fmt.Errorf("error creating channel: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}
	defer func() {
		if err != nil {
			// delete the channel if there was an error
			if err := b.state.DeleteChannel(c.ID, api.AuditLogReason(err.Error())); err != nil {
				log.Printf("error deleting channel %s: %v", c.ID, err)
			}
		}
	}()

	text := `Match between %[2]s and %[3]s (%[1]don%[1]d) scheduled at %[4]s

Please react with %[5]s to confirm your participation.
	`

	msg, err := b.state.SendMessage(
		c.ID,
		fmt.Sprintf(
			text,
			participantsPerTeam,
			team1.Mention(),
			team2.Mention(),
			discordutils.ToDiscordTimestamp(scheduledAt),
			ReactionEmoji,
		),
	)
	if err != nil {
		err = fmt.Errorf("error sending message: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}
	defer func() {
		if err != nil {
			// delete the message if there was an error
			if err := b.state.DeleteMessage(c.ID, msg.ID, api.AuditLogReason(err.Error())); err != nil {
				log.Printf("error deleting message %s: %v", msg.ID, err)
			}
		}
	}()

	err = b.state.React(c.ID, msg.ID, discord.APIEmoji(ReactionEmoji))
	if err != nil {
		err = fmt.Errorf("error reacting to message: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	var (
		channelID    = c.ID
		channelIDStr = channelID.String()
		// epoch seconds
		channelAccessibleAt = scheduledAt.Add(-1 * time.Second * time.Duration(cfg.ChannelAccessOffset)).Unix()
		channelDeleteAt     = scheduledAt.Add(time.Second * time.Duration(cfg.ChannelDeleteOffset)).Unix()
	)

	err = q.AddMatch(ctx, sqlc.AddMatchParams{
		GuildID:                        guildID.String(),
		ChannelID:                      channelIDStr,
		ChannelAccessibleAt:            max(nowUnix, channelAccessibleAt),
		ChannelDeleteAt:                max(nowUnix, channelDeleteAt),
		MessageID:                      msg.ID.String(),
		ScheduledAt:                    scheduledAt.Unix(),
		RequiredParticipantsPerTeam:    participantsPerTeam,
		ParticipationConfirmationUntil: max(nowUnix, scheduledAt.Add(-24*time.Hour).Unix()),
		CreatedAt:                      nowUnix,
		CreatedBy:                      userIDStr,
		UpdatedAt:                      nowUnix,
		UpdatedBy:                      userIDStr,
	})
	if err != nil {
		err = fmt.Errorf("error adding match: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	// team1
	err = q.AddMatchTeam(ctx, sqlc.AddMatchTeamParams{
		ChannelID: channelIDStr,
		RoleID:    team1.String(),
	})
	if err != nil {
		err = fmt.Errorf("error adding match team 1: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}
	// team 2
	err = q.AddMatchTeam(ctx, sqlc.AddMatchTeamParams{
		ChannelID: channelIDStr,
		RoleID:    team2.String(),
	})
	if err != nil {
		err = fmt.Errorf("error adding match team 2: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	err = q.AddMatchModerator(ctx, sqlc.AddMatchModeratorParams{
		ChannelID: channelIDStr,
		UserID:    moderatorID.String(),
	})
	if err != nil {
		err = fmt.Errorf("error adding match moderator: %w", err)
		return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
	}

	if okStreamer {
		err = q.AddMatchStreamer(ctx, sqlc.AddMatchStreamerParams{
			ChannelID: channelIDStr,
			UserID:    streamerID.String(),
			Url:       streamUrl,
		})
		if err != nil {
			err = fmt.Errorf("error adding match streamer: %w", err)
			return errorResponse(fmt.Errorf("%w, please contact the owner of the bot", err))
		}
	}

	return &api.InteractionResponseData{
		Content: option.NewNullableString(
			fmt.Sprintf(
				"Created a new match channel: https://discord.com/channels/%s/%s",
				guildID,
				c.ID,
			),
		),
		Flags:           discord.EphemeralMessage,
		AllowedMentions: &api.AllowedMentions{ /* none */ },
	}
}

func (b *Bot) commandRescheduleMatch(ctx context.Context, data cmdroute.CommandData) *api.InteractionResponseData {
	return nil
}
