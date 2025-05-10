package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/discordutils"
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

func (b *Bot) asyncGiveChannelAccess() (d time.Duration, err error) {
	defer func() {
		if err != nil {
			log.Printf("error in channel access routine: %v", err)
		}
	}()
	q, err := b.Queries(b.ctx)
	if err != nil {
		return 0, err
	}
	defer q.Close()

	nextChan, err := q.NextAccessibleChannel(b.ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// no channels to give access to
			return 0, nil
		}
		return 0, fmt.Errorf("error getting next accessible channel: %w", err)
	}

	var (
		channelID    = nextChan.ChannelID
		accessibleAt = time.Unix(nextChan.ChannelAccessibleAt, 0)
	)

	cid, err := discordutils.ParseChannelID(channelID)
	if err != nil {
		return 0, err
	}

	if time.Now().Before(accessibleAt) {
		// give the routine a hint that we might need to wait less than the usual interval.
		// or the usual interval in case the returned value it too large
		return time.Until(accessibleAt), nil
	}

	teamRoleIDs, err := b.listMatchTeamRoleIDs(b.ctx, q, cid)
	if err != nil {
		return 0, fmt.Errorf("error getting match team role IDs: %w", err)
	}

	modUserIDs, err := b.listMatchModeratorUserIDs(b.ctx, q, cid)
	if err != nil {
		return 0, fmt.Errorf("error getting match moderator role IDs: %w", err)
	}

	streamerUserIDs, err := b.listMatchStreamerUserIDs(b.ctx, q, cid)
	if err != nil {
		return 0, err
	}

	c, err := b.state.Channel(cid)
	if err != nil {
		return 0, fmt.Errorf("error getting channel: %w", err)
	}

	oldOverwrites := slices.Clone(c.Overwrites)
	overwrites := make([]discord.Overwrite, 0, len(oldOverwrites)+len(teamRoleIDs)+len(modUserIDs)+len(streamerUserIDs))
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

	for _, s := range streamerUserIDs {
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
		return 0, fmt.Errorf("error modifying channel: %w", err)
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
		b.ctx,
		sqlc.UpdateMatchChannelAccessibilityParams{
			ChannelID:         channelID,
			ChannelAccessible: 1,
		})

	// TODO: might need to check when the next channel is accessible
	// and return that time, in case it is shorter than the usual interval
	return 0, nil
}
