package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/internal/maputils"
	"github.com/jxs13/league-discord-bot/internal/model"
	"github.com/jxs13/league-discord-bot/internal/parse"
	"github.com/jxs13/league-discord-bot/sqlc"
)

var (
	PermissionBasicAccess = discord.PermissionViewChannel |
		discord.PermissionSendMessages |
		discord.PermissionSendTTSMessages |
		discord.PermissionEmbedLinks |
		discord.PermissionAttachFiles |
		discord.PermissionReadMessageHistory |
		discord.PermissionUseExternalEmojis |
		discord.PermissionAddReactions

	PermissionModerators = PermissionBasicAccess |
		discord.PermissionManageMessages |
		discord.PermissionMentionEveryone |
		discord.PermissionUseSlashCommands
)

func (b *Bot) asyncGrantChannelAccess(ctx context.Context) (d time.Duration, err error) {
	log.Println("checking for channel access changes")
	defer func() {
		if err != nil {
			log.Printf("error in channel access routine: %v", err)
		}
	}()

	err = b.TxQueries(b.ctx, func(ctx context.Context, q *sqlc.Queries) error {
		accessible, err := q.ListNowAccessibleChannels(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no channels to give access to
				return nil
			}
			return err
		}
		if len(accessible) == 0 {
			// no channels to give access to
			return nil
		}

		guildAccessible := make(map[discord.GuildID][]*GuildEventParam, len(accessible))
		orphanedMatches := make([]string, 0)

		for _, ac := range accessible {
			eventParam, err := b.grantSingleChannelAccess(ctx, q, ac)
			if err != nil {
				if discordutils.IsStatus4XX(err) {
					orphanedMatches = append(orphanedMatches, ac.ChannelID)
					continue
				}

				return fmt.Errorf("error giving channel access: %w", err)
			}
			log.Printf("granted access to channel %s", ac.ChannelID)

			guildAccessible[eventParam.GuildID] = append(guildAccessible[eventParam.GuildID], eventParam)
		}

		if len(orphanedMatches) > 0 {
			err = b.deleteOphanedMatches(ctx, q, orphanedMatches...)
			if err != nil {
				return err
			}
		}

		// at this point all guild matches should be valid and accessible
		for _, guildID := range maputils.SortedKeys(guildAccessible) {
			accessibleChannels := guildAccessible[guildID]

			cfg, err := q.GetGuildConfig(ctx, guildID.String())
			if err != nil {
				return fmt.Errorf("failed to get guild config for %s: %w", guildID, err)
			}

			// skip guilds that have explicitly disabled creating scheduled events
			if cfg.EventCreationEnabled == 0 {
				continue
			}

			for _, ac := range accessibleChannels {
				err = b.createGuildEvent(ctx, q, ac)
				if err != nil {
					return fmt.Errorf("failed to create guild event for %s: %w", ac.GuildID, err)
				}
			}
		}

		// TODO: might need to check when the next channel is accessible
		// and return that time, in case it is shorter than the usual interval
		return nil
	})
	if err != nil {
		return 0, err
	}

	// important that we do not overwrite this with 0,
	// because it might have been set in the transaction closure
	return d, nil
}

func (b *Bot) grantSingleChannelAccess(ctx context.Context, q *sqlc.Queries, a sqlc.ListNowAccessibleChannelsRow) (param *GuildEventParam, err error) {
	var (
		channelID = a.ChannelID
	)

	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to give channel access for channel %s: %w", channelID, err)
		}
	}()

	cid, err := parse.ChannelID(channelID)
	if err != nil {
		return nil, err
	}

	c, err := b.state.Channel(cid)
	if err != nil {
		return nil, fmt.Errorf("error getting channel: %w", err)
	}

	teamRoleIDs, err := b.listMatchTeamRoleIDs(ctx, q, cid)
	if err != nil {
		return nil, fmt.Errorf("error getting match team role IDs: %w", err)
	}

	modUserIDs, err := b.listMatchModeratorUserIDs(ctx, q, cid)
	if err != nil {
		return nil, fmt.Errorf("error getting match moderator role IDs: %w", err)
	}

	streamers, err := b.listMatchStreamerUserIDs(ctx, q, cid)
	if err != nil {
		return nil, err
	}

	oldOverwrites := slices.Clone(c.Overwrites)
	overwrites := make([]discord.Overwrite, 0, len(oldOverwrites)+len(teamRoleIDs)+len(modUserIDs)+len(streamers))
	overwrites = append(overwrites, oldOverwrites...)

	for _, rid := range teamRoleIDs {
		overwrites = append(overwrites, discord.Overwrite{
			ID:    discord.Snowflake(rid),
			Type:  discord.OverwriteRole,
			Allow: PermissionBasicAccess,
		})
	}

	for _, uid := range modUserIDs {
		overwrites = append(overwrites, discord.Overwrite{
			ID:    discord.Snowflake(uid),
			Type:  discord.OverwriteMember,
			Allow: PermissionModerators,
		})
	}

	for _, s := range streamers {
		overwrites = append(overwrites, discord.Overwrite{
			ID:    discord.Snowflake(s.UserID),
			Type:  discord.OverwriteMember,
			Allow: PermissionBasicAccess,
		})
	}

	err = b.state.ModifyChannel(cid, api.ModifyChannelData{
		Overwrites: &overwrites,
	})
	if err != nil {
		return nil, fmt.Errorf("error modifying channel: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		rerr := b.state.ModifyChannel(
			cid,
			api.ModifyChannelData{
				Overwrites: &oldOverwrites,
			})
		if rerr != nil {
			err = errors.Join(err, fmt.Errorf("error reverting modifying channel: %w", rerr))
		}
	}()

	// set accessible flag in database in order to prevent the routing from picking up
	// the already accessible channel
	err = q.UpdateMatchChannelAccessibility(
		ctx,
		sqlc.UpdateMatchChannelAccessibilityParams{
			ChannelID:         channelID,
			ChannelAccessible: 1,
		})
	if err != nil {
		return nil, fmt.Errorf("error updating match channel accessibility: %w", err)
	}

	param = &GuildEventParam{
		GuildID:          c.GuildID,
		ChannelID:        c.ID,
		ScheduledAt:      a.ScheduledAt,
		DeleteAt:         a.ChannelDeleteAt,
		TeamRoleIDs:      teamRoleIDs,
		ModeratorUserIDs: modUserIDs,
		Streamers:        streamers,
	}
	return param, nil
}

type GuildEventParam struct {
	GuildID          discord.GuildID
	ChannelID        discord.ChannelID
	ScheduledAt      int64
	DeleteAt         int64
	TeamRoleIDs      []discord.RoleID
	ModeratorUserIDs []discord.UserID
	Streamers        []model.Streamer
}

func (b *Bot) createGuildEvent(ctx context.Context, q *sqlc.Queries, param *GuildEventParam) (err error) {

	if len(param.Streamers) == 0 {
		// no streamers, no public event
		return nil
	}

	// first streamer with url
	urlStreamers := make([]model.Streamer, 0, len(param.Streamers))
	for _, s := range param.Streamers {
		if s.Info.Url != "" {
			urlStreamers = append(urlStreamers, s)
		}
	}
	if len(urlStreamers) == 0 {
		// no streamers with url, no public event
		return nil
	}
	streamer := urlStreamers[0]

	if len(param.TeamRoleIDs) == 0 {
		// no teams, no public event
		return nil
	}

	teamRoles := make([]*discord.Role, 0, len(param.TeamRoleIDs))
	tids := make([]string, 0, len(param.TeamRoleIDs))
	for _, tid := range param.TeamRoleIDs {
		tids = append(tids, tid.Mention())

		role, err := b.state.Role(param.GuildID, tid)
		if err != nil {
			return fmt.Errorf("failed to get role of team %s: %w", tid, err)
		}
		teamRoles = append(teamRoles, role)
	}

	teamMention := strings.Join(tids, " vs ") + "\n\nStreamed by " + streamer.Mention()
	roleNames := make([]string, 0, len(teamRoles))
	for _, role := range teamRoles {
		roleNames = append(roleNames, role.Name)
	}
	teamNameMention := strings.Join(roleNames, " vs ")

	const reason = "automatically created event because the participating teams were granted access to the match channel"
	var (
		now1       = time.Now().Add(time.Minute)
		startsAt   = time.Unix(param.ScheduledAt, 0)
		endsAt     = time.Unix(param.DeleteAt, 0)
		startsAtTs = discord.NewTimestamp(startsAt)
		endsAtTs   = discord.NewTimestamp(endsAt)
	)

	if startsAt.Before(now1) {
		// event is in the past, set it to now
		startsAtTs = discord.NewTimestamp(now1)
	}

	if endsAt.Before(now1) {
		// event is in the past, set it to now
		endsAtTs = discord.NewTimestamp(now1)
	}

	if startsAt.After(endsAt) {
		// event is in the past, set it to now
		endsAtTs = discord.NewTimestamp(startsAt.Add(time.Minute))
	}

	event, err := b.state.CreateScheduledEvent(param.GuildID, reason, api.CreateScheduledEventData{
		Name:        fmt.Sprintf("Match: %s", teamNameMention),
		Description: teamMention,
		EntityType:  discord.ExternalEntity,
		EntityMetadata: &discord.EntityMetadata{
			Location: streamer.Info.Url,
		},
		PrivacyLevel: discord.GuildOnly,
		StartTime:    startsAtTs,
		EndTime:      &endsAtTs,
	})
	if err != nil {
		// event is somehow in the past, which is why we omit creating the scheduled event again.
		if discordutils.IsStatus4XX(err) {
			log.Println("event is in the past, not creating scheduled event")
			return nil
		}
		return fmt.Errorf("failed to create scheduled event: %w", err)
	}

	err = q.UpdateMatchEventID(ctx, sqlc.UpdateMatchEventIDParams{
		ChannelID: param.ChannelID.String(),
		EventID:   event.ID.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to set match event ID %s for channel %s: %w", event.ID, param.ChannelID, err)
	}

	log.Printf("scheduled event in guild %s for channel %s: starts at %s, ends at %s",
		param.GuildID,
		param.ChannelID,
		startsAt.Local(),
		endsAt.Local(),
	)

	return nil
}
