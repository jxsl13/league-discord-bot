package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
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

		for _, ac := range accessible {
			err = b.giveSingleChannelAccess(ctx, q, ac)
			if err != nil {
				return fmt.Errorf("error giving channel access: %w", err)
			}
			log.Printf("granted access to channel %s", ac.ChannelID)
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

func (b *Bot) giveSingleChannelAccess(ctx context.Context, q *sqlc.Queries, a sqlc.ListNowAccessibleChannelsRow) (err error) {
	var (
		channelID    = a.ChannelID
		accessibleAt = time.Unix(a.ChannelAccessibleAt, 0)
	)

	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to give channel access for channel %s: %w", channelID, err)
		}
	}()

	// small sanity check
	if time.Now().Before(accessibleAt) {
		// give the routine a hint that we might need to wait less than the usual interval.
		// or the usual interval in case the returned value it too large
		return nil
	}

	cid, err := parse.ChannelID(channelID)
	if err != nil {
		return err
	}

	c, err := b.state.Channel(cid)
	if err != nil {
		return fmt.Errorf("error getting channel: %w", err)
	}

	teamRoleIDs, err := b.listMatchTeamRoleIDs(ctx, q, cid)
	if err != nil {
		return fmt.Errorf("error getting match team role IDs: %w", err)
	}

	modUserIDs, err := b.listMatchModeratorUserIDs(ctx, q, cid)
	if err != nil {
		return fmt.Errorf("error getting match moderator role IDs: %w", err)
	}

	streamerUserIDs, err := b.listMatchStreamerUserIDs(ctx, q, cid)
	if err != nil {
		return err
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
		return fmt.Errorf("error modifying channel: %w", err)
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
		return fmt.Errorf("error updating match channel accessibility: %w", err)
	}
	return nil
}
