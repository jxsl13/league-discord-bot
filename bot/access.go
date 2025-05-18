package bot

import (
	"context"
	"errors"
	"fmt"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/jxs13/league-discord-bot/internal/discordutils"
	"github.com/jxs13/league-discord-bot/sqlc"
)

const (
	READ  PermissionEnum = "READ"
	WRITE PermissionEnum = "WRITE"
	ADMIN PermissionEnum = "ADMIN"
)

var (
	ErrAccessForbidden = errors.New("access forbidden")
)

type PermissionEnum string

func (b *Bot) checkAccess(ctx context.Context, q *sqlc.Queries, e *discord.InteractionEvent, permission PermissionEnum, noGuild ...bool) error {

	withGuild := true
	if len(noGuild) > 0 {
		withGuild = !noGuild[0]
	}

	if withGuild {
		// first check if guild is enabled correctly
		err := b.checkGuildEnabled(ctx, q, e.GuildID)
		if err != nil {
			return err
		}
	}

	// check if admin or users have access
	ok, err := b.hasEventAccess(ctx, q, e, permission)
	if err != nil {
		return fmt.Errorf("%w, please contact the owner of the bot", err)
	}
	if !ok {
		return ErrAccessForbidden
	}

	return nil
}

func (b *Bot) hasEventAccess(ctx context.Context, q *sqlc.Queries, e *discord.InteractionEvent, permission PermissionEnum) (bool, error) {
	if e.Channel.SelfPermissions.Has(discord.PermissionAdministrator) {
		return true, nil
	}

	if permission == ADMIN {
		return e.Channel.SelfPermissions.Has(discord.PermissionAdministrator), nil
	}

	ok, err := b.hasAccess(ctx, q, e.GuildID, e.User, permission)
	if err != nil {
		return false, fmt.Errorf("error checking access: %w", err)
	}

	return ok, nil
}

func (b *Bot) hasAccess(ctx context.Context, q *sqlc.Queries, guildID discord.GuildID, user *discord.User, permission PermissionEnum) (bool, error) {

	ok, err := b.hasUserAccess(ctx, q, guildID, user, permission)
	if err != nil {
		return false, err
	}

	if ok {
		return true, nil
	}

	ok, err = b.hasRoleAccess(ctx, q, guildID, user, permission)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (b *Bot) hasUserAccess(ctx context.Context, q *sqlc.Queries, guildID discord.GuildID, user *discord.User, permission PermissionEnum) (bool, error) {
	if permission == "" || user == nil {
		return false, nil
	}

	i, err := q.HasUserAccess(ctx, sqlc.HasUserAccessParams{
		GuildID:    guildID.String(),
		UserID:     user.ID.String(),
		Permission: string(permission),
	})
	if err != nil {
		return false, err
	}

	return i == 1, nil
}

func (b *Bot) hasRoleAccess(ctx context.Context, q *sqlc.Queries, guildID discord.GuildID, user *discord.User, permission PermissionEnum) (bool, error) {
	if permission == "" || user == nil {
		return false, nil
	}

	member, err := b.state.Member(guildID, user.ID)
	if err != nil {
		return false, err
	}

	if len(member.RoleIDs) == 0 {
		return false, nil
	}

	memberRoleIDs := make([]string, 0, len(member.RoleIDs))
	for _, roleID := range member.RoleIDs {
		memberRoleIDs = append(memberRoleIDs, roleID.String())
	}

	i, err := q.HasRoleAccess(ctx, sqlc.HasRoleAccessParams{
		GuildID:    guildID.String(),
		Permission: string(permission),
		RoleIds:    memberRoleIDs,
	})
	if err != nil {
		return false, err
	}

	return i == 1, nil
}

func (b *Bot) checkGuildEnabled(ctx context.Context, q *sqlc.Queries, guildID discord.GuildID) error {
	enabled, err := q.IsGuildEnabled(ctx, guildID.String())
	if err != nil {
		return fmt.Errorf("failed to check if the guild is properly enabled: %w, please contact the bot owner", err)
	}

	if !int64ToBool(enabled) {
		return errors.New("bot is disabled until it has sufficient permissions: you can reenable the bot by using the `configure` slash command")
	}

	return nil
}

func (b *Bot) checkIsGuildChannel(event *discord.InteractionEvent, channelID discord.ChannelID) error {
	c, err := b.state.Channel(channelID)
	if err != nil {
		if discordutils.IsStatus4XX(err) {
			return errors.New("channel not found or bot not in channel")
		}
		return err
	}

	if c.GuildID != event.GuildID {
		return errors.New("channel must be in your server")
	}
	return nil
}
